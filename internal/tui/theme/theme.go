// Package theme provides theming support for the a9s TUI.
package theme

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/keanuharrell/a9s/internal/config"
)

// =============================================================================
// Theme Definition
// =============================================================================

// Theme contains all styles for the TUI.
type Theme struct {
	// Colors
	PrimaryColor    lipgloss.Color
	SecondaryColor  lipgloss.Color
	AccentColor     lipgloss.Color
	ErrorColor      lipgloss.Color
	WarningColor    lipgloss.Color
	SuccessColor    lipgloss.Color
	MutedColor      lipgloss.Color
	BackgroundColor lipgloss.Color

	// Layout styles
	Header      lipgloss.Style
	Footer      lipgloss.Style
	TabActive   lipgloss.Style
	TabInactive lipgloss.Style

	// Content styles
	Title   lipgloss.Style
	Info    lipgloss.Style
	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style
	Muted   lipgloss.Style
	Help    lipgloss.Style
}

// =============================================================================
// Theme Presets
// =============================================================================

// DefaultTheme returns the default dark theme.
func DefaultTheme() *Theme {
	t := &Theme{
		PrimaryColor:    lipgloss.Color("205"),
		SecondaryColor:  lipgloss.Color("62"),
		AccentColor:     lipgloss.Color("86"),
		ErrorColor:      lipgloss.Color("196"),
		WarningColor:    lipgloss.Color("214"),
		SuccessColor:    lipgloss.Color("82"),
		MutedColor:      lipgloss.Color("241"),
		BackgroundColor: lipgloss.Color("236"),
	}

	t.buildStyles()
	return t
}

// DarkTheme returns a dark theme.
func DarkTheme() *Theme {
	t := &Theme{
		PrimaryColor:    lipgloss.Color("205"),
		SecondaryColor:  lipgloss.Color("62"),
		AccentColor:     lipgloss.Color("86"),
		ErrorColor:      lipgloss.Color("196"),
		WarningColor:    lipgloss.Color("214"),
		SuccessColor:    lipgloss.Color("82"),
		MutedColor:      lipgloss.Color("241"),
		BackgroundColor: lipgloss.Color("236"),
	}

	t.buildStyles()
	return t
}

// LightTheme returns a light theme.
func LightTheme() *Theme {
	t := &Theme{
		PrimaryColor:    lipgloss.Color("33"),
		SecondaryColor:  lipgloss.Color("27"),
		AccentColor:     lipgloss.Color("39"),
		ErrorColor:      lipgloss.Color("160"),
		WarningColor:    lipgloss.Color("172"),
		SuccessColor:    lipgloss.Color("34"),
		MutedColor:      lipgloss.Color("245"),
		BackgroundColor: lipgloss.Color("255"),
	}

	t.buildStyles()
	return t
}

// MonochromeTheme returns a monochrome theme.
func MonochromeTheme() *Theme {
	t := &Theme{
		PrimaryColor:    lipgloss.Color("252"),
		SecondaryColor:  lipgloss.Color("248"),
		AccentColor:     lipgloss.Color("250"),
		ErrorColor:      lipgloss.Color("231"),
		WarningColor:    lipgloss.Color("250"),
		SuccessColor:    lipgloss.Color("252"),
		MutedColor:      lipgloss.Color("243"),
		BackgroundColor: lipgloss.Color("235"),
	}

	t.buildStyles()
	return t
}

// NordTheme returns a Nord-inspired theme.
func NordTheme() *Theme {
	t := &Theme{
		PrimaryColor:    lipgloss.Color("#88C0D0"),
		SecondaryColor:  lipgloss.Color("#81A1C1"),
		AccentColor:     lipgloss.Color("#5E81AC"),
		ErrorColor:      lipgloss.Color("#BF616A"),
		WarningColor:    lipgloss.Color("#EBCB8B"),
		SuccessColor:    lipgloss.Color("#A3BE8C"),
		MutedColor:      lipgloss.Color("#4C566A"),
		BackgroundColor: lipgloss.Color("#2E3440"),
	}

	t.buildStyles()
	return t
}

// DraculaTheme returns a Dracula-inspired theme.
func DraculaTheme() *Theme {
	t := &Theme{
		PrimaryColor:    lipgloss.Color("#BD93F9"),
		SecondaryColor:  lipgloss.Color("#8BE9FD"),
		AccentColor:     lipgloss.Color("#FF79C6"),
		ErrorColor:      lipgloss.Color("#FF5555"),
		WarningColor:    lipgloss.Color("#FFB86C"),
		SuccessColor:    lipgloss.Color("#50FA7B"),
		MutedColor:      lipgloss.Color("#6272A4"),
		BackgroundColor: lipgloss.Color("#282A36"),
	}

	t.buildStyles()
	return t
}

// =============================================================================
// Theme Building
// =============================================================================

// buildStyles creates all styles from colors.
func (t *Theme) buildStyles() {
	// Header style
	t.Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.PrimaryColor).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(t.SecondaryColor).
		Padding(0, 1)

	// Footer style
	t.Footer = lipgloss.NewStyle().
		Foreground(t.MutedColor).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(lipgloss.Color("238")).
		Padding(0, 1)

	// Tab styles
	t.TabActive = lipgloss.NewStyle().
		Padding(0, 1).
		Background(t.SecondaryColor).
		Foreground(lipgloss.Color("230"))

	t.TabInactive = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(t.MutedColor)

	// Content styles
	t.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.PrimaryColor)

	t.Info = lipgloss.NewStyle().
		Foreground(t.AccentColor)

	t.Success = lipgloss.NewStyle().
		Foreground(t.SuccessColor)

	t.Warning = lipgloss.NewStyle().
		Foreground(t.WarningColor)

	t.Error = lipgloss.NewStyle().
		Foreground(t.ErrorColor)

	t.Muted = lipgloss.NewStyle().
		Foreground(t.MutedColor)

	t.Help = lipgloss.NewStyle().
		Foreground(t.MutedColor)
}

// =============================================================================
// Theme from Config
// =============================================================================

// FromConfig creates a theme from configuration.
func FromConfig(cfg *config.Config) *Theme {
	if cfg == nil {
		return DefaultTheme()
	}

	// Get theme name from config
	themeName := cfg.TUI.Theme
	if themeName == "" {
		themeName = "default"
	}

	// Select base theme
	var theme *Theme
	switch themeName {
	case "light":
		theme = LightTheme()
	case "monochrome":
		theme = MonochromeTheme()
	case "nord":
		theme = NordTheme()
	case "dracula":
		theme = DraculaTheme()
	case "dark", "default":
		theme = DarkTheme()
	default:
		theme = DefaultTheme()
	}

	// Apply custom colors from config if present
	if cfg.Themes != nil {
		if custom, ok := cfg.Themes["custom"]; ok {
			needsRebuild := false

			if custom.Primary != "" {
				theme.PrimaryColor = lipgloss.Color(custom.Primary)
				needsRebuild = true
			}
			if custom.Secondary != "" {
				theme.SecondaryColor = lipgloss.Color(custom.Secondary)
				needsRebuild = true
			}
			if custom.Error != "" {
				theme.ErrorColor = lipgloss.Color(custom.Error)
			}
			if custom.Warning != "" {
				theme.WarningColor = lipgloss.Color(custom.Warning)
			}
			if custom.Success != "" {
				theme.SuccessColor = lipgloss.Color(custom.Success)
			}
			if custom.Muted != "" {
				theme.MutedColor = lipgloss.Color(custom.Muted)
			}
			if custom.Background != "" {
				theme.BackgroundColor = lipgloss.Color(custom.Background)
			}

			// Rebuild styles if primary colors changed
			if needsRebuild {
				theme.buildStyles()
			}
		}
	}

	return theme
}

// =============================================================================
// Theme Registry
// =============================================================================

// Available returns a list of available theme names.
func Available() []string {
	return []string{
		"default",
		"dark",
		"light",
		"monochrome",
		"nord",
		"dracula",
	}
}

// Get returns a theme by name.
func Get(name string) *Theme {
	switch name {
	case "light":
		return LightTheme()
	case "monochrome":
		return MonochromeTheme()
	case "nord":
		return NordTheme()
	case "dracula":
		return DraculaTheme()
	case "dark", "default":
		return DarkTheme()
	default:
		return DefaultTheme()
	}
}
