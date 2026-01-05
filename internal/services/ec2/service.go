// Package ec2 provides EC2 service implementation for the a9s application.
package ec2

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	awsfactory "github.com/keanuharrell/a9s/internal/aws"
	"github.com/keanuharrell/a9s/internal/core"
)

// =============================================================================
// Service Implementation
// =============================================================================

// Service implements EC2 operations.
type Service struct {
	factory    *awsfactory.ClientFactory
	dispatcher core.EventDispatcher
	testClient EC2API // Only used for testing
}

// EC2API defines the EC2 client interface for mocking.
type EC2API interface {
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	StartInstances(ctx context.Context, params *ec2.StartInstancesInput, optFns ...func(*ec2.Options)) (*ec2.StartInstancesOutput, error)
	StopInstances(ctx context.Context, params *ec2.StopInstancesInput, optFns ...func(*ec2.Options)) (*ec2.StopInstancesOutput, error)
	TerminateInstances(ctx context.Context, params *ec2.TerminateInstancesInput, optFns ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error)
	RebootInstances(ctx context.Context, params *ec2.RebootInstancesInput, optFns ...func(*ec2.Options)) (*ec2.RebootInstancesOutput, error)
}

// NewService creates a new EC2 service.
func NewService(factory *awsfactory.ClientFactory, dispatcher core.EventDispatcher) *Service {
	return &Service{
		factory:    factory,
		dispatcher: dispatcher,
	}
}

// NewServiceWithClient creates a service with a custom client (for testing).
func NewServiceWithClient(client EC2API, dispatcher core.EventDispatcher) *Service {
	return &Service{
		testClient: client,
		dispatcher: dispatcher,
	}
}

// client returns the EC2 client, fetching fresh from factory each time.
func (s *Service) client() EC2API {
	if s.testClient != nil {
		return s.testClient
	}
	return s.factory.EC2Client()
}

// =============================================================================
// AWSService Interface Implementation
// =============================================================================

// Name returns the service name.
func (s *Service) Name() string {
	return "ec2"
}

// Description returns the service description.
func (s *Service) Description() string {
	return "EC2 Instance Management"
}

// Icon returns the service icon.
func (s *Service) Icon() string {
	return "server"
}

// Initialize sets up the service.
func (s *Service) Initialize(ctx context.Context, cfg *core.AWSConfig) error {
	// Already initialized via factory
	return nil
}

// Close releases service resources.
func (s *Service) Close() error {
	return nil
}

// HealthCheck verifies the service can communicate with AWS.
func (s *Service) HealthCheck(ctx context.Context) error {
	_, err := s.client().DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		MaxResults: aws.Int32(1),
	})
	if err != nil {
		return core.NewServiceError("ec2", "health_check", err)
	}
	return nil
}

// =============================================================================
// ResourceLister Interface Implementation
// =============================================================================

// List returns EC2 instances matching the given options.
func (s *Service) List(ctx context.Context, opts core.ListOptions) ([]core.Resource, error) {
	start := time.Now()

	input := &ec2.DescribeInstancesInput{}

	// Apply filters
	if len(opts.Filters) > 0 {
		for key, value := range opts.Filters {
			input.Filters = append(input.Filters, types.Filter{
				Name:   aws.String(filterKeyToAWS(key)),
				Values: []string{value},
			})
		}
	}

	// Apply max results (capped to AWS limit)
	if opts.MaxResults > 0 {
		maxResults := opts.MaxResults
		if maxResults > 1000 {
			maxResults = 1000
		}
		input.MaxResults = aws.Int32(int32(maxResults)) //nolint:gosec // bounded above
	}

	// Apply pagination token
	if opts.NextToken != "" {
		input.NextToken = aws.String(opts.NextToken)
	}

	result, err := s.client().DescribeInstances(ctx, input)
	if err != nil {
		s.dispatchError(ctx, "list", err)
		return nil, core.NewServiceError("ec2", "list", err)
	}

	resources := make([]core.Resource, 0)
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			resources = append(resources, instanceToResource(instance))
		}
	}

	// Dispatch event
	s.dispatchEvent(ctx, core.EventResourceListed, core.ResourceEventData{
		ResourceType: "ec2:instance",
		Count:        len(resources),
	})

	// Log timing
	_ = time.Since(start)

	return resources, nil
}

// =============================================================================
// ResourceGetter Interface Implementation
// =============================================================================

// Get returns a specific EC2 instance by ID.
func (s *Service) Get(ctx context.Context, id string) (*core.Resource, error) {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{id},
	}

	result, err := s.client().DescribeInstances(ctx, input)
	if err != nil {
		return nil, core.NewServiceError("ec2", "get", err)
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return nil, core.ErrResourceNotFound
	}

	resource := instanceToResource(result.Reservations[0].Instances[0])
	return &resource, nil
}

// =============================================================================
// ActionExecutor Interface Implementation
// =============================================================================

// Actions returns the list of available actions for EC2.
func (s *Service) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "start",
			Description: "Start a stopped instance",
			Icon:        "play",
			Shortcut:    "s",
			Dangerous:   false,
			Category:    "lifecycle",
		},
		{
			Name:        "stop",
			Description: "Stop a running instance",
			Icon:        "stop",
			Shortcut:    "t",
			Dangerous:   false,
			Category:    "lifecycle",
		},
		{
			Name:        "reboot",
			Description: "Reboot an instance",
			Icon:        "refresh",
			Shortcut:    "b",
			Dangerous:   false,
			Category:    "lifecycle",
		},
		{
			Name:        "terminate",
			Description: "Terminate an instance (permanent)",
			Icon:        "trash",
			Shortcut:    "x",
			Dangerous:   true,
			Category:    "lifecycle",
			Parameters: []core.ActionParameter{
				{
					Name:        "confirm",
					Type:        "bool",
					Required:    true,
					Description: "Confirm termination",
				},
			},
		},
	}
}

// Execute runs the specified action on an EC2 instance.
func (s *Service) Execute(ctx context.Context, action string, resourceID string, params map[string]any) (*core.ActionResult, error) {
	start := time.Now()

	// Dispatch action started event
	s.dispatchEvent(ctx, core.EventActionStarted, core.ActionEventData{
		Action:     action,
		ResourceID: resourceID,
		Params:     params,
	})

	var result *core.ActionResult
	var err error

	switch action {
	case "start":
		result, err = s.startInstance(ctx, resourceID)
	case "stop":
		result, err = s.stopInstance(ctx, resourceID)
	case "reboot":
		result, err = s.rebootInstance(ctx, resourceID)
	case "terminate":
		if confirmed, _ := params["confirm"].(bool); !confirmed {
			return core.NewActionResult(false, "Termination not confirmed"), core.ErrConfirmationRequired
		}
		result, err = s.terminateInstance(ctx, resourceID)
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

	// Dispatch action executed event
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

func (s *Service) startInstance(ctx context.Context, instanceID string) (*core.ActionResult, error) {
	_, err := s.client().StartInstances(ctx, &ec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return core.NewActionResult(false, err.Error()), core.NewActionError("start", instanceID, err)
	}

	return core.NewActionResult(true, fmt.Sprintf("Instance %s is starting", instanceID)), nil
}

func (s *Service) stopInstance(ctx context.Context, instanceID string) (*core.ActionResult, error) {
	_, err := s.client().StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return core.NewActionResult(false, err.Error()), core.NewActionError("stop", instanceID, err)
	}

	return core.NewActionResult(true, fmt.Sprintf("Instance %s is stopping", instanceID)), nil
}

func (s *Service) rebootInstance(ctx context.Context, instanceID string) (*core.ActionResult, error) {
	_, err := s.client().RebootInstances(ctx, &ec2.RebootInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return core.NewActionResult(false, err.Error()), core.NewActionError("reboot", instanceID, err)
	}

	return core.NewActionResult(true, fmt.Sprintf("Instance %s is rebooting", instanceID)), nil
}

func (s *Service) terminateInstance(ctx context.Context, instanceID string) (*core.ActionResult, error) {
	_, err := s.client().TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return core.NewActionResult(false, err.Error()), core.NewActionError("terminate", instanceID, err)
	}

	return core.NewActionResult(true, fmt.Sprintf("Instance %s is terminating", instanceID)), nil
}

// =============================================================================
// Helper Functions
// =============================================================================

func instanceToResource(instance types.Instance) core.Resource {
	resource := core.Resource{
		ID:     aws.ToString(instance.InstanceId),
		Type:   "ec2:instance",
		Region: extractRegionFromAZ(aws.ToString(instance.Placement.AvailabilityZone)),
		State:  string(instance.State.Name),
		Tags:   make(map[string]string),
		Metadata: map[string]any{
			"instance_type":     string(instance.InstanceType),
			"availability_zone": aws.ToString(instance.Placement.AvailabilityZone),
			"public_ip":         aws.ToString(instance.PublicIpAddress),
			"private_ip":        aws.ToString(instance.PrivateIpAddress),
			"vpc_id":            aws.ToString(instance.VpcId),
			"subnet_id":         aws.ToString(instance.SubnetId),
			"architecture":      string(instance.Architecture),
			"platform":          aws.ToString(instance.PlatformDetails),
		},
	}

	// Extract tags
	for _, tag := range instance.Tags {
		key := aws.ToString(tag.Key)
		value := aws.ToString(tag.Value)
		resource.Tags[key] = value
		if key == "Name" {
			resource.Name = value
		}
	}

	// Set name to instance ID if no Name tag
	if resource.Name == "" {
		resource.Name = resource.ID
	}

	// Set timestamps
	if instance.LaunchTime != nil {
		resource.CreatedAt = instance.LaunchTime
		resource.Metadata["launch_time"] = instance.LaunchTime.Format(time.RFC3339)
	}

	return resource
}

func extractRegionFromAZ(az string) string {
	if len(az) > 0 {
		// Remove the last character (zone letter) to get the region
		return az[:len(az)-1]
	}
	return ""
}

func filterKeyToAWS(key string) string {
	// Map common filter keys to AWS filter names
	filterMap := map[string]string{
		"state":         "instance-state-name",
		"type":          "instance-type",
		"vpc":           "vpc-id",
		"subnet":        "subnet-id",
		"az":            "availability-zone",
		"architecture":  "architecture",
		"platform":      "platform",
	}

	if awsKey, ok := filterMap[key]; ok {
		return awsKey
	}
	return key
}

func (s *Service) dispatchEvent(ctx context.Context, eventType core.EventType, data any) {
	if s.dispatcher != nil {
		event := core.NewEvent(eventType, "ec2", data)
		_ = s.dispatcher.Dispatch(ctx, event)
	}
}

func (s *Service) dispatchError(ctx context.Context, op string, err error) {
	if s.dispatcher != nil {
		event := core.NewEvent(core.EventError, "ec2", map[string]string{
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
