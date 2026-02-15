package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"

	"github.com/rickihastings/spinner/internal/util"
)

// Client defines the interface for Docker operations.
// This interface enables dependency injection and mocking for testability.
type Client interface {
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

	// StartContainer starts a stopped container
	StartContainer(ctx context.Context, name string) (ContainerResult, error)

	// StopContainer stops a running container
	StopContainer(ctx context.Context, name string) error

	// LogsContainer returns the logs from a container
	LogsContainer(ctx context.Context, name string) ([]byte, error)

	// VerifyContainerStatus verifies that a container is running
	VerifyContainerStatus(ctx context.Context, name string) (ContainerResult, error)

	// StreamContainerLogs streams container logs in real-time.
	// It returns a channel of LogEvent that receives log entries as they arrive.
	// The channel is closed when the context is cancelled, the container stops,
	// or an error occurs. Check LogEvent.Error for any streaming errors.
	// Options can be used to configure the stream (follow, timestamps, tail, etc.)
	StreamContainerLogs(ctx context.Context, name string, opts LogStreamOptions) (<-chan LogEvent, error)
}

// RealDockerClient implements Client using the Docker SDK.
type RealDockerClient struct {
	sdk *sdkClient
}

// NewRealDockerClient creates a new RealDockerClient instance.
func NewRealDockerClient() *RealDockerClient {
	return &RealDockerClient{
		sdk: newSDKClient(),
	}
}

// getSDKClient returns the underlying Docker SDK client.
func (c *RealDockerClient) getSDKClient(ctx context.Context) (*client.Client, error) {
	return c.sdk.getClient(ctx)
}

// BuildImage builds a Docker image with the given configuration using the Docker SDK.
func (c *RealDockerClient) BuildImage(ctx context.Context, config BuildConfig) error {
	cli, err := c.getSDKClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Docker client: %w", err)
	}

	buildContextDir := filepath.Join(os.TempDir(), fmt.Sprintf("spinner-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(buildContextDir, 0755); err != nil {
		return fmt.Errorf("failed to create build context: %w", err)
	}

	defer func() { _ = os.RemoveAll(buildContextDir) }()

	// Determine the base image to use
	baseImage := config.BaseImage
	if baseImage == "" {
		baseImage = "ubuntu:22.04"
	}

	// If user provided a Dockerfile, build it first using SDK
	if config.Dockerfile != "" {
		userBaseImageTag := fmt.Sprintf("spinner-base:%s", config.Name)
		if err := c.buildUserDockerfile(ctx, cli, config.Dockerfile, userBaseImageTag); err != nil {
			return fmt.Errorf("failed to build user Dockerfile: %w", err)
		}

		baseImage = userBaseImageTag
	}

	// Generate the final Dockerfile
	dockerfilePath := filepath.Join(buildContextDir, "Dockerfile")

	dockerfileContent, err := generateDockerfile(dockerfileConfig{BaseImage: baseImage})
	if err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	// Write build files to build context
	templatesDir := filepath.Join(buildContextDir, "templates")
	for _, bf := range buildFiles {
		dstPath := filepath.Join(templatesDir, bf.dst)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", bf.dst, err)
		}

		if err := os.WriteFile(dstPath, []byte(bf.file), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", bf.dst, err)
		}
	}

	// Check if we're in development mode (local binary exists)
	// This happens when running from source after ./scripts/dev-setup.sh
	destPath := filepath.Join(buildContextDir, "spinner")
	localBuildDetected := false

	projectRoot, err := util.FindProjectRoot()
	if err == nil {
		localBinaryPath := filepath.Join(projectRoot, "dist", "spinner-linux-amd64")
		if _, statErr := os.Stat(localBinaryPath); statErr == nil {
			// Local binary exists - use it automatically
			fmt.Println("🔧 Local development binary detected, using dist/spinner-linux-amd64...")

			if err := copyFile(localBinaryPath, destPath); err != nil {
				return fmt.Errorf("failed to copy local binary: %w", err)
			}

			// Set LOCAL_BUILD for the Dockerfile
			err := os.Setenv("LOCAL_BUILD", "true")
			if err != nil {
				return fmt.Errorf("failed to set LOCAL_BUILD env var: %w", err)
			}

			localBuildDetected = true

			fmt.Println("✓ Using local binary for Docker image")
		}
	}

	// If no local build, create empty placeholder file so COPY doesn't fail
	if !localBuildDetected {
		if err := os.WriteFile(destPath, []byte(""), 0644); err != nil {
			return fmt.Errorf("failed to create placeholder spinner file: %w", err)
		}
	}

	// Build the final image using SDK
	imageName := fmt.Sprintf("spinner:%s", config.Name)
	if err := c.buildImageFromContext(ctx, cli, buildContextDir, imageName, baseImage); err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	return nil
}

// buildUserDockerfile builds a user-provided Dockerfile using the Docker SDK.
func (c *RealDockerClient) buildUserDockerfile(ctx context.Context, cli *client.Client, dockerfilePath, tag string) error {
	contextDir := filepath.Dir(dockerfilePath)
	dockerfileName := filepath.Base(dockerfilePath)

	// User dockerfiles don't need baseImage since they define their own FROM
	return c.buildImageWithOptions(ctx, cli, contextDir, dockerfileName, tag, "")
}

// buildImageFromContext builds a Docker image from a build context directory using the SDK.
func (c *RealDockerClient) buildImageFromContext(ctx context.Context, cli *client.Client, contextDir, tag, baseImage string) error {
	return c.buildImageWithOptions(ctx, cli, contextDir, "Dockerfile", tag, baseImage)
}

// buildImageWithOptions is the shared implementation for building Docker images.
func (c *RealDockerClient) buildImageWithOptions(ctx context.Context, cli *client.Client, contextDir, dockerfileName, tag, baseImage string) error {
	tarReader, err := createBuildContextTar(contextDir)
	if err != nil {
		return fmt.Errorf("failed to create build context tar: %w", err)
	}

	buildOptions := build.ImageBuildOptions{
		Tags:       []string{tag},
		Dockerfile: dockerfileName,
		Remove:     true,
		BuildArgs: map[string]*string{
			"LOCAL_BUILD": getEnvPtr("LOCAL_BUILD"),
			"BASE_IMAGE":  &baseImage,
		},
	}

	response, err := cli.ImageBuild(ctx, tarReader, buildOptions)
	if err != nil {
		return err
	}

	defer func() { _ = response.Body.Close() }()

	return c.processBuildOutput(response.Body)
}

// processBuildOutput reads and processes Docker build output, streaming to stdout.
// It uses Docker's own jsonmessage package which handles both legacy builder and
// BuildKit output formats, including terminal-aware progress bars.
func (c *RealDockerClient) processBuildOutput(reader io.Reader) error {
	fd := os.Stdout.Fd()
	isTerminal := term.IsTerminal(fd)

	return jsonmessage.DisplayJSONMessagesStream(reader, os.Stdout, fd, isTerminal, nil)
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

// ImageExists checks if a Docker image exists using the Docker SDK.
func (c *RealDockerClient) ImageExists(ctx context.Context, imageName string) (bool, error) {
	cli, err := c.getSDKClient(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get Docker client: %w", err)
	}

	_, err = cli.ImageInspect(ctx, imageName)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to inspect image: %w", err)
	}

	return true, nil
}

// ContainerExists checks if a container exists and returns its status using the Docker SDK.
func (c *RealDockerClient) ContainerExists(ctx context.Context, name string) (ContainerStatus, error) {
	cli, err := c.getSDKClient(ctx)
	if err != nil {
		return StatusNone, fmt.Errorf("failed to get Docker client: %w", err)
	}

	info, err := cli.ContainerInspect(ctx, name)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return StatusNone, nil
		}

		return StatusNone, fmt.Errorf("failed to inspect container: %w", err)
	}

	if info.State.Running {
		return StatusRunning, nil
	}

	return StatusStopped, nil
}

// RemoveContainer removes a container, forcing removal if it's running using the Docker SDK.
func (c *RealDockerClient) RemoveContainer(ctx context.Context, name string) (ContainerResult, error) {
	cli, err := c.getSDKClient(ctx)
	if err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: name,
			Error:         fmt.Sprintf("failed to get Docker client: %v", err),
		}, err
	}

	err = cli.ContainerRemove(ctx, name, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: false,
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

// StartContainer starts a stopped container using the Docker SDK.
func (c *RealDockerClient) StartContainer(ctx context.Context, name string) (ContainerResult, error) {
	cli, err := c.getSDKClient(ctx)
	if err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: name,
			Error:         fmt.Sprintf("failed to get Docker client: %v", err),
		}, err
	}

	err = cli.ContainerStart(ctx, name, container.StartOptions{})
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

// StopContainer stops a running container using the Docker SDK.
func (c *RealDockerClient) StopContainer(ctx context.Context, name string) error {
	cli, err := c.getSDKClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Docker client: %w", err)
	}

	timeout := 10 // seconds

	err = cli.ContainerStop(ctx, name, container.StopOptions{Timeout: &timeout})
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	return nil
}

// LogsContainer returns the logs from a container using the Docker SDK.
func (c *RealDockerClient) LogsContainer(ctx context.Context, name string) ([]byte, error) {
	cli, err := c.getSDKClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker client: %w", err)
	}

	logsReader, err := cli.ContainerLogs(ctx, name, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}

	defer func() { _ = logsReader.Close() }()

	output, err := io.ReadAll(logsReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read container logs: %w", err)
	}

	return output, nil
}

// VerifyContainerStatus verifies that a container is running using the Docker SDK.
func (c *RealDockerClient) VerifyContainerStatus(ctx context.Context, name string) (ContainerResult, error) {
	cli, err := c.getSDKClient(ctx)
	if err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: name,
			Error:         "Failed to verify container status",
		}, err
	}

	info, err := cli.ContainerInspect(ctx, name)
	if err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: name,
			Error:         "Failed to verify container status",
		}, err
	}

	if !info.State.Running {
		// Get logs to show what went wrong
		logsOutput := c.getContainerLogs(ctx, cli, name)

		return ContainerResult{
			Success:       false,
			ContainerName: name,
			Error:         fmt.Sprintf("Container exited. Logs: %s", logsOutput),
		}, fmt.Errorf("container not running: %s", info.State.Status)
	}

	return ContainerResult{
		Success:       true,
		ContainerName: name,
	}, nil
}

// getContainerLogs retrieves container logs using the Docker SDK.
func (c *RealDockerClient) getContainerLogs(ctx context.Context, cli *client.Client, name string) string {
	logsReader, err := cli.ContainerLogs(ctx, name, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "100",
	})
	if err != nil {
		return ""
	}

	defer func() { _ = logsReader.Close() }()

	logsBytes, err := io.ReadAll(logsReader)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(logsBytes))
}

// StreamContainerLogs streams container logs in real-time using the Docker SDK.
// It returns a channel that receives LogEvent entries as they arrive from the container.
// The channel is closed when:
//   - The context is cancelled
//   - The container stops (if Follow is true)
//   - An error occurs (error is sent in LogEvent.Error before closing)
//
// Usage example:
//
//	events, err := client.StreamContainerLogs(ctx, "my-container", LogStreamOptions{Follow: true})
//	if err != nil {
//	    return err
//	}
//	for event := range events {
//	    if event.Error != nil {
//	        log.Printf("Stream error: %v", event.Error)
//	        break
//	    }
//	    fmt.Printf("[%s] %s", event.Stream, event.Message)
//	}
func (c *RealDockerClient) StreamContainerLogs(ctx context.Context, name string, opts LogStreamOptions) (<-chan LogEvent, error) {
	cli, err := c.getSDKClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker client: %w", err)
	}

	// Build container logs options from our LogStreamOptions
	logsOpts := container.LogsOptions{
		ShowStdout: opts.Stdout,
		ShowStderr: opts.Stderr,
		Follow:     opts.Follow,
		Timestamps: opts.Timestamps,
		Tail:       opts.Tail,
		Since:      opts.Since,
		Until:      opts.Until,
	}

	logsReader, err := cli.ContainerLogs(ctx, name, logsOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}

	events := make(chan LogEvent)

	go func() {
		defer close(events)
		defer func() { _ = logsReader.Close() }()

		c.streamLogs(ctx, logsReader, events)
	}()

	return events, nil
}

// streamLogs reads from the Docker logs reader and sends events to the channel.
// Docker multiplexes stdout and stderr with an 8-byte header:
// [stream_type(1)][0(3)][size(4)][payload(size)]
// stream_type: 0=stdin, 1=stdout, 2=stderr
func (c *RealDockerClient) streamLogs(ctx context.Context, reader io.Reader, events chan<- LogEvent) {
	bufReader := bufio.NewReader(reader)
	header := make([]byte, 8)

	for {
		select {
		case <-ctx.Done():
			events <- LogEvent{
				Timestamp: time.Now(),
				Error:     ctx.Err(),
			}

			return
		default:
		}

		// Read the 8-byte header
		_, err := io.ReadFull(bufReader, header)
		if err != nil {
			if err != io.EOF {
				events <- LogEvent{
					Timestamp: time.Now(),
					Error:     fmt.Errorf("error reading log header: %w", err),
				}
			}

			return
		}

		// Parse stream type from header[0]
		streamType := "stdout"
		if header[0] == 2 {
			streamType = "stderr"
		}

		// Parse payload size from header[4:8] (big-endian)
		size := int(header[4])<<24 | int(header[5])<<16 | int(header[6])<<8 | int(header[7])

		if size == 0 {
			continue
		}

		// Read the payload
		payload := make([]byte, size)

		_, err = io.ReadFull(bufReader, payload)
		if err != nil {
			events <- LogEvent{
				Timestamp: time.Now(),
				Error:     fmt.Errorf("error reading log payload: %w", err),
			}

			return
		}

		events <- LogEvent{
			Timestamp: time.Now(),
			Stream:    streamType,
			Message:   string(payload),
		}
	}
}

// getEnvPtr returns a pointer to an environment variable value, or nil if not set
func getEnvPtr(key string) *string {
	val := os.Getenv(key)
	if val == "" {
		return nil
	}

	return &val
}
