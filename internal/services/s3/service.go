// Package s3 provides S3 service implementation for the a9s application.
package s3

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	awsfactory "github.com/keanuharrell/a9s/internal/aws"
	"github.com/keanuharrell/a9s/internal/core"
)

// =============================================================================
// Service Implementation
// =============================================================================

// Service implements S3 operations.
type Service struct {
	factory    *awsfactory.ClientFactory
	dispatcher core.EventDispatcher
	testClient S3API
}

// S3API defines the S3 client interface for mocking.
type S3API interface {
	ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error)
	GetBucketLocation(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	GetPublicAccessBlock(ctx context.Context, params *s3.GetPublicAccessBlockInput, optFns ...func(*s3.Options)) (*s3.GetPublicAccessBlockOutput, error)
	GetBucketTagging(ctx context.Context, params *s3.GetBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error)
	DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error)
	DeleteBucket(ctx context.Context, params *s3.DeleteBucketInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, error)
}

// NewService creates a new S3 service.
func NewService(factory *awsfactory.ClientFactory, dispatcher core.EventDispatcher) *Service {
	return &Service{
		factory:    factory,
		dispatcher: dispatcher,
	}
}

// NewServiceWithClient creates a service with a custom client (for testing).
func NewServiceWithClient(client S3API, dispatcher core.EventDispatcher) *Service {
	return &Service{
		testClient: client,
		dispatcher: dispatcher,
	}
}

// client returns the S3 client, fetching fresh from factory each time.
func (s *Service) client() S3API {
	if s.testClient != nil {
		return s.testClient
	}
	return s.factory.S3Client()
}

// =============================================================================
// AWSService Interface Implementation
// =============================================================================

// Name returns the service name.
func (s *Service) Name() string {
	return "s3"
}

// Description returns the service description.
func (s *Service) Description() string {
	return "S3 Bucket Management"
}

// Icon returns the service icon.
func (s *Service) Icon() string {
	return "bucket"
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
	_, err := s.client().ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return core.NewServiceError("s3", "health_check", err)
	}
	return nil
}

// =============================================================================
// ResourceLister Interface Implementation
// =============================================================================

// List returns S3 buckets with basic info (fast).
// Detailed analysis is done via EnrichResource or ListWithEnrichment.
func (s *Service) List(ctx context.Context, opts core.ListOptions) ([]core.Resource, error) {
	result, err := s.client().ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		s.dispatchError(ctx, "list", err)
		return nil, core.NewServiceError("s3", "list", err)
	}

	resources := make([]core.Resource, 0, len(result.Buckets))
	for _, bucket := range result.Buckets {
		bucketName := aws.ToString(bucket.Name)

		resource := core.Resource{
			ID:     bucketName,
			Type:   "s3:bucket",
			Name:   bucketName,
			Region: "loading...",
			State:  core.StatePending, // Not analyzed yet
			Tags:   make(map[string]string),
			Metadata: map[string]any{
				"is_empty":       false,
				"object_count":   0,
				"size_bytes":     int64(0),
				"size_human":     "...",
				"is_public":      false,
				"has_tags":       false,
				"should_cleanup": false,
				"cleanup_reason": "",
				"analyzed":       false,
			},
		}

		if bucket.CreationDate != nil {
			resource.CreatedAt = bucket.CreationDate
			resource.Metadata["created_date"] = bucket.CreationDate.Format("2006-01-02")
		}

		resources = append(resources, resource)
	}

	// Dispatch event
	s.dispatchEvent(ctx, core.EventResourceListed, core.ResourceEventData{
		ResourceType: "s3:bucket",
		Count:        len(resources),
	})

	return resources, nil
}

// EnrichResource adds detailed analysis to a single bucket.
func (s *Service) EnrichResource(ctx context.Context, resource *core.Resource) error {
	bucketName := resource.Name

	// Get bucket details (3 API calls per bucket - no ListObjectsV2 to avoid costs)
	region := s.getBucketRegion(ctx, bucketName)
	isPublic := s.isBucketPublic(ctx, bucketName)
	hasTags := s.hasTags(ctx, bucketName)

	// Determine cleanup status
	shouldCleanup, cleanupReason := s.shouldCleanup(isPublic, hasTags)

	// Determine state
	state := core.StateActive
	if shouldCleanup {
		state = core.StateWarning
	}

	// Update resource
	resource.Region = region
	resource.State = state
	resource.Metadata["is_public"] = isPublic
	resource.Metadata["has_tags"] = hasTags
	resource.Metadata["should_cleanup"] = shouldCleanup
	resource.Metadata["cleanup_reason"] = cleanupReason
	resource.Metadata["analyzed"] = true

	return nil
}

// ListWithEnrichment returns a channel that streams enriched resources.
func (s *Service) ListWithEnrichment(ctx context.Context, opts core.ListOptions) (<-chan core.ResourceUpdate, error) {
	// First get basic list
	resources, err := s.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	updateChan := make(chan core.ResourceUpdate, len(resources))

	// Send initial batch
	go func() {
		defer close(updateChan)

		// Send all basic resources first
		updateChan <- core.ResourceUpdate{
			Type:      core.UpdateTypeBatch,
			Resources: resources,
		}

		// Then enrich each one
		for i := range resources {
			select {
			case <-ctx.Done():
				return
			default:
				if err := s.EnrichResource(ctx, &resources[i]); err == nil {
					updateChan <- core.ResourceUpdate{
						Type:     core.UpdateTypeSingle,
						Resource: &resources[i],
						Index:    i,
					}
				}
			}
		}
	}()

	return updateChan, nil
}

// =============================================================================
// ResourceGetter Interface Implementation
// =============================================================================

// Get returns a specific S3 bucket by name.
func (s *Service) Get(ctx context.Context, id string) (*core.Resource, error) {
	// For S3, we need to list and find the bucket
	resources, err := s.List(ctx, core.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, r := range resources {
		if r.Name == id {
			return &r, nil
		}
	}

	return nil, core.ErrResourceNotFound
}

// =============================================================================
// ResourceMutator Interface Implementation
// =============================================================================

// Delete removes an S3 bucket.
func (s *Service) Delete(ctx context.Context, id string) error {
	// First, delete all objects
	listResult, err := s.client().ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(id),
	})
	if err != nil {
		return core.NewServiceError("s3", "delete", err)
	}

	if len(listResult.Contents) > 0 {
		var objectIDs []types.ObjectIdentifier
		for _, obj := range listResult.Contents {
			objectIDs = append(objectIDs, types.ObjectIdentifier{
				Key: obj.Key,
			})
		}

		_, err = s.client().DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(id),
			Delete: &types.Delete{
				Objects: objectIDs,
			},
		})
		if err != nil {
			return core.NewServiceError("s3", "delete_objects", err)
		}
	}

	// Then delete the bucket
	_, err = s.client().DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(id),
	})
	if err != nil {
		return core.NewServiceError("s3", "delete_bucket", err)
	}

	s.dispatchEvent(ctx, core.EventResourceDeleted, core.ResourceEventData{
		ResourceID:   id,
		ResourceType: "s3:bucket",
	})

	return nil
}

// =============================================================================
// ActionExecutor Interface Implementation
// =============================================================================

// Actions returns the list of available actions for S3.
func (s *Service) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "analyze",
			Description: "Analyze bucket contents and usage",
			Icon:        "search",
			Shortcut:    "a",
			Dangerous:   false,
			Category:    "info",
		},
		{
			Name:        "delete",
			Description: "Delete bucket and all contents",
			Icon:        "trash",
			Shortcut:    "d",
			Dangerous:   true,
			Category:    "lifecycle",
			Parameters: []core.ActionParameter{
				{
					Name:        "confirm",
					Type:        "bool",
					Required:    true,
					Description: "Confirm deletion",
				},
			},
		},
	}
}

// Execute runs the specified action on an S3 bucket.
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
	case "analyze":
		result, err = s.analyzeBucket(ctx, resourceID)
	case "delete":
		if confirmed, _ := params["confirm"].(bool); !confirmed {
			return core.NewActionResult(false, "Deletion not confirmed"), core.ErrConfirmationRequired
		}
		result, err = s.deleteBucket(ctx, resourceID)
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

func (s *Service) analyzeBucket(ctx context.Context, bucketName string) (*core.ActionResult, error) {
	isPublic := s.isBucketPublic(ctx, bucketName)
	hasTags := s.hasTags(ctx, bucketName)
	shouldCleanup, cleanupReason := s.shouldCleanup(isPublic, hasTags)

	result := core.NewActionResult(true, fmt.Sprintf("Analysis complete for %s", bucketName))
	result.Data = map[string]any{
		"bucket_name":    bucketName,
		"is_public":      isPublic,
		"has_tags":       hasTags,
		"should_cleanup": shouldCleanup,
		"cleanup_reason": cleanupReason,
	}

	return result, nil
}

func (s *Service) deleteBucket(ctx context.Context, bucketName string) (*core.ActionResult, error) {
	if err := s.Delete(ctx, bucketName); err != nil {
		return core.NewActionResult(false, err.Error()), err
	}

	return core.NewActionResult(true, fmt.Sprintf("Bucket %s deleted successfully", bucketName)), nil
}

// =============================================================================
// Helper Functions
// =============================================================================

func (s *Service) getBucketRegion(ctx context.Context, bucketName string) string {
	location, err := s.client().GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return "unknown"
	}

	if location.LocationConstraint == "" {
		return "us-east-1"
	}
	return string(location.LocationConstraint)
}

func (s *Service) isBucketPublic(ctx context.Context, bucketName string) bool {
	_, err := s.client().GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{
		Bucket: aws.String(bucketName),
	})
	// If we can't get public access block, assume it might be public
	return err != nil
}

func (s *Service) hasTags(ctx context.Context, bucketName string) bool {
	tags, err := s.client().GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucketName),
	})
	return err == nil && len(tags.TagSet) > 0
}

func (s *Service) shouldCleanup(isPublic, hasTags bool) (bool, string) {
	var reasons []string

	if isPublic && !hasTags {
		reasons = append(reasons, "public without tags")
	}

	if !hasTags {
		reasons = append(reasons, "untagged")
	}

	if len(reasons) > 0 {
		return true, strings.Join(reasons, ", ")
	}

	return false, ""
}

func (s *Service) dispatchEvent(ctx context.Context, eventType core.EventType, data any) {
	if s.dispatcher != nil {
		event := core.NewEvent(eventType, "s3", data)
		_ = s.dispatcher.Dispatch(ctx, event)
	}
}

func (s *Service) dispatchError(ctx context.Context, op string, err error) {
	if s.dispatcher != nil {
		event := core.NewEvent(core.EventError, "s3", map[string]string{
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
