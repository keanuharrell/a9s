// Package aws provides a factory for creating AWS service clients.
// It centralizes AWS configuration and client creation to avoid code duplication.
package aws

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/keanuharrell/a9s/internal/core"
)

// ClientFactory creates AWS service clients with shared configuration.
type ClientFactory struct {
	mu      sync.RWMutex
	cfg     aws.Config
	profile string
	region  string
	loaded  bool
}

// NewClientFactory creates a new AWS client factory.
func NewClientFactory(awsCfg *core.AWSConfig) (*ClientFactory, error) {
	factory := &ClientFactory{
		profile: awsCfg.Profile,
		region:  awsCfg.Region,
	}

	if err := factory.loadConfig(context.Background()); err != nil {
		return nil, err
	}

	return factory, nil
}

// loadConfig loads the AWS configuration.
func (f *ClientFactory) loadConfig(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.loaded {
		return nil
	}

	var opts []func(*config.LoadOptions) error

	if f.region != "" {
		opts = append(opts, config.WithRegion(f.region))
	}

	if f.profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(f.profile))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrAWSConfigFailed, err)
	}

	f.cfg = cfg
	f.loaded = true

	return nil
}

// Config returns the AWS configuration.
func (f *ClientFactory) Config() aws.Config {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.cfg
}

// Region returns the configured region.
func (f *ClientFactory) Region() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.region
}

// Profile returns the configured profile.
func (f *ClientFactory) Profile() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.profile
}

// Reload reloads the AWS configuration.
func (f *ClientFactory) Reload(ctx context.Context) error {
	f.mu.Lock()
	f.loaded = false
	f.mu.Unlock()

	return f.loadConfig(ctx)
}

// UpdateConfig updates the factory configuration and reloads.
func (f *ClientFactory) UpdateConfig(ctx context.Context, profile, region string) error {
	f.mu.Lock()
	f.profile = profile
	f.region = region
	f.loaded = false
	f.mu.Unlock()

	return f.loadConfig(ctx)
}

// =============================================================================
// Service Client Factories
// =============================================================================

// EC2Client creates an EC2 client.
func (f *ClientFactory) EC2Client() *ec2.Client {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return ec2.NewFromConfig(f.cfg)
}

// IAMClient creates an IAM client.
func (f *ClientFactory) IAMClient() *iam.Client {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return iam.NewFromConfig(f.cfg)
}

// S3Client creates an S3 client.
func (f *ClientFactory) S3Client() *s3.Client {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return s3.NewFromConfig(f.cfg)
}

// =============================================================================
// Generic Client Creation
// =============================================================================

// ClientType represents a type of AWS service client.
type ClientType string

const (
	ClientTypeEC2 ClientType = "ec2"
	ClientTypeIAM ClientType = "iam"
	ClientTypeS3  ClientType = "s3"
)

// Client returns an AWS client of the specified type.
func (f *ClientFactory) Client(clientType ClientType) (any, error) {
	switch clientType {
	case ClientTypeEC2:
		return f.EC2Client(), nil
	case ClientTypeIAM:
		return f.IAMClient(), nil
	case ClientTypeS3:
		return f.S3Client(), nil
	default:
		return nil, fmt.Errorf("unknown client type: %s", clientType)
	}
}

// =============================================================================
// Health Check
// =============================================================================

// HealthCheck verifies AWS connectivity by checking STS.
func (f *ClientFactory) HealthCheck(ctx context.Context) error {
	// Use EC2 DescribeRegions as a lightweight health check
	client := f.EC2Client()
	_, err := client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrAWSServiceError, err)
	}
	return nil
}

// =============================================================================
// Utility Functions
// =============================================================================

// StringPtr returns a pointer to a string value.
func StringPtr(v string) *string {
	return &v
}

// Int32Ptr returns a pointer to an int32 value.
func Int32Ptr(v int32) *int32 {
	return &v
}

// BoolPtr returns a pointer to a bool value.
func BoolPtr(v bool) *bool {
	return &v
}

// StringValue returns the value of a string pointer or empty string if nil.
func StringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// Int32Value returns the value of an int32 pointer or 0 if nil.
func Int32Value(v *int32) int32 {
	if v == nil {
		return 0
	}
	return *v
}
