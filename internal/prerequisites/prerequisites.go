package prerequisites

import (
	"os"
)

// EnvironmentVariableError is returned when a required environment variable is missing.
type EnvironmentVariableError struct {
	Variable string
	Message  string
}

func (e *EnvironmentVariableError) Error() string {
	return e.Message
}

// CheckEnvironmentVariables verifies that all required environment variables are set.
// This includes GITHUB_TOKEN and CLAUDE_CODE_OAUTH_TOKEN.
func CheckEnvironmentVariables() error {
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return &EnvironmentVariableError{
			Variable: "GITHUB_TOKEN",
			Message:  "GITHUB_TOKEN environment variable not set. Please set GITHUB_TOKEN before running spin.",
		}
	}

	claudeToken := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	if claudeToken == "" {
		return &EnvironmentVariableError{
			Variable: "CLAUDE_CODE_OAUTH_TOKEN",
			Message:  "CLAUDE_CODE_OAUTH_TOKEN environment variable not set. Please set CLAUDE_CODE_OAUTH_TOKEN before running spin.",
		}
	}

	return nil
}
