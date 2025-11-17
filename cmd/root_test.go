package cmd

import (
	"testing"
)

func TestVersionVariable(t *testing.T) {
	// Test that Version can be set
	originalVersion := Version
	defer func() { Version = originalVersion }()

	testVersion := "1.2.3"
	Version = testVersion

	if Version != testVersion {
		t.Errorf("Expected version '%s', got '%s'", testVersion, Version)
	}
}

func TestBuildTimeVariable(t *testing.T) {
	// Test that BuildTime can be set
	originalBuildTime := BuildTime
	defer func() { BuildTime = originalBuildTime }()

	testBuildTime := "2024-01-01_12:00:00"
	BuildTime = testBuildTime

	if BuildTime != testBuildTime {
		t.Errorf("Expected build time '%s', got '%s'", testBuildTime, BuildTime)
	}
}

func TestGetRootCommand(t *testing.T) {
	cmd := GetRootCommand()

	if cmd == nil {
		t.Fatal("GetRootCommand should not return nil")
	}

	if cmd.Use != "a9s" {
		t.Errorf("Expected command name 'a9s', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Command should have a short description")
	}

	if cmd.Long == "" {
		t.Error("Command should have a long description")
	}
}

func TestRootCommandFlags(t *testing.T) {
	cmd := GetRootCommand()

	// Test that persistent flags are registered
	flags := []string{"output", "profile", "region", "dry-run", "config"}

	for _, flagName := range flags {
		flag := cmd.PersistentFlags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag '%s' to be registered", flagName)
		}
	}
}

func TestRootCommandVersion(t *testing.T) {
	cmd := GetRootCommand()

	if cmd.Version == "" {
		t.Error("Command should have a version set")
	}

	// Version should match the Version variable
	if cmd.Version != Version {
		t.Errorf("Command version '%s' should match Version variable '%s'", cmd.Version, Version)
	}
}
