// Package aws provides AWS service clients and operations for EC2, IAM, and S3
package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	appconfig "github.com/keanuharrell/a9s/internal/config"
	"github.com/olekukonko/tablewriter"
)

// EC2Instance represents an EC2 instance with its metadata
type EC2Instance struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	State      string `json:"state"`
	PublicIP   string `json:"public_ip"`
	PrivateIP  string `json:"private_ip"`
	AZ         string `json:"availability_zone"`
	LaunchTime string `json:"launch_time"`
}

// EC2Service provides methods for interacting with AWS EC2
type EC2Service struct {
	client *ec2.Client
}

// NewEC2Service creates a new EC2 service instance with the specified AWS profile and region
func NewEC2Service(profile, region string) (*EC2Service, error) {
	ctx := context.Background()

	var opts []func(*config.LoadOptions) error

	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &EC2Service{
		client: ec2.NewFromConfig(cfg),
	}, nil
}

// ListInstances retrieves all EC2 instances in the configured region
func (s *EC2Service) ListInstances(ctx context.Context) ([]EC2Instance, error) {
	result, err := s.client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe instances: %w", err)
	}

	var instances []EC2Instance
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			ec2Instance := EC2Instance{
				ID:        aws.ToString(instance.InstanceId),
				Type:      string(instance.InstanceType),
				State:     string(instance.State.Name),
				PrivateIP: aws.ToString(instance.PrivateIpAddress),
				PublicIP:  aws.ToString(instance.PublicIpAddress),
				AZ:        aws.ToString(instance.Placement.AvailabilityZone),
			}

			if instance.LaunchTime != nil {
				ec2Instance.LaunchTime = instance.LaunchTime.Format("2006-01-02 15:04:05")
			}

			ec2Instance.Name = getInstanceName(instance.Tags)

			instances = append(instances, ec2Instance)
		}
	}

	return instances, nil
}

func getInstanceName(tags []types.Tag) string {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return "-"
}

// OutputEC2Instances outputs EC2 instances in the specified format (json or table)
func OutputEC2Instances(instances []EC2Instance, format string) error {
	switch strings.ToLower(format) {
	case appconfig.FormatJSON:
		return outputJSON(instances)
	case appconfig.FormatTable:
		return outputTable(instances)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func outputJSON(instances []EC2Instance) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(instances)
}

func outputTable(instances []EC2Instance) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Name", "Type", "State", "Public IP", "Private IP", "AZ", "Launch Time"})
	table.SetBorder(false)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)

	for _, instance := range instances {
		row := []string{
			instance.ID,
			instance.Name,
			instance.Type,
			instance.State,
			instance.PublicIP,
			instance.PrivateIP,
			instance.AZ,
			instance.LaunchTime,
		}
		table.Append(row)
	}

	table.Render()
	return nil
}
