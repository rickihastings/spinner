package cmd

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/rickihastings/spinner/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestSpinCommand_MissingImageFlag tests that spin command fails when --image flag is missing
func TestSpinCommand_MissingImageFlag(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	cmd := NewSpinCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--repo", "https://github.com/test/repo.git"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--image flag is required")
}

// TestSpinCommand_MissingRepoFlag tests that spin command fails when --repo flag is missing
func TestSpinCommand_MissingRepoFlag(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	cmd := NewSpinCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--image", "spinner:test"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--repo flag is required")
}

// TestSpinCommand_PromptFlagParsing tests that --prompt flag is correctly parsed
func TestSpinCommand_PromptFlagParsing(t *testing.T) {
	cmd := setupSpinCommandWithMocks(t)

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--image", "spinner:test", "--repo", "https://github.com/test/repo.git", "--prompt", "Fix the bug"})

	err := cmd.Execute()

	assert.NoError(t, err)
}

// TestSpinCommand_BranchFlagParsing tests that --branch flag is correctly parsed
func TestSpinCommand_BranchFlagParsing(t *testing.T) {
	cmd := setupSpinCommandWithMocks(t)

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--image", "spinner:test", "--repo", "https://github.com/test/repo.git", "--branch", "feature-branch"})

	err := cmd.Execute()

	assert.NoError(t, err)
}

// TestSpinCommand_MaxIterationsFlagParsing tests that --max-iterations flag is correctly parsed
func TestSpinCommand_MaxIterationsFlagParsing(t *testing.T) {
	cmd := setupSpinCommandWithMocks(t)

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--image", "spinner:test", "--repo", "https://github.com/test/repo.git", "--max-iterations", "50"})

	err := cmd.Execute()

	assert.NoError(t, err)
}

// TestSpinCommand_RecreateFlagParsing tests that --recreate flag triggers Remove then Create
func TestSpinCommand_RecreateFlagParsing(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")

	mockProvider := new(provider.MockProvider)
	mockProvider.On("InstanceName", mock.Anything).Return("test-container")
	mockProvider.On("Status", mock.Anything, "test-container").Return(provider.InstanceStatusRunning, nil)
	mockProvider.On("Remove", mock.Anything, "test-container").Return(nil)
	mockProvider.On("Create", mock.Anything, mock.Anything).Return(
		&provider.Instance{Name: "test-container", Status: provider.InstanceStatusRunning}, nil,
	)

	cmd := NewSpinCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--image", "spinner:test", "--repo", "https://github.com/test/repo.git", "--recreate"})

	err := cmd.Execute()

	assert.NoError(t, err)
	mockProvider.AssertCalled(t, "Remove", mock.Anything, "test-container")
}

// TestSpinCommand_SetupFlagParsing tests that --setup flag is correctly parsed
func TestSpinCommand_SetupFlagParsing(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")

	mockProvider := new(provider.MockProvider)
	mockProvider.On("Setup", mock.Anything, mock.Anything).Return(nil)
	mockProvider.On("InstanceName", mock.Anything).Return("test-container")
	mockProvider.On("Status", mock.Anything, "test-container").Return(provider.InstanceStatusNone, nil)
	mockProvider.On("Create", mock.Anything, mock.Anything).Return(
		&provider.Instance{Name: "test-container", Status: provider.InstanceStatusRunning}, nil,
	)

	cmd := NewSpinCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--setup", "--image", "test", "--repo", "https://github.com/test/repo.git"})

	err := cmd.Execute()

	assert.NoError(t, err)
	mockProvider.AssertCalled(t, "Setup", mock.Anything, mock.Anything)
}

// TestSpinCommand_ProviderArgsFlagParsing tests that --provider-args flag is correctly parsed
func TestSpinCommand_ProviderArgsFlagParsing(t *testing.T) {
	cmd := setupSpinCommandWithMocks(t)

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--image", "spinner:test",
		"--repo", "https://github.com/test/repo.git",
		"--provider-args", "-v /data:/data",
		"--provider-args", "--network=host",
	})

	err := cmd.Execute()

	assert.NoError(t, err)
}

// TestSpinCommand_ProviderArgsSingleFlag tests --provider-args with a single arg
func TestSpinCommand_ProviderArgsSingleFlag(t *testing.T) {
	cmd := setupSpinCommandWithMocks(t)

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--image", "spinner:test",
		"--repo", "https://github.com/test/repo.git",
		"--provider-args", "--privileged",
	})

	err := cmd.Execute()

	assert.NoError(t, err)
}

// TestSpinCommand_InvalidRepoURL tests that an invalid repo URL is rejected
func TestSpinCommand_InvalidRepoURL(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")

	mockProvider := new(provider.MockProvider)
	cmd := NewSpinCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--image", "spinner:test", "--repo", "not-a-valid-url"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repository must be a valid git URL")
}

// TestSpinCommand_FlagCombinations is a table-driven test for various flag combinations
func TestSpinCommand_FlagCombinations(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "missing image flag",
			args:      []string{"--repo", "https://github.com/test/repo.git"},
			wantError: true,
			errorMsg:  "--image flag is required",
		},
		{
			name:      "missing repo flag",
			args:      []string{"--image", "spinner:test"},
			wantError: true,
			errorMsg:  "--repo flag is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := new(provider.MockProvider)
			cmd := NewSpinCommand(testFactory(mockProvider))

			b := new(bytes.Buffer)
			cmd.SetOut(b)
			cmd.SetErr(b)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSpinCommand_GCPFlagsWithDockerBackend tests that GCP flags error with docker backend
func TestSpinCommand_GCPFlagsWithDockerBackend(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		errorMsg string
	}{
		{
			name:     "project flag with docker backend",
			args:     []string{"--image", "test", "--repo", "https://github.com/test/repo.git", "--project", "my-proj"},
			errorMsg: "--project requires --backend gcp",
		},
		{
			name:     "zone flag with docker backend",
			args:     []string{"--image", "test", "--repo", "https://github.com/test/repo.git", "--zone", "us-central1-a"},
			errorMsg: "--zone requires --backend gcp",
		},
		{
			name:     "state-bucket flag with docker backend",
			args:     []string{"--image", "test", "--repo", "https://github.com/test/repo.git", "--state-bucket", "my-bucket"},
			errorMsg: "--state-bucket requires --backend gcp",
		},
		{
			name:     "bake-script flag with docker backend",
			args:     []string{"--image", "test", "--repo", "https://github.com/test/repo.git", "--bake-script", "/tmp/bake.sh"},
			errorMsg: "--bake-script requires --backend gcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := new(provider.MockProvider)
			cmd := NewSpinCommand(testFactory(mockProvider))

			b := new(bytes.Buffer)
			cmd.SetOut(b)
			cmd.SetErr(b)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

// TestSpinCommand_UnknownBackend tests that an unknown backend produces a clear error
func TestSpinCommand_UnknownBackend(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")

	mockProvider := new(provider.MockProvider)
	cmd := NewSpinCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--image", "test", "--repo", "https://github.com/test/repo.git", "--backend", "kubernetes"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown backend")
}

// TestSpinCommand_BakeScriptRequiresSetup tests that --bake-script requires --setup on spin
func TestSpinCommand_BakeScriptRequiresSetup(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "bake-script-*.sh")
	assert.NoError(t, err)

	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_, _ = tmpfile.WriteString("#!/bin/bash\necho hello\n")
	_ = tmpfile.Close()

	mockProvider := new(provider.MockProvider)
	cmd := NewSpinCommand(testGCPFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--image", "test",
		"--repo", "https://github.com/test/repo.git",
		"--backend", "gcp",
		"--project", "my-proj",
		"--zone", "us-central1-a",
		"--state-bucket", "my-bucket",
		"--bake-script", tmpfile.Name(),
	})

	err = cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--bake-script requires --setup flag")
}

// TestSpinCommand_DockerBackendExplicit tests explicit --backend docker works
func TestSpinCommand_DockerBackendExplicit(t *testing.T) {
	cmd := setupSpinCommandWithMocks(t)

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--image", "spinner:test", "--repo", "https://github.com/test/repo.git", "--backend", "docker"})

	err := cmd.Execute()

	assert.NoError(t, err)
}

// TestSpinCommand_MissingRequiredGCPFlags tests that required GCP flags produce errors
func TestSpinCommand_MissingRequiredGCPFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		errorMsg string
	}{
		{
			name: "missing project",
			args: []string{
				"--image", "test",
				"--repo", "https://github.com/test/repo.git",
				"--backend", "gcp",
				"--zone", "us-central1-a",
				"--state-bucket", "bucket",
			},
			errorMsg: "--project is required for GCP backend",
		},
		{
			name: "missing zone",
			args: []string{
				"--image", "test",
				"--repo", "https://github.com/test/repo.git",
				"--backend", "gcp",
				"--project", "my-project",
				"--state-bucket", "bucket",
			},
			errorMsg: "--zone is required for GCP backend",
		},
		{
			name: "missing state-bucket",
			args: []string{
				"--image", "test",
				"--repo", "https://github.com/test/repo.git",
				"--backend", "gcp",
				"--project", "my-project",
				"--zone", "us-central1-a",
			},
			errorMsg: "--state-bucket is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_TOKEN", "test-token")
			t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")
			// Clear GCP env vars to ensure we're testing CLI validation
			t.Setenv("SPINNER_PROJECT", "")
			t.Setenv("SPINNER_ZONE", "")
			t.Setenv("SPINNER_STATE_BUCKET", "")

			mockProvider := new(provider.MockProvider)
			cmd := NewSpinCommand(testGCPFactory(mockProvider))

			b := new(bytes.Buffer)
			cmd.SetOut(b)
			cmd.SetErr(b)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

// TestSpinCommand_BackendFromEnvVar tests that SPINNER_BACKEND env var is respected
func TestSpinCommand_BackendFromEnvVar(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")
	t.Setenv("SPINNER_BACKEND", "gcp")
	// Clear GCP config env vars so validation fails
	t.Setenv("SPINNER_PROJECT", "")
	t.Setenv("SPINNER_ZONE", "")
	t.Setenv("SPINNER_STATE_BUCKET", "")

	mockProvider := new(provider.MockProvider)
	cmd := NewSpinCommand(testGCPFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--image", "test",
		"--repo", "https://github.com/test/repo.git",
	})

	err := cmd.Execute()

	// Should error about missing GCP flags because backend=gcp from env
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--project is required")
}

// TestSpinCommand_CLIFlagOverridesEnvVar tests that CLI flag takes precedence over env var
func TestSpinCommand_CLIFlagOverridesEnvVar(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")
	t.Setenv("SPINNER_BACKEND", "docker")
	// Clear GCP config env vars so validation fails
	t.Setenv("SPINNER_PROJECT", "")
	t.Setenv("SPINNER_ZONE", "")
	t.Setenv("SPINNER_STATE_BUCKET", "")

	mockProvider := new(provider.MockProvider)
	cmd := NewSpinCommand(testGCPFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	// CLI specifies --backend gcp, should override env SPINNER_BACKEND=docker
	cmd.SetArgs([]string{
		"--image", "test",
		"--repo", "https://github.com/test/repo.git",
		"--backend", "gcp",
	})

	err := cmd.Execute()

	// Should error about missing GCP flags because CLI --backend gcp overrides env
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--project is required")
}

// TestParseAndValidateEnvVars_ValidCases tests valid env var parsing
func TestParseAndValidateEnvVars_ValidCases(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]string
	}{
		{
			name:     "single env var",
			input:    []string{"NPM_TOKEN=abc123"},
			expected: map[string]string{"NPM_TOKEN": "abc123"},
		},
		{
			name:  "multiple env vars",
			input: []string{"NPM_TOKEN=abc", "MY_API_KEY=xyz"},
			expected: map[string]string{
				"NPM_TOKEN":  "abc",
				"MY_API_KEY": "xyz",
			},
		},
		{
			name:     "env var with equals in value",
			input:    []string{"CONFIG=key=value"},
			expected: map[string]string{"CONFIG": "key=value"},
		},
		{
			name:     "env var with empty value",
			input:    []string{"MY_VAR="},
			expected: map[string]string{"MY_VAR": ""},
		},
		{
			name:     "env var with spaces in value",
			input:    []string{"MESSAGE=hello world"},
			expected: map[string]string{"MESSAGE": "hello world"},
		},
		{
			name:     "no env vars",
			input:    []string{},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAndValidateEnvVars(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseAndValidateEnvVars_InvalidFormat tests invalid format handling
func TestParseAndValidateEnvVars_InvalidFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		errorMsg string
	}{
		{
			name:     "no equals sign",
			input:    []string{"INVALID"},
			errorMsg: "--env: invalid format \"INVALID\", expected KEY=VALUE",
		},
		{
			name:     "empty key",
			input:    []string{"=value"},
			errorMsg: "--env: key must not be empty",
		},
		{
			name:     "multiple invalid vars",
			input:    []string{"VALID=123", "INVALID"},
			errorMsg: "--env: invalid format \"INVALID\", expected KEY=VALUE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAndValidateEnvVars(tt.input)
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

// TestParseAndValidateEnvVars_ReservedVars tests that reserved variables are rejected
func TestParseAndValidateEnvVars_ReservedVars(t *testing.T) {
	reservedVars := []string{
		"GITHUB_TOKEN",
		"CLAUDE_CODE_OAUTH_TOKEN",
		"REPO_URL",
		"PROMPT",
		"BRANCH",
		"MAX_ITERATIONS",
		"LOG_DIR",
		"STATE_DIR",
		"SPINNER_LOG_BUCKET",
		"SPINNER_STATE_BUCKET",
		"SPINNER_INSTANCE_NAME",
		"ANTHROPIC_MODEL",
	}

	for _, reserved := range reservedVars {
		t.Run(reserved, func(t *testing.T) {
			result, err := parseAndValidateEnvVars([]string{reserved + "=override"})
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "cannot override reserved variable")
			assert.Contains(t, err.Error(), reserved)
		})
	}
}

// TestSpinCommand_EnvFlagParsing tests that --env flag is correctly parsed and passed through
func TestSpinCommand_EnvFlagParsing(t *testing.T) {
	cmd := setupSpinCommandWithMocks(t)

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--image", "spinner:test",
		"--repo", "https://github.com/test/repo.git",
		"--env", "NPM_TOKEN=abc123",
		"--env", "MY_API_KEY=xyz",
	})

	err := cmd.Execute()

	assert.NoError(t, err)
}

// TestSpinCommand_EnvFlagInvalidFormat tests that invalid --env format produces error
func TestSpinCommand_EnvFlagInvalidFormat(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	cmd := NewSpinCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--image", "spinner:test",
		"--repo", "https://github.com/test/repo.git",
		"--env", "INVALID",
	})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--env: invalid format")
}

// TestSpinCommand_EnvFlagReservedVar tests that reserved vars cannot be overridden
func TestSpinCommand_EnvFlagReservedVar(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	cmd := NewSpinCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--image", "spinner:test",
		"--repo", "https://github.com/test/repo.git",
		"--env", "GITHUB_TOKEN=override",
	})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot override reserved variable")
	assert.Contains(t, err.Error(), "GITHUB_TOKEN")
}

// TestSpinCommand_EnvFileFlagParsing tests that --env-file flag is correctly parsed
func TestSpinCommand_EnvFileFlagParsing(t *testing.T) {
	// Create a temporary env file
	tmpfile, err := os.CreateTemp("", "test-env-*.env")
	assert.NoError(t, err)

	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_, _ = tmpfile.WriteString("NPM_TOKEN=abc123\nAPI_KEY=xyz\n")
	_ = tmpfile.Close()

	cmd := setupSpinCommandWithMocks(t)

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--image", "spinner:test",
		"--repo", "https://github.com/test/repo.git",
		"--env-file", tmpfile.Name(),
	})

	err = cmd.Execute()

	assert.NoError(t, err)
}

// TestSpinCommand_EnvFileNotFound tests that --env-file errors when file doesn't exist
func TestSpinCommand_EnvFileNotFound(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	cmd := NewSpinCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--image", "spinner:test",
		"--repo", "https://github.com/test/repo.git",
		"--env-file", "/nonexistent/path/to/.env",
	})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--env-file: file does not exist")
}

// TestSpinCommand_StartFailsFallbackToCreate tests that when Start() fails on a stopped instance
// and Status() then reports InstanceStatusNone (TOCTOU race), the command falls through to Create().
func TestSpinCommand_StartFailsFallbackToCreate(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")

	mockProvider := new(provider.MockProvider)
	mockProvider.On("InstanceName", mock.Anything).Return("test-container")
	// First Status() call returns Stopped
	mockProvider.On("Status", mock.Anything, "test-container").Return(provider.InstanceStatusStopped, nil).Once()
	// Start() fails (instance was deleted between Status and Start)
	mockProvider.On("Start", mock.Anything, "test-container", mock.Anything).Return(
		(*provider.Instance)(nil), fmt.Errorf("instance not found"),
	)
	// Second Status() call (retry) returns None — instance is gone
	mockProvider.On("Status", mock.Anything, "test-container").Return(provider.InstanceStatusNone, nil).Once()
	// Create() succeeds
	mockProvider.On("Create", mock.Anything, mock.Anything).Return(
		&provider.Instance{Name: "test-container", Status: provider.InstanceStatusRunning}, nil,
	)

	cmd := NewSpinCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--image", "spinner:test", "--repo", "https://github.com/test/repo.git"})

	err := cmd.Execute()

	assert.NoError(t, err)
	mockProvider.AssertCalled(t, "Start", mock.Anything, "test-container", mock.Anything)
	mockProvider.AssertCalled(t, "Create", mock.Anything, mock.Anything)
}

// TestSpinCommand_ModelFlagParsing tests that --model flag is correctly parsed
func TestSpinCommand_ModelFlagParsing(t *testing.T) {
	cmd := setupSpinCommandWithMocks(t)

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--image", "spinner:test",
		"--repo", "https://github.com/test/repo.git",
		"--model", "claude-sonnet-4-5-20250929",
	})

	err := cmd.Execute()

	assert.NoError(t, err)
}

// TestSpinCommand_EnvFlagReservedVarAnthropicModel tests that ANTHROPIC_MODEL cannot be set via --env
func TestSpinCommand_EnvFlagReservedVarAnthropicModel(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	cmd := NewSpinCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--image", "spinner:test",
		"--repo", "https://github.com/test/repo.git",
		"--env", "ANTHROPIC_MODEL=claude-sonnet-4-5-20250929",
	})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot override reserved variable")
	assert.Contains(t, err.Error(), "ANTHROPIC_MODEL")
}

// TestSpinCommand_EnvFileAndEnvFlagCombination tests that both --env and --env-file work together
func TestSpinCommand_EnvFileAndEnvFlagCombination(t *testing.T) {
	// Create a temporary env file
	tmpfile, err := os.CreateTemp("", "test-env-*.env")
	assert.NoError(t, err)

	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_, _ = tmpfile.WriteString("NPM_TOKEN=abc123\n")
	_ = tmpfile.Close()

	cmd := setupSpinCommandWithMocks(t)

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--image", "spinner:test",
		"--repo", "https://github.com/test/repo.git",
		"--env", "MY_VAR=value",
		"--env-file", tmpfile.Name(),
	})

	err = cmd.Execute()

	assert.NoError(t, err)
}
