// Package tui provides terminal user interface components for a9s
package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the main TUI application model
type Model struct {
	currentView ViewType
	awsProfile  string
	awsRegion   string
	ec2View     *EC2Model
	iamView     *IAMModel
	s3View      *S3Model
	loading     bool
	err         error
	width       int
	height      int
}

// ViewType represents the different views available in the TUI
type ViewType int

const (
	// EC2View displays EC2 instances
	EC2View ViewType = iota
	// IAMView displays IAM roles
	IAMView
	// S3View displays S3 buckets
	S3View
	// HelpView displays help information
	HelpView
)

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// NewModel creates a new TUI model with the specified AWS profile and region
func NewModel(profile, region string) *Model {
	return &Model{
		currentView: EC2View,
		awsProfile:  profile,
		awsRegion:   region,
		ec2View:     NewEC2Model(profile, region),
		iamView:     NewIAMModel(profile, region),
		s3View:      NewS3Model(profile, region),
		loading:     false,
	}
}

// Init initializes the TUI model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tick(),
		m.ec2View.Init(),
	)
}

// Update handles messages and updates the model state
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.currentView = EC2View
			return m, m.ec2View.Refresh()
		case "2":
			m.currentView = IAMView
			return m, m.iamView.Refresh()
		case "3":
			m.currentView = S3View
			return m, m.s3View.Refresh()
		case "?", "h":
			m.currentView = HelpView
			return m, nil
		case "r":
			return m, m.refreshCurrentView()
		}

	case tickMsg:
		return m, tea.Batch(
			tick(),
			m.refreshCurrentView(),
		)

	case error:
		m.err = msg
		return m, nil
	}

	var cmd tea.Cmd
	switch m.currentView {
	case EC2View:
		ec2Model, ec2Cmd := m.ec2View.Update(msg)
		m.ec2View = ec2Model.(*EC2Model)
		cmd = ec2Cmd
	case IAMView:
		iamModel, iamCmd := m.iamView.Update(msg)
		m.iamView = iamModel.(*IAMModel)
		cmd = iamCmd
	case S3View:
		s3Model, s3Cmd := m.s3View.Update(msg)
		m.s3View = s3Model.(*S3Model)
		cmd = s3Cmd
	}

	return m, cmd
}

func (m *Model) refreshCurrentView() tea.Cmd {
	switch m.currentView {
	case EC2View:
		return m.ec2View.Refresh()
	case IAMView:
		return m.iamView.Refresh()
	case S3View:
		return m.s3View.Refresh()
	}
	return nil
}

// View renders the TUI view
func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	header := m.renderHeader()
	tabs := m.renderTabs()
	content := m.renderCurrentView()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		tabs,
		content,
		footer,
	)
}

func (m *Model) renderHeader() string {
	profile := m.awsProfile
	if profile == "" {
		profile = "default"
	}
	region := m.awsRegion
	if region == "" {
		region = "us-east-1"
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1).
		Width(m.width - 2)

	title := fmt.Sprintf("🚀 a9s - AWS Terminal UI (Profile: %s, Region: %s)", profile, region)
	return headerStyle.Render(title)
}

func (m *Model) renderTabs() string {
	tabs := []string{"[1] EC2", "[2] IAM", "[3] S3", "[?] Help"}

	renderedTabs := make([]string, 0, len(tabs))
	for i, tab := range tabs {
		style := lipgloss.NewStyle().Padding(0, 1)

		if ViewType(i) == m.currentView || (i == 3 && m.currentView == HelpView) {
			style = style.Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230"))
		}

		renderedTabs = append(renderedTabs, style.Render(tab))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
}

func (m *Model) renderCurrentView() string {
	contentStyle := lipgloss.NewStyle().
		Height(m.height - 6).
		Width(m.width - 2).
		Padding(1)

	var content string
	switch m.currentView {
	case EC2View:
		content = m.ec2View.View()
	case IAMView:
		content = m.iamView.View()
	case S3View:
		content = m.s3View.View()
	case HelpView:
		content = m.renderHelp()
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
		content = errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	return contentStyle.Render(content)
}

func (m *Model) renderHelp() string {
	help := `
🚀 a9s - The k9s for AWS

Navigation:
  [1,2,3]     Switch between services (EC2, IAM, S3)
  [r]         Refresh current view
  [q]         Quit
  [?/h]       Show this help

Service-specific keys:
  EC2 View:
    [j/k]     Navigate instances
    [enter]   Instance details
    [s]       Start instance
    [t]       Stop instance
    
  IAM View:
    [j/k]     Navigate roles
    [enter]   Role details
    
  S3 View:
    [j/k]     Navigate buckets
    [enter]   Bucket details
    [d]       Delete bucket (with confirmation)

Tip: Views refresh automatically every 5 seconds
`

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Render(help)
}

func (m *Model) renderFooter() string {
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(lipgloss.Color("238")).
		Padding(0, 1).
		Width(m.width - 2)

	status := ""
	if m.loading {
		status = "Loading..."
	} else {
		status = "Ready • [r] refresh • [q] quit • [?] help"
	}

	return footerStyle.Render(status)
}
