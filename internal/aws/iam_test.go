package aws

import (
	"testing"
)

func TestAssessRisk(t *testing.T) {
	service := &IAMService{} // Mock service

	tests := []struct {
		name           string
		policies       []string
		expectHighRisk bool
		expectedReason string
	}{
		{
			name:           "No policies - Low risk",
			policies:       []string{},
			expectHighRisk: false,
			expectedReason: "",
		},
		{
			name:           "Safe policies - Low risk",
			policies:       []string{"ReadOnlyAccess", "ViewOnlyAccess"},
			expectHighRisk: false,
			expectedReason: "",
		},
		{
			name:           "Administrator access - High risk",
			policies:       []string{"AdministratorAccess"},
			expectHighRisk: true,
			expectedReason: "Has AdministratorAccess policy",
		},
		{
			name:           "PowerUser access - High risk",
			policies:       []string{"PowerUserAccess"},
			expectHighRisk: true,
			expectedReason: "Has PowerUserAccess policy",
		},
		{
			name:           "Wildcard permissions - High risk",
			policies:       []string{"CustomPolicy*"},
			expectHighRisk: true,
			expectedReason: "Contains wildcard permissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isHighRisk, reason := service.assessRisk(tt.policies)

			if isHighRisk != tt.expectHighRisk {
				t.Errorf("Expected high risk: %v, got: %v", tt.expectHighRisk, isHighRisk)
			}

			if tt.expectHighRisk && reason == "" {
				t.Error("Expected a risk reason for high-risk policies")
			}

			if !tt.expectHighRisk && reason != "" {
				t.Errorf("Expected no risk reason, got: %s", reason)
			}

			if tt.expectedReason != "" && reason != tt.expectedReason {
				t.Errorf("Expected reason '%s', got '%s'", tt.expectedReason, reason)
			}
		})
	}
}

func TestIAMRole_Structure(t *testing.T) {
	role := IAMRole{
		Name:             "test-role",
		ARN:              "arn:aws:iam::123456789012:role/test-role",
		CreateDate:       "2024-01-01",
		AttachedPolicies: []string{"ReadOnlyAccess"},
		IsHighRisk:       false,
		RiskReason:       "",
	}

	if role.Name == "" {
		t.Error("Role name should not be empty")
	}

	if role.ARN == "" {
		t.Error("Role ARN should not be empty")
	}

	if len(role.AttachedPolicies) == 0 {
		t.Error("Expected at least one policy")
	}
}

func TestHighRiskPolicies(t *testing.T) {
	expectedPolicies := []string{
		"AdministratorAccess",
		"PowerUserAccess",
		"IAMFullAccess",
		"SecurityAudit",
	}

	if len(highRiskPolicies) != len(expectedPolicies) {
		t.Errorf("Expected %d high-risk policies, got %d", len(expectedPolicies), len(highRiskPolicies))
	}

	// Verify AdministratorAccess is in the list
	found := false
	for _, policy := range highRiskPolicies {
		if policy == "AdministratorAccess" {
			found = true
			break
		}
	}

	if !found {
		t.Error("AdministratorAccess should be in high-risk policies list")
	}
}
