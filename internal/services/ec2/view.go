package ec2

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

// View implements the TUI view for EC2 instances.
type View struct {
	*base.TableView
}

// NewView creates a new EC2 view.
func NewView() *View {
	columnDefs := []base.ColumnDef{
		{Title: "ID", MinWidth: 12, MaxWidth: 22, Weight: 1.0, Priority: 0},
		{Title: "Name", MinWidth: 10, MaxWidth: 30, Weight: 2.0, Priority: 1},
		{Title: "Type", MinWidth: 10, MaxWidth: 15, Weight: 0.5, Priority: 2},
		{Title: "State", MinWidth: 10, MaxWidth: 14, Weight: 0.5, Priority: 0},
		{Title: "Public IP", MinWidth: 12, MaxWidth: 16, Weight: 0.5, Priority: 3},
		{Title: "Private IP", MinWidth: 12, MaxWidth: 16, Weight: 0.5, Priority: 4},
		{Title: "AZ", MinWidth: 10, MaxWidth: 16, Weight: 0.5, Priority: 5},
	}

	return &View{
		TableView: base.NewTableView("EC2", "1", "ec2", columnDefs),
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
	return v.loadInstances()
}

// Update handles messages and updates the view state.
func (v *View) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			if row := v.GetSelectedResource(); row != nil {
				v.Message = fmt.Sprintf("Starting %s...", row.ID)
				return v, v.executeAction("start", row.ID)
			}
		case "t":
			if row := v.GetSelectedResource(); row != nil {
				v.Message = fmt.Sprintf("Stopping %s...", row.ID)
				return v, v.executeAction("stop", row.ID)
			}
		case "b":
			if row := v.GetSelectedResource(); row != nil {
				v.Message = fmt.Sprintf("Rebooting %s...", row.ID)
				return v, v.executeAction("reboot", row.ID)
			}
		case "enter":
			if row := v.GetSelectedResource(); row != nil {
				v.Message = fmt.Sprintf("Selected: %s (%s)", row.Name, row.ID)
			}
		}

	case ec2LoadedMsg:
		v.SetLoading(false)
		if msg.err != nil {
			v.SetError(msg.err)
			v.Message = fmt.Sprintf("Error: %v", msg.err)
		} else {
			v.SetError(nil)
			v.Resources = msg.resources
			v.updateTable()
			v.Message = fmt.Sprintf("Loaded %d instances", len(msg.resources))
		}

	case base.ActionResultMsg:
		if msg.Error != nil {
			v.Message = fmt.Sprintf("Action failed: %v", msg.Error)
		} else if msg.Result != nil {
			v.Message = msg.Result.Message
		}
		cmds = append(cmds, v.loadInstances())

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

	// Lines 3-N: Table or loading/error state
	if v.IsLoading() && len(v.Resources) == 0 {
		lines = append(lines, v.Styles.Muted.Render("Loading EC2 instances..."))
	} else if err := v.Error(); err != nil {
		lines = append(lines, v.Styles.Error.Render(fmt.Sprintf("Error: %v", err)))
	} else {
		lines = append(lines, v.TableViewString())
	}

	// Message line (or blank)
	if v.Message != "" {
		lines = append(lines, v.Styles.Info.Render(v.Message))
	} else {
		lines = append(lines, "")
	}

	// Help line
	lines = append(lines, v.Styles.Help.Render("[s]tart  [t]stop  [b]reboot  [↑/↓]navigate  [r]efresh"))

	return strings.Join(lines, "\n")
}

// =============================================================================
// core.View Interface Implementation
// =============================================================================

// Refresh reloads the instance data.
func (v *View) Refresh() tea.Cmd {
	return v.loadInstances()
}

// =============================================================================
// Internal Methods
// =============================================================================

type ec2LoadedMsg struct {
	resources []core.Resource
	err       error
}

func (v *View) loadInstances() tea.Cmd {
	v.SetLoading(true)
	return func() tea.Msg {
		service := v.Service()
		if service == nil {
			return ec2LoadedMsg{err: fmt.Errorf("service not initialized")}
		}

		lister, ok := service.(core.ResourceLister)
		if !ok {
			return ec2LoadedMsg{err: fmt.Errorf("service does not support listing")}
		}

		resources, err := lister.List(context.Background(), core.ListOptions{})
		return ec2LoadedMsg{resources: resources, err: err}
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
		rows[i] = table.Row{
			r.ID,
			base.TruncateString(r.Name, 30),
			r.GetMetadataString("instance_type"),
			base.FormatState(r.State),
			r.GetMetadataString("public_ip"),
			r.GetMetadataString("private_ip"),
			r.GetMetadataString("availability_zone"),
		}
	}
	v.SetRows(rows)
}

func (v *View) renderSummary() string {
	total := len(v.Resources)
	running := 0
	stopped := 0

	for _, r := range v.Resources {
		switch r.State {
		case core.StateRunning:
			running++
		case core.StateStopped:
			stopped++
		}
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		v.Styles.Title.Render("EC2 Instances"),
		"  ",
		v.Styles.Muted.Render(fmt.Sprintf("Total: %d", total)),
		"  ",
		v.Styles.Success.Render(fmt.Sprintf("Running: %d", running)),
		"  ",
		v.Styles.Error.Render(fmt.Sprintf("Stopped: %d", stopped)),
	)
}

// =============================================================================
// View Factory
// =============================================================================

// ViewFactory creates EC2 views.
type ViewFactory struct{}

// NewViewFactory creates a new EC2 view factory.
func NewViewFactory() *ViewFactory {
	return &ViewFactory{}
}

// Create creates a new EC2 view for the given service.
func (f *ViewFactory) Create(service core.AWSService) (core.View, error) {
	view := NewView()
	view.SetService(service)
	return view, nil
}

// ServiceName returns the service name this factory creates views for.
func (f *ViewFactory) ServiceName() string {
	return "ec2"
}

// =============================================================================
// Interface Assertions
// =============================================================================

var (
	_ tea.Model        = (*View)(nil)
	_ core.View        = (*View)(nil)
	_ core.ViewFactory = (*ViewFactory)(nil)
)
