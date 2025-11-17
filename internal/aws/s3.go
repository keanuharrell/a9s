package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	appconfig "github.com/keanuharrell/a9s/internal/config"
	"github.com/olekukonko/tablewriter"
)

// S3CleanupResult contains the results of an S3 bucket cleanup analysis
type S3CleanupResult struct {
	Buckets           []S3BucketInfo `json:"buckets"`
	CleanupCandidates []S3BucketInfo `json:"cleanup_candidates"`
	Summary           S3Summary      `json:"summary"`
}

// S3BucketInfo contains detailed information about an S3 bucket
type S3BucketInfo struct {
	Name          string `json:"name"`
	Region        string `json:"region"`
	CreatedDate   string `json:"created_date"`
	IsEmpty       bool   `json:"is_empty"`
	IsPublic      bool   `json:"is_public"`
	HasTags       bool   `json:"has_tags"`
	ObjectCount   int    `json:"object_count"`
	SizeBytes     int64  `json:"size_bytes"`
	ShouldCleanup bool   `json:"should_cleanup"`
	CleanupReason string `json:"cleanup_reason,omitempty"`
}

// S3Summary provides aggregated statistics from S3 bucket analysis
type S3Summary struct {
	TotalBuckets      int      `json:"total_buckets"`
	EmptyBuckets      int      `json:"empty_buckets"`
	PublicBuckets     int      `json:"public_buckets"`
	UntaggedBuckets   int      `json:"untagged_buckets"`
	CleanupCandidates []string `json:"cleanup_candidates"`
}

// S3Service provides methods for interacting with AWS S3
type S3Service struct {
	client *s3.Client
}

// NewS3Service creates a new S3 service instance with the specified AWS profile and region
func NewS3Service(profile, region string) (*S3Service, error) {
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

	return &S3Service{
		client: s3.NewFromConfig(cfg),
	}, nil
}

// AnalyzeBuckets analyzes all S3 buckets and identifies cleanup candidates
func (s *S3Service) AnalyzeBuckets(ctx context.Context) (*S3CleanupResult, error) {
	bucketsOutput, err := s.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	result := &S3CleanupResult{
		Buckets:           []S3BucketInfo{},
		CleanupCandidates: []S3BucketInfo{},
		Summary: S3Summary{
			CleanupCandidates: []string{},
		},
	}

	for _, bucket := range bucketsOutput.Buckets {
		bucketInfo := S3BucketInfo{
			Name: aws.ToString(bucket.Name),
		}

		if bucket.CreationDate != nil {
			bucketInfo.CreatedDate = bucket.CreationDate.Format("2006-01-02")
		}

		bucketInfo.Region = s.getBucketRegion(ctx, aws.ToString(bucket.Name))

		bucketInfo.IsEmpty, bucketInfo.ObjectCount, bucketInfo.SizeBytes =
			s.getBucketStats(ctx, aws.ToString(bucket.Name))

		bucketInfo.IsPublic = s.isBucketPublic(ctx, aws.ToString(bucket.Name))

		bucketInfo.HasTags = s.hasTags(ctx, aws.ToString(bucket.Name))

		bucketInfo.ShouldCleanup, bucketInfo.CleanupReason =
			s.shouldCleanup(bucketInfo)

		if bucketInfo.IsEmpty {
			result.Summary.EmptyBuckets++
		}
		if bucketInfo.IsPublic {
			result.Summary.PublicBuckets++
		}
		if !bucketInfo.HasTags {
			result.Summary.UntaggedBuckets++
		}

		if bucketInfo.ShouldCleanup {
			result.CleanupCandidates = append(result.CleanupCandidates, bucketInfo)
			result.Summary.CleanupCandidates = append(
				result.Summary.CleanupCandidates, bucketInfo.Name)
		}

		result.Buckets = append(result.Buckets, bucketInfo)
	}

	result.Summary.TotalBuckets = len(result.Buckets)

	return result, nil
}

func (s *S3Service) getBucketRegion(ctx context.Context, bucketName string) string {
	location, err := s.client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
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

func (s *S3Service) getBucketStats(ctx context.Context, bucketName string) (bool, int, int64) {
	var objectCount int
	var totalSize int64

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucketName),
		MaxKeys: aws.Int32(1000),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return false, 0, 0
		}

		objectCount += len(page.Contents)
		for _, obj := range page.Contents {
			totalSize += aws.ToInt64(obj.Size)
		}

		if objectCount > 100 {
			break
		}
	}

	return objectCount == 0, objectCount, totalSize
}

func (s *S3Service) isBucketPublic(ctx context.Context, bucketName string) bool {
	_, err := s.client.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{
		Bucket: aws.String(bucketName),
	})

	return err != nil
}

func (s *S3Service) hasTags(ctx context.Context, bucketName string) bool {
	tags, err := s.client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucketName),
	})

	return err == nil && len(tags.TagSet) > 0
}

func (s *S3Service) shouldCleanup(bucket S3BucketInfo) (bool, string) {
	var reasons []string

	if bucket.IsEmpty {
		reasons = append(reasons, "empty")
	}

	if bucket.IsPublic && !bucket.HasTags {
		reasons = append(reasons, "public without tags")
	}

	if !bucket.HasTags && bucket.ObjectCount < 10 {
		reasons = append(reasons, "untagged with few objects")
	}

	if len(reasons) > 0 {
		return true, strings.Join(reasons, ", ")
	}

	return false, ""
}

// DeleteBucket deletes an S3 bucket and all its objects
func (s *S3Service) DeleteBucket(ctx context.Context, bucketName string) error {
	objects, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return fmt.Errorf("failed to list objects: %w", err)
	}

	if len(objects.Contents) > 0 {
		var objectIDs []types.ObjectIdentifier
		for _, object := range objects.Contents {
			objectIDs = append(objectIDs, types.ObjectIdentifier{
				Key: object.Key,
			})
		}

		_, err = s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(bucketName),
			Delete: &types.Delete{
				Objects: objectIDs,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to delete objects: %w", err)
		}
	}

	_, err = s.client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	return nil
}

// OutputS3Cleanup outputs the S3 cleanup results in the specified format (json or table)
func OutputS3Cleanup(result *S3CleanupResult, format string) error {
	switch strings.ToLower(format) {
	case appconfig.FormatJSON:
		return outputS3JSON(result)
	case appconfig.FormatTable:
		return outputS3Table(result)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func outputS3JSON(result *S3CleanupResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func outputS3Table(result *S3CleanupResult) error {
	fmt.Printf("\n=== S3 Cleanup Analysis ===\n")
	fmt.Printf("Total Buckets: %d\n", result.Summary.TotalBuckets)
	fmt.Printf("Empty Buckets: %d\n", result.Summary.EmptyBuckets)
	fmt.Printf("Public Buckets: %d\n", result.Summary.PublicBuckets)
	fmt.Printf("Untagged Buckets: %d\n", result.Summary.UntaggedBuckets)
	fmt.Printf("Cleanup Candidates: %d\n", len(result.Summary.CleanupCandidates))

	if len(result.CleanupCandidates) > 0 {
		fmt.Printf("\n=== Buckets Recommended for Cleanup ===\n")
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Bucket Name", "Region", "Created", "Objects", "Size", "Reason"})
		table.SetBorder(false)
		table.SetAutoWrapText(false)

		for _, bucket := range result.CleanupCandidates {
			sizeStr := formatBytes(bucket.SizeBytes)
			row := []string{
				bucket.Name,
				bucket.Region,
				bucket.CreatedDate,
				fmt.Sprintf("%d", bucket.ObjectCount),
				sizeStr,
				bucket.CleanupReason,
			}
			table.Append(row)
		}

		table.Render()
	}

	fmt.Printf("\n=== All Buckets ===\n")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Bucket Name", "Region", "Created", "Empty", "Public", "Tagged"})
	table.SetBorder(false)
	table.SetAutoWrapText(false)

	for _, bucket := range result.Buckets {
		empty := appconfig.S3No
		if bucket.IsEmpty {
			empty = appconfig.S3Yes
		}
		public := appconfig.S3No
		if bucket.IsPublic {
			public = appconfig.S3Yes
		}
		tagged := appconfig.S3No
		if bucket.HasTags {
			tagged = appconfig.S3Yes
		}

		row := []string{
			bucket.Name,
			bucket.Region,
			bucket.CreatedDate,
			empty,
			public,
			tagged,
		}
		table.Append(row)
	}

	table.Render()
	return nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// CleanupBuckets performs cleanup of buckets identified as candidates (dry-run mode available)
func (s *S3Service) CleanupBuckets(ctx context.Context, dryRun bool) (*S3CleanupResult, error) {
	result, err := s.AnalyzeBuckets(ctx)
	if err != nil {
		return nil, err
	}

	if !dryRun && len(result.CleanupCandidates) > 0 {
		fmt.Printf("\n=== Performing Cleanup ===\n")
		for _, bucket := range result.CleanupCandidates {
			fmt.Printf("Deleting bucket: %s... ", bucket.Name)
			if err := s.DeleteBucket(ctx, bucket.Name); err != nil {
				fmt.Printf("FAILED: %v\n", err)
			} else {
				fmt.Printf("SUCCESS\n")
			}

			time.Sleep(500 * time.Millisecond)
		}
	}

	return result, nil
}
