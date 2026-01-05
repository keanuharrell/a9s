// Package tui provides the terminal user interface for a9s.
// It implements a dynamic, registry-based TUI that supports hot-reloading of views.
package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	awsfactory "github.com/keanuharrell/a9s/internal/aws"
	"github.com/keanuharrell/a9s/internal/config"
	"github.com/keanuharrell/a9s/internal/core"
	"github.com/keanuharrell/a9s/internal/registry"
	"github.com/keanuharrell/a9s/internal/tui/components"
	"github.com/keanuharrell/a9s/internal/tui/theme"
)

// =============================================================================
// Layout Constants - SINGLE SOURCE OF TRUTH
// =============================================================================

const (
	// Chrome heights (fixed)
	chromeHeight = 6 // header(3) + tabs(1) + footer(2)
)

// =============================================================================
// Application Model
// =============================================================================

// SelectorType indicates which selector is being shown.
type SelectorType int

const (
	SelectorNone SelectorType = iota
	SelectorProfile
	SelectorRegion
)

// App is the main TUI application model.
// It manages views dynamically through the registry pattern.
type App struct {
	// Dependencies
	registry *registry.Registry
	config   *config.Config
	theme    *theme.Theme
	factory  *awsfactory.ClientFactory

	// State
	currentView core.View
	viewIndex   int
	views       []core.View
	shortcuts   map[string]core.View

	// UI state
	width        int
	height       int
	showHelp     bool
	message      string
	msgTime      time.Time
	selectorType SelectorType
	selector     *components.Selector

	// Event dispatcher
	dispatcher core.EventDispatcher

	// Callback for config changes (set by root.go)
	OnConfigChange func(profile, region string) error
}

// NewApp creates a new TUI application.
func NewApp(reg *registry.Registry, cfg *config.Config, dispatcher core.EventDispatcher) *App {
	app := &App{
		registry:     reg,
		config:       cfg,
		theme:        theme.FromConfig(cfg),
		shortcuts:    make(map[string]core.View),
		dispatcher:   dispatcher,
		selectorType: SelectorNone,
	}

	// Load initial views
	app.refreshViews()

	// Watch for registry changes
	reg.Watch(func(_ core.RegistryEvent) {
		app.refreshViews()
	})

	return app
}

// SetFactory sets the AWS client factory for dynamic config changes.
func (a *App) SetFactory(factory *awsfactory.ClientFactory) {
	a.factory = factory
}

// SetOnConfigChange sets the callback for config changes.
func (a *App) SetOnConfigChange(fn func(profile, region string) error) {
	a.OnConfigChange = fn
}

// refreshViews updates the view list from registry.
func (a *App) refreshViews() {
	a.views = a.registry.ListViewsOrdered()
	a.shortcuts = make(map[string]core.View)

	for _, view := range a.views {
		a.shortcuts[view.Shortcut()] = view
	}

	// Set current view if not set
	if a.currentView == nil && len(a.views) > 0 {
		a.currentView = a.views[0]
		a.viewIndex = 0
	}
}

// contentHeight returns the available height for view content
func (a *App) contentHeight() int {
	h := a.height - chromeHeight
	if h < 5 {
		h = 5
	}
	return h
}

// contentWidth returns the available width for content area
func (a *App) contentWidth() int {
	w := a.width - 2 // Small margin
	if w < 20 {
		w = 20
	}
	return w
}

// updateViewDimensions updates all views with current dimensions
func (a *App) updateViewDimensions() {
	w := a.contentWidth()
	h := a.contentHeight()
	for _, view := range a.views {
		view.SetDimensions(w, h)
	}
}

// =============================================================================
// Messages
// =============================================================================

// tickMsg is sent periodically for auto-refresh.
type tickMsg time.Time

// viewChangedMsg signals a view change.
type viewChangedMsg struct {
	view core.View
}

// =============================================================================
// tea.Model Implementation
// =============================================================================

// Init initializes the application.
func (a *App) Init() tea.Cmd {
	var cmds []tea.Cmd

	// Start tick timer
	cmds = append(cmds, a.tick())

	// Initialize current view
	if a.currentView != nil {
		cmds = append(cmds, a.currentView.Init())
	}

	return tea.Batch(cmds...)
}

// Update handles messages.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle selector mode first
	if a.selectorType != SelectorNone && a.selector != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			selector, cmd := a.selector.Update(msg)
			a.selector = selector
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return a, tea.Batch(cmds...)

		case components.SelectorResultMsg:
			return a.handleSelectorResult(msg)
		}
		return a, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.updateViewDimensions()
		// Don't return - forward to views

	case tea.KeyMsg:
		cmd := a.handleKeyPress(msg)
		if cmd != nil {
			return a, cmd
		}

	case tickMsg:
		cmds = append(cmds, a.tick())
		if a.currentView != nil && a.config.TUI.RefreshInterval > 0 {
			cmds = append(cmds, a.currentView.Refresh())
		}
		return a, tea.Batch(cmds...)

	case viewChangedMsg:
		a.currentView = msg.view
		return a, a.currentView.Init()

	case configChangedMsg:
		profile := a.config.AWS.Profile
		if profile == "" {
			profile = "default"
		}
		a.setMessage(fmt.Sprintf("Switched to %s / %s", profile, a.config.AWS.Region))

		for _, view := range a.views {
			if resettable, ok := view.(interface{ Reset() }); ok {
				resettable.Reset()
			}
		}

		for _, view := range a.views {
			cmds = append(cmds, view.Init())
		}
		return a, tea.Batch(cmds...)

	case components.SelectorResultMsg:
		return a.handleSelectorResult(msg)
	}

	// Forward message to ALL views
	for _, view := range a.views {
		model, cmd := view.Update(msg)
		if v, ok := model.(core.View); ok {
			for i, existing := range a.views {
				if existing.Name() == v.Name() {
					a.views[i] = v
					if v == a.currentView || existing == a.currentView {
						a.currentView = v
					}
					break
				}
			}
		}
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

// handleKeyPress processes keyboard input.
func (a *App) handleKeyPress(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	switch key {
	case "q", "ctrl+c":
		return tea.Quit

	case "?":
		a.showHelp = !a.showHelp
		return nil

	case "P":
		return a.showProfileSelector()

	case "G":
		return a.showRegionSelector()

	case "r":
		if a.currentView != nil {
			a.setMessage("Refreshing...")
			return a.currentView.Refresh()
		}
		return nil

	case "tab":
		return a.nextView()

	case "shift+tab":
		return a.prevView()

	case "esc":
		if a.showHelp {
			a.showHelp = false
			return nil
		}
	}

	// View shortcuts (1, 2, 3, etc.)
	if view, ok := a.shortcuts[key]; ok {
		if view != a.currentView {
			return a.switchToView(view)
		}
	}

	return nil
}

// =============================================================================
// Profile/Region Selector
// =============================================================================

type configChangedMsg struct {
	profile string
	region  string
}

func (a *App) showProfileSelector() tea.Cmd {
	profiles := awsfactory.ListProfiles()
	items := components.StringsToItems(profiles)

	current := a.config.AWS.Profile
	if current == "" {
		current = "default"
	}

	a.selector = components.NewSelector("Select AWS Profile", items, current)
	a.selector.SetDimensions(a.width, a.height)
	a.selectorType = SelectorProfile

	return nil
}

func (a *App) showRegionSelector() tea.Cmd {
	regions := awsfactory.ListRegions()
	items := components.StringsToItemsWithLabels(regions, func(r string) string {
		return fmt.Sprintf("%s (%s)", r, awsfactory.GetRegionName(r))
	})

	current := a.config.AWS.Region
	if current == "" {
		current = "us-east-1"
	}

	a.selector = components.NewSelector("Select AWS Region", items, current)
	a.selector.SetDimensions(a.width, a.height)
	a.selectorType = SelectorRegion

	return nil
}

func (a *App) handleSelectorResult(msg components.SelectorResultMsg) (tea.Model, tea.Cmd) {
	selectorType := a.selectorType
	a.selectorType = SelectorNone
	a.selector = nil

	if msg.Canceled {
		return a, nil
	}

	profile := a.config.AWS.Profile
	region := a.config.AWS.Region

	switch selectorType {
	case SelectorProfile:
		profile = msg.Value
		if profile == "default" {
			profile = ""
		}
	case SelectorRegion:
		region = msg.Value
	}

	if profile == a.config.AWS.Profile && region == a.config.AWS.Region {
		return a, nil
	}

	a.config.AWS.Profile = profile
	a.config.AWS.Region = region

	if a.factory != nil {
		a.setMessage("Updating AWS configuration...")
		return a, a.updateAWSConfig(profile, region)
	}

	if a.OnConfigChange != nil {
		if err := a.OnConfigChange(profile, region); err != nil {
			a.setMessage(fmt.Sprintf("Error: %v", err))
			return a, nil
		}
	}

	return a, func() tea.Msg {
		return configChangedMsg{profile: profile, region: region}
	}
}

func (a *App) updateAWSConfig(profile, region string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		_ = a.factory.UpdateConfig(ctx, profile, region)
		return configChangedMsg{profile: profile, region: region}
	}
}

// switchToView switches to a different view immediately.
func (a *App) switchToView(view core.View) tea.Cmd {
	for i, v := range a.views {
		if v == view {
			a.viewIndex = i
			break
		}
	}
	a.currentView = view
	view.SetDimensions(a.contentWidth(), a.contentHeight())
	return view.Init()
}

func (a *App) nextView() tea.Cmd {
	if len(a.views) == 0 {
		return nil
	}
	a.viewIndex = (a.viewIndex + 1) % len(a.views)
	return a.switchToView(a.views[a.viewIndex])
}

func (a *App) prevView() tea.Cmd {
	if len(a.views) == 0 {
		return nil
	}
	a.viewIndex = (a.viewIndex - 1 + len(a.views)) % len(a.views)
	return a.switchToView(a.views[a.viewIndex])
}

func (a *App) tick() tea.Cmd {
	interval := a.config.TUI.RefreshInterval
	if interval == 0 {
		interval = 30 * time.Second
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (a *App) setMessage(msg string) {
	a.message = msg
	a.msgTime = time.Now()
}

// =============================================================================
// View - SINGLE ROOT LAYOUT
// =============================================================================

func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	if a.selectorType != SelectorNone && a.selector != nil {
		return a.renderWithSelector()
	}

	if a.showHelp {
		return a.renderHelp()
	}

	// ROOT LAYOUT - Use lipgloss for proper styling
	header := a.renderHeader()
	tabs := a.renderTabs()
	content := a.renderContent()
	footer := a.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, content, footer)
}

func (a *App) renderHeader() string {
	profile := a.config.AWS.Profile
	if profile == "" {
		profile = "default"
	}
	region := a.config.AWS.Region
	if region == "" {
		region = "us-east-1"
	}

	title := fmt.Sprintf("🚀 a9s - AWS Terminal UI  ⎔ %s  ⎔ %s", profile, region)

	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(a.theme.PrimaryColor).
		Background(a.theme.BackgroundColor).
		Padding(0, 1).
		Width(a.width - 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.SecondaryColor)

	return style.Render(title)
}

func (a *App) renderTabs() string {
	if len(a.views) == 0 {
		return ""
	}

	sortedViews := make([]core.View, len(a.views))
	copy(sortedViews, a.views)
	sort.Slice(sortedViews, func(i, j int) bool {
		return sortedViews[i].Shortcut() < sortedViews[j].Shortcut()
	})

	var parts []string
	for _, view := range sortedViews {
		label := fmt.Sprintf(" [%s] %s ", view.Shortcut(), view.Name())
		if view == a.currentView {
			parts = append(parts, a.theme.TabActive.Render(label))
		} else {
			parts = append(parts, a.theme.TabInactive.Render(label))
		}
	}
	parts = append(parts, a.theme.TabInactive.Render(" [?] Help "))

	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (a *App) renderContent() string {
	h := a.contentHeight()
	w := a.contentWidth()

	var content string
	if a.currentView != nil {
		content = a.currentView.View()
	} else {
		content = a.theme.Muted.Render("No services registered.")
	}

	// IMPORTANT: lipgloss.Height() does NOT truncate content!
	// We must manually truncate to exactly h lines
	lines := strings.Split(content, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	// Pad with empty lines if too short
	for len(lines) < h {
		lines = append(lines, "")
	}

	// Truncate width if needed
	for i, line := range lines {
		if lipgloss.Width(line) > w {
			lines[i] = line[:w]
		}
	}

	return strings.Join(lines, "\n")
}

func (a *App) renderFooter() string {
	status := "Ready"
	if a.currentView != nil && a.currentView.IsLoading() {
		status = "⏳ Loading..."
	} else if a.message != "" && time.Since(a.msgTime) < 3*time.Second {
		status = a.message
	}

	help := "[r] refresh  [P] profile  [G] region  [q] quit  [?] help"

	style := lipgloss.NewStyle().
		Foreground(a.theme.MutedColor).
		Width(a.width-2).
		Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(lipgloss.Color("238"))

	return style.Render(fmt.Sprintf("%s  │  %s", status, help))
}

func (a *App) renderWithSelector() string {
	selectorContent := a.selector.View()

	bgStyle := lipgloss.NewStyle().
		Width(a.width).
		Height(a.height).
		Align(lipgloss.Center, lipgloss.Center)

	return bgStyle.Render(selectorContent)
}

func (a *App) renderHelp() string {
	help := `🚀 a9s - The k9s for AWS

Navigation:
  [1-4]       Switch services
  [Tab]       Next service
  [r]         Refresh
  [P]         Change profile
  [G]         Change region
  [?]         Toggle help
  [q]         Quit

EC2: [s]tart [t]stop [b]reboot
IAM: [a]udit [p]olicies
S3:  [a]nalyze [d]elete [D]confirm
Lambda: [i]nvoke [c]onfig

Press [?] or [Esc] to close.`

	style := lipgloss.NewStyle().
		Width(a.width-4).
		Height(a.height-2).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.AccentColor)

	return style.Render(a.theme.Muted.Render(help))
}

var _ tea.Model = (*App)(nil)
