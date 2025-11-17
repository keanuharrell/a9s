package cmd

import (
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
	RunE: func(_ *cobra.Command, _ []string) error {
		return runTUI()
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
