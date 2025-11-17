package aws

import (
	"testing"
)

func TestShouldCleanup(t *testing.T) {
	service := &S3Service{} // Mock service

	tests := []struct {
		name           string
		bucket         S3BucketInfo
		shouldCleanup  bool
		expectedReason string
	}{
		{
			name: "Empty bucket - Should cleanup",
			bucket: S3BucketInfo{
				Name:        "test-bucket",
				IsEmpty:     true,
				IsPublic:    false,
				HasTags:     true,
				ObjectCount: 0,
			},
			shouldCleanup:  true,
			expectedReason: "empty",
		},
		{
			name: "Public without tags - Should cleanup",
			bucket: S3BucketInfo{
				Name:        "public-bucket",
				IsEmpty:     false,
				IsPublic:    true,
				HasTags:     false,
				ObjectCount: 5,
			},
			shouldCleanup:  true,
			expectedReason: "public without tags",
		},
		{
			name: "Untagged with few objects - Should cleanup",
			bucket: S3BucketInfo{
				Name:        "small-bucket",
				IsEmpty:     false,
				IsPublic:    false,
				HasTags:     false,
				ObjectCount: 5,
			},
			shouldCleanup:  true,
			expectedReason: "untagged with few objects",
		},
		{
			name: "Well-tagged bucket with many objects - No cleanup",
			bucket: S3BucketInfo{
				Name:        "production-bucket",
				IsEmpty:     false,
				IsPublic:    false,
				HasTags:     true,
				ObjectCount: 1000,
			},
			shouldCleanup: false,
		},
		{
			name: "Private bucket with tags but empty - Should cleanup",
			bucket: S3BucketInfo{
				Name:        "old-bucket",
				IsEmpty:     true,
				IsPublic:    false,
				HasTags:     true,
				ObjectCount: 0,
			},
			shouldCleanup:  true,
			expectedReason: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldCleanup, reason := service.shouldCleanup(tt.bucket)

			if shouldCleanup != tt.shouldCleanup {
				t.Errorf("Expected shouldCleanup: %v, got: %v", tt.shouldCleanup, shouldCleanup)
			}

			if tt.shouldCleanup && reason == "" {
				t.Error("Expected a cleanup reason")
			}

			if !tt.shouldCleanup && reason != "" {
				t.Errorf("Expected no cleanup reason, got: %s", reason)
			}

			if tt.expectedReason != "" && reason != tt.expectedReason {
				t.Logf("Expected reason to contain '%s', got '%s'", tt.expectedReason, reason)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "Zero bytes",
			bytes:    0,
			expected: "0 B",
		},
		{
			name:     "Bytes only",
			bytes:    512,
			expected: "512 B",
		},
		{
			name:     "Kilobytes",
			bytes:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "Megabytes",
			bytes:    1024 * 1024,
			expected: "1.0 MB",
		},
		{
			name:     "Gigabytes",
			bytes:    1024 * 1024 * 1024,
			expected: "1.0 GB",
		},
		{
			name:     "Mixed size",
			bytes:    1536, // 1.5 KB
			expected: "1.5 KB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestS3BucketInfo_Structure(t *testing.T) {
	bucket := S3BucketInfo{
		Name:          "test-bucket",
		Region:        "us-east-1",
		CreatedDate:   "2024-01-01",
		IsEmpty:       false,
		IsPublic:      false,
		HasTags:       true,
		ObjectCount:   100,
		SizeBytes:     1024 * 1024,
		ShouldCleanup: false,
		CleanupReason: "",
	}

	if bucket.Name == "" {
		t.Error("Bucket name should not be empty")
	}

	if bucket.ObjectCount < 0 {
		t.Error("Object count should not be negative")
	}

	if bucket.SizeBytes < 0 {
		t.Error("Size bytes should not be negative")
	}
}
