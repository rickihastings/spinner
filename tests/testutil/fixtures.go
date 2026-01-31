package testutil

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"
)

// GenerateTestID creates a unique identifier for test resources
func GenerateTestID(t *testing.T) string {
	t.Helper()

	timestamp := time.Now().Unix()
	random := rand.Intn(10000)

	return fmt.Sprintf("test-%d-%d", timestamp, random)
}

// GenerateTestImageName creates a unique Docker image name for testing
// Returns the full image name (repository:tag format)
func GenerateTestImageName(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("spinner:test-%d", time.Now().Unix())
}

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

// CleanupTestResources removes test Docker resources (images and containers)
func CleanupTestResources(t *testing.T, imageName string, containerName string) {
	t.Helper()

	if containerName != "" {
		RemoveDockerContainer(t, containerName)
	}

	if imageName != "" {
		RemoveDockerImage(t, imageName)
	}
}

// SkipIfDockerNotAvailable skips the test if Docker is not available
func SkipIfDockerNotAvailable(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	EnsureDockerRunning(t)
}

// GetHomeDir returns the user's home directory
func GetHomeDir(t *testing.T) string {
	t.Helper()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	return homeDir
}

// FileExists checks if a file exists at the given path
func FileExists(t *testing.T, path string) bool {
	t.Helper()

	_, err := os.Stat(path)

	return err == nil
}

// WriteFile writes content to a file at the given path
func WriteFile(t *testing.T, path string, content string) error {
	t.Helper()

	return os.WriteFile(path, []byte(content), 0644)
}
