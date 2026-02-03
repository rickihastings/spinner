package testutil

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// GenerateTestImageTag creates a unique Docker image tag for testing
// Returns just the tag part (without the "spinner:" prefix)
// Use this with the setup command's --name flag
func GenerateTestImageTag(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("test-%d", time.Now().Unix())
}

// GenerateTestContainerName creates a unique Docker container name for testing
func GenerateTestContainerName(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("spinner-test-%d", time.Now().Unix())
}

// SkipIfDockerNotAvailable skips the test if Docker is not available
func SkipIfDockerNotAvailable(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	EnsureDockerRunning(t)
}

// WriteFile writes content to a file at the given path
func WriteFile(t *testing.T, path string, content string) error {
	t.Helper()

	return os.WriteFile(path, []byte(content), 0644)
}
