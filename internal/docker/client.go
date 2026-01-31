package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// DockerClient defines the interface for Docker operations.
// This interface enables dependency injection and mocking for testability.
type DockerClient interface {
	// BuildImage builds a Docker image with the given configuration
	BuildImage(ctx context.Context, config BuildConfig) error

	// RunContainer creates and starts a container with the given configuration
	RunContainer(ctx context.Context, config SpinConfig, containerName string, hasNpmrc bool) (ContainerResult, error)

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

// RealDockerClient implements DockerClient using the Docker SDK.
type RealDockerClient struct {
	cli *client.Client
}

// NewRealDockerClient creates a new RealDockerClient instance.
func NewRealDockerClient() *RealDockerClient {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		// Return a client that will fail gracefully on operations
		return &RealDockerClient{cli: nil}
	}

	return &RealDockerClient{cli: cli}
}

// ensureClient checks if the Docker client is initialized and returns an error if not.
func (c *RealDockerClient) ensureClient() error {
	if c.cli == nil {
		return fmt.Errorf("docker client not initialized - is Docker running?")
	}

	return nil
}

// BuildImage builds a Docker image with the given configuration.
func (c *RealDockerClient) BuildImage(ctx context.Context, config BuildConfig) error {
	if err := c.ensureClient(); err != nil {
		return err
	}

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

		// Build user's Dockerfile from current directory
		if err := c.buildFromDockerfile(ctx, ".", config.Dockerfile, userBaseImageTag); err != nil {
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
	if err := c.buildFromDockerfile(ctx, buildContext, "Dockerfile", imageName); err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	return nil
}

// buildFromDockerfile builds an image from a Dockerfile using the SDK.
func (c *RealDockerClient) buildFromDockerfile(ctx context.Context, contextDir, dockerfilePath, tag string) error {
	// Create a tar archive of the build context
	tarBuffer, err := createTarArchive(contextDir)
	if err != nil {
		return fmt.Errorf("failed to create build context tar: %w", err)
	}

	buildOptions := types.ImageBuildOptions{
		Tags:       []string{tag},
		Dockerfile: dockerfilePath,
		Remove:     true,
	}

	resp, err := c.cli.ImageBuild(ctx, tarBuffer, buildOptions)
	if err != nil {
		return fmt.Errorf("image build failed: %w", err)
	}
	defer resp.Body.Close()

	// Stream build output to stdout
	if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
		return fmt.Errorf("failed to read build output: %w", err)
	}

	return nil
}

// createTarArchive creates a tar archive of a directory for Docker build context.
func createTarArchive(srcDir string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// Write file content if it's a regular file
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		}

		return nil
	})

	return buf, err
}

// RunContainer creates and starts a container with the given configuration.
func (c *RealDockerClient) RunContainer(ctx context.Context, config SpinConfig, containerName string, hasNpmrc bool) (ContainerResult, error) {
	if err := c.ensureClient(); err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: containerName,
			Error:         err.Error(),
		}, err
	}

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

	// Build container configuration directly from SpinConfig
	containerConfig, hostConfig := buildContainerConfigs(config, containerName, homeDir, hasNpmrc)

	// Create the container
	createResp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName)
	if err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: containerName,
			Error:         fmt.Sprintf("Failed to create container: %s", err.Error()),
		}, err
	}

	// Start the container
	if err := c.cli.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{}); err != nil {
		// Try to get logs to show what went wrong
		logs, _ := c.getContainerLogs(ctx, containerName)

		return ContainerResult{
			Success:       false,
			ContainerName: containerName,
			Error:         fmt.Sprintf("Git clone failed: %s", strings.TrimSpace(logs)),
		}, err
	}

	// Wait briefly and check if container is still running (clone might fail immediately)
	time.Sleep(2 * time.Second)

	inspect, err := c.cli.ContainerInspect(ctx, createResp.ID)
	if err == nil && !inspect.State.Running && inspect.State.ExitCode != 0 {
		logs, _ := c.getContainerLogs(ctx, containerName)

		return ContainerResult{
			Success:       false,
			ContainerName: containerName,
			Error:         fmt.Sprintf("Git clone failed: %s", strings.TrimSpace(logs)),
		}, fmt.Errorf("container exited with code %d", inspect.State.ExitCode)
	}

	return ContainerResult{
		Success:       true,
		ContainerName: containerName,
	}, nil
}

// buildContainerConfigs creates Docker container and host configs from SpinConfig.
func buildContainerConfigs(config SpinConfig, containerName, homeDir string, hasNpmrc bool) (*container.Config, *container.HostConfig) {
	// Convert SSH URLs to HTTPS for GitHub PAT authentication
	repoURL := convertSshToHttps(config.Repo)

	// Build environment variables
	env := []string{
		fmt.Sprintf("GITHUB_TOKEN=%s", os.Getenv("GITHUB_TOKEN")),
		fmt.Sprintf("CLAUDE_CODE_OAUTH_TOKEN=%s", os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")),
		fmt.Sprintf("REPO_URL=%s", repoURL),
	}

	// Add Ralph loop environment variables if prompt is provided
	if config.Prompt != "" {
		env = append(env, fmt.Sprintf("PROMPT=%s", escapeShellArg(config.Prompt)))

		maxIterations := config.MaxIterations
		if maxIterations == "" {
			maxIterations = DefaultMaxIterations
		}

		env = append(env, fmt.Sprintf("MAX_ITERATIONS=%s", maxIterations))

		if config.Branch != "" {
			env = append(env, fmt.Sprintf("BRANCH=%s", escapeShellArg(config.Branch)))
		}
	}

	// Build volume binds
	binds := []string{
		fmt.Sprintf("%s/.spinner/%s/logs:/logs", homeDir, containerName),
	}

	if hasNpmrc {
		npmrcPath := filepath.Join(homeDir, ".npmrc")
		binds = append(binds, fmt.Sprintf("%s:/home/spinner/.npmrc", npmrcPath))
	}

	containerConfig := &container.Config{
		Image: config.Image,
		Env:   env,
	}

	hostConfig := &container.HostConfig{
		Binds: binds,
	}

	return containerConfig, hostConfig
}

// getContainerLogs retrieves logs from a container.
func (c *RealDockerClient) getContainerLogs(ctx context.Context, containerName string) (string, error) {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "100",
	}

	logs, err := c.cli.ContainerLogs(ctx, containerName, options)
	if err != nil {
		return "", err
	}
	defer logs.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, logs); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ImageExists checks if a Docker image exists.
func (c *RealDockerClient) ImageExists(ctx context.Context, imageName string) (bool, error) {
	if err := c.ensureClient(); err != nil {
		return false, err
	}

	_, _, err := c.cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// ContainerExists checks if a container exists and returns its status.
func (c *RealDockerClient) ContainerExists(ctx context.Context, name string) (ContainerStatus, error) {
	if err := c.ensureClient(); err != nil {
		return StatusNone, err
	}

	inspect, err := c.cli.ContainerInspect(ctx, name)
	if err != nil {
		if client.IsErrNotFound(err) {
			return StatusNone, nil
		}

		return StatusNone, err
	}

	if inspect.State.Running {
		return StatusRunning, nil
	}

	return StatusStopped, nil
}

// RemoveContainer removes a container, forcing removal if it's running.
func (c *RealDockerClient) RemoveContainer(ctx context.Context, name string) (ContainerResult, error) {
	if err := c.ensureClient(); err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: name,
			Error:         err.Error(),
		}, err
	}

	err := c.cli.ContainerRemove(ctx, name, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: name,
			Error:         err.Error(),
		}, err
	}

	return ContainerResult{
		Success:       true,
		ContainerName: name,
	}, nil
}

// RestartContainer restarts a stopped container.
func (c *RealDockerClient) RestartContainer(ctx context.Context, name string) (ContainerResult, error) {
	if err := c.ensureClient(); err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: name,
			Error:         err.Error(),
		}, err
	}

	err := c.cli.ContainerStart(ctx, name, types.ContainerStartOptions{})
	if err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: name,
			Error:         err.Error(),
		}, err
	}

	return ContainerResult{
		Success:       true,
		ContainerName: name,
	}, nil
}

// VerifyContainerStatus verifies that a container is running.
func (c *RealDockerClient) VerifyContainerStatus(ctx context.Context, name string) (ContainerResult, error) {
	if err := c.ensureClient(); err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: name,
			Error:         err.Error(),
		}, err
	}

	inspect, err := c.cli.ContainerInspect(ctx, name)
	if err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: name,
			Error:         "Failed to verify container status",
		}, err
	}

	if !inspect.State.Running {
		// Get logs to show what went wrong
		logs, _ := c.getContainerLogs(ctx, name)

		return ContainerResult{
			Success:       false,
			ContainerName: name,
			Error:         fmt.Sprintf("Container exited. Logs: %s", strings.TrimSpace(logs)),
		}, fmt.Errorf("container not running: %s", inspect.State.Status)
	}

	return ContainerResult{
		Success:       true,
		ContainerName: name,
	}, nil
}
