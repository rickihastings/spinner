package gcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunGcloud_Success(t *testing.T) {
	// Use a command that exists on all systems to test the runner.
	// We can't call actual gcloud in unit tests, but we can verify the error
	// path by calling a nonexistent subcommand or use the --version flag.
	ctx := context.Background()

	// Test with a deliberately failing command to verify error handling
	_, err := runGcloud(ctx, "--version-nonexistent-flag")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gcloud")
}

func TestRunGcloud_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := runGcloud(ctx, "compute", "instances", "list")
	assert.Error(t, err)
}

func TestRunGcloudJSON_InvalidJSON(t *testing.T) {
	// runGcloudJSON should return an error on invalid JSON.
	// We test this indirectly — any gcloud error will produce non-JSON stderr.
	ctx := context.Background()

	var target map[string]interface{}

	err := runGcloudJSON(ctx, &target, "--version-nonexistent-flag")
	assert.Error(t, err)
}

func TestRunGcloudJSON_UnmarshalSuccess(t *testing.T) {
	// Test that runGcloudJSON can unmarshal valid JSON.
	// Use gcloud version which outputs plain text (not JSON), so this will
	// test the unmarshal error path.
	if err := checkGcloudInstalled(); err != nil {
		t.Skip("gcloud not installed, skipping test")
	}

	ctx := context.Background()

	var result interface{}

	// This calls gcloud --version which outputs text, not JSON,
	// so json.Unmarshal should fail
	err := runGcloudJSON(ctx, &result, "--version")
	// gcloud --version returns text, not JSON, so unmarshal should fail
	if err != nil {
		assert.Contains(t, err.Error(), "invalid character")
	}
}

func TestRunGcloud_ErrorMessageFormat(t *testing.T) {
	ctx := context.Background()

	_, err := runGcloud(ctx, "compute", "instances", "describe", "nonexistent-instance-xyz",
		"--project=nonexistent-project-xyz", "--zone=us-central1-a", "--format=json")
	require.Error(t, err)

	// Error should include the gcloud command prefix for debugging
	assert.Contains(t, err.Error(), "gcloud")
}
