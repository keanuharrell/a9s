// Package base provides base implementations for service views.
package base

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/keanuharrell/a9s/internal/core"
)

// =============================================================================
// Layout Constants for Views
// =============================================================================

const (
	// View layout: summary(1) + blank(1) + table + message(1) + help(1) = table + 4
	// BUT table adds header(1) + border(1) beyond SetHeight, so actual = table + 6
	viewNonTableLines = 6
	minTableHeight    = 3
)

// =============================================================================
// Generic Table View
// =============================================================================

// TableView provides a reusable table-based view with responsive columns.
type TableView struct {
	*View
	Table      table.Model
	ColumnDefs []ColumnDef
	Styles     Styles
	Resources  []core.Resource
	Message    string
}

// NewTableView creates a new table view with responsive columns.
func NewTableView(name, shortcut, serviceName string, columnDefs []ColumnDef) *TableView {
	columns := CalculateColumnWidths(columnDefs, 100)

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	styles := DefaultStyles()
	t.SetStyles(styles.Table)

	return &TableView{
		View:       NewView(name, shortcut, serviceName),
		Table:      t,
		ColumnDefs: columnDefs,
		Styles:     styles,
	}
}

// HandleWindowSize updates table dimensions based on available space.
// Dimensions come from App via SetDimensions().
func (tv *TableView) HandleWindowSize(_ tea.WindowSizeMsg) {
	width := tv.Width()
	height := tv.Height()

	if width == 0 || height == 0 {
		return
	}

	// Table gets all space minus non-table lines
	tableHeight := height - viewNonTableLines
	if tableHeight < minTableHeight {
		tableHeight = minTableHeight
	}
	tv.Table.SetHeight(tableHeight)

	// Update column widths
	columns := CalculateColumnWidths(tv.ColumnDefs, width)
	tv.Table.SetColumns(columns)
}

// UpdateTable passes a message to the table and returns the command.
func (tv *TableView) UpdateTable(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	tv.Table, cmd = tv.Table.Update(msg)
	return cmd
}

// SetRows sets the table rows.
func (tv *TableView) SetRows(rows []table.Row) {
	tv.Table.SetRows(rows)
}

// Cursor returns the current cursor position.
func (tv *TableView) Cursor() int {
	return tv.Table.Cursor()
}

// GetSelectedResource returns the currently selected resource.
func (tv *TableView) GetSelectedResource() *core.Resource {
	cursor := tv.Table.Cursor()
	if cursor >= 0 && cursor < len(tv.Resources) {
		return &tv.Resources[cursor]
	}
	return nil
}

// SetMessage sets the status message.
func (tv *TableView) SetMessage(msg string) {
	tv.Message = msg
}

// Reset clears the view data, forcing a reload on next Init.
func (tv *TableView) Reset() {
	tv.Resources = nil
	tv.Message = ""
	tv.SetRows(nil)
}

// TableViewString returns the rendered table.
func (tv *TableView) TableViewString() string {
	return tv.Table.View()
}
