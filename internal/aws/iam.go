package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/olekukonko/tablewriter"
)

type IAMAuditResult struct {
	Roles         []IAMRole    `json:"roles"`
	HighRiskCount int          `json:"high_risk_count"`
	Summary       AuditSummary `json:"summary"`
}

type IAMRole struct {
	Name             string   `json:"name"`
	ARN              string   `json:"arn"`
	CreateDate       string   `json:"create_date"`
	AttachedPolicies []string `json:"attached_policies"`
	IsHighRisk       bool     `json:"is_high_risk"`
	RiskReason       string   `json:"risk_reason,omitempty"`
}

type AuditSummary struct {
	TotalRoles           int      `json:"total_roles"`
	HighRiskRoles        int      `json:"high_risk_roles"`
	RolesWithAdminAccess []string `json:"roles_with_admin_access"`
	RolesWithWildcards   []string `json:"roles_with_wildcards"`
}

type IAMService struct {
	client *iam.Client
}

var highRiskPolicies = []string{
	"AdministratorAccess",
	"PowerUserAccess",
	"IAMFullAccess",
	"SecurityAudit",
}

func NewIAMService(profile, region string) (*IAMService, error) {
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

	return &IAMService{
		client: iam.NewFromConfig(cfg),
	}, nil
}

func (s *IAMService) AuditRoles(ctx context.Context) (*IAMAuditResult, error) {
	rolesOutput, err := s.client.ListRoles(ctx, &iam.ListRolesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	result := &IAMAuditResult{
		Roles: []IAMRole{},
		Summary: AuditSummary{
			RolesWithAdminAccess: []string{},
			RolesWithWildcards:   []string{},
		},
	}

	for _, role := range rolesOutput.Roles {
		iamRole := IAMRole{
			Name:             aws.ToString(role.RoleName),
			ARN:              aws.ToString(role.Arn),
			AttachedPolicies: []string{},
		}

		if role.CreateDate != nil {
			iamRole.CreateDate = role.CreateDate.Format("2006-01-02")
		}

		policies, err := s.getAttachedPolicies(ctx, aws.ToString(role.RoleName))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to get policies for role %s: %v\n",
				aws.ToString(role.RoleName), err)
			continue
		}

		iamRole.AttachedPolicies = policies
		iamRole.IsHighRisk, iamRole.RiskReason = s.assessRisk(policies)

		if iamRole.IsHighRisk {
			result.HighRiskCount++
			result.Summary.HighRiskRoles++

			if strings.Contains(iamRole.RiskReason, "Administrator") {
				result.Summary.RolesWithAdminAccess = append(
					result.Summary.RolesWithAdminAccess, iamRole.Name)
			}
			if strings.Contains(iamRole.RiskReason, "wildcard") {
				result.Summary.RolesWithWildcards = append(
					result.Summary.RolesWithWildcards, iamRole.Name)
			}
		}

		result.Roles = append(result.Roles, iamRole)
	}

	result.Summary.TotalRoles = len(result.Roles)

	return result, nil
}

func (s *IAMService) getAttachedPolicies(ctx context.Context, roleName string) ([]string, error) {
	output, err := s.client.ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return nil, err
	}

	var policies []string
	for _, policy := range output.AttachedPolicies {
		policies = append(policies, aws.ToString(policy.PolicyName))
	}

	inlineOutput, err := s.client.ListRolePolicies(ctx, &iam.ListRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err == nil {
		for _, policyName := range inlineOutput.PolicyNames {
			policies = append(policies, fmt.Sprintf("%s (inline)", policyName))
		}
	}

	return policies, nil
}

func (s *IAMService) assessRisk(policies []string) (bool, string) {
	for _, policy := range policies {
		for _, highRisk := range highRiskPolicies {
			if strings.Contains(policy, highRisk) {
				return true, fmt.Sprintf("Has %s policy", highRisk)
			}
		}

		if strings.Contains(policy, "*") {
			return true, "Contains wildcard permissions"
		}
	}

	return false, ""
}

func OutputIAMAudit(result *IAMAuditResult, format string) error {
	switch strings.ToLower(format) {
	case "json":
		return outputIAMJSON(result)
	case "table":
		return outputIAMTable(result)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func outputIAMJSON(result *IAMAuditResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func outputIAMTable(result *IAMAuditResult) error {
	fmt.Printf("\n=== IAM Audit Summary ===\n")
	fmt.Printf("Total Roles: %d\n", result.Summary.TotalRoles)
	fmt.Printf("High Risk Roles: %d\n", result.Summary.HighRiskRoles)

	if len(result.Summary.RolesWithAdminAccess) > 0 {
		fmt.Printf("Roles with Admin Access: %s\n",
			strings.Join(result.Summary.RolesWithAdminAccess, ", "))
	}

	fmt.Printf("\n=== Role Details ===\n")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Role Name", "Created", "Policies", "Risk Level", "Risk Reason"})
	table.SetBorder(false)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, role := range result.Roles {
		riskLevel := "Low"
		if role.IsHighRisk {
			riskLevel = "HIGH"
		}

		policies := strings.Join(role.AttachedPolicies, ", ")
		if len(policies) > 50 {
			policies = policies[:47] + "..."
		}

		row := []string{
			role.Name,
			role.CreateDate,
			policies,
			riskLevel,
			role.RiskReason,
		}
		table.Append(row)
	}

	table.Render()
	return nil
}
