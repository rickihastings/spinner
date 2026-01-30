package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/rickihastings/spinner/internal/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestSetupCommand_MissingNameFlag tests that the setup command returns an error when --name is missing
func TestSetupCommand_MissingNameFlag(t *testing.T) {
	mockClient := new(docker.MockDockerClient)
	cmd := NewSetupCommand(mockClient)

	// Capture output
	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required flag: --name")
	// Verify no Docker operations were attempted
	mockClient.AssertNotCalled(t, "BuildImage")
}

// TestSetupCommand_MutuallyExclusiveFlags tests that --base-image and --dockerfile are mutually exclusive
func TestSetupCommand_MutuallyExclusiveFlags(t *testing.T) {
	mockClient := new(docker.MockDockerClient)
	cmd := NewSetupCommand(mockClient)

	// Capture output
	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--name", "test-env",
		"--base-image", "ubuntu:22.04",
		"--dockerfile", "Dockerfile.custom",
	})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
	// Verify no Docker operations were attempted
	mockClient.AssertNotCalled(t, "BuildImage")
}

// TestSetupCommand_NameOnly tests successful setup with only --name flag
func TestSetupCommand_NameOnly(t *testing.T) {
	mockClient := new(docker.MockDockerClient)

	// Mock the BuildImage call to succeed
	mockClient.On("BuildImage", mock.Anything, mock.MatchedBy(func(cfg docker.BuildConfig) bool {
		return cfg.Name == "test-env" && cfg.BaseImage == "" && cfg.Dockerfile == ""
	})).Return(nil)

	cmd := NewSetupCommand(mockClient)

	// Capture output
	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--name", "test-env"})

	err := cmd.Execute()

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

// TestSetupCommand_WithBaseImage tests successful setup with --name and --base-image
func TestSetupCommand_WithBaseImage(t *testing.T) {
	mockClient := new(docker.MockDockerClient)

	// Mock the BuildImage call to succeed
	mockClient.On("BuildImage", mock.Anything, mock.MatchedBy(func(cfg docker.BuildConfig) bool {
		return cfg.Name == "test-env" && cfg.BaseImage == "node:20" && cfg.Dockerfile == ""
	})).Return(nil)

	cmd := NewSetupCommand(mockClient)

	// Capture output
	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--name", "test-env",
		"--base-image", "node:20",
	})

	err := cmd.Execute()

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

// TestSetupCommand_WithDockerfile tests successful setup with --name and --dockerfile
func TestSetupCommand_WithDockerfile(t *testing.T) {
	// Create a temporary Dockerfile
	tmpfile, err := os.CreateTemp("", "Dockerfile.*")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	mockClient := new(docker.MockDockerClient)

	// Mock the BuildImage call to succeed
	mockClient.On("BuildImage", mock.Anything, mock.MatchedBy(func(cfg docker.BuildConfig) bool {
		return cfg.Name == "test-env" && cfg.BaseImage == "" && cfg.Dockerfile == tmpfile.Name()
	})).Return(nil)

	cmd := NewSetupCommand(mockClient)

	// Capture output
	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--name", "test-env",
		"--dockerfile", tmpfile.Name(),
	})

	err = cmd.Execute()

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

// TestSetupCommand_NonExistentDockerfile tests validation of non-existent Dockerfile path
func TestSetupCommand_NonExistentDockerfile(t *testing.T) {
	mockClient := new(docker.MockDockerClient)
	cmd := NewSetupCommand(mockClient)

	// Capture output
	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--name", "test-env",
		"--dockerfile", "/nonexistent/path/Dockerfile",
	})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Dockerfile not found at path")
	// Verify no Docker operations were attempted
	mockClient.AssertNotCalled(t, "BuildImage")
}

// TestSetupCommand_FlagCombinations is a table-driven test for various flag combinations
func TestSetupCommand_FlagCombinations(t *testing.T) {
	// Create a temporary Dockerfile for valid Dockerfile tests
	tmpfile, err := os.CreateTemp("", "Dockerfile.*")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	tests := []struct {
		name        string
		args        []string
		wantError   bool
		errorString string
		mockSetup   func(*docker.MockDockerClient)
	}{
		{
			name:        "missing name flag",
			args:        []string{},
			wantError:   true,
			errorString: "missing required flag: --name",
			mockSetup:   func(m *docker.MockDockerClient) {},
		},
		{
			name:      "name only (valid)",
			args:      []string{"--name", "test"},
			wantError: false,
			mockSetup: func(m *docker.MockDockerClient) {
				m.On("BuildImage", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:      "name with base-image (valid)",
			args:      []string{"--name", "test", "--base-image", "ubuntu:22.04"},
			wantError: false,
			mockSetup: func(m *docker.MockDockerClient) {
				m.On("BuildImage", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:      "name with dockerfile (valid)",
			args:      []string{"--name", "test", "--dockerfile", tmpfile.Name()},
			wantError: false,
			mockSetup: func(m *docker.MockDockerClient) {
				m.On("BuildImage", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:        "mutually exclusive flags",
			args:        []string{"--name", "test", "--base-image", "ubuntu:22.04", "--dockerfile", tmpfile.Name()},
			wantError:   true,
			errorString: "mutually exclusive",
			mockSetup:   func(m *docker.MockDockerClient) {},
		},
		{
			name:        "non-existent dockerfile",
			args:        []string{"--name", "test", "--dockerfile", "/nonexistent/Dockerfile"},
			wantError:   true,
			errorString: "Dockerfile not found",
			mockSetup:   func(m *docker.MockDockerClient) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(docker.MockDockerClient)
			tt.mockSetup(mockClient)

			cmd := NewSetupCommand(mockClient)
			b := new(bytes.Buffer)
			cmd.SetOut(b)
			cmd.SetErr(b)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				mockClient.AssertExpectations(t)
			}
		})
	}
}
