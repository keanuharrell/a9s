// Package base provides base implementations for service views.
package base

import (
	"context"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/keanuharrell/a9s/internal/core"
)

// =============================================================================
// Base View
// =============================================================================

// View provides common functionality for all service views.
type View struct {
	name        string
	shortcut    string
	serviceName string
	service     core.AWSService
	width       int
	height      int
	loading     bool
	err         error
}

// NewView creates a new base view.
func NewView(name, shortcut, serviceName string) *View {
	return &View{
		name:        name,
		shortcut:    shortcut,
		serviceName: serviceName,
	}
}

// Name returns the view name.
func (v *View) Name() string {
	return v.name
}

// Shortcut returns the keyboard shortcut.
func (v *View) Shortcut() string {
	return v.shortcut
}

// ServiceName returns the associated service name.
func (v *View) ServiceName() string {
	return v.serviceName
}

// SetService sets the AWS service.
func (v *View) SetService(service core.AWSService) {
	v.service = service
}

// Service returns the AWS service.
func (v *View) Service() core.AWSService {
	return v.service
}

// SetDimensions sets the view dimensions.
func (v *View) SetDimensions(width, height int) {
	v.width = width
	v.height = height
}

// Width returns the view width.
func (v *View) Width() int {
	return v.width
}

// Height returns the view height.
func (v *View) Height() int {
	return v.height
}

// SetLoading sets the loading state.
func (v *View) SetLoading(loading bool) {
	v.loading = loading
}

// IsLoading returns the loading state.
func (v *View) IsLoading() bool {
	return v.loading
}

// SetError sets the error state.
func (v *View) SetError(err error) {
	v.err = err
}

// Error returns the error state.
func (v *View) Error() error {
	return v.err
}

// =============================================================================
// Common Styles
// =============================================================================

// Styles holds common styling for views.
type Styles struct {
	Title     lipgloss.Style
	Subtitle  lipgloss.Style
	Info      lipgloss.Style
	Error     lipgloss.Style
	Success   lipgloss.Style
	Warning   lipgloss.Style
	Muted     lipgloss.Style
	Help      lipgloss.Style
	StatusBar lipgloss.Style
	Table     table.Styles
}

// DefaultStyles returns the default view styles.
func DefaultStyles() Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")),

		Subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),

		Info: lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")),

		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")),

		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")),

		Warning: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")),

		Muted: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),

		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),

		StatusBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Background(lipgloss.Color("236")).
			Padding(0, 1),

		Table: DefaultTableStyles(),
	}
}

// DefaultTableStyles returns default table styling.
func DefaultTableStyles() table.Styles {
	s := table.DefaultStyles()

	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("229"))

	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	s.Cell = s.Cell.
		Foreground(lipgloss.Color("252"))

	return s
}

// =============================================================================
// Common Messages
// =============================================================================

// LoadingMsg indicates data is being loaded.
type LoadingMsg struct {
	ViewName string
}

// LoadedMsg indicates data has been loaded.
type LoadedMsg struct {
	ViewName  string
	Resources []core.Resource
	Error     error
}

// ActionMsg indicates an action should be executed.
type ActionMsg struct {
	Action     string
	ResourceID string
	Params     map[string]any
}

// ActionResultMsg indicates an action has completed.
type ActionResultMsg struct {
	Action string
	Result *core.ActionResult
	Error  error
}

// RefreshMsg triggers a refresh of the current view.
type RefreshMsg struct{}

// =============================================================================
// Common Commands
// =============================================================================

// LoadResourcesCmd creates a command to load resources.
func LoadResourcesCmd(viewName string, lister core.ResourceLister) tea.Cmd {
	return func() tea.Msg {
		resources, err := lister.List(context.Background(), core.ListOptions{})
		return LoadedMsg{
			ViewName:  viewName,
			Resources: resources,
			Error:     err,
		}
	}
}

// ExecuteActionCmd creates a command to execute an action.
func ExecuteActionCmd(executor core.ActionExecutor, action, resourceID string, params map[string]any) tea.Cmd {
	return func() tea.Msg {
		result, err := executor.Execute(context.Background(), action, resourceID, params)
		return ActionResultMsg{
			Action: action,
			Result: result,
			Error:  err,
		}
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// StateIcon returns an icon for a resource state.
func StateIcon(state string) string {
	switch state {
	case core.StateRunning, core.StateActive:
		return "🟢"
	case core.StateStopped, core.StateInactive:
		return "🔴"
	case core.StatePending, core.StateCreating, core.StateUpdating:
		return "🟡"
	case core.StateTerminated, core.StateDeleting:
		return "⚫"
	case core.StateError:
		return "🔴"
	default:
		return "⚪"
	}
}

// FormatState formats a state with an icon.
func FormatState(state string) string {
	return StateIcon(state) + " " + state
}

// TruncateString truncates a string to a maximum length.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// =============================================================================
// Responsive Table Helpers
// =============================================================================

// ColumnDef defines a responsive column with min/max/weight.
type ColumnDef struct {
	Title    string
	MinWidth int     // Minimum width (column hidden if can't fit)
	MaxWidth int     // Maximum width (0 = no max)
	Weight   float64 // Relative weight for distributing extra space
	Priority int     // Lower = more important (hidden last)
}

// CalculateColumnWidths calculates responsive column widths based on available space.
func CalculateColumnWidths(defs []ColumnDef, availableWidth int) []table.Column {
	// Account for table borders and padding (roughly 4 chars)
	availableWidth -= 4

	if availableWidth < 20 {
		availableWidth = 20
	}

	// Sort by priority to determine which columns to show
	type indexedDef struct {
		def   ColumnDef
		index int
	}

	indexed := make([]indexedDef, len(defs))
	for i, d := range defs {
		indexed[i] = indexedDef{def: d, index: i}
	}

	// Calculate minimum total width
	minTotal := 0
	for _, d := range defs {
		minTotal += d.MinWidth
	}

	// If we can't fit all columns at minimum, hide low priority ones
	visibleDefs := make([]indexedDef, 0, len(defs))
	currentMin := 0

	// Add columns by priority (lower priority value = more important)
	for priority := 0; priority <= 10; priority++ {
		for _, id := range indexed {
			if id.def.Priority == priority {
				if currentMin+id.def.MinWidth <= availableWidth {
					visibleDefs = append(visibleDefs, id)
					currentMin += id.def.MinWidth
				}
			}
		}
	}

	// Sort back to original order
	result := make([]table.Column, 0, len(visibleDefs))
	for i := range defs {
		for _, vd := range visibleDefs {
			if vd.index == i {
				result = append(result, table.Column{
					Title: vd.def.Title,
					Width: vd.def.MinWidth,
				})
				break
			}
		}
	}

	// Distribute remaining space based on weights
	usedWidth := 0
	totalWeight := 0.0
	for _, col := range result {
		usedWidth += col.Width
	}
	for _, vd := range visibleDefs {
		totalWeight += vd.def.Weight
	}

	remaining := availableWidth - usedWidth
	if remaining > 0 && totalWeight > 0 {
		// Map original indices to result indices
		resultIdx := 0
		for i := range defs {
			for _, vd := range visibleDefs {
				if vd.index == i {
					extra := int(float64(remaining) * (vd.def.Weight / totalWeight))
					newWidth := result[resultIdx].Width + extra

					// Apply max width constraint
					if vd.def.MaxWidth > 0 && newWidth > vd.def.MaxWidth {
						newWidth = vd.def.MaxWidth
					}

					result[resultIdx].Width = newWidth
					resultIdx++
					break
				}
			}
		}
	}

	return result
}

// MinTableHeight returns the minimum height for a table.
func MinTableHeight(availableHeight, headerLines, footerLines int) int {
	height := availableHeight - headerLines - footerLines
	if height < 3 {
		height = 3
	}
	return height
}
