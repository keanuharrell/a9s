package lambda

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

type View struct {
	*base.TableView
}

func NewView() *View {
	columnDefs := []base.ColumnDef{
		{Title: "Name", MinWidth: 15, MaxWidth: 40, Weight: 2.0, Priority: 0},
		{Title: "Runtime", MinWidth: 10, MaxWidth: 18, Weight: 0.5, Priority: 1},
		{Title: "Memory", MinWidth: 8, MaxWidth: 12, Weight: 0.3, Priority: 2},
		{Title: "Timeout", MinWidth: 8, MaxWidth: 12, Weight: 0.3, Priority: 3},
		{Title: "Last Modified", MinWidth: 12, MaxWidth: 20, Weight: 0.5, Priority: 4},
	}

	return &View{
		TableView: base.NewTableView("Lambda", "4", "lambda", columnDefs),
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
	return v.loadFunctions()
}

func (v *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "i":
			if row := v.GetSelectedResource(); row != nil {
				v.Message = fmt.Sprintf("Invoking %s...", row.Name)
				return v, v.executeAction("invoke", row.Name)
			}
		case "c":
			if row := v.GetSelectedResource(); row != nil {
				v.Message = fmt.Sprintf("Loading config for %s...", row.Name)
				return v, v.executeAction("view_config", row.Name)
			}
		case "enter":
			if row := v.GetSelectedResource(); row != nil {
				v.Message = fmt.Sprintf("%s: %s", row.Name, row.GetMetadataString("runtime"))
			}
		}

	case lambdaLoadedMsg:
		v.SetLoading(false)
		if msg.err != nil {
			v.SetError(msg.err)
			v.Message = fmt.Sprintf("Error: %v", msg.err)
		} else {
			v.SetError(nil)
			v.Resources = msg.resources
			v.updateTable()
			v.Message = fmt.Sprintf("Loaded %d functions", len(msg.resources))
		}

	case base.ActionResultMsg:
		if msg.Error != nil {
			v.Message = fmt.Sprintf("Action failed: %v", msg.Error)
		} else if msg.Result != nil {
			v.Message = msg.Result.Message
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
		lines = append(lines, v.Styles.Muted.Render("Loading Lambda functions..."))
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
	lines = append(lines, v.Styles.Help.Render("[i]nvoke  [c]onfig  [↑/↓]navigate  [r]efresh"))
	return strings.Join(lines, "\n")
}

// =============================================================================
// core.View Interface Implementation
// =============================================================================

func (v *View) Refresh() tea.Cmd {
	return v.loadFunctions()
}

// =============================================================================
// Internal Methods
// =============================================================================

type lambdaLoadedMsg struct {
	resources []core.Resource
	err       error
}

func (v *View) loadFunctions() tea.Cmd {
	v.SetLoading(true)
	return func() tea.Msg {
		service := v.Service()
		if service == nil {
			return lambdaLoadedMsg{err: fmt.Errorf("service not initialized")}
		}
		lister, ok := service.(core.ResourceLister)
		if !ok {
			return lambdaLoadedMsg{err: fmt.Errorf("service does not support listing")}
		}
		resources, err := lister.List(context.Background(), core.ListOptions{})
		return lambdaLoadedMsg{resources: resources, err: err}
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
	for i, r := range v.Resources {
		runtime := r.GetMetadataString("runtime")

		memoryMB := "0 MB"
		if m, ok := r.Metadata["memory_mb"].(int32); ok {
			memoryMB = fmt.Sprintf("%d MB", m)
		}

		timeoutSec := "0 s"
		if t, ok := r.Metadata["timeout_sec"].(int32); ok {
			timeoutSec = fmt.Sprintf("%d s", t)
		}

		lastModified := r.GetMetadataString("last_modified")
		if len(lastModified) > 19 {
			lastModified = lastModified[:19]
		}

		rows[i] = table.Row{
			base.TruncateString(r.Name, 40),
			runtime,
			memoryMB,
			timeoutSec,
			lastModified,
		}
	}
	v.SetRows(rows)
}

func (v *View) renderSummary() string {
	total := len(v.Resources)
	runtimes := make(map[string]int)
	for _, r := range v.Resources {
		runtime := r.GetMetadataString("runtime")
		runtimes[runtime]++
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		v.Styles.Title.Render("Lambda Functions"),
		"  ",
		v.Styles.Muted.Render(fmt.Sprintf("Total: %d", total)),
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

func (f *ViewFactory) ServiceName() string { return "lambda" }

var (
	_ tea.Model        = (*View)(nil)
	_ core.View        = (*View)(nil)
	_ core.ViewFactory = (*ViewFactory)(nil)
)
