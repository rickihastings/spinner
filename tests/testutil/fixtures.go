package testutil

import (
	"fmt"
	"math/rand"
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
func GenerateTestImageName(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("spinner:test-%d", time.Now().Unix())
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
