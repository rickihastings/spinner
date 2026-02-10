package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rickihastings/spinner/tests/testutil"
)

// TestGCPSpin_NonExistentImage tests that referencing a non-existent image produces a clear error.
func TestGCPSpin_NonExistentImage(t *testing.T) {
	cfg := testutil.SkipIfGCPNotAvailable(t)

	args := []string{
		"spin",
		"--backend", "gcp",
		"--image", "nonexistent-image-12345",
		"--repo", testRepo,
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
	}
	stdout, stderr, exitCode := testutil.RunCommandExpectError(t, args...)
	output := stdout + stderr

	assert.NotEqual(t, 0, exitCode, "should fail with non-existent image")
	assert.Contains(t, output, "not found", "should indicate image was not found")
}

// TestGCPSpin_EnvVarsInMetadata tests that --env flags are passed as SPINNER_ENV_ prefixed instance metadata.
func TestGCPSpin_EnvVarsInMetadata(t *testing.T) {
	cfg := testutil.SkipIfGCPNotAvailable(t)

	imageName := testutil.GetSharedGCPImage(t)

	spinArgs := []string{
		"spin",
		"--backend", "gcp",
		"--image", imageName,
		"--repo", testRepo,
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
		"--env", "MY_CUSTOM_VAR=hello_world",
		"--env", "API_KEY=secret123",
	}

	instanceName, _, _ := testutil.RunGCPSpinCommand(t, spinArgs...)
	require.NotEmpty(t, instanceName, "should get instance name")

	t.Cleanup(func() {
		testutil.RemoveGCPInstance(t, cfg.Project, cfg.Zone, instanceName)
		testutil.CleanupGCSPrefix(t, cfg.Bucket, instanceName+"/")
	})

	// Verify env vars are stored as SPINNER_ENV_ prefixed metadata
	val, found := testutil.GCPInstanceMetadata(t, cfg.Project, cfg.Zone, instanceName, "SPINNER_ENV_MY_CUSTOM_VAR")
	assert.True(t, found, "SPINNER_ENV_MY_CUSTOM_VAR metadata should exist")
	assert.Equal(t, "hello_world", val, "MY_CUSTOM_VAR should have correct value")

	val, found = testutil.GCPInstanceMetadata(t, cfg.Project, cfg.Zone, instanceName, "SPINNER_ENV_API_KEY")
	assert.True(t, found, "SPINNER_ENV_API_KEY metadata should exist")
	assert.Equal(t, "secret123", val, "API_KEY should have correct value")
}
