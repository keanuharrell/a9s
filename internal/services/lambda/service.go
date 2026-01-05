// Package lambda provides Lambda service implementation for the a9s application.
package lambda

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"

	awsfactory "github.com/keanuharrell/a9s/internal/aws"
	"github.com/keanuharrell/a9s/internal/core"
)

// =============================================================================
// Service Implementation
// =============================================================================

// Service implements Lambda operations.
type Service struct {
	factory    *awsfactory.ClientFactory
	dispatcher core.EventDispatcher
	testClient LambdaAPI
}

// LambdaAPI defines the Lambda client interface for mocking.
type LambdaAPI interface {
	ListFunctions(ctx context.Context, params *lambda.ListFunctionsInput, optFns ...func(*lambda.Options)) (*lambda.ListFunctionsOutput, error)
	GetFunction(ctx context.Context, params *lambda.GetFunctionInput, optFns ...func(*lambda.Options)) (*lambda.GetFunctionOutput, error)
	Invoke(ctx context.Context, params *lambda.InvokeInput, optFns ...func(*lambda.Options)) (*lambda.InvokeOutput, error)
}

// NewService creates a new Lambda service.
func NewService(factory *awsfactory.ClientFactory, dispatcher core.EventDispatcher) *Service {
	return &Service{
		factory:    factory,
		dispatcher: dispatcher,
	}
}

// NewServiceWithClient creates a service with a custom client (for testing).
func NewServiceWithClient(client LambdaAPI, dispatcher core.EventDispatcher) *Service {
	return &Service{
		testClient: client,
		dispatcher: dispatcher,
	}
}

// client returns the Lambda client, fetching fresh from factory each time.
func (s *Service) client() LambdaAPI {
	if s.testClient != nil {
		return s.testClient
	}
	return lambda.NewFromConfig(s.factory.Config())
}

// =============================================================================
// AWSService Interface Implementation
// =============================================================================

// Name returns the service name.
func (s *Service) Name() string {
	return "lambda"
}

// Description returns the service description.
func (s *Service) Description() string {
	return "AWS Lambda Functions"
}

// Icon returns the service icon.
func (s *Service) Icon() string {
	return "function"
}

// Initialize sets up the service.
func (s *Service) Initialize(_ context.Context, _ *core.AWSConfig) error {
	return nil
}

// Close releases service resources.
func (s *Service) Close() error {
	return nil
}

// HealthCheck verifies the service can communicate with AWS.
func (s *Service) HealthCheck(ctx context.Context) error {
	_, err := s.client().ListFunctions(ctx, &lambda.ListFunctionsInput{
		MaxItems: aws.Int32(1),
	})
	if err != nil {
		return core.NewServiceError("lambda", "health_check", err)
	}
	return nil
}

// =============================================================================
// ResourceLister Interface Implementation
// =============================================================================

// List returns Lambda functions.
func (s *Service) List(ctx context.Context, opts core.ListOptions) ([]core.Resource, error) {
	start := time.Now()

	input := &lambda.ListFunctionsInput{}
	if opts.MaxResults > 0 {
		maxResults := opts.MaxResults
		if maxResults > 1000 {
			maxResults = 1000
		}
		input.MaxItems = aws.Int32(int32(maxResults)) //nolint:gosec // bounded above
	}

	result, err := s.client().ListFunctions(ctx, input)
	if err != nil {
		s.dispatchError(ctx, "list", err)
		return nil, core.NewServiceError("lambda", "list", err)
	}

	resources := make([]core.Resource, 0, len(result.Functions))
	for _, fn := range result.Functions {
		resource := s.functionToResource(fn)
		resources = append(resources, resource)
	}

	// Dispatch event
	s.dispatchEvent(ctx, core.EventResourceListed, core.ResourceEventData{
		ResourceType: "lambda:function",
		Count:        len(resources),
	})

	_ = time.Since(start)

	return resources, nil
}

func (s *Service) functionToResource(fn types.FunctionConfiguration) core.Resource {
	runtime := string(fn.Runtime)
	if runtime == "" {
		runtime = "unknown"
	}

	memoryMB := int32(0)
	if fn.MemorySize != nil {
		memoryMB = *fn.MemorySize
	}

	timeout := int32(0)
	if fn.Timeout != nil {
		timeout = *fn.Timeout
	}

	resource := core.Resource{
		ID:    aws.ToString(fn.FunctionArn),
		Type:  "lambda:function",
		Name:  aws.ToString(fn.FunctionName),
		ARN:   aws.ToString(fn.FunctionArn),
		State: core.StateActive,
		Tags:  make(map[string]string),
		Metadata: map[string]any{
			"runtime":       runtime,
			"memory_mb":     memoryMB,
			"timeout_sec":   timeout,
			"handler":       aws.ToString(fn.Handler),
			"code_size":     fn.CodeSize,
			"description":   aws.ToString(fn.Description),
			"last_modified": aws.ToString(fn.LastModified),
		},
	}

	return resource
}

// =============================================================================
// ResourceGetter Interface Implementation
// =============================================================================

// Get returns a specific Lambda function by name.
func (s *Service) Get(ctx context.Context, name string) (*core.Resource, error) {
	result, err := s.client().GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(name),
	})
	if err != nil {
		return nil, core.NewServiceError("lambda", "get", err)
	}

	config := result.Configuration
	resource := &core.Resource{
		ID:    aws.ToString(config.FunctionArn),
		Type:  "lambda:function",
		Name:  aws.ToString(config.FunctionName),
		ARN:   aws.ToString(config.FunctionArn),
		State: core.StateActive,
		Tags:  make(map[string]string),
		Metadata: map[string]any{
			"runtime":       string(config.Runtime),
			"memory_mb":     config.MemorySize,
			"timeout_sec":   config.Timeout,
			"handler":       aws.ToString(config.Handler),
			"code_size":     config.CodeSize,
			"description":   aws.ToString(config.Description),
			"last_modified": aws.ToString(config.LastModified),
		},
	}

	return resource, nil
}

// =============================================================================
// ActionExecutor Interface Implementation
// =============================================================================

// Actions returns the list of available actions for Lambda.
func (s *Service) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "invoke",
			Description: "Invoke the function",
			Icon:        "play",
			Shortcut:    "i",
			Dangerous:   false,
			Category:    "execute",
		},
		{
			Name:        "view_config",
			Description: "View function configuration",
			Icon:        "info",
			Shortcut:    "c",
			Dangerous:   false,
			Category:    "info",
		},
	}
}

// Execute runs the specified action on a Lambda function.
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
	case "invoke":
		result, err = s.invokeFunction(ctx, resourceID, params)
	case "view_config":
		result, err = s.viewConfig(ctx, resourceID)
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

func (s *Service) invokeFunction(ctx context.Context, functionName string, params map[string]any) (*core.ActionResult, error) {
	input := &lambda.InvokeInput{
		FunctionName: aws.String(functionName),
	}

	// Add payload if provided
	if payload, ok := params["payload"].([]byte); ok {
		input.Payload = payload
	}

	result, err := s.client().Invoke(ctx, input)
	if err != nil {
		return core.NewActionResult(false, err.Error()), err
	}

	actionResult := core.NewActionResult(true, fmt.Sprintf("Function invoked successfully, status: %d", result.StatusCode))
	actionResult.Data = map[string]any{
		"status_code": result.StatusCode,
		"payload":     string(result.Payload),
	}

	return actionResult, nil
}

func (s *Service) viewConfig(ctx context.Context, functionName string) (*core.ActionResult, error) {
	result, err := s.client().GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(functionName),
	})
	if err != nil {
		return core.NewActionResult(false, err.Error()), err
	}

	config := result.Configuration
	actionResult := core.NewActionResult(true, fmt.Sprintf("Configuration for %s", aws.ToString(config.FunctionName)))
	actionResult.Data = map[string]any{
		"runtime":     string(config.Runtime),
		"handler":     aws.ToString(config.Handler),
		"memory_mb":   config.MemorySize,
		"timeout_sec": config.Timeout,
		"description": aws.ToString(config.Description),
	}

	return actionResult, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

func (s *Service) dispatchEvent(ctx context.Context, eventType core.EventType, data any) {
	if s.dispatcher != nil {
		event := core.NewEvent(eventType, "lambda", data)
		_ = s.dispatcher.Dispatch(ctx, event)
	}
}

func (s *Service) dispatchError(ctx context.Context, op string, err error) {
	if s.dispatcher != nil {
		event := core.NewEvent(core.EventError, "lambda", map[string]string{
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
