package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/rickihastings/spinner/internal/provider"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupSpinCommandWithMocks(t *testing.T) *cobra.Command {
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")

	mockProvider := new(provider.MockProvider)
	mockProvider.On("InstanceName", mock.Anything).Return("test-container")
	mockProvider.On("Status", mock.Anything, "test-container").Return(provider.InstanceStatusNone, nil)
	mockProvider.On("Create", mock.Anything, mock.Anything).Return(
		&provider.Instance{Name: "test-container", Status: provider.InstanceStatusRunning}, nil,
	)

	return NewSpinCommand(testFactory(mockProvider))
}

// TestSpinCommand_MissingImageFlag tests that spin command fails when --image flag is missing
func TestSpinCommand_MissingImageFlag(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	cmd := NewSpinCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--repo", "git@github.com:test/repo.git"})

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

// TestSpinCommand_SetupWithBaseImageValidation tests that --base-image requires --setup flag
func TestSpinCommand_SetupWithBaseImageValidation(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	cmd := NewSpinCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--image", "test", "--repo", "https://github.com/test/repo.git", "--base-image", "ubuntu:22.04"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--base-image requires --setup flag")
}

// TestSpinCommand_SetupMutuallyExclusiveFlags tests that --base-image and --dockerfile are mutually exclusive with --setup
func TestSpinCommand_SetupMutuallyExclusiveFlags(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	cmd := NewSpinCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--setup",
		"--image", "test",
		"--repo", "https://github.com/test/repo.git",
		"--base-image", "ubuntu:22.04",
		"--dockerfile", "Dockerfile.custom",
	})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
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
		{
			name:      "base-image without setup",
			args:      []string{"--image", "test", "--repo", "https://github.com/test/repo.git", "--base-image", "ubuntu:22.04"},
			wantError: true,
			errorMsg:  "--base-image requires --setup flag",
		},
		{
			name:      "dockerfile without setup",
			args:      []string{"--image", "test", "--repo", "https://github.com/test/repo.git", "--dockerfile", "Dockerfile.custom"},
			wantError: true,
			errorMsg:  "--dockerfile requires --setup flag",
		},
		{
			name: "setup with base-image and dockerfile (mutually exclusive)",
			args: []string{
				"--setup",
				"--image", "test",
				"--repo", "https://github.com/test/repo.git",
				"--base-image", "ubuntu:22.04",
				"--dockerfile", "Dockerfile.custom",
			},
			wantError: true,
			errorMsg:  "mutually exclusive",
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
