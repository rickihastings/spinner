package prerequisites

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckPrerequisites_AllPass tests when all prerequisites are available
func TestCheckPrerequisites_AllPass(t *testing.T) {
	// This test assumes Docker, Git, and Claude CLI are installed on the test system
	// In CI environments, these should be pre-installed
	err := CheckPrerequisites()

	// If running in an environment without these tools, skip rather than fail
	if err != nil {
		t.Skipf("Skipping test - prerequisite not available: %v", err)
	}

	assert.NoError(t, err, "CheckPrerequisites should pass when all tools are installed")
}

// TestCheckPrerequisites_MissingTool tests when a prerequisite tool is missing
func TestCheckPrerequisites_MissingTool(t *testing.T) {
	// Save original prerequisites
	original := prerequisites

	defer func() { prerequisites = original }()

	// Create a prerequisite for a command that definitely doesn't exist
	prerequisites = []Prerequisite{
		{
			Name:         "NonExistentTool",
			Command:      "this-command-definitely-does-not-exist-12345",
			ErrorMessage: "Test error: NonExistentTool is not installed.",
		},
	}

	err := CheckPrerequisites()

	require.Error(t, err, "CheckPrerequisites should return error for missing tool")

	var prereqErr *PrerequisiteError

	ok := errors.As(err, &prereqErr)
	require.True(t, ok, "Error should be of type *PrerequisiteError")
	assert.Equal(t, "NonExistentTool", prereqErr.Prerequisite.Name)
	assert.Equal(t, "Test error: NonExistentTool is not installed.", prereqErr.Error())
}

// TestCheckPrerequisites_Docker tests Docker availability check
func TestCheckPrerequisites_Docker(t *testing.T) {
	// Save original prerequisites
	original := prerequisites

	defer func() { prerequisites = original }()

	// Test only Docker prerequisite
	prerequisites = []Prerequisite{
		{
			Name:         "Docker",
			Command:      "docker",
			ErrorMessage: "Docker is not installed. Please install Docker Desktop.",
		},
	}

	err := CheckPrerequisites()

	// If Docker is not installed, we expect an error
	if err != nil {
		var prereqErr *PrerequisiteError

		ok := errors.As(err, &prereqErr)
		require.True(t, ok, "Error should be of type *PrerequisiteError")
		assert.Equal(t, "Docker", prereqErr.Prerequisite.Name)
		t.Skipf("Docker not available on this system: %v", err)
	} else {
		assert.NoError(t, err, "Docker should be available")
	}
}

// TestCheckPrerequisites_TableDriven tests various prerequisite scenarios
func TestCheckPrerequisites_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		prerequisites []Prerequisite
		expectError   bool
		errorContains string
	}{
		{
			name: "All common tools available",
			prerequisites: []Prerequisite{
				{Name: "Git", Command: "git", ErrorMessage: "Git not found"},
			},
			expectError: false,
		},
		{
			name: "Nonexistent tool",
			prerequisites: []Prerequisite{
				{Name: "FakeTool", Command: "nonexistent-tool-xyz", ErrorMessage: "FakeTool not found"},
			},
			expectError:   true,
			errorContains: "FakeTool not found",
		},
		{
			name: "Multiple tools - first missing",
			prerequisites: []Prerequisite{
				{Name: "Missing", Command: "missing-cmd", ErrorMessage: "Missing tool"},
				{Name: "Git", Command: "git", ErrorMessage: "Git not found"},
			},
			expectError:   true,
			errorContains: "Missing tool",
		},
		{
			name: "Multiple tools - second missing",
			prerequisites: []Prerequisite{
				{Name: "Git", Command: "git", ErrorMessage: "Git not found"},
				{Name: "Missing", Command: "missing-cmd", ErrorMessage: "Missing tool"},
			},
			expectError:   true,
			errorContains: "Missing tool",
		},
		{
			name:          "Empty prerequisites list",
			prerequisites: []Prerequisite{},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original prerequisites
			original := prerequisites

			defer func() { prerequisites = original }()

			prerequisites = tt.prerequisites

			err := CheckPrerequisites()

			if tt.expectError {
				require.Error(t, err, "Expected error for test case: %s", tt.name)
				assert.Contains(t, err.Error(), tt.errorContains, "Error message should contain expected text")
			} else {
				// For tests expecting no error, we still need to handle cases where
				// the tool might not be installed (like Docker, Claude)
				if err != nil {
					t.Skipf("Skipping - tool not available: %v", err)
				}

				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)
			}
		})
	}
}

// TestCheckEnvironmentVariables tests GitHub and Claude token validation
func TestCheckEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name          string
		githubToken   *string // nil means don't set, empty string means set to ""
		claudeToken   *string
		expectError   bool
		errorContains string
	}{
		{
			name:        "Both tokens set",
			githubToken: stringPtr("test-github-token"),
			claudeToken: stringPtr("test-claude-token"),
			expectError: false,
		},
		{
			name:          "GitHub token missing",
			githubToken:   stringPtr(""),
			claudeToken:   stringPtr("test-claude-token"),
			expectError:   true,
			errorContains: "GITHUB_TOKEN",
		},
		{
			name:          "Claude token missing",
			githubToken:   stringPtr("test-github-token"),
			claudeToken:   stringPtr(""),
			expectError:   true,
			errorContains: "CLAUDE_CODE_OAUTH_TOKEN",
		},
		{
			name:          "Both tokens missing",
			githubToken:   stringPtr(""),
			claudeToken:   stringPtr(""),
			expectError:   true,
			errorContains: "GITHUB_TOKEN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			origGitHub := os.Getenv("GITHUB_TOKEN")
			origClaude := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")

			defer func() {
				_ = os.Setenv("GITHUB_TOKEN", origGitHub)
				_ = os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", origClaude)
			}()

			// Set test environment
			if tt.githubToken != nil {
				_ = os.Setenv("GITHUB_TOKEN", *tt.githubToken)
			} else {
				_ = os.Unsetenv("GITHUB_TOKEN")
			}

			if tt.claudeToken != nil {
				_ = os.Setenv("CLAUDE_CODE_OAUTH_TOKEN", *tt.claudeToken)
			} else {
				_ = os.Unsetenv("CLAUDE_CODE_OAUTH_TOKEN")
			}

			err := CheckEnvironmentVariables()

			if tt.expectError {
				require.Error(t, err, "Expected error for test case: %s", tt.name)
				assert.Contains(t, err.Error(), tt.errorContains, "Error message should contain expected text")
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)
			}
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
