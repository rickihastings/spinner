package docker

import (
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
