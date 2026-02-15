package integration

import (
	"encoding/base64"
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
		"--service-account", cfg.ServiceAccount,
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
		"--branch", "test-env-vars",
		"--project", cfg.Project,
		"--service-account", cfg.ServiceAccount,
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

// TestGCPSpin_ModelFlagSetsMetadata tests that --model sets ANTHROPIC_MODEL in GCP instance metadata.
func TestGCPSpin_ModelFlagSetsMetadata(t *testing.T) {
	cfg := testutil.SkipIfGCPNotAvailable(t)

	imageName := testutil.GetSharedGCPImage(t)

	spinArgs := []string{
		"spin",
		"--backend", "gcp",
		"--image", imageName,
		"--repo", testRepo,
		"--branch", "test-model-flag",
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
		"--model", "claude-sonnet-4-5-20250929",
	}

	instanceName, _, _ := testutil.RunGCPSpinCommand(t, spinArgs...)
	require.NotEmpty(t, instanceName, "should get instance name")

	t.Cleanup(func() {
		testutil.RemoveGCPInstance(t, cfg.Project, cfg.Zone, instanceName)
		testutil.CleanupGCSPrefix(t, cfg.Bucket, instanceName+"/")
	})

	// Verify ANTHROPIC_MODEL is set in instance metadata
	val, found := testutil.GCPInstanceMetadata(t, cfg.Project, cfg.Zone, instanceName, "ANTHROPIC_MODEL")
	assert.True(t, found, "ANTHROPIC_MODEL metadata should exist")
	assert.Equal(t, "claude-sonnet-4-5-20250929", val, "ANTHROPIC_MODEL should match the --model flag value")
}

// TestGCPSpin_ModelFlagUpdatedOnRestart tests that restarting with a different --model updates the metadata.
func TestGCPSpin_ModelFlagUpdatedOnRestart(t *testing.T) {
	cfg := testutil.SkipIfGCPNotAvailable(t)

	imageName := testutil.GetSharedGCPImage(t)

	// 1. Spin with initial model
	spinArgs := []string{
		"spin",
		"--backend", "gcp",
		"--image", imageName,
		"--repo", testRepo,
		"--branch", "test-model-restart",
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
		"--model", "claude-sonnet-4-5-20250929",
	}

	instanceName, _, _ := testutil.RunGCPSpinCommand(t, spinArgs...)
	require.NotEmpty(t, instanceName, "should get instance name")

	t.Cleanup(func() {
		testutil.RemoveGCPInstance(t, cfg.Project, cfg.Zone, instanceName)
		testutil.CleanupGCSPrefix(t, cfg.Bucket, instanceName+"/")
	})

	// Verify initial model
	val, found := testutil.GCPInstanceMetadata(t, cfg.Project, cfg.Zone, instanceName, "ANTHROPIC_MODEL")
	assert.True(t, found, "ANTHROPIC_MODEL metadata should exist after initial spin")
	assert.Equal(t, "claude-sonnet-4-5-20250929", val, "initial model should be sonnet")

	// 2. Stop the instance
	testutil.StopGCPInstance(t, cfg.Project, cfg.Zone, instanceName)
	testutil.WaitForGCPInstanceStatus(t, cfg.Project, cfg.Zone, instanceName, "TERMINATED", 120)

	// 3. Spin again with a different model (triggers restart with metadata update)
	restartArgs := []string{
		"spin",
		"--backend", "gcp",
		"--image", imageName,
		"--repo", testRepo,
		"--branch", "test-model-restart",
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
		"--model", "claude-opus-4-6",
	}

	testutil.RunCommandExpectSuccess(t, restartArgs...)

	// 4. Verify model metadata was updated
	val, found = testutil.GCPInstanceMetadata(t, cfg.Project, cfg.Zone, instanceName, "ANTHROPIC_MODEL")
	assert.True(t, found, "ANTHROPIC_MODEL metadata should exist after restart")
	assert.Equal(t, "claude-opus-4-6", val, "model should be updated to opus after restart")
}

// TestGCPSpin_EnvFileInMetadata tests that --env-file content is base64-encoded
// and stored as SPINNER_ENV_FILE metadata on the GCP instance.
func TestGCPSpin_EnvFileInMetadata(t *testing.T) {
	cfg := testutil.SkipIfGCPNotAvailable(t)

	imageName := testutil.GetSharedGCPImage(t)

	// Create a temporary env file
	tmpDir := t.TempDir()
	envFilePath := tmpDir + "/.env"
	envContent := "DATABASE_URL=postgres://localhost\nSECRET_KEY=mysecret\n"
	err := testutil.WriteFile(t, envFilePath, envContent)
	require.NoError(t, err, "should create env file")

	spinArgs := []string{
		"spin",
		"--backend", "gcp",
		"--image", imageName,
		"--repo", testRepo,
		"--branch", "test-env-file",
		"--project", cfg.Project,
		"--service-account", cfg.ServiceAccount,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
		"--env-file", envFilePath,
	}

	instanceName, _, _ := testutil.RunGCPSpinCommand(t, spinArgs...)
	require.NotEmpty(t, instanceName, "should get instance name")

	t.Cleanup(func() {
		testutil.RemoveGCPInstance(t, cfg.Project, cfg.Zone, instanceName)
		testutil.CleanupGCSPrefix(t, cfg.Bucket, instanceName+"/")
	})

	// Verify env file content is base64-encoded in SPINNER_ENV_FILE metadata
	val, found := testutil.GCPInstanceMetadata(t, cfg.Project, cfg.Zone, instanceName, "SPINNER_ENV_FILE")
	assert.True(t, found, "SPINNER_ENV_FILE metadata should exist")

	// Decode and verify content matches
	decoded, err := base64.StdEncoding.DecodeString(val)
	require.NoError(t, err, "SPINNER_ENV_FILE should be valid base64")
	assert.Equal(t, envContent, string(decoded), "decoded env file content should match original")
}
