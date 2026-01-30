package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGenerateTestID verifies unique ID generation
func TestGenerateTestID(t *testing.T) {
	id1 := GenerateTestID(t)
	id2 := GenerateTestID(t)

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.Contains(t, id1, "test-")
	assert.Contains(t, id2, "test-")
}

// TestGenerateTestImageName verifies image name generation
func TestGenerateTestImageName(t *testing.T) {
	imageName := GenerateTestImageName(t)

	assert.NotEmpty(t, imageName)
	assert.Contains(t, imageName, "spinner:test-")
}

// TestGenerateTestContainerName verifies container name generation
func TestGenerateTestContainerName(t *testing.T) {
	containerName := GenerateTestContainerName(t)

	assert.NotEmpty(t, containerName)
	assert.Contains(t, containerName, "spinner-test-")
}

// TestDockerImageExists_NonExistentImage verifies image existence check for non-existent image
func TestDockerImageExists_NonExistentImage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Docker test in short mode")
	}

	imageName := "nonexistent-image:never-exists-123456789"
	exists := DockerImageExists(t, imageName)

	assert.False(t, exists)
}
