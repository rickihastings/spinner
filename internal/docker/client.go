package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DockerClient defines the interface for Docker operations.
// This interface enables dependency injection and mocking for testability.
type DockerClient interface {
	// BuildImage builds a Docker image with the given configuration
	BuildImage(ctx context.Context, config BuildConfig) error

	// RunContainer creates and starts a container with the given arguments
	RunContainer(ctx context.Context, args []string, containerName string) (ContainerResult, error)

	// ImageExists checks if a Docker image exists
	ImageExists(ctx context.Context, image string) (bool, error)

	// ContainerExists checks if a container exists and returns its status
	ContainerExists(ctx context.Context, name string) (ContainerStatus, error)

	// RemoveContainer removes a container, forcing removal if it's running
	RemoveContainer(ctx context.Context, name string) (ContainerResult, error)

	// RestartContainer restarts a stopped container
	RestartContainer(ctx context.Context, name string) (ContainerResult, error)

	// VerifyContainerStatus verifies that a container is running
	VerifyContainerStatus(ctx context.Context, name string) (ContainerResult, error)
}

// RealDockerClient implements DockerClient using actual Docker CLI commands.
type RealDockerClient struct{}

// NewRealDockerClient creates a new RealDockerClient instance.
func NewRealDockerClient() *RealDockerClient {
	return &RealDockerClient{}
}

// BuildImage builds a Docker image with the given configuration.
func (c *RealDockerClient) BuildImage(ctx context.Context, config BuildConfig) error {
	buildContext := filepath.Join(os.TempDir(), fmt.Sprintf("spinner-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(buildContext, 0755); err != nil {
		return fmt.Errorf("failed to create build context: %w", err)
	}

	defer func() { _ = os.RemoveAll(buildContext) }()

	// Determine the base image to use
	baseImage := config.BaseImage
	if baseImage == "" {
		baseImage = "ubuntu:22.04"
	}

	// If user provided a Dockerfile, build it first
	if config.Dockerfile != "" {
		userBaseImageTag := fmt.Sprintf("spinner-base:%s", config.Name)
		cmd := exec.CommandContext(ctx, "docker", "build", "-t", userBaseImageTag, "-f", config.Dockerfile, ".")
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

	// Build spinner CLI binary for linux/amd64
	spinnerBinaryPath := filepath.Join(buildContext, "spinner")
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", spinnerBinaryPath)

	// Find project root to ensure go build runs from the correct directory
	projectRoot, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	buildCmd.Dir = projectRoot

	buildCmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build spinner binary: %w", err)
	}

	// Build the final image
	imageName := fmt.Sprintf("spinner:%s", config.Name)
	cmd := exec.CommandContext(ctx, "docker", "build", "-t", imageName, ".")
	cmd.Dir = buildContext
	cmd.Stdout = os.Stdout

	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	return nil
}

// RunContainer creates and starts a container with the given arguments.
func (c *RealDockerClient) RunContainer(ctx context.Context, args []string, containerName string) (ContainerResult, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: containerName,
			Error:         fmt.Sprintf("Failed to get home directory: %s", err.Error()),
		}, err
	}

	logsDir := filepath.Join(homeDir, ".spinner", containerName, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: containerName,
			Error:         fmt.Sprintf("Failed to create logs directory: %s", err.Error()),
		}, err
	}

	cmd := exec.CommandContext(ctx, "docker", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Container may have started but clone failed
		// Try to get the git error message from container logs
		logsCmd := exec.CommandContext(ctx, "docker", "logs", containerName)
		if logsOutput, logsErr := logsCmd.CombinedOutput(); logsErr == nil {
			return ContainerResult{
				Success:       false,
				ContainerName: containerName,
				Error:         fmt.Sprintf("Git clone failed: %s", strings.TrimSpace(string(logsOutput))),
			}, err
		}

		return ContainerResult{
			Success:       false,
			ContainerName: containerName,
			Error:         strings.TrimSpace(string(output)),
		}, err
	}

	return ContainerResult{
		Success:       true,
		ContainerName: containerName,
	}, nil
}

// ImageExists checks if a Docker image exists.
func (c *RealDockerClient) ImageExists(ctx context.Context, image string) (bool, error) {
	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", image)
	if err := cmd.Run(); err != nil {
		return false, nil
	}

	return true, nil
}

// ContainerExists checks if a container exists and returns its status.
func (c *RealDockerClient) ContainerExists(ctx context.Context, name string) (ContainerStatus, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Status}}", name)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Container doesn't exist
		return StatusNone, nil
	}

	status := strings.TrimSpace(string(output))
	if status == string(StatusRunning) {
		return StatusRunning, nil
	}

	return StatusStopped, nil
}

// RemoveContainer removes a container, forcing removal if it's running.
func (c *RealDockerClient) RemoveContainer(ctx context.Context, name string) (ContainerResult, error) {
	cmd := exec.CommandContext(ctx, "docker", "rm", "-f", name)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: name,
			Error:         strings.TrimSpace(string(output)),
		}, err
	}

	return ContainerResult{
		Success:       true,
		ContainerName: name,
	}, nil
}

// RestartContainer restarts a stopped container.
func (c *RealDockerClient) RestartContainer(ctx context.Context, name string) (ContainerResult, error) {
	cmd := exec.CommandContext(ctx, "docker", "start", name)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: name,
			Error:         strings.TrimSpace(string(output)),
		}, err
	}

	return ContainerResult{
		Success:       true,
		ContainerName: name,
	}, nil
}

// VerifyContainerStatus verifies that a container is running.
func (c *RealDockerClient) VerifyContainerStatus(ctx context.Context, name string) (ContainerResult, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Status}}", name)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: name,
			Error:         "Failed to verify container status",
		}, err
	}

	status := strings.TrimSpace(string(output))
	if status != string(StatusRunning) {
		// Get logs to show what went wrong
		logsCmd := exec.CommandContext(ctx, "docker", "logs", name)
		logsOutput, _ := logsCmd.CombinedOutput()

		return ContainerResult{
			Success:       false,
			ContainerName: name,
			Error:         fmt.Sprintf("Container exited. Logs: %s", strings.TrimSpace(string(logsOutput))),
		}, fmt.Errorf("container not running: %s", status)
	}

	return ContainerResult{
		Success:       true,
		ContainerName: name,
	}, nil
}
