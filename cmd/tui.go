package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/keanuharrell/a9s/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive Terminal UI",
	Long: `Launch the interactive Terminal User Interface for AWS resource management.
	
The TUI provides an intuitive way to:
- Navigate EC2 instances and manage them
- Audit IAM roles for security issues
- Analyze S3 buckets and cleanup candidates
- Real-time refresh and interactive controls

Navigation:
  [1,2,3]     Switch between services (EC2, IAM, S3)
  [↑↓]        Navigate items in current view
  [Enter]     View details
  [r]         Refresh current view
  [q]         Quit
  [?]         Help`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTUI() error {
	model := tui.NewModel(awsProfile, awsRegion)
	
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	
	_, err := program.Run()
	if err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}
	
	return nil
}