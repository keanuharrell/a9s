package cmd

import (
	"context"
	"fmt"

	"github.com/keanuharrell/a9s/internal/aws"
	"github.com/spf13/cobra"
)

var s3Cmd = &cobra.Command{
	Use:   "s3",
	Short: "Manage S3 buckets",
	Long:  `Commands for managing and optimizing S3 buckets`,
}

var s3CleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Identify and clean up S3 buckets",
	Long: `Analyze S3 buckets and identify candidates for cleanup.
	
This command will:
- List all S3 buckets in the account
- Identify empty buckets
- Flag public buckets without tags
- Recommend buckets for cleanup
- Optionally delete buckets (with --dry-run flag for safety)`,
	RunE: func(_ *cobra.Command, _ []string) error {
		return runS3Cleanup()
	},
}

func init() {
	rootCmd.AddCommand(s3Cmd)
	s3Cmd.AddCommand(s3CleanupCmd)
}

func runS3Cleanup() error {
	s3Service, err := aws.NewS3Service(awsProfile, awsRegion)
	if err != nil {
		return fmt.Errorf("failed to initialize S3 service: %w", err)
	}

	ctx := context.Background()

	if dryRun {
		fmt.Println("Running in dry-run mode - no changes will be made")
	}

	result, err := s3Service.CleanupBuckets(ctx, dryRun)
	if err != nil {
		return fmt.Errorf("failed to analyze buckets: %w", err)
	}

	if len(result.Buckets) == 0 {
		fmt.Println("No S3 buckets found")
		return nil
	}

	if err := aws.OutputS3Cleanup(result, outputFormat); err != nil {
		return fmt.Errorf("failed to output results: %w", err)
	}

	if !dryRun && len(result.CleanupCandidates) > 0 {
		fmt.Printf("\n%d buckets were processed for cleanup\n",
			len(result.CleanupCandidates))
	}

	return nil
}
