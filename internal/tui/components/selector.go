// Package components provides reusable TUI components.
package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Selector Component
// =============================================================================

// SelectorItem represents an item in the selector.
type SelectorItem struct {
	Value       string
	Label       string
	Description string
}

// Selector is a modal component for selecting from a list of options.
type Selector struct {
	title    string
	items    []SelectorItem
	cursor   int
	selected string
	width    int
	height   int

	// Styles
	titleStyle       lipgloss.Style
	itemStyle        lipgloss.Style
	selectedStyle    lipgloss.Style
	descriptionStyle lipgloss.Style
	borderStyle      lipgloss.Style
}

// NewSelector creates a new selector component.
func NewSelector(title string, items []SelectorItem, current string) *Selector {
	s := &Selector{
		title:    title,
		items:    items,
		selected: current,
		width:    60,
		height:   20,
	}

	// Find cursor position for current selection
	for i, item := range items {
		if item.Value == current {
			s.cursor = i
			break
		}
	}

	// Initialize styles
	s.titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF79C6")).
		MarginBottom(1)

	s.itemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F8F8F2")).
		PaddingLeft(2)

	s.selectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#50FA7B")).
		Bold(true).
		PaddingLeft(2)

	s.descriptionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6272A4")).
		PaddingLeft(4)

	s.borderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#BD93F9")).
		Padding(1, 2)

	return s
}

// SetDimensions sets the selector dimensions.
func (s *Selector) SetDimensions(width, height int) {
	s.width = width
	s.height = height
}

// Selected returns the currently selected value.
func (s *Selector) Selected() string {
	if s.cursor >= 0 && s.cursor < len(s.items) {
		return s.items[s.cursor].Value
	}
	return s.selected
}

// =============================================================================
// tea.Model Implementation
// =============================================================================

// SelectorResultMsg is sent when a selection is made.
type SelectorResultMsg struct {
	Type     string // "profile" or "region"
	Value    string
	Canceled bool
}

// Init initializes the selector.
func (s *Selector) Init() tea.Cmd {
	return nil
}

// Update handles input.
func (s *Selector) Update(msg tea.Msg) (*Selector, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if s.cursor > 0 {
				s.cursor--
			}
		case "down", "j":
			if s.cursor < len(s.items)-1 {
				s.cursor++
			}
		case "home":
			s.cursor = 0
		case "end":
			s.cursor = len(s.items) - 1
		case "pgup":
			s.cursor -= 10
			if s.cursor < 0 {
				s.cursor = 0
			}
		case "pgdown":
			s.cursor += 10
			if s.cursor >= len(s.items) {
				s.cursor = len(s.items) - 1
			}
		case "enter":
			return s, func() tea.Msg {
				return SelectorResultMsg{
					Type:  s.title,
					Value: s.Selected(),
				}
			}
		case "esc", "q":
			return s, func() tea.Msg {
				return SelectorResultMsg{
					Canceled: true,
				}
			}
		}
	}
	return s, nil
}

// View renders the selector.
func (s *Selector) View() string {
	var b strings.Builder

	// Title
	b.WriteString(s.titleStyle.Render(s.title))
	b.WriteString("\n\n")

	// Calculate visible range (scrolling)
	maxVisible := s.height - 10
	if maxVisible < 5 {
		maxVisible = 5
	}

	start := 0
	if s.cursor >= maxVisible {
		start = s.cursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(s.items) {
		end = len(s.items)
	}

	// Show scroll indicator if needed
	if start > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render("  ↑ more above"))
		b.WriteString("\n")
	}

	// Items
	for i := start; i < end; i++ {
		item := s.items[i]
		prefix := "  "
		style := s.itemStyle

		if i == s.cursor {
			prefix = "→ "
			style = s.selectedStyle
		}

		// Check mark for current selection
		check := ""
		if item.Value == s.selected {
			check = " ✓"
		}

		label := item.Label
		if label == "" {
			label = item.Value
		}

		b.WriteString(style.Render(fmt.Sprintf("%s%s%s", prefix, label, check)))
		b.WriteString("\n")

		// Description on hover
		if i == s.cursor && item.Description != "" {
			b.WriteString(s.descriptionStyle.Render(item.Description))
			b.WriteString("\n")
		}
	}

	// Show scroll indicator if needed
	if end < len(s.items) {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render("  ↓ more below"))
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
	b.WriteString(helpStyle.Render("[↑/↓] navigate  [Enter] select  [Esc] cancel"))

	// Apply border
	content := b.String()
	boxWidth := s.width - 4
	if boxWidth < 40 {
		boxWidth = 40
	}

	return s.borderStyle.Width(boxWidth).Render(content)
}

// =============================================================================
// Helper Functions
// =============================================================================

// StringsToItems converts a slice of strings to selector items.
func StringsToItems(values []string) []SelectorItem {
	items := make([]SelectorItem, len(values))
	for i, v := range values {
		items[i] = SelectorItem{Value: v, Label: v}
	}
	return items
}

// StringsToItemsWithLabels converts strings to items with custom labels.
func StringsToItemsWithLabels(values []string, labelFn func(string) string) []SelectorItem {
	items := make([]SelectorItem, len(values))
	for i, v := range values {
		items[i] = SelectorItem{
			Value: v,
			Label: labelFn(v),
		}
	}
	return items
}
