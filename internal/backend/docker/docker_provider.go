package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rickihastings/spinner/internal/provider"
)

// Provider implements provider.Provider using Docker containers as the backend.
type Provider struct {
	client Client
}

// NewDockerProvider creates a new Provider backed by the given Client.
func NewDockerProvider(client Client) *Provider {
	return &Provider{client: client}
}

// Setup provisions a named environment by building a Docker image.
// Options: "base-image" (Docker base image), "dockerfile" (path to custom Dockerfile).
func (p *Provider) Setup(ctx context.Context, config provider.SetupConfig) error {
	baseImage := config.Options["base-image"]
	dockerfile := config.Options["dockerfile"]

	if dockerfile != "" {
		if _, err := os.Stat(dockerfile); os.IsNotExist(err) {
			return fmt.Errorf("dockerfile not found at path: %s", dockerfile)
		}
	}

	return p.client.BuildImage(ctx, BuildConfig{
		Name:       config.Name,
		BaseImage:  baseImage,
		Dockerfile: dockerfile,
	})
}

// InstanceName returns the deterministic container name for the given config.
// Options: "image" (Docker image name).
func (p *Provider) InstanceName(config provider.CreateConfig) string {
	return generateContainerName(spinConfig{
		Image:  config.Options["image"],
		Repo:   config.Repo,
		Branch: config.Branch,
	})
}

// Create creates and starts a new container from a provisioned image.
// Options: "image" (Docker image name).
func (p *Provider) Create(ctx context.Context, config provider.CreateConfig) (*provider.Instance, error) {
	image := config.Options["image"]
	containerName := p.InstanceName(config)

	exists, err := p.client.ImageExists(ctx, image)
	if err != nil {
		return nil, fmt.Errorf("failed to check image: %w", err)
	}

	if !exists {
		return nil, fmt.Errorf("image '%s' not found", image)
	}

	hasNpmrc := p.detectNpmrc()

	sc := spinConfig{
		Image:         image,
		Repo:          config.Repo,
		Prompt:        config.Prompt,
		Branch:        config.Branch,
		MaxIterations: config.MaxIterations,
	}

	args, err := buildDockerRunCommand(sc, containerName, hasNpmrc)
	if err != nil {
		return nil, fmt.Errorf("failed to build run command: %w", err)
	}

	result, err := p.client.RunContainer(ctx, args, containerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("failed to create instance: %s", result.Error)
	}

	verifyResult, err := p.client.VerifyContainerStatus(ctx, containerName)
	if err != nil {
		return nil, fmt.Errorf("instance verification failed: %w", err)
	}

	if !verifyResult.Success {
		return nil, fmt.Errorf("instance verification failed: %s", verifyResult.Error)
	}

	return &provider.Instance{Name: containerName, Status: provider.InstanceStatusRunning}, nil
}

// Start starts a stopped container and verifies it is running.
func (p *Provider) Start(ctx context.Context, name string) (*provider.Instance, error) {
	result, err := p.client.StartContainer(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to start instance: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("failed to start instance: %s", result.Error)
	}

	verifyResult, err := p.client.VerifyContainerStatus(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("instance verification failed: %w", err)
	}

	if !verifyResult.Success {
		return nil, fmt.Errorf("instance verification failed: %s", verifyResult.Error)
	}

	return &provider.Instance{Name: name, Status: provider.InstanceStatusRunning}, nil
}

// Restart stops then starts a container.
func (p *Provider) Restart(ctx context.Context, name string) (*provider.Instance, error) {
	if err := p.Stop(ctx, name); err != nil {
		return nil, err
	}

	return p.Start(ctx, name)
}

// Stop stops a running container.
func (p *Provider) Stop(ctx context.Context, name string) error {
	return p.client.StopContainer(ctx, name)
}

// Remove force-removes a container.
func (p *Provider) Remove(ctx context.Context, name string) error {
	result, err := p.client.RemoveContainer(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to remove instance: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("failed to remove instance: %s", result.Error)
	}

	return nil
}

// Logs returns the container's log output.
func (p *Provider) Logs(ctx context.Context, name string) (io.ReadCloser, error) {
	output, err := p.client.LogsContainer(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	return io.NopCloser(bytes.NewReader(output)), nil
}

// Status maps the container's state to a provider.InstanceStatus.
func (p *Provider) Status(ctx context.Context, name string) (provider.InstanceStatus, error) {
	status, err := p.client.ContainerExists(ctx, name)
	if err != nil {
		return provider.InstanceStatusNone, err
	}

	switch status {
	case StatusRunning:
		return provider.InstanceStatusRunning, nil
	case StatusStopped:
		return provider.InstanceStatusStopped, nil
	default:
		return provider.InstanceStatusNone, nil
	}
}

// WatchLogs implements provider.Provider.WatchLogs using docker.LogWatcher.
func (p *Provider) WatchLogs(ctx context.Context, name string, tailLines int, ch chan<- string) error {
	// Create LogWatcher with nil parser (raw mode)
	logWatcher, err := newLogWatcher(name, nil)
	if err != nil {
		return fmt.Errorf("failed to create log watcher: %w", err)
	}

	// Tail existing lines
	existingLines, err := logWatcher.tailExistingLines(ctx, tailLines)
	if err != nil {
		return fmt.Errorf("failed to tail logs: %w", err)
	}

	// Send existing lines
	for _, line := range existingLines {
		select {
		case ch <- line:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Watch for new lines
	return logWatcher.watchLines(ctx, ch)
}

// WatchMetrics implements provider.Provider.WatchMetrics for Docker containers.
func (p *Provider) WatchMetrics(ctx context.Context, name string, ch chan<- provider.ContainerMetrics) error {
	// Create Docker client for metrics
	cli, err := createMetricsClient()
	if err != nil {
		return fmt.Errorf("failed to create metrics client: %w", err)
	}

	defer func() {
		_ = cli.Close()
	}()

	// Stream metrics using the helper function
	return streamMetrics(ctx, cli, name, ch)
}

// GetInstanceMetadata returns metadata about a Docker container.
func (p *Provider) GetInstanceMetadata(ctx context.Context, name string) (*provider.InstanceMetadata, error) {
	// Check if container exists first
	status, err := p.client.ContainerExists(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to check container: %w", err)
	}

	if status == StatusNone {
		return nil, fmt.Errorf("container not found: %s", name)
	}

	metadata := &provider.InstanceMetadata{
		Backend:    "docker",
		InstanceID: getDockerContainerID(name),
		ImageID:    getDockerImageID(name),
	}

	// Try to get agent and max iterations from environment variables
	// These are set in the container at creation time
	if model := os.Getenv("ANTHROPIC_MODEL"); model != "" {
		metadata.Agent = model
	}

	if maxIter := os.Getenv("MAX_ITERATIONS"); maxIter != "" {
		if val, parseErr := fmt.Sscanf(maxIter, "%d", &metadata.MaxIterations); parseErr != nil || val != 1 {
			_ = fmt.Errorf("failed to parse max iterations: %s", maxIter)
		}
	}

	return metadata, nil
}

// getDockerContainerID gets the container ID for a given container name using docker inspect.
func getDockerContainerID(containerName string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*1000000000) // 5 seconds
	defer cancel()

	cmd := fmt.Sprintf("docker inspect --format={{.Id}} %s", containerName)

	result, err := execCommand(ctx, cmd)
	if err != nil {
		return ""
	}

	return result
}

// getDockerImageID gets the image ID for a given container name using docker inspect.
func getDockerImageID(containerName string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*1000000000) // 5 seconds
	defer cancel()

	cmd := fmt.Sprintf("docker inspect --format={{.Image}} %s", containerName)

	result, err := execCommand(ctx, cmd)
	if err != nil {
		return ""
	}

	return result
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

// detectNpmrc checks whether ~/.npmrc exists on the host.
func (p *Provider) detectNpmrc() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	_, err = os.Stat(filepath.Join(homeDir, ".npmrc"))

	return err == nil
}
