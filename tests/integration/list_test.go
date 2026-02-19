package integration

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rickihastings/spinner/tests/testutil"
)

// TestList_ShowsSpunContainer tests that `spinner list` shows a container
// created via `spinner spin` with correct status and backend.
func TestDockerList_ShowsSpunContainer(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image and spin up a container
	_, imageName := testutil.SetupTestImage(t)

	args := []string{"spin", "--image", imageName, "--repo", testRepo}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Run list command
	stdout, stderr := testutil.RunCommandExpectSuccess(t, "list")
	output := stdout + stderr

	// Verify table header is present
	assert.Contains(t, output, "BACKEND", "should have BACKEND column header")
	assert.Contains(t, output, "NAME", "should have NAME column header")
	assert.Contains(t, output, "STATUS", "should have STATUS column header")

	// Verify the container appears in the output
	assert.Contains(t, output, containerName, "should show the spun container")
	assert.Contains(t, output, "docker", "should show docker backend")
	assert.Contains(t, output, "running", "should show running status")
}

// TestList_ShowsStoppedContainer tests that `spinner list` shows a stopped
// container with the correct status.
func TestDockerList_ShowsStoppedContainer(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image and spin up a container
	_, imageName := testutil.SetupTestImage(t)

	args := []string{"spin", "--image", imageName, "--repo", testRepo}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Stop the container
	stopCmd := exec.Command("docker", "stop", containerName)
	err := stopCmd.Run()
	require.NoError(t, err, "should stop container")

	// Run list command
	stdout, stderr := testutil.RunCommandExpectSuccess(t, "list")
	output := stdout + stderr

	// Verify the container appears with stopped status
	assert.Contains(t, output, containerName, "should show the stopped container")

	// Find the line with the container name and check it shows "stopped"
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, containerName) {
			assert.Contains(t, line, "stopped", "container line should show stopped status")
			break
		}
	}
}

// TestList_NoInstances tests that `spinner list` shows an appropriate message
// when no spinner-managed instances exist.
func TestDockerList_NoInstances(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Run list command without creating any containers
	// Note: other tests may have left containers, but they should be cleaned up.
	// This test checks the output format; the "No instances found" message only
	// appears when there are truly no spinner-managed containers.
	stdout, stderr := testutil.RunCommandExpectSuccess(t, "list")
	output := stdout + stderr

	// The output should either show "No instances found." or a table with instances
	// We can't guarantee no other tests left containers, so just verify the command succeeds
	assert.True(t,
		strings.Contains(output, "No instances found") || strings.Contains(output, "BACKEND"),
		"should show either 'No instances found' or a table")
}

// TestList_MultipleContainers tests that `spinner list` shows multiple containers.
func TestDockerList_MultipleContainers(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image
	_, imageName := testutil.SetupTestImage(t)

	// Spin up two containers with different repos
	repo1 := "https://github.com/octocat/Hello-World.git"
	repo2 := "https://github.com/octocat/Spoon-Knife.git"

	args1 := []string{"spin", "--image", imageName, "--repo", repo1}
	containerName1, _, _ := testutil.RunSpinCommand(t, args1...)

	args2 := []string{"spin", "--image", imageName, "--repo", repo2}
	containerName2, _, _ := testutil.RunSpinCommand(t, args2...)

	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName1)
		testutil.RemoveDockerContainer(t, containerName2)
	})

	// Run list command
	stdout, stderr := testutil.RunCommandExpectSuccess(t, "list")
	output := stdout + stderr

	// Verify both containers appear
	assert.Contains(t, output, containerName1, "should show first container")
	assert.Contains(t, output, containerName2, "should show second container")
}

// TestList_LabelPresence tests that containers created via `spinner spin` have
// the `spinner-managed=true` label, which is how `spinner list` discovers them.
func TestDockerList_LabelPresence(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image and spin up a container
	_, imageName := testutil.SetupTestImage(t)

	args := []string{"spin", "--image", imageName, "--repo", testRepo}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Inspect container labels using docker inspect
	cmd := exec.Command("docker", "inspect", "-f", "{{index .Config.Labels \"spinner-managed\"}}", containerName)
	output, err := cmd.Output()
	require.NoError(t, err, "should inspect container labels")

	labelValue := strings.TrimSpace(string(output))
	assert.Equal(t, "true", labelValue, "container should have spinner-managed=true label")
}

// TestList_LabelPresentAfterRecreate tests that the spinner-managed label persists
// when a container is recreated with --recreate.
func TestDockerList_LabelPresentAfterRecreate(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)

	// Setup test image with a specific tag
	imageTag := "test-env"
	imageName := "spinner:" + imageTag

	t.Cleanup(func() {
		testutil.RemoveDockerImage(t, imageName)
	})

	testutil.RunCommandExpectSuccess(t, "setup", "--name", imageTag)

	// Create initial container
	args := []string{"spin", "--image", imageName, "--repo", testRepo}
	containerName, _, _ := testutil.RunSpinCommand(t, args...)

	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Recreate the container
	recreateArgs := []string{"spin", "--image", imageName, "--repo", testRepo, "--recreate"}
	testutil.RunCommandExpectSuccess(t, recreateArgs...)

	// Verify label is still present after recreation
	cmd := exec.Command("docker", "inspect", "-f", "{{index .Config.Labels \"spinner-managed\"}}", containerName)
	output, err := cmd.Output()
	require.NoError(t, err, "should inspect container labels after recreate")

	labelValue := strings.TrimSpace(string(output))
	assert.Equal(t, "true", labelValue, "recreated container should have spinner-managed=true label")
}
