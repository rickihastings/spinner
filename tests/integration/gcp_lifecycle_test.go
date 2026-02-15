package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rickihastings/spinner/tests/testutil"
)

// TestGCPLifecycle_FullCycle tests the complete lifecycle: setup → spin → stop → start → remove.
// This is the most comprehensive integration test for the GCP backend.
func TestGCPLifecycle_FullCycle(t *testing.T) {
	cfg := testutil.SkipIfGCPNotAvailable(t)

	// 1. Setup: bake a GCP image
	imageName := testutil.GenerateTestImageName(t)

	t.Cleanup(func() {
		testutil.RemoveGCPImage(t, cfg.Project, imageName)
	})

	setupArgs := []string{
		"setup",
		"--backend", "gcp",
		"--name", imageName,
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
	}
	stdout, stderr := testutil.RunCommandExpectSuccess(t, setupArgs...)
	assert.Contains(t, stdout+stderr, "Environment provisioned", "setup should succeed")

	// 2. Spin: create a VM instance
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

	t.Cleanup(func() {
		testutil.RemoveGCPInstance(t, cfg.Project, cfg.Zone, instanceName)
		testutil.CleanupGCSPrefix(t, cfg.Bucket, instanceName+"/")
	})

	assert.Contains(t, spinStdout+spinStderr, "Instance created successfully",
		"spin should create instance")

	// Verify VM is running
	status := testutil.GCPInstanceStatus(t, cfg.Project, cfg.Zone, instanceName)
	assert.Equal(t, "RUNNING", status, "VM should be RUNNING after spin")

	// 3. Stop: stop the VM
	testutil.StopGCPInstance(t, cfg.Project, cfg.Zone, instanceName)

	status = testutil.GCPInstanceStatus(t, cfg.Project, cfg.Zone, instanceName)
	assert.Equal(t, "TERMINATED", status, "VM should be TERMINATED after stop")

	// 4. Spin again: should restart the stopped VM
	stdout, stderr = testutil.RunCommandExpectSuccess(t, spinArgs...)
	assert.Contains(t, stdout+stderr, "Instance restarted: "+instanceName,
		"spin should restart stopped VM")

	status = testutil.GCPInstanceStatus(t, cfg.Project, cfg.Zone, instanceName)
	assert.Equal(t, "RUNNING", status, "VM should be RUNNING after restart")

	// 5. Spin again: should reuse running VM
	stdout, stderr = testutil.RunCommandExpectSuccess(t, spinArgs...)
	assert.Contains(t, stdout+stderr, "Reusing running instance: "+instanceName,
		"spin should reuse running VM")

	// 6. Spin with --recreate: should destroy and recreate VM
	recreateArgs := append(spinArgs, "--recreate")
	stdout, stderr = testutil.RunCommandExpectSuccess(t, recreateArgs...)
	assert.Contains(t, stdout+stderr, "Instance created successfully: "+instanceName,
		"spin --recreate should destroy and recreate VM")

	status = testutil.GCPInstanceStatus(t, cfg.Project, cfg.Zone, instanceName)
	assert.Equal(t, "RUNNING", status, "VM should be RUNNING after recreate")

	// 7. Remove: delete the VM
	testutil.RemoveGCPInstance(t, cfg.Project, cfg.Zone, instanceName)

	assert.False(t, testutil.GCPInstanceExists(t, cfg.Project, cfg.Zone, instanceName),
		"VM should not exist after removal")
}

// TestGCPLifecycle_VMAutoStopsOnCompletion tests that VMs automatically stop when execution completes successfully.
func TestGCPLifecycle_VMAutoStopsOnCompletion(t *testing.T) {
	cfg := testutil.SkipIfGCPNotAvailable(t)

	imageName := testutil.GetSharedGCPImage(t)

	// Create a simple test that completes quickly
	// Note: This test requires the completion detection to work correctly
	spinArgs := []string{
		"spin",
		"--backend", "gcp",
		"--image", imageName,
		"--repo", testRepo,
		"--branch", "test-auto-stop",
		"--prompt", "list files and output ~~ FEATURE_COMPLETED ~~",
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
		"--max-iterations", "1",
	}

	instanceName, _, _ := testutil.RunGCPSpinCommand(t, spinArgs...)
	require.NotEmpty(t, instanceName, "should get instance name")

	t.Cleanup(func() {
		testutil.RemoveGCPInstance(t, cfg.Project, cfg.Zone, instanceName)
		testutil.CleanupGCSPrefix(t, cfg.Bucket, instanceName+"/")
	})

	// Poll GCS state until status is "completed" (with timeout)
	testutil.WaitForGCSStateStatus(t, cfg.Bucket, instanceName, "completed", 300)

	// Poll VM status until it becomes TERMINATED (with timeout)
	// GCP can take a while to report TERMINATED after poweroff
	testutil.WaitForGCPInstanceStatus(t, cfg.Project, cfg.Zone, instanceName, "TERMINATED", 300)

	// Verify final state
	status := testutil.GCPInstanceStatus(t, cfg.Project, cfg.Zone, instanceName)
	assert.Equal(t, "TERMINATED", status, "VM should be TERMINATED after successful completion")
}

// TestGCPLifecycle_VMStaysRunningOnError tests that VMs stay running when errors occur.
func TestGCPLifecycle_VMStaysRunningOnError(t *testing.T) {
	cfg := testutil.SkipIfGCPNotAvailable(t)

	imageName := testutil.GetSharedGCPImage(t)

	// Create VM with invalid repo URL to cause an error
	spinArgs := []string{
		"spin",
		"--backend", "gcp",
		"--image", imageName,
		"--repo", "https://github.com/invalid/nonexistent-repo-12345",
		"--prompt", "do something",
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
		"--max-iterations", "1",
	}

	instanceName, _, _ := testutil.RunGCPSpinCommand(t, spinArgs...)
	require.NotEmpty(t, instanceName, "should get instance name")

	t.Cleanup(func() {
		testutil.RemoveGCPInstance(t, cfg.Project, cfg.Zone, instanceName)
		testutil.CleanupGCSPrefix(t, cfg.Bucket, instanceName+"/")
	})

	// Wait for error state in GCS (repo clone will fail)
	testutil.WaitForGCSStateStatus(t, cfg.Bucket, instanceName, "error", 120)

	// Wait a bit to ensure VM doesn't shut down
	testutil.Sleep(t, 10)

	// Verify VM is still running
	status := testutil.GCPInstanceStatus(t, cfg.Project, cfg.Zone, instanceName)
	assert.Equal(t, "RUNNING", status, "VM should still be RUNNING after error for debugging")
}

// TestGCPLifecycle_VMStaysRunningWithoutPrompt tests that VMs stay running when no prompt is specified.
func TestGCPLifecycle_VMStaysRunningWithoutPrompt(t *testing.T) {
	cfg := testutil.SkipIfGCPNotAvailable(t)

	imageName := testutil.GetSharedGCPImage(t)

	// Create VM without a prompt (interactive mode)
	spinArgs := []string{
		"spin",
		"--backend", "gcp",
		"--image", imageName,
		"--repo", testRepo,
		"--branch", "test-no-prompt",
		// Note: NO --prompt flag
		"--project", cfg.Project,
		"--zone", cfg.Zone,
		"--state-bucket", cfg.Bucket,
	}

	instanceName, _, _ := testutil.RunGCPSpinCommand(t, spinArgs...)
	require.NotEmpty(t, instanceName, "should get instance name")

	t.Cleanup(func() {
		testutil.RemoveGCPInstance(t, cfg.Project, cfg.Zone, instanceName)
		testutil.CleanupGCSPrefix(t, cfg.Bucket, instanceName+"/")
	})

	// Wait for VM to finish booting and running startup script
	// Without a prompt, startup.sh should run `tail -f /dev/null` and keep running
	testutil.Sleep(t, 30)

	// Verify VM is still running
	status := testutil.GCPInstanceStatus(t, cfg.Project, cfg.Zone, instanceName)
	assert.Equal(t, "RUNNING", status, "VM should stay RUNNING without a prompt for interactive use")
}
