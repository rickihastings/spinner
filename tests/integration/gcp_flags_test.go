package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rickihastings/spinner/tests/testutil"
)

// TestGCPFlags_WrongBackendFlagsError tests that GCP-specific flags fail with Docker backend.
func TestGCPFlags_WrongBackendFlagsError(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "project flag with docker backend",
			args:    []string{"setup", "--name", "test", "--project", "my-project"},
			wantErr: "--project requires --backend gcp",
		},
		{
			name:    "zone flag with docker backend",
			args:    []string{"setup", "--name", "test", "--zone", "us-central1-a"},
			wantErr: "--zone requires --backend gcp",
		},
		{
			name:    "state-bucket flag with docker backend",
			args:    []string{"setup", "--name", "test", "--state-bucket", "my-bucket"},
			wantErr: "--state-bucket requires --backend gcp",
		},
		{
			name:    "project flag on spin with docker backend",
			args:    []string{"spin", "--image", "spinner:test", "--repo", "https://github.com/octocat/Hello-World.git", "--project", "my-project"},
			wantErr: "--project requires --backend gcp",
		},
		{
			name:    "zone flag on spin with docker backend",
			args:    []string{"spin", "--image", "spinner:test", "--repo", "https://github.com/octocat/Hello-World.git", "--zone", "us-central1-a"},
			wantErr: "--zone requires --backend gcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, _ := testutil.RunCommandExpectError(t, tt.args...)
			output := stdout + stderr
			assert.Contains(t, output, tt.wantErr, "should show error for wrong-backend flag")
		})
	}
}

// TestGCPFlags_UnknownBackendError tests that unknown backend names produce clear errors.
func TestGCPFlags_UnknownBackendError(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "unknown backend on setup",
			args:    []string{"setup", "--backend", "kubernetes", "--name", "test"},
			wantErr: "unknown backend",
		},
		{
			name:    "unknown backend on spin",
			args:    []string{"spin", "--backend", "aws", "--image", "test", "--repo", "https://github.com/octocat/Hello-World.git"},
			wantErr: "unknown backend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, _ := testutil.RunCommandExpectError(t, tt.args...)
			output := stdout + stderr
			assert.Contains(t, output, tt.wantErr, "should show unknown backend error")
		})
	}
}

// TestGCPFlags_PrecedenceChain tests the flag precedence: CLI > env > config > defaults.
func TestGCPFlags_PrecedenceChain(t *testing.T) {
	t.Run("CLI flag overrides env var", func(t *testing.T) {
		// Set env var for backend
		env := map[string]string{
			"SPINNER_BACKEND": "docker",
		}

		// Use --backend gcp on CLI (should override env var)
		// This will fail because required GCP flags are missing, but
		// the error should be about GCP flags, not about "unknown backend"
		args := []string{"setup", "--backend", "gcp", "--name", "test"}
		stdout, stderr, _ := testutil.RunCommandWithEnv(t, env, args...)
		output := stdout + stderr

		// If backend=gcp was picked up (overriding env), we should see
		// GCP-specific validation errors (not Docker errors)
		assert.Contains(t, output, "--project is required for GCP backend",
			"CLI --backend gcp should override SPINNER_BACKEND=docker env var")
	})

	t.Run("env var provides default backend", func(t *testing.T) {
		// Set env var for backend
		env := map[string]string{
			"SPINNER_BACKEND": "gcp",
		}

		// Don't provide --backend on CLI
		args := []string{"setup", "--name", "test"}
		stdout, stderr, _ := testutil.RunCommandWithEnv(t, env, args...)
		output := stdout + stderr

		// Should use GCP backend from env var (show GCP-specific errors)
		assert.Contains(t, output, "--project is required for GCP backend",
			"SPINNER_BACKEND env var should set default backend to gcp")
	})

	t.Run("default backend is docker", func(t *testing.T) {
		// No env var, no --backend flag — default is docker.
		// We just verify it doesn't require GCP-specific flags.
		args := []string{"setup", "--name", "test"}
		stdout, stderr, _ := testutil.RunCommand(t, args...)
		output := stdout + stderr

		// Should NOT show GCP-specific errors (default backend is docker)
		assert.NotContains(t, output, "--project is required",
			"default backend should be docker, not gcp")
		assert.NotContains(t, output, "--zone is required",
			"default backend should be docker, not gcp")
		assert.NotContains(t, output, "--state-bucket is required",
			"default backend should be docker, not gcp")
	})
}

// TestGCPFlags_BackendFromEnvVar tests that SPINNER_BACKEND env var is respected.
func TestGCPFlags_BackendFromEnvVar(t *testing.T) {
	// Set SPINNER_BACKEND=gcp via env
	env := map[string]string{
		"SPINNER_BACKEND": "gcp",
	}

	args := []string{"setup", "--name", "test-env-backend"}
	stdout, stderr, _ := testutil.RunCommandWithEnv(t, env, args...)
	output := stdout + stderr

	// Should use GCP backend (shows GCP-specific validation errors)
	assert.Contains(t, output, "--project is required",
		"should use GCP backend from SPINNER_BACKEND env var")
}

// TestGCPFlags_SetupMissingName tests that --name is always required.
func TestGCPFlags_SetupMissingName(t *testing.T) {
	args := []string{
		"setup",
		"--backend", "gcp",
		"--project", "p",
		"--zone", "z",
		"--state-bucket", "b",
	}
	stdout, stderr, _ := testutil.RunCommandExpectError(t, args...)
	output := stdout + stderr

	assert.Contains(t, output, "--name", "should error about missing --name flag")
}

// TestGCPFlags_SpinMissingImage tests that --image is always required for spin.
func TestGCPFlags_SpinMissingImage(t *testing.T) {
	args := []string{
		"spin",
		"--backend", "gcp",
		"--repo", "https://github.com/octocat/Hello-World.git",
		"--project", "p",
		"--zone", "z",
		"--state-bucket", "b",
	}
	stdout, stderr, _ := testutil.RunCommandExpectError(t, args...)
	output := stdout + stderr

	assert.Contains(t, output, "--image", "should error about missing --image flag")
}

// TestGCPFlags_SpinMissingRepo tests that --repo is always required for spin.
func TestGCPFlags_SpinMissingRepo(t *testing.T) {
	args := []string{
		"spin",
		"--backend", "gcp",
		"--image", "test-img",
		"--project", "p",
		"--zone", "z",
		"--state-bucket", "b",
	}
	stdout, stderr, _ := testutil.RunCommandExpectError(t, args...)
	output := stdout + stderr

	assert.Contains(t, output, "--repo", "should error about missing --repo flag")
}

// TestGCPFlags_BakeScriptRequiresSetupOnSpin tests that --bake-script requires --setup on spin.
func TestGCPFlags_BakeScriptRequiresSetupOnSpin(t *testing.T) {
	// Create a valid bake script file
	tmpDir := t.TempDir()
	bakeScriptPath := filepath.Join(tmpDir, "bake.sh")

	err := os.WriteFile(bakeScriptPath, []byte("#!/bin/bash\necho ok"), 0644)
	if err != nil {
		t.Fatalf("failed to write bake script: %v", err)
	}

	args := []string{
		"spin",
		"--backend", "gcp",
		"--image", "test-img",
		"--repo", "https://github.com/octocat/Hello-World.git",
		"--project", "p",
		"--zone", "z",
		"--state-bucket", "b",
		"--bake-script", bakeScriptPath,
		// Note: no --setup flag
	}
	stdout, stderr, _ := testutil.RunCommandExpectError(t, args...)
	output := stdout + stderr

	assert.Contains(t, output, "--bake-script requires --setup",
		"should error that --bake-script requires --setup on spin")
}
