package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
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

// BuildImage builds a Docker image with the given configuration
func BuildImage(config BuildConfig) error {
	buildContext := filepath.Join(os.TempDir(), fmt.Sprintf("spinner-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(buildContext, 0755); err != nil {
		return fmt.Errorf("failed to create build context: %w", err)
	}
	defer os.RemoveAll(buildContext)

	// Determine the base image to use
	baseImage := config.BaseImage
	if baseImage == "" {
		baseImage = "ubuntu:22.04"
	}

	// If user provided a Dockerfile, build it first
	if config.Dockerfile != "" {
		userBaseImageTag := fmt.Sprintf("spinner-base:%s", config.Name)
		cmd := exec.Command("docker", "build", "-t", userBaseImageTag, "-f", config.Dockerfile, ".")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to build user Dockerfile: %w", err)
		}
		baseImage = userBaseImageTag
	}

	// Generate the final Dockerfile
	dockerfilePath := filepath.Join(buildContext, "Dockerfile")
	dockerfileContent, err := GenerateDockerfile(DockerfileConfig{BaseImage: baseImage})
	if err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}
	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	// Copy build files to build context
	templatesDir := filepath.Join(buildContext, "templates")
	for _, bf := range buildFiles {
		srcPath, err := resolveTemplatePath(filepath.Join("templates", bf.src))
		if err != nil {
			return fmt.Errorf("failed to find %s: %w", bf.src, err)
		}

		dstPath := filepath.Join(templatesDir, bf.dst)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", bf.dst, err)
		}

		if err := copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to copy %s: %w", bf.src, err)
		}
	}

	// Build the final image
	imageName := fmt.Sprintf("spinner:%s", config.Name)
	cmd := exec.Command("docker", "build", "-t", imageName, ".")
	cmd.Dir = buildContext
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
