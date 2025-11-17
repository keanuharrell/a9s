package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/keanuharrell/a9s/internal/tui"
	"github.com/spf13/cobra"
)

var (
	// Version is set via ldflags during build
	Version   = "dev"
	BuildTime = "unknown"

	// CLI flags
	outputFormat string
	awsProfile   string
	awsRegion    string
	dryRun       bool
	configFile   string
)

var rootCmd = &cobra.Command{
	Use:   "a9s",
	Short: "Interactive Terminal UI for AWS infrastructure management",
	Long: `a9s is the k9s for AWS - an interactive Terminal UI that simplifies AWS infrastructure management.

It provides commands for:
- EC2 instance management and monitoring
- IAM security auditing and compliance checks
- S3 bucket cleanup and optimization
- Interactive Terminal UI (TUI) for real-time management
- And more DevOps automation tasks

Usage:
  a9s          Launch interactive TUI (default)
  a9s tui      Launch interactive TUI explicitly
  a9s [cmd]    Run specific CLI commands`,
	Version: Version,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runTUI(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// GetRootCommand returns the root command for external use (e.g., man page generation)
func GetRootCommand() *cobra.Command {
	return rootCmd
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

func init() {
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format (json|table)")
	rootCmd.PersistentFlags().StringVar(&awsProfile, "profile", "", "AWS profile to use")
	rootCmd.PersistentFlags().StringVar(&awsRegion, "region", "", "AWS region")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Simulate actions without making changes")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file path (optional)")
}
