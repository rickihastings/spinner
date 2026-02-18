package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rickihastings/spinner/internal/util"
	"github.com/rickihastings/spinner/internal/version"
)

const (
	// defaultStopTimeoutSecs is the grace period before force-killing a container.
	defaultStopTimeoutSecs = "10"

	// defaultLogTailLines is the number of trailing log lines to fetch for diagnostics.
	defaultLogTailLines = "100"
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

	// ListContainers lists containers matching the given label filters.
	// The filters map keys are filter names (e.g. "label") and values are filter values.
	ListContainers(ctx context.Context, filterLabels map[string]string) ([]ContainerListEntry, error)

	// ContainerEnvVars reads environment variables from a container.
	ContainerEnvVars(ctx context.Context, name string) (map[string]string, error)
}

// RealDockerClient implements Client using the Docker CLI.
type RealDockerClient struct{}

// NewRealDockerClient creates a new RealDockerClient instance.
func NewRealDockerClient() *RealDockerClient {
	return &RealDockerClient{}
}

// BuildImage builds a Docker image with the given configuration using the Docker CLI.
func (c *RealDockerClient) BuildImage(ctx context.Context, config BuildConfig) error {
	buildContextDir := filepath.Join(os.TempDir(), fmt.Sprintf("spinner-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(buildContextDir, 0755); err != nil {
		return fmt.Errorf("failed to create build context: %w", err)
	}

	defer func() { _ = os.RemoveAll(buildContextDir) }()

	// Generate the final Dockerfile
	dockerfilePath := filepath.Join(buildContextDir, "Dockerfile")

	dockerfileContent, err := generateDockerfile(dockerfileConfig{BaseImage: "ubuntu:22.04"})
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

	// Build the final image using CLI
	imageName := fmt.Sprintf("spinner:%s", config.Name)
	if err := c.buildWithCLI(ctx, buildContextDir, "Dockerfile", imageName, config.ExtraArgs); err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	return nil
}

// buildWithCLI builds a Docker image using the docker build CLI command.
// ExtraArgs are injected before the context directory argument.
func (c *RealDockerClient) buildWithCLI(ctx context.Context, contextDir, dockerfileName, tag string, extraArgs []string) error {
	args := []string{"build", "-t", tag, "-f", filepath.Join(contextDir, dockerfileName)}

	// Add build args
	localBuild := os.Getenv("LOCAL_BUILD")
	if localBuild != "" {
		args = append(args, "--build-arg", fmt.Sprintf("LOCAL_BUILD=%s", localBuild))
	}

	// Pin the container's spinner version to match the host binary.
	// Skip for dev builds — they use LOCAL_BUILD instead.
	if localBuild == "" && version.IsRelease() {
		args = append(args, "--build-arg", fmt.Sprintf("SPINNER_VERSION=%s", version.Tag()))
	}

	// Append provider pass-through args before the context directory
	args = append(args, extraArgs...)

	args = append(args, contextDir)

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
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

// ImageExists checks if a Docker image exists using the Docker CLI.
func (c *RealDockerClient) ImageExists(ctx context.Context, imageName string) (bool, error) {
	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", imageName)
	if err := cmd.Run(); err != nil {
		// Exit code != 0 means image doesn't exist (or docker error)
		return false, nil
	}

	return true, nil
}

// ContainerExists checks if a container exists and returns its status using the Docker CLI.
func (c *RealDockerClient) ContainerExists(ctx context.Context, name string) (ContainerStatus, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Running}}", name)

	output, err := cmd.Output()
	if err != nil {
		// Container doesn't exist
		return StatusNone, nil
	}

	if strings.TrimSpace(string(output)) == "true" {
		return StatusRunning, nil
	}

	return StatusStopped, nil
}

// RemoveContainer removes a container, forcing removal if it's running using the Docker CLI.
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

// StartContainer starts a stopped container using the Docker CLI.
func (c *RealDockerClient) StartContainer(ctx context.Context, name string) (ContainerResult, error) {
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

// StopContainer stops a running container using the Docker CLI.
func (c *RealDockerClient) StopContainer(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "docker", "stop", "-t", defaultStopTimeoutSecs, name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop container: %s", strings.TrimSpace(string(output)))
	}

	return nil
}

// LogsContainer returns the logs from a container using the Docker CLI.
func (c *RealDockerClient) LogsContainer(ctx context.Context, name string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "docker", "logs", name)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %s", strings.TrimSpace(string(output)))
	}

	return output, nil
}

// VerifyContainerStatus verifies that a container is running using the Docker CLI.
func (c *RealDockerClient) VerifyContainerStatus(ctx context.Context, name string) (ContainerResult, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Running}} {{.State.Status}}", name)

	output, err := cmd.Output()
	if err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: name,
			Error:         "Failed to verify container status",
		}, err
	}

	parts := strings.Fields(strings.TrimSpace(string(output)))
	running := len(parts) > 0 && parts[0] == "true"

	stateStatus := ""
	if len(parts) > 1 {
		stateStatus = parts[1]
	}

	if !running {
		// Get last 100 lines of logs to show what went wrong
		logsCmd := exec.CommandContext(ctx, "docker", "logs", "--tail", defaultLogTailLines, name)
		logsOutput, _ := logsCmd.CombinedOutput()

		return ContainerResult{
			Success:       false,
			ContainerName: name,
			Error:         fmt.Sprintf("Container exited. Logs: %s", strings.TrimSpace(string(logsOutput))),
		}, fmt.Errorf("container not running: %s", stateStatus)
	}

	return ContainerResult{
		Success:       true,
		ContainerName: name,
	}, nil
}

// StreamContainerLogs streams container logs in real-time using the Docker CLI.
// It returns a channel that receives LogEvent entries as they arrive from the container.
// The channel is closed when:
//   - The context is cancelled
//   - The container stops (if Follow is true)
//   - An error occurs (error is sent in LogEvent.Error before closing)
func (c *RealDockerClient) StreamContainerLogs(ctx context.Context, name string, opts LogStreamOptions) (<-chan LogEvent, error) {
	args := []string{"logs"}

	if opts.Follow {
		args = append(args, "--follow")
	}

	if opts.Timestamps {
		args = append(args, "--timestamps")
	}

	if opts.Tail != "" && opts.Tail != "all" {
		args = append(args, "--tail", opts.Tail)
	}

	if opts.Since != "" {
		args = append(args, "--since", opts.Since)
	}

	if opts.Until != "" {
		args = append(args, "--until", opts.Until)
	}

	args = append(args, name)

	cmd := exec.CommandContext(ctx, "docker", args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start docker logs: %w", err)
	}

	events := make(chan LogEvent)

	go func() {
		defer close(events)

		var wg sync.WaitGroup

		if opts.Stdout {
			wg.Add(1)

			go func() {
				defer wg.Done()

				scanner := bufio.NewScanner(stdoutPipe)
				for scanner.Scan() {
					select {
					case events <- LogEvent{
						Timestamp: time.Now(),
						Stream:    "stdout",
						Message:   scanner.Text() + "\n",
					}:
					case <-ctx.Done():
						return
					}
				}
			}()
		}

		if opts.Stderr {
			wg.Add(1)

			go func() {
				defer wg.Done()

				scanner := bufio.NewScanner(stderrPipe)
				for scanner.Scan() {
					select {
					case events <- LogEvent{
						Timestamp: time.Now(),
						Stream:    "stderr",
						Message:   scanner.Text() + "\n",
					}:
					case <-ctx.Done():
						return
					}
				}
			}()
		}

		wg.Wait()
		_ = cmd.Wait()
	}()

	return events, nil
}

// ListContainers lists containers matching the given label filters using the Docker CLI.
func (c *RealDockerClient) ListContainers(ctx context.Context, filterLabels map[string]string) ([]ContainerListEntry, error) {
	args := []string{"ps", "-a", "--no-trunc", "--format", "json"}

	for key, value := range filterLabels {
		args = append(args, "--filter", fmt.Sprintf("label=%s=%s", key, value))
	}

	cmd := exec.CommandContext(ctx, "docker", args...)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	return parseContainerListOutput(string(output))
}

// dockerPSJSON represents the JSON output from docker ps --format json.
type dockerPSJSON struct {
	ID     string `json:"ID"`
	Names  string `json:"Names"`
	Image  string `json:"Image"`
	State  string `json:"State"`
	Labels string `json:"Labels"`
}

// parseContainerListOutput parses the line-delimited JSON from docker ps --format json.
func parseContainerListOutput(output string) ([]ContainerListEntry, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil, nil
	}

	var containers []ContainerListEntry

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var raw dockerPSJSON
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			return nil, fmt.Errorf("failed to parse container entry: %w", err)
		}

		containers = append(containers, containerListEntryFromJSON(raw))
	}

	return containers, nil
}

// containerListEntryFromJSON converts docker ps JSON output to a ContainerListEntry.
func containerListEntryFromJSON(raw dockerPSJSON) ContainerListEntry {
	names := strings.Split(raw.Names, ",")

	labels := make(map[string]string)

	if raw.Labels != "" {
		for _, label := range strings.Split(raw.Labels, ",") {
			if idx := strings.IndexByte(label, '='); idx >= 0 {
				labels[label[:idx]] = label[idx+1:]
			}
		}
	}

	return ContainerListEntry{
		ID:     raw.ID,
		Names:  names,
		Image:  raw.Image,
		State:  raw.State,
		Labels: labels,
	}
}

// ContainerEnvVars reads environment variables from a container using docker inspect.
func (c *RealDockerClient) ContainerEnvVars(ctx context.Context, name string) (map[string]string, error) {
	inspectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(inspectCtx, "docker", "inspect", "--format", "{{range .Config.Env}}{{println .}}{{end}}", name)

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	envMap := make(map[string]string)

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if idx := strings.IndexByte(line, '='); idx >= 0 {
			envMap[line[:idx]] = line[idx+1:]
		}
	}

	return envMap, nil
}

// execCommand executes a shell command and returns the output.
func execCommand(ctx context.Context, command string) (string, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
