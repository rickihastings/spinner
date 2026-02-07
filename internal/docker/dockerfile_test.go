package docker

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGenerateDockerfile_DefaultBaseImage tests Dockerfile generation with default base image
func TestGenerateDockerfile_DefaultBaseImage(t *testing.T) {
	config := dockerfileConfig{
		BaseImage: "ubuntu:22.04",
	}

	content, err := generateDockerfile(config)

	assert.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Contains(t, content, "FROM ubuntu:22.04")
}

// TestGenerateDockerfile_CustomBaseImage tests Dockerfile generation with custom base image
func TestGenerateDockerfile_CustomBaseImage(t *testing.T) {
	config := dockerfileConfig{
		BaseImage: "node:20",
	}

	content, err := generateDockerfile(config)

	assert.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Contains(t, content, "FROM node:20")
}

// TestGenerateDockerfile_TemplateNotFound tests error handling when template doesn't exist
func TestGenerateDockerfile_TemplateNotFound(t *testing.T) {
	// Save original working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = os.Chdir(originalWd) }()

	// Change to a temporary directory without templates
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	config := dockerfileConfig{
		BaseImage: "ubuntu:22.04",
	}

	_, err = generateDockerfile(config)

	assert.Error(t, err)
}
