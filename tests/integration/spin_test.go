package integration

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rickihastings/spinner/tests/testutil"
)

const (
	// Public test repository
	testRepo = "https://github.com/octocat/Hello-World.git"
	// Container workspace path
	workspacePath = "/home/spinner/workspace"
)

// setupSpinTestEnvironment sets up a test image for spin tests
func setupSpinTestEnvironment(t *testing.T) (imageTag string, imageName string) {
	t.Helper()
	return setupTestImage(t)
}

// runSpinCommand executes the spin command and returns the container name from output
func runSpinCommand(t *testing.T, args ...string) (containerName string, stdout string, stderr string) {
	t.Helper()

	stdout, stderr = testutil.RunCommandExpectSuccess(t, args...)
	output := stdout + stderr

	// Extract container name from output
	// Expected format: "Container created successfully: <container-name>"
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Container created successfully:") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				containerName = parts[len(parts)-1]
				break
			}
		}
	}

	require.NotEmpty(t, containerName, "should extract container name from output: %s", output)

	return containerName, stdout, stderr
}

// TestSpin_SuccessfulContainerCreation tests successful container creation with valid flags
func TestSpin_SuccessfulContainerCreation(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)
	testutil.BuildCLI(t)

	// Setup test image
	_, imageName := setupSpinTestEnvironment(t)

	// Run spin command
	args := []string{"spin", "--image", imageName, "--repo", testRepo}
	containerName, stdout, stderr := runSpinCommand(t, args...)
	output := stdout + stderr

	// Register cleanup
	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Verify success message in output
	assert.Contains(t, output, "Container created successfully", "should show success message")

	// Verify container was created
	assert.True(t, testutil.DockerContainerExists(t, containerName), "container should exist")
}

// TestSpin_ContainerNaming tests that container is named deterministically based on image + repo
func TestSpin_ContainerNaming(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)
	testutil.BuildCLI(t)

	// Setup test image with a specific tag
	imageTag := "test-env"
	imageName := "spinner:" + imageTag

	// Build the test image first
	testutil.SkipIfDockerNotAvailable(t)
	testutil.BuildCLI(t)

	t.Cleanup(func() {
		testutil.RemoveDockerImage(t, imageName)
	})

	testutil.RunCommandExpectSuccess(t, "setup", "--name", imageTag)

	// Run spin command
	args := []string{"spin", "--image", imageName, "--repo", testRepo}
	containerName, _, _ := runSpinCommand(t, args...)

	// Register cleanup
	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Expected deterministic name format: spinner-<image-tag>-<repo-name>
	// For image "spinner:test-env" and repo "Hello-World", expect "spinner-test-env-hello-world"
	expectedName := "spinner-test-env-hello-world"

	assert.Equal(t, expectedName, containerName, "container name should be deterministic based on image and repo")
}

// TestSpin_ContainerRunning tests that container is running after spin command completes
func TestSpin_ContainerRunning(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)
	testutil.BuildCLI(t)

	// Setup test image
	_, imageName := setupSpinTestEnvironment(t)

	// Run spin command
	args := []string{"spin", "--image", imageName, "--repo", testRepo}
	containerName, _, _ := runSpinCommand(t, args...)

	// Register cleanup
	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Verify container is running
	assert.True(t, testutil.DockerContainerRunning(t, containerName), "container should be running")

	// Verify container status is "running"
	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Status}}", containerName)
	output, err := cmd.Output()
	require.NoError(t, err, "should get container status")

	status := strings.TrimSpace(string(output))
	assert.Equal(t, "running", status, "container status should be 'running'")
}

// TestSpin_RepositoryCloned tests that repository is cloned into /home/spinner/workspace inside container
func TestSpin_RepositoryCloned(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)
	testutil.BuildCLI(t)

	// Setup test image
	_, imageName := setupSpinTestEnvironment(t)

	// Run spin command
	args := []string{"spin", "--image", imageName, "--repo", testRepo}
	containerName, _, _ := runSpinCommand(t, args...)

	// Register cleanup
	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Wait a moment for clone to complete
	time.Sleep(2 * time.Second)

	// Check if /home/spinner/workspace exists and has .git directory
	cmd := exec.Command("docker", "exec", containerName, "test", "-d", workspacePath+"/.git")
	err := cmd.Run()
	assert.NoError(t, err, "repository should be cloned into %s", workspacePath)

	// Also verify the workspace directory exists
	cmd = exec.Command("docker", "exec", containerName, "test", "-d", workspacePath)
	err = cmd.Run()
	assert.NoError(t, err, "workspace directory should exist")
}

// TestSpin_ContainerExec tests that container can be exec'd into with bash
func TestSpin_ContainerExec(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)
	testutil.BuildCLI(t)

	// Setup test image
	_, imageName := setupSpinTestEnvironment(t)

	// Run spin command
	args := []string{"spin", "--image", imageName, "--repo", testRepo}
	containerName, _, _ := runSpinCommand(t, args...)

	// Register cleanup
	t.Cleanup(func() {
		testutil.RemoveDockerContainer(t, containerName)
	})

	// Try to exec into container with bash
	cmd := exec.Command("docker", "exec", containerName, "bash", "-c", "pwd")
	output, err := cmd.Output()
	require.NoError(t, err, "should be able to exec into container with bash")

	// Verify we got output (pwd should return the current directory)
	assert.NotEmpty(t, output, "exec command should produce output")
	assert.Contains(t, string(output), "/", "pwd should return a valid path")
}

// TestSpin_NonExistentImage tests that non-existent Docker image exits with error
func TestSpin_NonExistentImage(t *testing.T) {
	testutil.SkipIfDockerNotAvailable(t)
	testutil.BuildCLI(t)

	// Use a non-existent image name
	nonExistentImage := "non-existent-image:latest"

	// Run spin command and expect it to fail
	args := []string{"spin", "--image", nonExistentImage, "--repo", testRepo}
	stdout, stderr, exitCode := testutil.RunCommandExpectError(t, args...)
	output := stdout + stderr

	// Verify error message
	assert.Contains(t, output, "Docker image '"+nonExistentImage+"' not found", "should show image not found error")

	// Verify exit code is 1
	assert.Equal(t, 1, exitCode, "should exit with code 1")
}
