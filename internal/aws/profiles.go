// Package aws provides AWS profile and region utilities.
package aws

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// =============================================================================
// AWS Profiles
// =============================================================================

// ListProfiles returns all available AWS profiles from credentials and config files.
func ListProfiles() []string {
	profileSet := make(map[string]bool)

	// Read from ~/.aws/credentials
	if home, err := os.UserHomeDir(); err == nil {
		credentialsPath := filepath.Join(home, ".aws", "credentials")
		for _, p := range parseAWSFile(credentialsPath, false) {
			profileSet[p] = true
		}

		// Read from ~/.aws/config
		configPath := filepath.Join(home, ".aws", "config")
		for _, p := range parseAWSFile(configPath, true) {
			profileSet[p] = true
		}
	}

	// Always include "default"
	profileSet["default"] = true

	// Convert to sorted slice
	profiles := make([]string, 0, len(profileSet))
	for p := range profileSet {
		profiles = append(profiles, p)
	}
	sort.Strings(profiles)

	// Move "default" to first position
	for i, p := range profiles {
		if p == "default" && i > 0 {
			profiles = append([]string{"default"}, append(profiles[:i], profiles[i+1:]...)...)
			break
		}
	}

	return profiles
}

// parseAWSFile parses an AWS credentials or config file and extracts profile names.
// isConfig indicates if this is the config file (profiles are prefixed with "profile ")
func parseAWSFile(path string, isConfig bool) []string {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil
	}
	defer func() { _ = file.Close() }()

	var profiles []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Check for section headers [profile name] or [name]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := line[1 : len(line)-1]

			if isConfig {
				// In config file, profiles are "[profile name]" except for "[default]"
				if strings.HasPrefix(section, "profile ") {
					profiles = append(profiles, strings.TrimPrefix(section, "profile "))
				} else if section == "default" {
					profiles = append(profiles, "default")
				}
				// Skip other sections like [sso-session ...]
			} else {
				// In credentials file, profiles are just "[name]"
				profiles = append(profiles, section)
			}
		}
	}

	return profiles
}

// =============================================================================
// AWS Regions
// =============================================================================

// CommonRegions returns a list of commonly used AWS regions.
var CommonRegions = []string{
	"us-east-1",      // N. Virginia
	"us-east-2",      // Ohio
	"us-west-1",      // N. California
	"us-west-2",      // Oregon
	"eu-west-1",      // Ireland
	"eu-west-2",      // London
	"eu-west-3",      // Paris
	"eu-central-1",   // Frankfurt
	"eu-north-1",     // Stockholm
	"ap-northeast-1", // Tokyo
	"ap-northeast-2", // Seoul
	"ap-northeast-3", // Osaka
	"ap-southeast-1", // Singapore
	"ap-southeast-2", // Sydney
	"ap-south-1",     // Mumbai
	"sa-east-1",      // Sao Paulo
	"ca-central-1",   // Canada
	"me-south-1",     // Bahrain
	"af-south-1",     // Cape Town
}

// RegionNames provides human-readable names for AWS regions.
var RegionNames = map[string]string{
	"us-east-1":      "US East (N. Virginia)",
	"us-east-2":      "US East (Ohio)",
	"us-west-1":      "US West (N. California)",
	"us-west-2":      "US West (Oregon)",
	"eu-west-1":      "Europe (Ireland)",
	"eu-west-2":      "Europe (London)",
	"eu-west-3":      "Europe (Paris)",
	"eu-central-1":   "Europe (Frankfurt)",
	"eu-north-1":     "Europe (Stockholm)",
	"ap-northeast-1": "Asia Pacific (Tokyo)",
	"ap-northeast-2": "Asia Pacific (Seoul)",
	"ap-northeast-3": "Asia Pacific (Osaka)",
	"ap-southeast-1": "Asia Pacific (Singapore)",
	"ap-southeast-2": "Asia Pacific (Sydney)",
	"ap-south-1":     "Asia Pacific (Mumbai)",
	"sa-east-1":      "South America (Sao Paulo)",
	"ca-central-1":   "Canada (Central)",
	"me-south-1":     "Middle East (Bahrain)",
	"af-south-1":     "Africa (Cape Town)",
}

// GetRegionName returns the human-readable name for a region.
func GetRegionName(region string) string {
	if name, ok := RegionNames[region]; ok {
		return name
	}
	return region
}

// ListRegions returns all common AWS regions.
func ListRegions() []string {
	return CommonRegions
}
