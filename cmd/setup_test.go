package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/rickihastings/spinner/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestSetupCommand_MissingNameFlag tests that the setup command returns an error when --name is missing
func TestSetupCommand_MissingNameFlag(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	cmd := NewSetupCommand(mockProvider)

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required flag: --name")
	mockProvider.AssertNotCalled(t, "Setup")
}

// TestSetupCommand_MutuallyExclusiveFlags tests that --base-image and --dockerfile are mutually exclusive
func TestSetupCommand_MutuallyExclusiveFlags(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	cmd := NewSetupCommand(mockProvider)

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
	mockProvider.AssertNotCalled(t, "Setup")
}

// TestSetupCommand_NameOnly tests successful setup with only --name flag
func TestSetupCommand_NameOnly(t *testing.T) {
	mockProvider := new(provider.MockProvider)

	mockProvider.On("Setup", mock.Anything, mock.MatchedBy(func(cfg provider.SetupConfig) bool {
		return cfg.Name == "test-env" && cfg.Options["base-image"] == "" && cfg.Options["dockerfile"] == ""
	})).Return(nil)

	cmd := NewSetupCommand(mockProvider)

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--name", "test-env"})

	err := cmd.Execute()

	assert.NoError(t, err)
	mockProvider.AssertExpectations(t)
}

// TestSetupCommand_WithBaseImage tests successful setup with --name and --base-image
func TestSetupCommand_WithBaseImage(t *testing.T) {
	mockProvider := new(provider.MockProvider)

	mockProvider.On("Setup", mock.Anything, mock.MatchedBy(func(cfg provider.SetupConfig) bool {
		return cfg.Name == "test-env" && cfg.Options["base-image"] == "node:20" && cfg.Options["dockerfile"] == ""
	})).Return(nil)

	cmd := NewSetupCommand(mockProvider)

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--name", "test-env",
		"--base-image", "node:20",
	})

	err := cmd.Execute()

	assert.NoError(t, err)
	mockProvider.AssertExpectations(t)
}

// TestSetupCommand_WithDockerfile tests successful setup with --name and --dockerfile
func TestSetupCommand_WithDockerfile(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "Dockerfile.*")
	assert.NoError(t, err)

	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_ = tmpfile.Close()

	mockProvider := new(provider.MockProvider)

	mockProvider.On("Setup", mock.Anything, mock.MatchedBy(func(cfg provider.SetupConfig) bool {
		return cfg.Name == "test-env" && cfg.Options["base-image"] == "" && cfg.Options["dockerfile"] == tmpfile.Name()
	})).Return(nil)

	cmd := NewSetupCommand(mockProvider)

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--name", "test-env",
		"--dockerfile", tmpfile.Name(),
	})

	err = cmd.Execute()

	assert.NoError(t, err)
	mockProvider.AssertExpectations(t)
}

// TestSetupCommand_FlagCombinations is a table-driven test for various flag combinations
func TestSetupCommand_FlagCombinations(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "Dockerfile.*")
	assert.NoError(t, err)

	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_ = tmpfile.Close()

	tests := []struct {
		name        string
		args        []string
		wantError   bool
		errorString string
		mockSetup   func(*provider.MockProvider)
	}{
		{
			name:        "missing name flag",
			args:        []string{},
			wantError:   true,
			errorString: "missing required flag: --name",
			mockSetup:   func(m *provider.MockProvider) {},
		},
		{
			name:      "name only (valid)",
			args:      []string{"--name", "test"},
			wantError: false,
			mockSetup: func(m *provider.MockProvider) {
				m.On("Setup", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:      "name with base-image (valid)",
			args:      []string{"--name", "test", "--base-image", "ubuntu:22.04"},
			wantError: false,
			mockSetup: func(m *provider.MockProvider) {
				m.On("Setup", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:      "name with dockerfile (valid)",
			args:      []string{"--name", "test", "--dockerfile", tmpfile.Name()},
			wantError: false,
			mockSetup: func(m *provider.MockProvider) {
				m.On("Setup", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name:        "mutually exclusive flags",
			args:        []string{"--name", "test", "--base-image", "ubuntu:22.04", "--dockerfile", tmpfile.Name()},
			wantError:   true,
			errorString: "mutually exclusive",
			mockSetup:   func(m *provider.MockProvider) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := new(provider.MockProvider)
			tt.mockSetup(mockProvider)

			cmd := NewSetupCommand(mockProvider)
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
				mockProvider.AssertExpectations(t)
			}
		})
	}
}
