package s3

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/keanuharrell/a9s/internal/core"
	"github.com/keanuharrell/a9s/internal/services/base"
)

// =============================================================================
// View Implementation
// =============================================================================

// View implements the TUI view for S3 buckets.
type View struct {
	*base.TableView
	enriching  bool
	analyzed   int
	cancelFunc context.CancelFunc
	cache      map[string]*core.Resource
}

// NewView creates a new S3 view.
func NewView() *View {
	columnDefs := []base.ColumnDef{
		{Title: "Name", MinWidth: 20, MaxWidth: 50, Weight: 2.0, Priority: 0},
		{Title: "Region", MinWidth: 10, MaxWidth: 16, Weight: 0.5, Priority: 1},
		{Title: "Created", MinWidth: 10, MaxWidth: 12, Weight: 0.3, Priority: 3},
		{Title: "Public", MinWidth: 8, MaxWidth: 12, Weight: 0.3, Priority: 0},
		{Title: "Tagged", MinWidth: 8, MaxWidth: 12, Weight: 0.3, Priority: 2},
		{Title: "Cleanup", MinWidth: 8, MaxWidth: 12, Weight: 0.3, Priority: 2},
	}

	return &View{
		TableView: base.NewTableView("S3", "3", "s3", columnDefs),
		cache:     make(map[string]*core.Resource),
	}
}

// =============================================================================
// tea.Model Interface Implementation
// =============================================================================

func (v *View) Init() tea.Cmd {
	// Don't reload if we already have data or are currently loading
	if len(v.Resources) > 0 || v.IsLoading() {
		return nil
	}
	return v.loadBuckets()
}

func (v *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "R":
			v.Message = "Full refresh..."
			return v, v.hardRefresh()
		case "a":
			if row := v.GetSelectedResource(); row != nil {
				v.Message = fmt.Sprintf("Analyzing %s...", row.Name)
				return v, v.analyzeSelected()
			}
		case "d":
			if row := v.GetSelectedResource(); row != nil {
				v.Message = fmt.Sprintf("Press 'D' to confirm deletion of %s", row.Name)
			}
		case "D":
			if row := v.GetSelectedResource(); row != nil {
				v.Message = fmt.Sprintf("Deleting %s...", row.Name)
				return v, v.executeAction("delete", row.Name)
			}
		case "enter":
			if row := v.GetSelectedResource(); row != nil {
				v.Message = fmt.Sprintf("%s: %s", row.Name, row.GetMetadataString("size_human"))
			}
		}

	case s3LoadedMsg:
		v.SetLoading(false)
		if msg.err != nil {
			v.SetError(msg.err)
			v.Message = fmt.Sprintf("Error: %v", msg.err)
		} else {
			v.SetError(nil)
			if msg.hardRefresh {
				v.cache = make(map[string]*core.Resource)
				v.analyzed = 0
				v.Resources = msg.resources
				v.updateTable()
				v.Message = fmt.Sprintf("Loaded %d buckets, analyzing...", len(msg.resources))
				cmds = append(cmds, v.startEnrichment())
			} else {
				newCount := 0
				v.Resources = msg.resources
				for i := range v.Resources {
					if cached, ok := v.cache[v.Resources[i].Name]; ok {
						v.Resources[i] = *cached
					} else {
						newCount++
					}
				}
				v.updateTable()
				if newCount > 0 {
					v.Message = fmt.Sprintf("Found %d new buckets, analyzing...", newCount)
					cmds = append(cmds, v.startEnrichmentForNew())
				} else {
					v.Message = fmt.Sprintf("Refreshed %d buckets", len(msg.resources))
				}
			}
		}

	case s3ResourceEnrichedMsg:
		if msg.index >= 0 && msg.index < len(v.Resources) {
			v.Resources[msg.index] = msg.resource
			v.cache[msg.resource.Name] = &v.Resources[msg.index]
			v.analyzed++
			v.updateTableRow(msg.index)
			v.Message = fmt.Sprintf("Analyzing... %d/%d", v.analyzed, len(v.Resources))
			cmds = append(cmds, v.continueEnrichment())
		}

	case s3EnrichmentDoneMsg:
		v.enriching = false
		v.Message = fmt.Sprintf("Loaded %d buckets", len(v.Resources))

	case base.ActionResultMsg:
		if msg.Error != nil {
			v.Message = fmt.Sprintf("Action failed: %v", msg.Error)
		} else if msg.Result != nil {
			v.Message = msg.Result.Message
		}
		if msg.Action == "delete" {
			cmds = append(cmds, v.loadBuckets())
		}

	case tea.WindowSizeMsg:
		v.HandleWindowSize(msg)
	}

	cmds = append(cmds, v.UpdateTable(msg))
	return v, tea.Batch(cmds...)
}

func (v *View) View() string {
	var lines []string

	// Line 1: Summary
	lines = append(lines, v.renderSummary())
	// Line 2: Blank
	lines = append(lines, "")

	// Table or loading/error
	if v.IsLoading() && len(v.Resources) == 0 {
		lines = append(lines, v.Styles.Muted.Render("Loading S3 buckets..."))
	} else if err := v.Error(); err != nil {
		lines = append(lines, v.Styles.Error.Render(fmt.Sprintf("Error: %v", err)))
	} else {
		lines = append(lines, v.TableViewString())
	}

	// Message or blank
	if v.Message != "" {
		lines = append(lines, v.Styles.Info.Render(v.Message))
	} else {
		lines = append(lines, "")
	}

	// Help
	lines = append(lines, v.Styles.Help.Render("[a]nalyze  [d]elete  [r]efresh  [R]e-analyze  [↑/↓]nav"))
	return strings.Join(lines, "\n")
}

// =============================================================================
// core.View Interface Implementation
// =============================================================================

func (v *View) Refresh() tea.Cmd {
	return v.softRefresh()
}

// Reset clears all view data including cache.
func (v *View) Reset() {
	v.TableView.Reset()
	v.cache = make(map[string]*core.Resource)
	v.analyzed = 0
	v.enriching = false
	if v.cancelFunc != nil {
		v.cancelFunc()
		v.cancelFunc = nil
	}
}

func (v *View) softRefresh() tea.Cmd {
	v.SetLoading(true)
	return func() tea.Msg {
		service := v.Service()
		if service == nil {
			return s3LoadedMsg{err: fmt.Errorf("service not initialized"), hardRefresh: false}
		}
		lister, ok := service.(core.ResourceLister)
		if !ok {
			return s3LoadedMsg{err: fmt.Errorf("service does not support listing"), hardRefresh: false}
		}
		resources, err := lister.List(context.Background(), core.ListOptions{})
		return s3LoadedMsg{resources: resources, err: err, hardRefresh: false}
	}
}

func (v *View) hardRefresh() tea.Cmd {
	v.cache = make(map[string]*core.Resource)
	v.analyzed = 0
	return v.loadBuckets()
}

func (v *View) analyzeSelected() tea.Cmd {
	cursor := v.Cursor()
	if cursor < 0 || cursor >= len(v.Resources) {
		return nil
	}
	service := v.Service()
	if service == nil {
		return nil
	}
	s3Svc, ok := service.(*Service)
	if !ok {
		return nil
	}

	index := cursor
	return func() tea.Msg {
		resource := v.Resources[index]
		delete(v.cache, resource.Name)
		resource.Metadata["analyzed"] = false
		if err := s3Svc.EnrichResource(context.Background(), &resource); err == nil {
			return s3ResourceEnrichedMsg{index: index, resource: resource}
		}
		return s3EnrichmentDoneMsg{}
	}
}

// =============================================================================
// Internal Methods
// =============================================================================

type s3LoadedMsg struct {
	resources   []core.Resource
	err         error
	hardRefresh bool
}

type s3ResourceEnrichedMsg struct {
	index    int
	resource core.Resource
}

type s3EnrichmentDoneMsg struct{}

func (v *View) loadBuckets() tea.Cmd {
	if v.cancelFunc != nil {
		v.cancelFunc()
	}
	v.SetLoading(true)
	v.enriching = false

	return func() tea.Msg {
		service := v.Service()
		if service == nil {
			return s3LoadedMsg{err: fmt.Errorf("service not initialized"), hardRefresh: true}
		}
		lister, ok := service.(core.ResourceLister)
		if !ok {
			return s3LoadedMsg{err: fmt.Errorf("service does not support listing"), hardRefresh: true}
		}
		resources, err := lister.List(context.Background(), core.ListOptions{})
		return s3LoadedMsg{resources: resources, err: err, hardRefresh: true}
	}
}

func (v *View) startEnrichment() tea.Cmd {
	service := v.Service()
	if service == nil {
		return nil
	}
	s3Svc, ok := service.(*Service)
	if !ok {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	v.cancelFunc = cancel
	v.enriching = true

	return func() tea.Msg {
		for i := range v.Resources {
			select {
			case <-ctx.Done():
				return s3EnrichmentDoneMsg{}
			default:
				resource := v.Resources[i]
				if err := s3Svc.EnrichResource(ctx, &resource); err == nil {
					return s3ResourceEnrichedMsg{index: i, resource: resource}
				}
			}
		}
		return s3EnrichmentDoneMsg{}
	}
}

func (v *View) startEnrichmentForNew() tea.Cmd {
	service := v.Service()
	if service == nil {
		return nil
	}
	s3Svc, ok := service.(*Service)
	if !ok {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	v.cancelFunc = cancel
	v.enriching = true

	return func() tea.Msg {
		for i := range v.Resources {
			if _, inCache := v.cache[v.Resources[i].Name]; inCache {
				continue
			}
			if analyzed, ok := v.Resources[i].Metadata["analyzed"].(bool); ok && analyzed {
				continue
			}
			select {
			case <-ctx.Done():
				return s3EnrichmentDoneMsg{}
			default:
				resource := v.Resources[i]
				if err := s3Svc.EnrichResource(ctx, &resource); err == nil {
					return s3ResourceEnrichedMsg{index: i, resource: resource}
				}
			}
		}
		return s3EnrichmentDoneMsg{}
	}
}

func (v *View) continueEnrichment() tea.Cmd {
	service := v.Service()
	if service == nil || !v.enriching {
		return nil
	}
	s3Svc, ok := service.(*Service)
	if !ok {
		return nil
	}

	nextIndex := -1
	for i, r := range v.Resources {
		if analyzed, ok := r.Metadata["analyzed"].(bool); !ok || !analyzed {
			nextIndex = i
			break
		}
	}

	if nextIndex == -1 {
		v.enriching = false
		return func() tea.Msg { return s3EnrichmentDoneMsg{} }
	}

	ctx := context.Background()
	if v.cancelFunc != nil {
		ctx, v.cancelFunc = context.WithCancel(context.Background())
	}

	return func() tea.Msg {
		resource := v.Resources[nextIndex]
		if err := s3Svc.EnrichResource(ctx, &resource); err == nil {
			return s3ResourceEnrichedMsg{index: nextIndex, resource: resource}
		}
		return s3EnrichmentDoneMsg{}
	}
}

func (v *View) executeAction(action, resourceID string) tea.Cmd {
	return func() tea.Msg {
		service := v.Service()
		if service == nil {
			return base.ActionResultMsg{Error: fmt.Errorf("service not initialized")}
		}
		executor, ok := service.(core.ActionExecutor)
		if !ok {
			return base.ActionResultMsg{Error: fmt.Errorf("service does not support actions")}
		}
		params := map[string]any{}
		if action == "delete" {
			params["confirm"] = true
		}
		result, err := executor.Execute(context.Background(), action, resourceID, params)
		return base.ActionResultMsg{Action: action, Result: result, Error: err}
	}
}

func (v *View) updateTable() {
	rows := make([]table.Row, len(v.Resources))
	for i := range v.Resources {
		rows[i] = v.buildRow(i)
	}
	v.SetRows(rows)
}

func (v *View) updateTableRow(index int) {
	if index < 0 || index >= len(v.Resources) {
		return
	}
	rows := v.Table.Rows()
	if index < len(rows) {
		rows[index] = v.buildRow(index)
		v.SetRows(rows)
	}
}

func (v *View) buildRow(index int) table.Row {
	r := v.Resources[index]

	isPublic, _ := r.Metadata["is_public"].(bool)
	hasTags, _ := r.Metadata["has_tags"].(bool)
	shouldCleanup, _ := r.Metadata["should_cleanup"].(bool)
	createdDate, _ := r.Metadata["created_date"].(string)
	analyzed, _ := r.Metadata["analyzed"].(bool)

	publicIcon, taggedIcon, cleanupIcon := "...", "...", "..."
	if analyzed {
		publicIcon = "🟢 No"
		if isPublic {
			publicIcon = "🔴 Yes"
		}
		taggedIcon = "🔴 No"
		if hasTags {
			taggedIcon = "🟢 Yes"
		}
		cleanupIcon = "🟢 No"
		if shouldCleanup {
			cleanupIcon = "🟡 Yes"
		}
	}

	return table.Row{
		base.TruncateString(r.Name, 50),
		r.Region,
		createdDate,
		publicIcon,
		taggedIcon,
		cleanupIcon,
	}
}

func (v *View) renderSummary() string {
	total := len(v.Resources)
	public, cleanup, analyzed := 0, 0, 0

	for _, r := range v.Resources {
		if isAnalyzed, ok := r.Metadata["analyzed"].(bool); ok && isAnalyzed {
			analyzed++
		}
		if isPublic, ok := r.Metadata["is_public"].(bool); ok && isPublic {
			public++
		}
		if shouldCleanup, ok := r.Metadata["should_cleanup"].(bool); ok && shouldCleanup {
			cleanup++
		}
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		v.Styles.Title.Render("S3 Buckets"),
		"  ",
		v.Styles.Muted.Render(fmt.Sprintf("Analyzed: %d/%d", analyzed, total)),
		"  ",
		v.Styles.Error.Render(fmt.Sprintf("Public: %d", public)),
		"  ",
		v.Styles.Warning.Render(fmt.Sprintf("Cleanup: %d", cleanup)),
	)
}

// =============================================================================
// View Factory
// =============================================================================

type ViewFactory struct{}

func NewViewFactory() *ViewFactory { return &ViewFactory{} }

func (f *ViewFactory) Create(service core.AWSService) (core.View, error) {
	view := NewView()
	view.SetService(service)
	return view, nil
}

func (f *ViewFactory) ServiceName() string { return "s3" }

var (
	_ tea.Model        = (*View)(nil)
	_ core.View        = (*View)(nil)
	_ core.ViewFactory = (*ViewFactory)(nil)
)
