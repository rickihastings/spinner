package prerequisites

import (
	"os"
	"os/exec"
)

// Prerequisite defines a required tool that must be installed.
type Prerequisite struct {
	Name         string
	Command      string
	ErrorMessage string
}

var prerequisites = []Prerequisite{
	{
		Name:         "Docker",
		Command:      "docker",
		ErrorMessage: "Docker is not installed. Please install Docker Desktop.",
	},
	{
		Name:         "Git",
		Command:      "git",
		ErrorMessage: "Git is not installed. Please install Git.",
	},
	{
		Name:         "Claude",
		Command:      "claude",
		ErrorMessage: "Claude CLI is not installed. Please install claude-code.",
	},
}

// PrerequisiteError is returned when a required prerequisite is missing.
type PrerequisiteError struct {
	Prerequisite Prerequisite
}

func (e *PrerequisiteError) Error() string {
	return e.Prerequisite.ErrorMessage
}

// CheckPrerequisites verifies that all required tools are installed.
func CheckPrerequisites() error {
	for _, prereq := range prerequisites {
		cmd := exec.Command(prereq.Command, "--version")
		if err := cmd.Run(); err != nil {
			return &PrerequisiteError{Prerequisite: prereq}
		}
	}

	return nil
}

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
