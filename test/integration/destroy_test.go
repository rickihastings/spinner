package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rickihastings/spinner/test/testutil"
)

// TestDestroy_Docker tests the destroy command with Docker backend
func TestDockerDestroy(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image
	_, imageName := testutil.SetupTestImage(t)

	// Create a container first using spin
	args := []string{"spin", "--image", imageName, "--repo", testRepo}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	// Verify container exists
	require.True(t, testutil.DockerContainerExists(t, containerName), "container should exist before destroy")

	// Verify state directory exists
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err, "should get home directory")

	stateDir := filepath.Join(homeDir, ".spinner", containerName)
	require.DirExists(t, stateDir, "state directory should exist before destroy")

	// Run destroy command
	destroyArgs := []string{"destroy", containerName}
	stdout, stderr := testutil.RunCommandExpectSuccess(t, destroyArgs...)
	output := stdout + stderr

	// Verify success message
	assert.Contains(t, output, "Instance '"+containerName+"' destroyed", "should show success message")

	// Verify container is removed
	assert.False(t, testutil.DockerContainerExists(t, containerName), "container should not exist after destroy")

	// Verify state directory is removed
	_, err = os.Stat(stateDir)
	assert.True(t, os.IsNotExist(err), "state directory should not exist after destroy")
}

// TestDestroy_Docker_MultipleInstances tests destroying multiple Docker instances
func TestDockerDestroy_MultipleInstances(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image
	_, imageName := testutil.SetupTestImage(t)

	// Create multiple containers with different repos
	repo1 := "https://github.com/octocat/Hello-World.git"
	repo2 := "https://github.com/octocat/Spoon-Knife.git"

	args1 := []string{"spin", "--image", imageName, "--repo", repo1}
	containerName1, _, _ := testutil.RunSpinCommand(t, args1...)

	args2 := []string{"spin", "--image", imageName, "--repo", repo2}
	containerName2, _, _ := testutil.RunSpinCommand(t, args2...)

	// Verify both containers exist
	require.True(t, testutil.DockerContainerExists(t, containerName1), "container1 should exist")
	require.True(t, testutil.DockerContainerExists(t, containerName2), "container2 should exist")

	// Run destroy command with both container names
	destroyArgs := []string{"destroy", containerName1, containerName2}
	stdout, stderr := testutil.RunCommandExpectSuccess(t, destroyArgs...)
	output := stdout + stderr

	// Verify success messages for both
	assert.Contains(t, output, "Instance '"+containerName1+"' destroyed", "should show success for container1")
	assert.Contains(t, output, "Instance '"+containerName2+"' destroyed", "should show success for container2")

	// Verify both containers are removed
	assert.False(t, testutil.DockerContainerExists(t, containerName1), "container1 should not exist after destroy")
	assert.False(t, testutil.DockerContainerExists(t, containerName2), "container2 should not exist after destroy")
}

// TestDestroy_Docker_NonExistent tests destroy command with non-existent instance
func TestDockerDestroy_NonExistent(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Run destroy command with non-existent container
	destroyArgs := []string{"destroy", "non-existent-container"}
	stdout, stderr, exitCode := testutil.RunCommandExpectError(t, destroyArgs...)
	require.NotEqual(t, 0, exitCode, "should fail when destroying non-existent instance")

	output := stdout + stderr

	// Verify error message
	assert.Contains(t, output, "Instance 'non-existent-container' not found", "should show instance not found error")
	assert.Contains(t, output, "failed to destroy 1 of 1 instance(s)", "should show aggregate error")
}
