package testutil

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

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

// WaitForContainer waits for a container to reach a specific state
func WaitForContainer(t *testing.T, name string, timeout time.Duration) error {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for container %s", name)
		case <-ticker.C:
			if DockerContainerExists(t, name) {
				return nil
			}
		}
	}
}

// GetContainerID retrieves the container ID by name
func GetContainerID(t *testing.T, name string) string {
	t.Helper()

	cmd := exec.Command("docker", "inspect", "-f", "{{.Id}}", name)
	output, err := cmd.Output()
	require.NoError(t, err, "failed to get container ID for %s", name)

	return strings.TrimSpace(string(output))
}

// EnsureDockerRunning checks that Docker daemon is accessible
func EnsureDockerRunning(t *testing.T) {
	t.Helper()

	cmd := exec.Command("docker", "info")
	err := cmd.Run()
	require.NoError(t, err, "Docker daemon is not accessible. Please ensure Docker is running.")
}
