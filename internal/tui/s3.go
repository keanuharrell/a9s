package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/keanuharrell/a9s/internal/aws"
)

type S3Model struct {
	table          table.Model
	cleanupResult  *aws.S3CleanupResult
	awsService     *aws.S3Service
	loading        bool
	err            error
}

type s3LoadedMsg struct {
	result *aws.S3CleanupResult
	err    error
}

func NewS3Model(profile, region string) *S3Model {
	columns := []table.Column{
		{Title: "Bucket Name", Width: 35},
		{Title: "Region", Width: 15},
		{Title: "Created", Width: 12},
		{Title: "Objects", Width: 10},
		{Title: "Public", Width: 8},
		{Title: "Tagged", Width: 8},
		{Title: "Cleanup", Width: 8},
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
	
	awsService, _ := aws.NewS3Service(profile, region)
	
	return &S3Model{
		table:      t,
		awsService: awsService,
		loading:    true,
	}
}

func (m *S3Model) Init() tea.Cmd {
	return m.loadBuckets()
}

func (m *S3Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.cleanupResult != nil && len(m.cleanupResult.Buckets) > 0 && m.table.Cursor() < len(m.cleanupResult.Buckets) {
				return m, m.showBucketDetails()
			}
		case "d":
			if m.cleanupResult != nil && len(m.cleanupResult.Buckets) > 0 && m.table.Cursor() < len(m.cleanupResult.Buckets) {
				return m, m.confirmDeleteBucket()
			}
		}
		
	case s3LoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.cleanupResult = msg.result
			m.updateTable()
		}
		return m, nil
	}
	
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *S3Model) View() string {
	if m.loading {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Render("Loading S3 buckets and analyzing cleanup candidates...")
	}
	
	if m.err != nil {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Render(fmt.Sprintf("Error loading S3 data: %v", m.err))
	}
	
	if m.cleanupResult == nil || len(m.cleanupResult.Buckets) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("No S3 buckets found")
	}
	
	summary := fmt.Sprintf("🪣 %d buckets", m.cleanupResult.Summary.TotalBuckets)
	if m.cleanupResult.Summary.EmptyBuckets > 0 {
		summary += fmt.Sprintf(" • 📭 %d empty", m.cleanupResult.Summary.EmptyBuckets)
	}
	if m.cleanupResult.Summary.PublicBuckets > 0 {
		summary += fmt.Sprintf(" • 🌐 %d public", m.cleanupResult.Summary.PublicBuckets)
	}
	if len(m.cleanupResult.CleanupCandidates) > 0 {
		summary += fmt.Sprintf(" • 🧹 %d cleanup candidates", len(m.cleanupResult.CleanupCandidates))
	}
	
	help := "\n💡 [↑↓] navigate • [enter] bucket details • [d] delete bucket • [r] refresh"
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color("110")).Render(summary),
		m.table.View(),
		lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(help),
	)
}

func (m *S3Model) Refresh() tea.Cmd {
	m.loading = true
	return m.loadBuckets()
}

func (m *S3Model) loadBuckets() tea.Cmd {
	return func() tea.Msg {
		if m.awsService == nil {
			return s3LoadedMsg{
				result: nil,
				err:    fmt.Errorf("AWS S3 service not initialized"),
			}
		}
		
		ctx := context.Background()
		result, err := m.awsService.AnalyzeBuckets(ctx)
		return s3LoadedMsg{
			result: result,
			err:    err,
		}
	}
}

func (m *S3Model) updateTable() {
	if m.cleanupResult == nil {
		return
	}
	
	var rows []table.Row
	
	for _, bucket := range m.cleanupResult.Buckets {
		public := "🔒 No"
		if bucket.IsPublic {
			public = "🌐 Yes"
		}
		
		tagged := "❌ No"
		if bucket.HasTags {
			tagged = "✅ Yes"
		}
		
		cleanup := "❌ No"
		if bucket.ShouldCleanup {
			cleanup = "⚠️ Yes"
		}
		
		objectCount := fmt.Sprintf("%d", bucket.ObjectCount)
		if bucket.IsEmpty {
			objectCount = "📭 0"
		}
		
		rows = append(rows, table.Row{
			bucket.Name,
			bucket.Region,
			bucket.CreatedDate,
			objectCount,
			public,
			tagged,
			cleanup,
		})
	}
	
	m.table.SetRows(rows)
}

func (m *S3Model) showBucketDetails() tea.Cmd {
	if m.cleanupResult == nil || m.table.Cursor() >= len(m.cleanupResult.Buckets) {
		return nil
	}
	
	bucket := m.cleanupResult.Buckets[m.table.Cursor()]
	
	details := fmt.Sprintf("S3 Bucket Details:\nName: %s\nRegion: %s\nCreated: %s\nObject Count: %d\nSize: %s\nPublic: %v\nTagged: %v\n",
		bucket.Name, bucket.Region, bucket.CreatedDate,
		bucket.ObjectCount, formatBytes(bucket.SizeBytes),
		bucket.IsPublic, bucket.HasTags)
	
	if bucket.ShouldCleanup {
		details += fmt.Sprintf("\n🧹 Cleanup Recommended: %s\n", bucket.CleanupReason)
	}
	
	return tea.Printf(details)
}

func (m *S3Model) confirmDeleteBucket() tea.Cmd {
	if m.cleanupResult == nil || m.table.Cursor() >= len(m.cleanupResult.Buckets) {
		return nil
	}
	
	bucket := m.cleanupResult.Buckets[m.table.Cursor()]
	
	return tea.Printf("⚠️ Delete Bucket Confirmation\nBucket: %s\nObjects: %d\n\nNote: This is a demo - actual delete would require confirmation dialog\nUse the CLI with --dry-run for safe testing", 
		bucket.Name, bucket.ObjectCount)
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}