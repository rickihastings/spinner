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
