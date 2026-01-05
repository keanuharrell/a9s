package iam

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

// View implements the TUI view for IAM roles.
type View struct {
	*base.TableView
	enriching  bool
	analyzed   int
	cancelFunc context.CancelFunc
	cache      map[string]*core.Resource
}

// NewView creates a new IAM view.
func NewView() *View {
	columnDefs := []base.ColumnDef{
		{Title: "Name", MinWidth: 15, MaxWidth: 40, Weight: 2.0, Priority: 0},
		{Title: "Created", MinWidth: 10, MaxWidth: 12, Weight: 0.3, Priority: 3},
		{Title: "Policies", MinWidth: 8, MaxWidth: 10, Weight: 0.2, Priority: 1},
		{Title: "Risk", MinWidth: 8, MaxWidth: 12, Weight: 0.2, Priority: 0},
		{Title: "Risk Reason", MinWidth: 15, MaxWidth: 50, Weight: 2.0, Priority: 2},
	}

	return &View{
		TableView: base.NewTableView("IAM", "2", "iam", columnDefs),
		cache:     make(map[string]*core.Resource),
	}
}

// =============================================================================
// tea.Model Interface Implementation
// =============================================================================

// Init initializes the view and starts loading data.
func (v *View) Init() tea.Cmd {
	// Don't reload if we already have data or are currently loading
	if len(v.Resources) > 0 || v.IsLoading() {
		return nil
	}
	return v.loadRoles()
}

// Update handles messages and updates the view state.
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
				v.Message = fmt.Sprintf("Auditing %s...", row.Name)
				return v, v.analyzeSelected()
			}
		case "p":
			if row := v.GetSelectedResource(); row != nil {
				v.Message = fmt.Sprintf("Loading policies for %s...", row.Name)
				return v, v.executeAction("view_policies", row.Name)
			}
		case "enter":
			if row := v.GetSelectedResource(); row != nil {
				policies, _ := row.Metadata["policies"].([]string)
				v.Message = fmt.Sprintf("%s: %d policies", row.Name, len(policies))
			}
		}

	case iamLoadedMsg:
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
				v.Message = fmt.Sprintf("Loaded %d roles, analyzing...", len(msg.resources))
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
					v.Message = fmt.Sprintf("Found %d new roles, analyzing...", newCount)
					cmds = append(cmds, v.startEnrichmentForNew())
				} else {
					v.Message = fmt.Sprintf("Refreshed %d roles", len(msg.resources))
				}
			}
		}

	case iamResourceEnrichedMsg:
		if msg.index >= 0 && msg.index < len(v.Resources) {
			v.Resources[msg.index] = msg.resource
			v.cache[msg.resource.Name] = &v.Resources[msg.index]
			v.analyzed++
			v.updateTableRow(msg.index)
			v.Message = fmt.Sprintf("Analyzing... %d/%d", v.analyzed, len(v.Resources))
			cmds = append(cmds, v.continueEnrichment())
		}

	case iamEnrichmentDoneMsg:
		v.enriching = false
		v.Message = fmt.Sprintf("Loaded %d roles", len(v.Resources))

	case base.ActionResultMsg:
		if msg.Error != nil {
			v.Message = fmt.Sprintf("Action failed: %v", msg.Error)
		} else if msg.Result != nil {
			if data, ok := msg.Result.Data.(map[string]any); ok {
				if policies, ok := data["policies"].([]string); ok {
					v.Message = fmt.Sprintf("Policies: %s", strings.Join(policies, ", "))
				} else {
					v.Message = msg.Result.Message
				}
			} else {
				v.Message = msg.Result.Message
			}
		}

	case tea.WindowSizeMsg:
		v.HandleWindowSize(msg)
	}

	cmds = append(cmds, v.UpdateTable(msg))
	return v, tea.Batch(cmds...)
}

// View renders the view.
func (v *View) View() string {
	var lines []string

	// Line 1: Summary
	lines = append(lines, v.renderSummary())
	// Line 2: Blank
	lines = append(lines, "")

	// Table or loading/error
	if v.IsLoading() && len(v.Resources) == 0 {
		lines = append(lines, v.Styles.Muted.Render("Loading IAM roles..."))
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
	lines = append(lines, v.Styles.Help.Render("[a]udit  [p]olicies  [r]efresh  [R]e-analyze  [↑/↓]nav"))
	return strings.Join(lines, "\n")
}

// =============================================================================
// core.View Interface Implementation
// =============================================================================

// Refresh does a soft refresh.
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
			return iamLoadedMsg{err: fmt.Errorf("service not initialized"), hardRefresh: false}
		}
		lister, ok := service.(core.ResourceLister)
		if !ok {
			return iamLoadedMsg{err: fmt.Errorf("service does not support listing"), hardRefresh: false}
		}
		resources, err := lister.List(context.Background(), core.ListOptions{})
		return iamLoadedMsg{resources: resources, err: err, hardRefresh: false}
	}
}

func (v *View) hardRefresh() tea.Cmd {
	v.cache = make(map[string]*core.Resource)
	v.analyzed = 0
	return v.loadRoles()
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
	iamSvc, ok := service.(*Service)
	if !ok {
		return nil
	}

	index := cursor
	return func() tea.Msg {
		resource := v.Resources[index]
		delete(v.cache, resource.Name)
		resource.Metadata["analyzed"] = false
		if err := iamSvc.EnrichResource(context.Background(), &resource); err == nil {
			return iamResourceEnrichedMsg{index: index, resource: resource}
		}
		return iamEnrichmentDoneMsg{}
	}
}

// =============================================================================
// Internal Methods
// =============================================================================

type iamLoadedMsg struct {
	resources   []core.Resource
	err         error
	hardRefresh bool
}

type iamResourceEnrichedMsg struct {
	index    int
	resource core.Resource
}

type iamEnrichmentDoneMsg struct{}

func (v *View) loadRoles() tea.Cmd {
	if v.cancelFunc != nil {
		v.cancelFunc()
	}
	v.SetLoading(true)
	v.enriching = false

	return func() tea.Msg {
		service := v.Service()
		if service == nil {
			return iamLoadedMsg{err: fmt.Errorf("service not initialized"), hardRefresh: true}
		}
		lister, ok := service.(core.ResourceLister)
		if !ok {
			return iamLoadedMsg{err: fmt.Errorf("service does not support listing"), hardRefresh: true}
		}
		resources, err := lister.List(context.Background(), core.ListOptions{})
		return iamLoadedMsg{resources: resources, err: err, hardRefresh: true}
	}
}

func (v *View) startEnrichment() tea.Cmd {
	service := v.Service()
	if service == nil {
		return nil
	}
	iamSvc, ok := service.(*Service)
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
				return iamEnrichmentDoneMsg{}
			default:
				resource := v.Resources[i]
				if err := iamSvc.EnrichResource(ctx, &resource); err == nil {
					return iamResourceEnrichedMsg{index: i, resource: resource}
				}
			}
		}
		return iamEnrichmentDoneMsg{}
	}
}

func (v *View) startEnrichmentForNew() tea.Cmd {
	service := v.Service()
	if service == nil {
		return nil
	}
	iamSvc, ok := service.(*Service)
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
				return iamEnrichmentDoneMsg{}
			default:
				resource := v.Resources[i]
				if err := iamSvc.EnrichResource(ctx, &resource); err == nil {
					return iamResourceEnrichedMsg{index: i, resource: resource}
				}
			}
		}
		return iamEnrichmentDoneMsg{}
	}
}

func (v *View) continueEnrichment() tea.Cmd {
	service := v.Service()
	if service == nil || !v.enriching {
		return nil
	}
	iamSvc, ok := service.(*Service)
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
		return func() tea.Msg { return iamEnrichmentDoneMsg{} }
	}

	ctx := context.Background()
	if v.cancelFunc != nil {
		ctx, v.cancelFunc = context.WithCancel(context.Background())
	}

	return func() tea.Msg {
		resource := v.Resources[nextIndex]
		if err := iamSvc.EnrichResource(ctx, &resource); err == nil {
			return iamResourceEnrichedMsg{index: nextIndex, resource: resource}
		}
		return iamEnrichmentDoneMsg{}
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
		result, err := executor.Execute(context.Background(), action, resourceID, nil)
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

	policyCount := 0
	if count, ok := r.Metadata["policy_count"].(int); ok {
		policyCount = count
	}

	riskLevel := "Low"
	riskIcon := "🟢"
	if isHighRisk, ok := r.Metadata["is_high_risk"].(bool); ok && isHighRisk {
		riskLevel = "HIGH"
		riskIcon = "🔴"
	}

	riskReason := ""
	if reason, ok := r.Metadata["risk_reason"].(string); ok {
		riskReason = reason
	}

	createDate := ""
	if date, ok := r.Metadata["create_date"].(string); ok {
		createDate = date
	}

	analyzed := false
	if a, ok := r.Metadata["analyzed"].(bool); ok {
		analyzed = a
	}

	policyStr := "..."
	riskStr := "..."
	if analyzed {
		policyStr = fmt.Sprintf("%d", policyCount)
		riskStr = riskIcon + " " + riskLevel
	}

	return table.Row{
		base.TruncateString(r.Name, 40),
		createDate,
		policyStr,
		riskStr,
		base.TruncateString(riskReason, 50),
	}
}

func (v *View) renderSummary() string {
	total := len(v.Resources)
	highRisk := 0
	for _, r := range v.Resources {
		if isHighRisk, ok := r.Metadata["is_high_risk"].(bool); ok && isHighRisk {
			highRisk++
		}
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		v.Styles.Title.Render("IAM Roles"),
		"  ",
		v.Styles.Muted.Render(fmt.Sprintf("Total: %d", total)),
		"  ",
		v.Styles.Error.Render(fmt.Sprintf("High Risk: %d", highRisk)),
	)
}

// =============================================================================
// View Factory
// =============================================================================

type ViewFactory struct{}

func NewViewFactory() *ViewFactory {
	return &ViewFactory{}
}

func (f *ViewFactory) Create(service core.AWSService) (core.View, error) {
	view := NewView()
	view.SetService(service)
	return view, nil
}

func (f *ViewFactory) ServiceName() string {
	return "iam"
}

var (
	_ tea.Model        = (*View)(nil)
	_ core.View        = (*View)(nil)
	_ core.ViewFactory = (*ViewFactory)(nil)
)
