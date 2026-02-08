package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rickihastings/spinner/tests/testutil"
)

// TestGCPSpin_SuccessfulVMCreation tests successful VM creation with valid flags.
func TestGCPSpin_SuccessfulVMCreation(t *testing.T) {
	cfg := testutil.SkipIfGCPNotAvailable(t)

	// Use shared test image (created once in TestMain)
	imageName := testutil.GetSharedGCPImage(t)

	// Run spin command with GCP backend
	args := []string{
		"spin",
		"--backend", "gcp",
		"--image", imageName,
		"--repo", testRepo,
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
	}
	instanceName, stdout, stderr := testutil.RunGCPSpinCommand(t, args...)
	output := stdout + stderr

	require.NotEmpty(t, instanceName, "should extract instance name from output: %s", output)

	// Register cleanup
	t.Cleanup(func() {
		testutil.RemoveGCPInstance(t, cfg.Project, cfg.Zone, instanceName)
		testutil.CleanupGCSPrefix(t, cfg.Bucket, instanceName+"/")
	})

	// Verify success message
	assert.Contains(t, output, "Instance created successfully", "should show success message")

	// Verify VM exists and is running
	status := testutil.GCPInstanceStatus(t, cfg.Project, cfg.Zone, instanceName)
	assert.Equal(t, "RUNNING", status, "VM should be in RUNNING state")
}

// TestGCPSpin_DeterministicNaming tests that VM names are deterministic based on image + repo.
func TestGCPSpin_DeterministicNaming(t *testing.T) {
	cfg := testutil.SkipIfGCPNotAvailable(t)

	imageName := testutil.GetSharedGCPImage(t)

	args := []string{
		"spin",
		"--backend", "gcp",
		"--image", imageName,
		"--repo", testRepo,
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
	}
	instanceName, _, _ := testutil.RunGCPSpinCommand(t, args...)

	require.NotEmpty(t, instanceName, "should get instance name")

	t.Cleanup(func() {
		testutil.RemoveGCPInstance(t, cfg.Project, cfg.Zone, instanceName)
		testutil.CleanupGCSPrefix(t, cfg.Bucket, instanceName+"/")
	})

	// Instance name should follow the pattern: spinner-{image}-{repo}
	// For repo "Hello-World" -> "hello-world"
	assert.Contains(t, instanceName, "spinner-", "should start with spinner prefix")
	assert.Contains(t, instanceName, "hello-world", "should contain sanitized repo name")

	// Verify it's a valid GCP instance name: lowercase, [a-z0-9-], max 63 chars
	assert.Regexp(t, `^[a-z][a-z0-9-]*[a-z0-9]$`, instanceName,
		"instance name should be a valid GCP name")
	assert.LessOrEqual(t, len(instanceName), 63,
		"instance name should be at most 63 characters")
}

// TestGCPSpin_DeterministicNamingWithBranch tests instance naming includes branch.
func TestGCPSpin_DeterministicNamingWithBranch(t *testing.T) {
	cfg := testutil.SkipIfGCPNotAvailable(t)

	imageName := testutil.GetSharedGCPImage(t)

	args := []string{
		"spin",
		"--backend", "gcp",
		"--image", imageName,
		"--repo", testRepo,
		"--branch", "master",
		"--prompt", "echo test",
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
	}
	instanceName, _, _ := testutil.RunGCPSpinCommand(t, args...)

	require.NotEmpty(t, instanceName, "should get instance name")

	t.Cleanup(func() {
		testutil.RemoveGCPInstance(t, cfg.Project, cfg.Zone, instanceName)
		testutil.CleanupGCSPrefix(t, cfg.Bucket, instanceName+"/")
	})

	// Should include branch in the name
	assert.Contains(t, instanceName, "master", "instance name should include branch")
}

// TestGCPSpin_ReuseRunningInstance tests that a running VM is reused on second spin.
func TestGCPSpin_ReuseRunningInstance(t *testing.T) {
	cfg := testutil.SkipIfGCPNotAvailable(t)

	imageName := testutil.GetSharedGCPImage(t)

	args := []string{
		"spin",
		"--backend", "gcp",
		"--image", imageName,
		"--repo", testRepo,
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
	}

	// First run: create VM
	instanceName, _, _ := testutil.RunGCPSpinCommand(t, args...)
	require.NotEmpty(t, instanceName, "should get instance name")

	t.Cleanup(func() {
		testutil.RemoveGCPInstance(t, cfg.Project, cfg.Zone, instanceName)
		testutil.CleanupGCSPrefix(t, cfg.Bucket, instanceName+"/")
	})

	// Verify VM is running
	status := testutil.GCPInstanceStatus(t, cfg.Project, cfg.Zone, instanceName)
	assert.Equal(t, "RUNNING", status, "VM should be running")

	// Second run: should reuse existing running VM
	stdout, stderr := testutil.RunCommandExpectSuccess(t, args...)
	output := stdout + stderr

	assert.Contains(t, output, "Reusing running instance: "+instanceName,
		"should reuse running VM")
}

// TestGCPSpin_RestartStoppedInstance tests that a stopped VM is restarted on spin.
func TestGCPSpin_RestartStoppedInstance(t *testing.T) {
	cfg := testutil.SkipIfGCPNotAvailable(t)

	imageName := testutil.GetSharedGCPImage(t)

	args := []string{
		"spin",
		"--backend", "gcp",
		"--image", imageName,
		"--repo", testRepo,
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
	}

	// Create VM
	instanceName, _, _ := testutil.RunGCPSpinCommand(t, args...)
	require.NotEmpty(t, instanceName, "should get instance name")

	t.Cleanup(func() {
		testutil.RemoveGCPInstance(t, cfg.Project, cfg.Zone, instanceName)
		testutil.CleanupGCSPrefix(t, cfg.Bucket, instanceName+"/")
	})

	// Stop the VM using gcloud
	testutil.StopGCPInstance(t, cfg.Project, cfg.Zone, instanceName)

	// Verify VM is stopped
	status := testutil.GCPInstanceStatus(t, cfg.Project, cfg.Zone, instanceName)
	assert.Equal(t, "TERMINATED", status, "VM should be stopped")

	// Second run: should restart stopped VM
	stdout, stderr := testutil.RunCommandExpectSuccess(t, args...)
	output := stdout + stderr

	assert.Contains(t, output, "Instance restarted: "+instanceName,
		"should restart stopped VM")

	// Verify VM is running again
	status = testutil.GCPInstanceStatus(t, cfg.Project, cfg.Zone, instanceName)
	assert.Equal(t, "RUNNING", status, "VM should be running after restart")
}

// TestGCPSpin_RecreateFlag tests that --recreate destroys and recreates the VM.
func TestGCPSpin_RecreateFlag(t *testing.T) {
	cfg := testutil.SkipIfGCPNotAvailable(t)

	imageName := testutil.GetSharedGCPImage(t)

	args := []string{
		"spin",
		"--backend", "gcp",
		"--image", imageName,
		"--repo", testRepo,
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
	}

	// First run: create VM
	instanceName, _, _ := testutil.RunGCPSpinCommand(t, args...)
	require.NotEmpty(t, instanceName, "should get instance name")

	t.Cleanup(func() {
		testutil.RemoveGCPInstance(t, cfg.Project, cfg.Zone, instanceName)
		testutil.CleanupGCSPrefix(t, cfg.Bucket, instanceName+"/")
	})

	// Second run: recreate VM
	recreateArgs := append(args, "--recreate")
	stdout, stderr := testutil.RunCommandExpectSuccess(t, recreateArgs...)
	output := stdout + stderr

	// Should show creation (not reuse)
	assert.Contains(t, output, "Instance created successfully: "+instanceName,
		"should show VM creation message after recreate")
}

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

// TestGCPSpin_SetupFlag tests that --setup builds a GCP image before creating the VM.
func TestGCPSpin_SetupFlag(t *testing.T) {
	cfg := testutil.SkipIfGCPNotAvailable(t)

	imageName := testutil.GenerateTestImageName(t)

	t.Cleanup(func() {
		testutil.RemoveGCPImage(t, cfg.Project, imageName)
	})

	args := []string{
		"spin",
		"--setup",
		"--backend", "gcp",
		"--image", imageName,
		"--repo", testRepo,
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
	}
	instanceName, stdout, stderr := testutil.RunGCPSpinCommand(t, args...)
	output := stdout + stderr

	// Register VM cleanup
	if instanceName != "" {
		t.Cleanup(func() {
			testutil.RemoveGCPInstance(t, cfg.Project, cfg.Zone, instanceName)
			testutil.CleanupGCSPrefix(t, cfg.Bucket, instanceName+"/")
		})
	}

	// Verify both setup and spin succeeded
	assert.Contains(t, output, "Environment provisioned", "should show image build success")
	assert.Contains(t, output, "Instance created successfully", "should show VM creation success")

	// Verify image exists
	assert.True(t, testutil.GCPImageExists(t, cfg.Project, imageName),
		"image should exist after --setup")
}
