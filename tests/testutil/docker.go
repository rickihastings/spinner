package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// DockerImageExists checks if a Docker image exists
func DockerImageExists(t *testing.T, image string) bool {
	t.Helper()

	cmd := exec.Command("docker", "image", "inspect", image)
	err := cmd.Run()

	return err == nil
}

// DockerContainerExists checks if a container exists (running or stopped)
func DockerContainerExists(t *testing.T, name string) bool {
	t.Helper()

	cmd := exec.Command("docker", "container", "inspect", name)
	err := cmd.Run()

	return err == nil
}

// DockerContainerRunning checks if a container is currently running
func DockerContainerRunning(t *testing.T, name string) bool {
	t.Helper()

	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", name)

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) == "true"
}

// RemoveDockerImage removes a Docker image if it exists
func RemoveDockerImage(t *testing.T, image string) {
	t.Helper()

	if !DockerImageExists(t, image) {
		return
	}

	cmd := exec.Command("docker", "rmi", "-f", image)

	err := cmd.Run()
	if err != nil {
		t.Logf("Warning: failed to remove image %s: %v", image, err)
	}
}

// RemoveDockerContainer removes a Docker container if it exists
func RemoveDockerContainer(t *testing.T, name string) {
	t.Helper()

	if !DockerContainerExists(t, name) {
		return
	}

	cmd := exec.Command("docker", "rm", "-f", name)

	err := cmd.Run()
	if err != nil {
		t.Logf("Warning: failed to remove container %s: %v", name, err)
	}
}

// IsDockerAvailable returns true if the Docker daemon is accessible.
func IsDockerAvailable() bool {
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}

// EnsureDockerRunning checks that Docker daemon is accessible
func EnsureDockerRunning(t *testing.T) {
	t.Helper()

	cmd := exec.Command("docker", "info")
	err := cmd.Run()
	require.NoError(t, err, "Docker daemon is not accessible. Please ensure Docker is running.")
}

// CleanupTestSpinnerDirs removes all test-related spinner data directories.
// Matches directories containing "-test-" in the name (e.g., spinner-dockerfile-test-hello-world).
// This is intended to be called from TestMain for global cleanup after all tests.
func CleanupTestSpinnerDirs() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	spinnerDir := filepath.Join(homeDir, ".spinner")
	if _, err := os.Stat(spinnerDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(spinnerDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.Contains(entry.Name(), "-test-") {
			dirPath := filepath.Join(spinnerDir, entry.Name())
			if err := os.RemoveAll(dirPath); err != nil {
				// Log but don't fail on cleanup errors
				continue
			}
		}
	}

	return nil
}

// SetupTestImage builds the CLI, creates a test image with the given setup args, and sets up cleanup
func SetupTestImage(t *testing.T, setupArgs ...string) (imageTag string, imageName string) {
	t.Helper()

	SkipIfDockerNotAvailable(t)
	BuildCLI(t)

	imageTag = GenerateTestImageTag(t)
	imageName = "spinner:" + imageTag

	t.Cleanup(func() {
		RemoveDockerImage(t, imageName)
	})

	// Run setup command with provided args plus the name flag
	args := append([]string{"setup", "--name", imageTag}, setupArgs...)
	RunCommandExpectSuccess(t, args...)

	return imageTag, imageName
}

// RunContainerWithImage starts a container with the given image and returns the container name
func RunContainerWithImage(t *testing.T, imageName string) string {
	t.Helper()

	containerName := GenerateTestContainerName(t)

	t.Cleanup(func() {
		RemoveDockerContainer(t, containerName)
	})

	cmd := exec.Command("docker", "run", "-d", "--name", containerName, imageName, "tail", "-f", "/dev/null")
	err := cmd.Run()
	require.NoError(t, err, "should start container")

	return containerName
}

// ExecInContainer runs a command in a container and returns the output
func ExecInContainer(t *testing.T, containerName string, command ...string) string {
	t.Helper()

	args := append([]string{"exec", containerName}, command...)
	cmd := exec.Command("docker", args...)
	output, err := cmd.Output()
	require.NoError(t, err, "command should succeed in container")

	return string(output)
}

// RunSpinCommand executes the spin command and returns the container name from output
func RunSpinCommand(t *testing.T, args ...string) (containerName string, stdout string, stderr string) {
	t.Helper()

	stdout, stderr = RunCommandExpectSuccess(t, args...)
	output := stdout + stderr

	// Extract container name from output
	// Expected format: "Instance created successfully: <container-name>"
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Instance created successfully:") {
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
