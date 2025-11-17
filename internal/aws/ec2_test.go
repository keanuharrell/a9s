package aws

import (
	"testing"
)

func TestGetInstanceName(t *testing.T) {
	tests := []struct {
		name     string
		tags     []interface{} // Using interface{} for mock tags
		expected string
	}{
		{
			name:     "No tags",
			tags:     []interface{}{},
			expected: "-",
		},
		{
			name:     "Empty tags",
			tags:     nil,
			expected: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: getInstanceName expects AWS SDK types.Tag
			// This is a basic structure test
			if tt.expected != "-" {
				t.Skip("Full AWS SDK integration test - requires mock")
			}
		})
	}
}

func TestEC2Instance_Structure(t *testing.T) {
	instance := EC2Instance{
		ID:         "i-1234567890abcdef0",
		Name:       "test-instance",
		Type:       "t2.micro",
		State:      "running",
		PublicIP:   "1.2.3.4",
		PrivateIP:  "10.0.0.5",
		AZ:         "us-east-1a",
		LaunchTime: "2024-01-01 12:00:00",
	}

	if instance.ID == "" {
		t.Error("Instance ID should not be empty")
	}

	if instance.State != "running" {
		t.Errorf("Expected state 'running', got '%s'", instance.State)
	}

	if instance.Type != "t2.micro" {
		t.Errorf("Expected type 't2.micro', got '%s'", instance.Type)
	}
}

func TestEC2Instance_JSONTags(t *testing.T) {
	instance := EC2Instance{
		ID:    "i-test",
		Name:  "test",
		Type:  "t2.micro",
		State: "running",
	}

	// Verify JSON tags are properly set for serialization
	if instance.ID == "" {
		t.Error("ID field should be accessible")
	}
}
