package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rickihastings/spinner/tests/testutil"
)

// TestGCPDestroy tests the destroy command with GCP backend
func TestGCPDestroy(t *testing.T) {
	cfg := testutil.SkipIfGCPNotAvailable(t)

	// Use shared image to avoid baking overhead
	imageName := testutil.GetSharedGCPImage(t)

	// Create a VM instance using spin
	spinArgs := []string{
		"spin",
		"--backend", "gcp",
		"--image", imageName,
		"--repo", testRepo,
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
	}
	instanceName, spinStdout, spinStderr := testutil.RunGCPSpinCommand(t, spinArgs...)
	require.NotEmpty(t, instanceName, "should get instance name from spin output: %s", spinStdout+spinStderr)

	// Verify instance exists
	require.True(t, testutil.GCPInstanceExists(t, cfg.Project, cfg.Zone, instanceName),
		"instance should exist before destroy")

	// Run destroy command
	destroyArgs := []string{
		"destroy",
		instanceName,
		"--backend", "gcp",
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
	}
	stdout, stderr := testutil.RunCommandExpectSuccess(t, destroyArgs...)
	output := stdout + stderr

	// Verify success message
	assert.Contains(t, output, "Instance '"+instanceName+"' destroyed", "should show success message")

	// Verify instance is removed
	assert.False(t, testutil.GCPInstanceExists(t, cfg.Project, cfg.Zone, instanceName),
		"instance should not exist after destroy")

	// Verify local state directory is removed
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err, "should get home directory")

	stateDir := filepath.Join(homeDir, ".spinner", instanceName)
	_, err = os.Stat(stateDir)
	assert.True(t, os.IsNotExist(err), "local state directory should not exist after destroy")

	// Note: GCS bucket state cleanup verification would require additional GCS client calls
	// The cleanup is handled by the GCP provider's Remove method
}

// TestGCPDestroy_NonExistent tests destroy command with non-existent GCP instance
func TestGCPDestroy_NonExistent(t *testing.T) {
	cfg := testutil.SkipIfGCPNotAvailable(t)

	// Run destroy command with non-existent instance
	destroyArgs := []string{
		"destroy",
		"non-existent-instance",
		"--backend", "gcp",
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
	}
	stdout, stderr, exitCode := testutil.RunCommandExpectError(t, destroyArgs...)
	require.NotEqual(t, 0, exitCode, "should fail when destroying non-existent instance")

	output := stdout + stderr

	// Verify error message
	assert.Contains(t, output, "Instance 'non-existent-instance' not found", "should show instance not found error")
	assert.Contains(t, output, "failed to destroy 1 of 1 instance(s)", "should show aggregate error")
}
