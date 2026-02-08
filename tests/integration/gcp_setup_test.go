package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rickihastings/spinner/tests/testutil"
)

// TestGCPSetup_MissingRequiredFlags tests that missing required GCP flags produce clear errors.
// Does not require actual GCP credentials — tests CLI flag validation only.
func TestGCPSetup_MissingRequiredFlags(t *testing.T) {

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name: "missing project",
			args: []string{
				"setup", "--backend", "gcp",
				"--name", "test-missing-project",
				"--zone", "us-central1-a",
				"--state-bucket", "test-bucket",
			},
			wantErr: "--project is required for GCP backend",
		},
		{
			name: "missing zone",
			args: []string{
				"setup", "--backend", "gcp",
				"--name", "test-missing-zone",
				"--project", "test-project",
				"--state-bucket", "test-bucket",
			},
			wantErr: "--zone is required for GCP backend",
		},
		{
			name: "missing state-bucket",
			args: []string{
				"setup", "--backend", "gcp",
				"--name", "test-missing-bucket",
				"--project", "test-project",
				"--zone", "us-central1-a",
			},
			wantErr: "--state-bucket is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, _ := testutil.RunCommandExpectError(t, tt.args...)
			output := stdout + stderr
			assert.Contains(t, output, tt.wantErr, "should show error for missing flag")
		})
	}
}

// TestGCPSetup_BakeScriptFileNotFound tests that a non-existent bake script path is rejected.
// Does not require actual GCP credentials — tests CLI flag validation only.
func TestGCPSetup_BakeScriptFileNotFound(t *testing.T) {

	args := []string{
		"setup",
		"--backend", "gcp",
		"--name", "test-bad-bake-script",
		"--project", "test-project",
		"--zone", "us-central1-a",
		"--state-bucket", "test-bucket",
		"--bake-script", "/nonexistent/path/bake.sh",
	}
	stdout, stderr, _ := testutil.RunCommandExpectError(t, args...)
	output := stdout + stderr

	assert.Contains(t, output, "bake script not found", "should error on non-existent bake script")
}
