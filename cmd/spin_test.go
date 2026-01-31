package cmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/rickihastings/spinner/internal/docker"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDockerClient is a mock implementation of docker.DockerClient for testing
type MockDockerClient struct {
	mock.Mock
}

func (m *MockDockerClient) BuildImage(ctx context.Context, config docker.BuildConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockDockerClient) ImageExists(ctx context.Context, image string) (bool, error) {
	args := m.Called(ctx, image)
	return args.Bool(0), args.Error(1)
}

func (m *MockDockerClient) ContainerExists(ctx context.Context, name string) (docker.ContainerStatus, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(docker.ContainerStatus), args.Error(1)
}

func (m *MockDockerClient) RunContainer(ctx context.Context, config docker.SpinConfig, containerName string, hasNpmrc bool) (docker.ContainerResult, error) {
	args := m.Called(ctx, config, containerName, hasNpmrc)
	return args.Get(0).(docker.ContainerResult), args.Error(1)
}

func (m *MockDockerClient) VerifyContainerStatus(ctx context.Context, containerName string) (docker.ContainerResult, error) {
	args := m.Called(ctx, containerName)
	return args.Get(0).(docker.ContainerResult), args.Error(1)
}

func (m *MockDockerClient) RemoveContainer(ctx context.Context, containerName string) (docker.ContainerResult, error) {
	args := m.Called(ctx, containerName)
	return args.Get(0).(docker.ContainerResult), args.Error(1)
}

func (m *MockDockerClient) RestartContainer(ctx context.Context, containerName string) (docker.ContainerResult, error) {
	args := m.Called(ctx, containerName)
	return args.Get(0).(docker.ContainerResult), args.Error(1)
}

func setupSpinCommandWithMocks() *cobra.Command {
	mockClient := new(MockDockerClient)
	cmd := NewSpinCommand(mockClient)

	// Mock the prerequisites validation to pass
	mockClient.On("ImageExists", mock.Anything, "spinner:test").Return(true, nil)

	// Mock container operations
	mockClient.On("ContainerExists", mock.Anything, mock.Anything).Return(docker.StatusNone, nil)
	mockClient.On("RunContainer", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(docker.ContainerResult{Success: true}, nil)
	mockClient.On("VerifyContainerStatus", mock.Anything, mock.Anything).Return(docker.ContainerResult{Success: true}, nil)

	return cmd
}

// TestSpinCommand_MissingImageFlag tests that spin command fails when --image flag is missing
func TestSpinCommand_MissingImageFlag(t *testing.T) {
	mockClient := new(MockDockerClient)
	cmd := NewSpinCommand(mockClient)

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
	mockClient := new(MockDockerClient)
	cmd := NewSpinCommand(mockClient)

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
	cmd := setupSpinCommandWithMocks()

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--image", "spinner:test", "--repo", "https://github.com/test/repo.git", "--prompt", "Fix the bug"})

	err := cmd.Execute()

	// The command should succeed (the prompt is optional and doesn't cause errors)
	assert.NoError(t, err)
}

// TestSpinCommand_BranchFlagParsing tests that --branch flag is correctly parsed
func TestSpinCommand_BranchFlagParsing(t *testing.T) {
	cmd := setupSpinCommandWithMocks()

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--image", "spinner:test", "--repo", "https://github.com/test/repo.git", "--branch", "feature-branch"})

	err := cmd.Execute()

	assert.NoError(t, err)
}

// TestSpinCommand_MaxIterationsFlagParsing tests that --max-iterations flag is correctly parsed
func TestSpinCommand_MaxIterationsFlagParsing(t *testing.T) {
	cmd := setupSpinCommandWithMocks()

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--image", "spinner:test", "--repo", "https://github.com/test/repo.git", "--max-iterations", "50"})

	err := cmd.Execute()

	assert.NoError(t, err)
}

// TestSpinCommand_RecreateFlagParsing tests that --recreate flag is correctly parsed
func TestSpinCommand_RecreateFlagParsing(t *testing.T) {
	mockClient := new(MockDockerClient)
	cmd := NewSpinCommand(mockClient)

	// Mock the prerequisites validation to pass
	mockClient.On("ImageExists", mock.Anything, "spinner:test").Return(true, nil)

	// Mock container operations - container exists and should be removed
	mockClient.On("ContainerExists", mock.Anything, mock.Anything).Return(docker.StatusRunning, nil)
	mockClient.On("RemoveContainer", mock.Anything, mock.Anything).Return(docker.ContainerResult{Success: true}, nil)
	mockClient.On("RunContainer", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(docker.ContainerResult{Success: true}, nil)
	mockClient.On("VerifyContainerStatus", mock.Anything, mock.Anything).Return(docker.ContainerResult{Success: true}, nil)

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--image", "spinner:test", "--repo", "https://github.com/test/repo.git", "--recreate"})

	err := cmd.Execute()

	assert.NoError(t, err)
	mockClient.AssertCalled(t, "RemoveContainer", mock.Anything, mock.Anything)
}

// TestSpinCommand_SetupFlagParsing tests that --setup flag is correctly parsed
func TestSpinCommand_SetupFlagParsing(t *testing.T) {
	mockClient := new(MockDockerClient)
	cmd := NewSpinCommand(mockClient)

	// Mock setup operations
	mockClient.On("BuildImage", mock.Anything, mock.Anything).Return(nil)

	// Mock the prerequisites validation to pass
	mockClient.On("ImageExists", mock.Anything, "spinner:test").Return(true, nil)

	// Mock container operations
	mockClient.On("ContainerExists", mock.Anything, mock.Anything).Return(docker.StatusNone, nil)
	mockClient.On("RunContainer", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(docker.ContainerResult{Success: true}, nil)
	mockClient.On("VerifyContainerStatus", mock.Anything, mock.Anything).Return(docker.ContainerResult{Success: true}, nil)

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--setup", "--image", "test", "--repo", "https://github.com/test/repo.git"})

	err := cmd.Execute()

	assert.NoError(t, err)
	mockClient.AssertCalled(t, "BuildImage", mock.Anything, mock.Anything)
}

// TestSpinCommand_SetupWithBaseImageValidation tests that --base-image requires --setup flag
func TestSpinCommand_SetupWithBaseImageValidation(t *testing.T) {
	mockClient := new(MockDockerClient)
	cmd := NewSpinCommand(mockClient)

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
	mockClient := new(MockDockerClient)
	cmd := NewSpinCommand(mockClient)

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
			mockClient := new(MockDockerClient)
			cmd := NewSpinCommand(mockClient)

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
