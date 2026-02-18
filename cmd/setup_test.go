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
	cmd := NewSetupCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required flag: --name")
	mockProvider.AssertNotCalled(t, "Setup")
}

// TestSetupCommand_NameOnly tests successful setup with only --name flag
func TestSetupCommand_NameOnly(t *testing.T) {
	mockProvider := new(provider.MockProvider)

	mockProvider.On("Setup", mock.Anything, mock.MatchedBy(func(cfg provider.SetupConfig) bool {
		return cfg.Name == "test-env" && len(cfg.ProviderArgs) == 0
	})).Return(nil)

	cmd := NewSetupCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--name", "test-env"})

	err := cmd.Execute()

	assert.NoError(t, err)
	mockProvider.AssertExpectations(t)
}

// TestSetupCommand_WithProviderArgs tests setup with --provider-args
func TestSetupCommand_WithProviderArgs(t *testing.T) {
	mockProvider := new(provider.MockProvider)

	mockProvider.On("Setup", mock.Anything, mock.MatchedBy(func(cfg provider.SetupConfig) bool {
		return cfg.Name == "test-env" &&
			len(cfg.ProviderArgs) == 2 &&
			cfg.ProviderArgs[0] == "--build-arg=BASE_IMAGE=node:20" &&
			cfg.ProviderArgs[1] == "--no-cache"
	})).Return(nil)

	cmd := NewSetupCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--name", "test-env",
		"--provider-args", "--build-arg=BASE_IMAGE=node:20",
		"--provider-args", "--no-cache",
	})

	err := cmd.Execute()

	assert.NoError(t, err)
	mockProvider.AssertExpectations(t)
}

// TestSetupCommand_FlagCombinations is a table-driven test for various flag combinations
func TestSetupCommand_FlagCombinations(t *testing.T) {
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
			name:      "name with provider-args (valid)",
			args:      []string{"--name", "test", "--provider-args", "--build-arg=BASE_IMAGE=ubuntu:22.04"},
			wantError: false,
			mockSetup: func(m *provider.MockProvider) {
				m.On("Setup", mock.Anything, mock.Anything).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := new(provider.MockProvider)
			tt.mockSetup(mockProvider)

			cmd := NewSetupCommand(testFactory(mockProvider))
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

// TestSetupCommand_GCPFlagsWithDockerBackend tests that GCP flags error with docker backend
func TestSetupCommand_GCPFlagsWithDockerBackend(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		errorMsg string
	}{
		{
			name:     "project flag with docker backend",
			args:     []string{"--name", "test", "--project", "my-proj"},
			errorMsg: "--project requires --backend gcp",
		},
		{
			name:     "zone flag with docker backend",
			args:     []string{"--name", "test", "--zone", "us-central1-a"},
			errorMsg: "--zone requires --backend gcp",
		},
		{
			name:     "state-bucket flag with docker backend",
			args:     []string{"--name", "test", "--state-bucket", "my-bucket"},
			errorMsg: "--state-bucket requires --backend gcp",
		},
		{
			name:     "bake-script flag with docker backend",
			args:     []string{"--name", "test", "--bake-script", "/tmp/bake.sh"},
			errorMsg: "--bake-script requires --backend gcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := new(provider.MockProvider)
			cmd := NewSetupCommand(testFactory(mockProvider))

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

// TestSetupCommand_UnknownBackend tests that an unknown backend errors with available list
func TestSetupCommand_UnknownBackend(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	cmd := NewSetupCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--name", "test", "--backend", "kubernetes"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown backend")
	assert.Contains(t, err.Error(), "docker")
}

// TestSetupCommand_BakeScriptFileNotFound tests that --bake-script errors when file doesn't exist
func TestSetupCommand_BakeScriptFileNotFound(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	cmd := NewSetupCommand(testGCPFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--name", "test",
		"--backend", "gcp",
		"--project", "my-proj",
		"--zone", "us-central1-a",
		"--state-bucket", "my-bucket",
		"--bake-script", "/nonexistent/bake.sh",
	})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bake script not found")
}

// TestSetupCommand_BakeScriptWithGCPBackend tests that --bake-script is accepted with GCP backend
func TestSetupCommand_BakeScriptWithGCPBackend(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "bake-script-*.sh")
	assert.NoError(t, err)

	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_, _ = tmpfile.WriteString("#!/bin/bash\napt-get install -y python3\n")
	_ = tmpfile.Close()

	mockProvider := new(provider.MockProvider)
	mockProvider.On("Setup", mock.Anything, mock.MatchedBy(func(cfg provider.SetupConfig) bool {
		return cfg.Name == "test" && cfg.Options["bake-script"] == tmpfile.Name()
	})).Return(nil)

	cmd := NewSetupCommand(testGCPFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--name", "test",
		"--backend", "gcp",
		"--project", "my-proj",
		"--zone", "us-central1-a",
		"--state-bucket", "my-bucket",
		"--bake-script", tmpfile.Name(),
	})

	err = cmd.Execute()

	assert.NoError(t, err)
	mockProvider.AssertExpectations(t)
}

// TestSetupCommand_DockerBackendExplicit tests explicit --backend docker works
func TestSetupCommand_DockerBackendExplicit(t *testing.T) {
	mockProvider := new(provider.MockProvider)
	mockProvider.On("Setup", mock.Anything, mock.Anything).Return(nil)

	cmd := NewSetupCommand(testFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--name", "test", "--backend", "docker"})

	err := cmd.Execute()

	assert.NoError(t, err)
	mockProvider.AssertExpectations(t)
}

// TestSetupCommand_MissingRequiredGCPFlags tests that required GCP flags produce errors
func TestSetupCommand_MissingRequiredGCPFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		errorMsg string
	}{
		{
			name: "missing project",
			args: []string{
				"--name", "test",
				"--backend", "gcp",
				"--zone", "us-central1-a",
				"--state-bucket", "bucket",
			},
			errorMsg: "--project is required for GCP backend",
		},
		{
			name: "missing zone",
			args: []string{
				"--name", "test",
				"--backend", "gcp",
				"--project", "my-project",
				"--state-bucket", "bucket",
			},
			errorMsg: "--zone is required for GCP backend",
		},
		{
			name: "missing state-bucket",
			args: []string{
				"--name", "test",
				"--backend", "gcp",
				"--project", "my-project",
				"--zone", "us-central1-a",
			},
			errorMsg: "--state-bucket is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear GCP env vars to ensure we're testing CLI validation
			t.Setenv("SPINNER_PROJECT", "")
			t.Setenv("SPINNER_ZONE", "")
			t.Setenv("SPINNER_STATE_BUCKET", "")

			mockProvider := new(provider.MockProvider)
			cmd := NewSetupCommand(testGCPFactory(mockProvider))

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

// TestSetupCommand_BackendFromEnvVar tests that SPINNER_BACKEND env var is respected
func TestSetupCommand_BackendFromEnvVar(t *testing.T) {
	t.Setenv("SPINNER_BACKEND", "gcp")
	// Clear GCP config env vars so validation fails
	t.Setenv("SPINNER_PROJECT", "")
	t.Setenv("SPINNER_ZONE", "")
	t.Setenv("SPINNER_STATE_BUCKET", "")

	mockProvider := new(provider.MockProvider)
	cmd := NewSetupCommand(testGCPFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"--name", "test"})

	err := cmd.Execute()

	// Should error about missing GCP flags because backend=gcp from env
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--project is required")
}

// TestSetupCommand_CLIFlagOverridesEnvVar tests that CLI flag takes precedence over env var
func TestSetupCommand_CLIFlagOverridesEnvVar(t *testing.T) {
	t.Setenv("SPINNER_BACKEND", "docker")
	// Clear GCP config env vars so validation fails
	t.Setenv("SPINNER_PROJECT", "")
	t.Setenv("SPINNER_ZONE", "")
	t.Setenv("SPINNER_STATE_BUCKET", "")

	mockProvider := new(provider.MockProvider)
	cmd := NewSetupCommand(testGCPFactory(mockProvider))

	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	// CLI specifies --backend gcp, should override env SPINNER_BACKEND=docker
	cmd.SetArgs([]string{
		"--name", "test",
		"--backend", "gcp",
	})

	err := cmd.Execute()

	// Should error about missing GCP flags because CLI --backend gcp overrides env
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--project is required")
}
