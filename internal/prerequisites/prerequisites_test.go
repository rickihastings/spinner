package prerequisites

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
