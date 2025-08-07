package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/keanuharrell/a9s/internal/aws"
)

type IAMModel struct {
	table       table.Model
	auditResult *aws.IAMAuditResult
	awsService  *aws.IAMService
	loading     bool
	err         error
}

type iamLoadedMsg struct {
	result *aws.IAMAuditResult
	err    error
}

func NewIAMModel(profile, region string) *IAMModel {
	columns := []table.Column{
		{Title: "Role Name", Width: 30},
		{Title: "Created", Width: 12},
		{Title: "Risk Level", Width: 12},
		{Title: "Policies", Width: 25},
		{Title: "Risk Reason", Width: 30},
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
	
	awsService, _ := aws.NewIAMService(profile, region)
	
	return &IAMModel{
		table:      t,
		awsService: awsService,
		loading:    true,
	}
}

func (m *IAMModel) Init() tea.Cmd {
	return m.loadRoles()
}

func (m *IAMModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.auditResult != nil && len(m.auditResult.Roles) > 0 && m.table.Cursor() < len(m.auditResult.Roles) {
				return m, m.showRoleDetails()
			}
		}
		
	case iamLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.auditResult = msg.result
			m.updateTable()
		}
		return m, nil
	}
	
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *IAMModel) View() string {
	if m.loading {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Render("Loading IAM roles and running security audit...")
	}
	
	if m.err != nil {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Render(fmt.Sprintf("Error loading IAM data: %v", m.err))
	}
	
	if m.auditResult == nil || len(m.auditResult.Roles) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("No IAM roles found")
	}
	
	summary := fmt.Sprintf("🔐 %d roles", m.auditResult.Summary.TotalRoles)
	if m.auditResult.Summary.HighRiskRoles > 0 {
		summary += fmt.Sprintf(" • 🚨 %d high-risk", m.auditResult.Summary.HighRiskRoles)
	}
	if len(m.auditResult.Summary.RolesWithAdminAccess) > 0 {
		summary += fmt.Sprintf(" • ⚡ %d admin access", len(m.auditResult.Summary.RolesWithAdminAccess))
	}
	
	help := "\n💡 [↑↓] navigate • [enter] role details • [r] refresh"
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color("110")).Render(summary),
		m.table.View(),
		lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(help),
	)
}

func (m *IAMModel) Refresh() tea.Cmd {
	m.loading = true
	return m.loadRoles()
}

func (m *IAMModel) loadRoles() tea.Cmd {
	return func() tea.Msg {
		if m.awsService == nil {
			return iamLoadedMsg{
				result: nil,
				err:    fmt.Errorf("AWS IAM service not initialized"),
			}
		}
		
		ctx := context.Background()
		result, err := m.awsService.AuditRoles(ctx)
		return iamLoadedMsg{
			result: result,
			err:    err,
		}
	}
}

func (m *IAMModel) updateTable() {
	if m.auditResult == nil {
		return
	}
	
	var rows []table.Row
	
	for _, role := range m.auditResult.Roles {
		riskLevel := "Low"
		riskIcon := "🟢"
		
		if role.IsHighRisk {
			riskLevel = "HIGH"
			riskIcon = "🔴"
		}
		
		policies := strings.Join(role.AttachedPolicies, ", ")
		if len(policies) > 25 {
			policies = policies[:22] + "..."
		}
		
		reason := role.RiskReason
		if len(reason) > 30 {
			reason = reason[:27] + "..."
		}
		
		rows = append(rows, table.Row{
			role.Name,
			role.CreateDate,
			riskIcon + " " + riskLevel,
			policies,
			reason,
		})
	}
	
	m.table.SetRows(rows)
}

func (m *IAMModel) showRoleDetails() tea.Cmd {
	if m.auditResult == nil || m.table.Cursor() >= len(m.auditResult.Roles) {
		return nil
	}
	
	role := m.auditResult.Roles[m.table.Cursor()]
	
	details := fmt.Sprintf("IAM Role Details:\nName: %s\nARN: %s\nCreated: %s\nRisk Level: %s\n",
		role.Name, role.ARN, role.CreateDate, 
		map[bool]string{true: "HIGH", false: "Low"}[role.IsHighRisk])
	
	if role.RiskReason != "" {
		details += fmt.Sprintf("Risk Reason: %s\n", role.RiskReason)
	}
	
	details += "\nAttached Policies:\n"
	for _, policy := range role.AttachedPolicies {
		details += fmt.Sprintf("  • %s\n", policy)
	}
	
	return tea.Printf(details)
}