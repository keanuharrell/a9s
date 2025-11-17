package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/keanuharrell/a9s/internal/aws"
	"github.com/spf13/cobra"
)

var ec2Cmd = &cobra.Command{
	Use:   "ec2",
	Short: "Manage EC2 instances",
	Long:  `Commands for managing and monitoring EC2 instances`,
}

var ec2ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List EC2 instances",
	Long:  `List all EC2 instances in the specified region with details`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEC2List()
	},
}

func init() {
	rootCmd.AddCommand(ec2Cmd)
	ec2Cmd.AddCommand(ec2ListCmd)
}

func runEC2List() error {
	ec2Service, err := aws.NewEC2Service(awsProfile, awsRegion)
	if err != nil {
		return fmt.Errorf("failed to initialize EC2 service: %w", err)
	}

	ctx := context.Background()
	instances, err := ec2Service.ListInstances(ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	if len(instances) == 0 {
		fmt.Println("No EC2 instances found")
		return nil
	}

	if err := aws.OutputEC2Instances(instances, outputFormat); err != nil {
		fmt.Fprintf(os.Stderr, "Error outputting instances: %v\n", err)
		return err
	}

	return nil
}
