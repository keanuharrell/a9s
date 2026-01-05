// Package iam provides IAM service implementation for the a9s application.
package iam

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"

	awsfactory "github.com/keanuharrell/a9s/internal/aws"
	"github.com/keanuharrell/a9s/internal/core"
)

// =============================================================================
// High Risk Policies
// =============================================================================

var highRiskPolicies = []string{
	"AdministratorAccess",
	"PowerUserAccess",
	"IAMFullAccess",
	"SecurityAudit",
}

// =============================================================================
// Service Implementation
// =============================================================================

// Service implements IAM operations.
type Service struct {
	factory    *awsfactory.ClientFactory
	dispatcher core.EventDispatcher
	testClient IAMAPI
}

// IAMAPI defines the IAM client interface for mocking.
type IAMAPI interface {
	ListRoles(ctx context.Context, params *iam.ListRolesInput, optFns ...func(*iam.Options)) (*iam.ListRolesOutput, error)
	ListAttachedRolePolicies(ctx context.Context, params *iam.ListAttachedRolePoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedRolePoliciesOutput, error)
	ListRolePolicies(ctx context.Context, params *iam.ListRolePoliciesInput, optFns ...func(*iam.Options)) (*iam.ListRolePoliciesOutput, error)
	GetRole(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error)
}

// NewService creates a new IAM service.
func NewService(factory *awsfactory.ClientFactory, dispatcher core.EventDispatcher) *Service {
	return &Service{
		factory:    factory,
		dispatcher: dispatcher,
	}
}

// NewServiceWithClient creates a service with a custom client (for testing).
func NewServiceWithClient(client IAMAPI, dispatcher core.EventDispatcher) *Service {
	return &Service{
		testClient: client,
		dispatcher: dispatcher,
	}
}

// client returns the IAM client, fetching fresh from factory each time.
func (s *Service) client() IAMAPI {
	if s.testClient != nil {
		return s.testClient
	}
	return s.factory.IAMClient()
}

// =============================================================================
// AWSService Interface Implementation
// =============================================================================

// Name returns the service name.
func (s *Service) Name() string {
	return "iam"
}

// Description returns the service description.
func (s *Service) Description() string {
	return "IAM Role Security Audit"
}

// Icon returns the service icon.
func (s *Service) Icon() string {
	return "shield"
}

// Initialize sets up the service.
func (s *Service) Initialize(ctx context.Context, cfg *core.AWSConfig) error {
	return nil
}

// Close releases service resources.
func (s *Service) Close() error {
	return nil
}

// HealthCheck verifies the service can communicate with AWS.
func (s *Service) HealthCheck(ctx context.Context) error {
	_, err := s.client().ListRoles(ctx, &iam.ListRolesInput{
		MaxItems: aws.Int32(1),
	})
	if err != nil {
		return core.NewServiceError("iam", "health_check", err)
	}
	return nil
}

// =============================================================================
// ResourceLister Interface Implementation
// =============================================================================

// List returns IAM roles with basic info (fast).
// Detailed analysis is done via EnrichResource.
func (s *Service) List(ctx context.Context, opts core.ListOptions) ([]core.Resource, error) {
	input := &iam.ListRolesInput{}
	if opts.MaxResults > 0 {
		maxResults := opts.MaxResults
		if maxResults > 1000 {
			maxResults = 1000
		}
		input.MaxItems = aws.Int32(int32(maxResults)) //nolint:gosec // bounded above
	}

	result, err := s.client().ListRoles(ctx, input)
	if err != nil {
		s.dispatchError(ctx, "list", err)
		return nil, core.NewServiceError("iam", "list", err)
	}

	resources := make([]core.Resource, 0, len(result.Roles))
	for _, role := range result.Roles {
		roleName := aws.ToString(role.RoleName)

		resource := core.Resource{
			ID:    aws.ToString(role.RoleId),
			Type:  "iam:role",
			Name:  roleName,
			ARN:   aws.ToString(role.Arn),
			State: core.StatePending, // Not analyzed yet
			Tags:  make(map[string]string),
			Metadata: map[string]any{
				"policies":     []string{},
				"policy_count": 0,
				"is_high_risk": false,
				"risk_reason":  "",
				"path":         aws.ToString(role.Path),
				"analyzed":     false,
			},
		}

		if role.CreateDate != nil {
			resource.CreatedAt = role.CreateDate
			resource.Metadata["create_date"] = role.CreateDate.Format("2006-01-02")
		}

		// Extract tags
		for _, tag := range role.Tags {
			resource.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}

		resources = append(resources, resource)
	}

	// Dispatch event
	s.dispatchEvent(ctx, core.EventResourceListed, core.ResourceEventData{
		ResourceType: "iam:role",
		Count:        len(resources),
	})

	return resources, nil
}

// EnrichResource adds detailed policy analysis to a single role.
func (s *Service) EnrichResource(ctx context.Context, resource *core.Resource) error {
	roleName := resource.Name

	// Get attached policies (2 API calls per role)
	policies, err := s.getAttachedPolicies(ctx, roleName)
	if err != nil {
		policies = []string{}
	}

	// Assess risk
	isHighRisk, riskReason := assessRisk(policies)

	// Determine state based on risk
	state := core.StateActive
	if isHighRisk {
		state = core.StateWarning
	}

	// Update resource
	resource.State = state
	resource.Metadata["policies"] = policies
	resource.Metadata["policy_count"] = len(policies)
	resource.Metadata["is_high_risk"] = isHighRisk
	resource.Metadata["risk_reason"] = riskReason
	resource.Metadata["analyzed"] = true

	return nil
}

// =============================================================================
// ResourceGetter Interface Implementation
// =============================================================================

// Get returns a specific IAM role by name.
func (s *Service) Get(ctx context.Context, id string) (*core.Resource, error) {
	result, err := s.client().GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(id),
	})
	if err != nil {
		return nil, core.NewServiceError("iam", "get", err)
	}

	role := result.Role
	policies, _ := s.getAttachedPolicies(ctx, aws.ToString(role.RoleName))
	isHighRisk, riskReason := assessRisk(policies)

	state := core.StateActive
	if isHighRisk {
		state = core.StateWarning
	}

	resource := &core.Resource{
		ID:    aws.ToString(role.RoleId),
		Type:  "iam:role",
		Name:  aws.ToString(role.RoleName),
		ARN:   aws.ToString(role.Arn),
		State: state,
		Tags:  make(map[string]string),
		Metadata: map[string]any{
			"policies":     policies,
			"policy_count": len(policies),
			"is_high_risk": isHighRisk,
			"risk_reason":  riskReason,
			"path":         aws.ToString(role.Path),
			"description":  aws.ToString(role.Description),
		},
	}

	if role.CreateDate != nil {
		resource.CreatedAt = role.CreateDate
	}

	return resource, nil
}

// =============================================================================
// ActionExecutor Interface Implementation
// =============================================================================

// Actions returns the list of available actions for IAM.
func (s *Service) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "audit",
			Description: "Perform security audit on role",
			Icon:        "search",
			Shortcut:    "a",
			Dangerous:   false,
			Category:    "security",
		},
		{
			Name:        "view_policies",
			Description: "View attached policies",
			Icon:        "list",
			Shortcut:    "p",
			Dangerous:   false,
			Category:    "info",
		},
	}
}

// Execute runs the specified action on an IAM role.
func (s *Service) Execute(ctx context.Context, action string, resourceID string, params map[string]any) (*core.ActionResult, error) {
	start := time.Now()

	s.dispatchEvent(ctx, core.EventActionStarted, core.ActionEventData{
		Action:     action,
		ResourceID: resourceID,
		Params:     params,
	})

	var result *core.ActionResult
	var err error

	switch action {
	case "audit":
		result, err = s.auditRole(ctx, resourceID)
	case "view_policies":
		result, err = s.viewPolicies(ctx, resourceID)
	default:
		return nil, core.NewActionError(action, resourceID, core.ErrActionNotFound)
	}

	if err != nil {
		s.dispatchEvent(ctx, core.EventActionFailed, core.ActionEventData{
			Action:     action,
			ResourceID: resourceID,
			Error:      err.Error(),
		})
		return result, err
	}

	result.Duration = time.Since(start)

	s.dispatchEvent(ctx, core.EventActionExecuted, core.ActionEventData{
		Action:     action,
		ResourceID: resourceID,
		Result:     result,
	})

	return result, nil
}

// =============================================================================
// Action Implementations
// =============================================================================

func (s *Service) auditRole(ctx context.Context, roleName string) (*core.ActionResult, error) {
	policies, err := s.getAttachedPolicies(ctx, roleName)
	if err != nil {
		return core.NewActionResult(false, err.Error()), err
	}

	isHighRisk, riskReason := assessRisk(policies)

	result := core.NewActionResult(true, fmt.Sprintf("Audit complete for %s", roleName))
	result.Data = map[string]any{
		"role_name":    roleName,
		"policies":     policies,
		"is_high_risk": isHighRisk,
		"risk_reason":  riskReason,
	}

	return result, nil
}

func (s *Service) viewPolicies(ctx context.Context, roleName string) (*core.ActionResult, error) {
	policies, err := s.getAttachedPolicies(ctx, roleName)
	if err != nil {
		return core.NewActionResult(false, err.Error()), err
	}

	result := core.NewActionResult(true, fmt.Sprintf("Found %d policies", len(policies)))
	result.Data = map[string]any{
		"role_name": roleName,
		"policies":  policies,
	}

	return result, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

func (s *Service) getAttachedPolicies(ctx context.Context, roleName string) ([]string, error) {
	output, err := s.client().ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return nil, err
	}

	policies := make([]string, 0, len(output.AttachedPolicies))
	for _, policy := range output.AttachedPolicies {
		policies = append(policies, aws.ToString(policy.PolicyName))
	}

	// Also get inline policies
	inlineOutput, err := s.client().ListRolePolicies(ctx, &iam.ListRolePoliciesInput{
		RoleName: aws.String(roleName),
	})
	if err == nil {
		for _, policyName := range inlineOutput.PolicyNames {
			policies = append(policies, fmt.Sprintf("%s (inline)", policyName))
		}
	}

	return policies, nil
}

func assessRisk(policies []string) (bool, string) {
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

func (s *Service) dispatchEvent(ctx context.Context, eventType core.EventType, data any) {
	if s.dispatcher != nil {
		event := core.NewEvent(eventType, "iam", data)
		_ = s.dispatcher.Dispatch(ctx, event)
	}
}

func (s *Service) dispatchError(ctx context.Context, op string, err error) {
	if s.dispatcher != nil {
		event := core.NewEvent(core.EventError, "iam", map[string]string{
			"operation": op,
			"error":     err.Error(),
		})
		_ = s.dispatcher.Dispatch(ctx, event)
	}
}

// =============================================================================
// Interface Assertions
// =============================================================================

var (
	_ core.AWSService     = (*Service)(nil)
	_ core.ResourceLister = (*Service)(nil)
	_ core.ResourceGetter = (*Service)(nil)
	_ core.ActionExecutor = (*Service)(nil)
)
