package docker

import (
	"context"
	"os"
)

// BuildConfig contains configuration for building a Docker image.
type BuildConfig struct {
	Name       string
	BaseImage  string
	Dockerfile string
}

// buildFile defines a file to copy into the Docker build context
type buildFile struct {
	// Source path relative to the templates directory
	src string
	// Destination path relative to the build context templates directory
	dst string
}

// buildFiles lists all files to copy into the Docker build context
var buildFiles = []buildFile{
	{src: "scripts/startup.sh", dst: "scripts/startup.sh"},
	{src: "scripts/ralph-loop.sh", dst: "scripts/ralph-loop.sh"},
}

// BuildImage builds a Docker image with the given configuration.
// This is a convenience function that uses the default RealDockerClient.
func BuildImage(config BuildConfig) error {
	client := NewRealDockerClient()
	return client.BuildImage(context.Background(), config)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0644)
}
