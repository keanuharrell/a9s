package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/keanuharrell/a9s/internal/aws"
)

type EC2Model struct {
	table       table.Model
	instances   []aws.EC2Instance
	awsService  *aws.EC2Service
	loading     bool
	err         error
	selectedRow int
}

type ec2LoadedMsg struct {
	instances []aws.EC2Instance
	err       error
}

func NewEC2Model(profile, region string) *EC2Model {
	columns := []table.Column{
		{Title: "ID", Width: 20},
		{Title: "Name", Width: 25},
		{Title: "Type", Width: 15},
		{Title: "State", Width: 12},
		{Title: "Public IP", Width: 15},
		{Title: "AZ", Width: 15},
	}
	
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(15),
	)
	
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)
	
	awsService, _ := aws.NewEC2Service(profile, region)
	
	return &EC2Model{
		table:      t,
		awsService: awsService,
		loading:    true,
	}
}

func (m *EC2Model) Init() tea.Cmd {
	return m.loadInstances()
}

func (m *EC2Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if len(m.instances) > 0 && m.table.Cursor() < len(m.instances) {
				return m, m.showInstanceDetails()
			}
		case "s":
			if len(m.instances) > 0 && m.table.Cursor() < len(m.instances) {
				return m, m.startInstance()
			}
		case "t":
			if len(m.instances) > 0 && m.table.Cursor() < len(m.instances) {
				return m, m.stopInstance()
			}
		}
		
	case ec2LoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.instances = msg.instances
			m.updateTable()
		}
		return m, nil
	}
	
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *EC2Model) View() string {
	if m.loading {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Render("Loading EC2 instances...")
	}
	
	if m.err != nil {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Render(fmt.Sprintf("Error loading instances: %v", m.err))
	}
	
	if len(m.instances) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("No EC2 instances found in this region")
	}
	
	summary := fmt.Sprintf("📦 %d instances", len(m.instances))
	running := 0
	stopped := 0
	for _, instance := range m.instances {
		switch instance.State {
		case "running":
			running++
		case "stopped":
			stopped++
		}
	}
	
	if running > 0 {
		summary += fmt.Sprintf(" • ✅ %d running", running)
	}
	if stopped > 0 {
		summary += fmt.Sprintf(" • ⏹️ %d stopped", stopped)
	}
	
	help := "\n💡 [↑↓] navigate • [enter] details • [s] start • [t] stop • [r] refresh"
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color("110")).Render(summary),
		m.table.View(),
		lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(help),
	)
}

func (m *EC2Model) Refresh() tea.Cmd {
	m.loading = true
	return m.loadInstances()
}

func (m *EC2Model) loadInstances() tea.Cmd {
	return func() tea.Msg {
		if m.awsService == nil {
			return ec2LoadedMsg{
				instances: nil,
				err:       fmt.Errorf("AWS service not initialized"),
			}
		}
		
		ctx := context.Background()
		instances, err := m.awsService.ListInstances(ctx)
		return ec2LoadedMsg{
			instances: instances,
			err:       err,
		}
	}
}

func (m *EC2Model) updateTable() {
	var rows []table.Row
	
	for _, instance := range m.instances {
		state := instance.State
		switch state {
		case "running":
			state = "🟢 " + state
		case "stopped":
			state = "🔴 " + state
		case "pending":
			state = "🟡 " + state
		case "stopping":
			state = "🟠 " + state
		}
		
		publicIP := instance.PublicIP
		if publicIP == "" {
			publicIP = "-"
		}
		
		rows = append(rows, table.Row{
			instance.ID,
			instance.Name,
			instance.Type,
			state,
			publicIP,
			instance.AZ,
		})
	}
	
	m.table.SetRows(rows)
}

func (m *EC2Model) showInstanceDetails() tea.Cmd {
	if m.table.Cursor() >= len(m.instances) {
		return nil
	}
	
	instance := m.instances[m.table.Cursor()]
	
	return tea.Printf("Instance Details:\nID: %s\nName: %s\nType: %s\nState: %s\nPublic IP: %s\nPrivate IP: %s\nAZ: %s\nLaunch Time: %s\n",
		instance.ID, instance.Name, instance.Type, instance.State,
		instance.PublicIP, instance.PrivateIP, instance.AZ, instance.LaunchTime)
}

func (m *EC2Model) startInstance() tea.Cmd {
	if m.table.Cursor() >= len(m.instances) {
		return nil
	}
	
	instance := m.instances[m.table.Cursor()]
	
	if instance.State == "running" {
		return tea.Printf("Instance %s is already running", instance.ID)
	}
	
	return tea.Printf("🚀 Starting instance %s (%s)...\nNote: This is a demo - actual start command would be implemented here", 
		instance.ID, instance.Name)
}

func (m *EC2Model) stopInstance() tea.Cmd {
	if m.table.Cursor() >= len(m.instances) {
		return nil
	}
	
	instance := m.instances[m.table.Cursor()]
	
	if instance.State == "stopped" {
		return tea.Printf("Instance %s is already stopped", instance.ID)
	}
	
	return tea.Printf("⏹️ Stopping instance %s (%s)...\nNote: This is a demo - actual stop command would be implemented here", 
		instance.ID, instance.Name)
}