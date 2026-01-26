package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type BuildConfig struct {
	Name       string
	BaseImage  string
	Dockerfile string
}

// BuildImage builds a Docker image with the given configuration
func BuildImage(config BuildConfig) error {
	buildContext := filepath.Join(os.TempDir(), fmt.Sprintf("spinner-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(buildContext, 0755); err != nil {
		return fmt.Errorf("failed to create build context: %w", err)
	}

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

	// Copy startup scripts to build context
	templatesDir := filepath.Join(buildContext, "templates")
	scriptsDir := filepath.Join(templatesDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		return fmt.Errorf("failed to create scripts directory: %w", err)
	}

	// Copy startup.sh
	startupScriptSrc, err := resolveTemplatePath(filepath.Join("templates", "scripts", "startup.sh"))
	if err != nil {
		return fmt.Errorf("failed to find startup.sh: %w", err)
	}
	startupScriptDest := filepath.Join(scriptsDir, "startup.sh")
	if err := copyFile(startupScriptSrc, startupScriptDest); err != nil {
		return fmt.Errorf("failed to copy startup.sh: %w", err)
	}

	// Copy ralph-loop.sh
	ralphLoopScriptSrc, err := resolveTemplatePath(filepath.Join("templates", "scripts", "ralph-loop.sh"))
	if err != nil {
		return fmt.Errorf("failed to find ralph-loop.sh: %w", err)
	}
	ralphLoopScriptDest := filepath.Join(scriptsDir, "ralph-loop.sh")
	if err := copyFile(ralphLoopScriptSrc, ralphLoopScriptDest); err != nil {
		return fmt.Errorf("failed to copy ralph-loop.sh: %w", err)
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
