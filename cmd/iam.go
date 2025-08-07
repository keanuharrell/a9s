package cmd

import (
	"context"
	"fmt"

	"github.com/keanuharrell/a9s/internal/aws"
	"github.com/spf13/cobra"
)

var iamCmd = &cobra.Command{
	Use:   "iam",
	Short: "Manage IAM resources",
	Long:  `Commands for managing and auditing IAM roles, users, and policies`,
}

var iamAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit IAM roles for security risks",
	Long: `Audit all IAM roles and identify high-risk permissions.
	
This command will:
- List all IAM roles in the account
- Identify roles with administrative access
- Flag roles with wildcard permissions
- Provide a summary of security findings`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runIAMAudit()
	},
}

func init() {
	rootCmd.AddCommand(iamCmd)
	iamCmd.AddCommand(iamAuditCmd)
}

func runIAMAudit() error {
	iamService, err := aws.NewIAMService(awsProfile, awsRegion)
	if err != nil {
		return fmt.Errorf("failed to initialize IAM service: %w", err)
	}
	
	ctx := context.Background()
	result, err := iamService.AuditRoles(ctx)
	if err != nil {
		return fmt.Errorf("failed to audit roles: %w", err)
	}
	
	if len(result.Roles) == 0 {
		fmt.Println("No IAM roles found")
		return nil
	}
	
	if err := aws.OutputIAMAudit(result, outputFormat); err != nil {
		return fmt.Errorf("failed to output audit results: %w", err)
	}
	
	return nil
}