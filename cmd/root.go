package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
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
	Version: "0.1.0",
	Run: func(cmd *cobra.Command, args []string) {
		runTUI()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format (json|table)")
	rootCmd.PersistentFlags().StringVar(&awsProfile, "profile", "", "AWS profile to use")
	rootCmd.PersistentFlags().StringVar(&awsRegion, "region", "", "AWS region")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Simulate actions without making changes")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file path (optional)")
}