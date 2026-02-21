package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

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
// ProviderArgs are forwarded as extra arguments to the docker build command.
func (p *Provider) Setup(ctx context.Context, config provider.SetupConfig) error {
	// Validate provider args don't conflict with Spinner-managed flags
	if err := ValidateDockerBuildArgs(config.ProviderArgs); err != nil {
		return err
	}

	return p.client.BuildImage(ctx, BuildConfig{
		Name:       config.Name,
		Dockerfile: config.Options["dockerfile"],
		ExtraArgs:  config.ProviderArgs,
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

	// Validate provider args don't conflict with Spinner-managed flags
	if err := ValidateDockerRunArgs(config.ProviderArgs); err != nil {
		return nil, err
	}

	sc := spinConfig{
		Image:         image,
		Repo:          config.Repo,
		Prompt:        config.Prompt,
		Branch:        config.Branch,
		MaxIterations: config.MaxIterations,
		Model:         config.Model,
		EnvVars:       config.EnvVars,
		EnvFile:       config.EnvFile,
		ExtraArgs:     config.ProviderArgs,
		SecretBlob:    config.SecretBlob,
		SecretKey:     config.SecretKey,
	}

	args, tmpFile, err := buildDockerRunCommand(sc, containerName, hasNpmrc)
	if err != nil {
		return nil, fmt.Errorf("failed to build run command: %w", err)
	}

	// Ensure temp file cleanup happens after docker run
	if tmpFile != "" {
		defer func() { _ = os.Remove(tmpFile) }()
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
// Before starting, it writes configuration overrides (prompt, max-iterations)
// to the host-mounted state directory so startup.sh picks up the latest values.
func (p *Provider) Start(ctx context.Context, name string, config provider.CreateConfig) (*provider.Instance, error) {
	// Write prompt override to the host-mounted state directory.
	// Docker containers cannot have their env vars updated after creation,
	// so we write to a file that startup.sh reads on boot.
	if err := writeConfigOverrides(name, config); err != nil {
		return nil, fmt.Errorf("failed to write config overrides: %w", err)
	}

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

// writeConfigOverrides writes prompt, max-iterations, and model to the host-mounted
// state directory so the container picks up updated values on restart.
// In inception mode (SPINNER_HOST_STATE_DIR set), writes to the outer container's
// bind-mounted /state/ dir so the host Docker daemon can resolve the path.
func writeConfigOverrides(containerName string, config provider.CreateConfig) error {
	var stateDir string

	if parentStateDir := os.Getenv("SPINNER_HOST_STATE_DIR"); parentStateDir != "" {
		// Inception mode: write under /state/<containerName>/state/ which maps to a
		// host-accessible path via the outer container's bind mount.
		stateDir = filepath.Join(inceptionMountBase, containerName, "state")
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		stateDir = filepath.Join(homeDir, ".spinner", containerName, "state")
	}

	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Write prompt override
	promptFile := filepath.Join(stateDir, "prompt.txt")
	if err := os.WriteFile(promptFile, []byte(config.Prompt), 0644); err != nil {
		return fmt.Errorf("failed to write prompt override: %w", err)
	}

	// Write max-iterations override
	maxIterations := config.MaxIterations
	if maxIterations == "" {
		maxIterations = defaultMaxIterations
	}

	maxIterFile := filepath.Join(stateDir, "max-iterations.txt")
	if err := os.WriteFile(maxIterFile, []byte(maxIterations), 0644); err != nil {
		return fmt.Errorf("failed to write max-iterations override: %w", err)
	}

	// Write model override
	modelFile := filepath.Join(stateDir, "model.txt")
	if err := os.WriteFile(modelFile, []byte(config.Model), 0644); err != nil {
		return fmt.Errorf("failed to write model override: %w", err)
	}

	return nil
}

// Stop stops a running container.
func (p *Provider) Stop(ctx context.Context, name string) error {
	return p.client.StopContainer(ctx, name)
}

// Remove force-removes a container and cleans up its state directory.
func (p *Provider) Remove(ctx context.Context, name string) error {
	result, err := p.client.RemoveContainer(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to remove instance: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("failed to remove instance: %s", result.Error)
	}

	// Clean up state directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	stateDir := filepath.Join(homeDir, ".spinner", name)
	if err := os.RemoveAll(stateDir); err != nil {
		return fmt.Errorf("failed to remove state directory: %w", err)
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
	return streamMetrics(ctx, name, ch)
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
		InstanceID: dockerInspectField(name, "{{.Id}}"),
		ImageID:    dockerInspectField(name, "{{.Image}}"),
	}

	// Read environment variables from the container (not the host process)
	envVars, _ := p.client.ContainerEnvVars(ctx, name)

	if model, ok := envVars["ANTHROPIC_MODEL"]; ok && model != "" {
		metadata.Agent = model
	}

	if maxIter, ok := envVars["MAX_ITERATIONS"]; ok && maxIter != "" {
		_, _ = fmt.Sscanf(maxIter, "%d", &metadata.MaxIterations)
	}

	if branch, ok := envVars["BRANCH"]; ok && branch != "" {
		metadata.Branch = branch
	}

	return metadata, nil
}

// List discovers all spinner-managed Docker containers and returns their info.
func (p *Provider) List(ctx context.Context) ([]provider.InstanceInfo, error) {
	containers, err := p.client.ListContainers(ctx, map[string]string{
		"spinner-managed": "true",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var instances []provider.InstanceInfo

	for _, c := range containers {
		name := c.Names[0]

		status := provider.InstanceStatusStopped
		if c.State == "running" {
			status = provider.InstanceStatusRunning
		}

		info := provider.InstanceInfo{
			Name:    name,
			Status:  status,
			Backend: "docker",
			Image:   c.Image,
		}

		// Read env vars for repo, branch, agent, max-iterations
		envVars, _ := p.client.ContainerEnvVars(ctx, name)
		if repo, ok := envVars["REPO_URL"]; ok {
			info.Repo = repo
		}

		if branch, ok := envVars["BRANCH"]; ok {
			info.Branch = branch
		}

		if model, ok := envVars["ANTHROPIC_MODEL"]; ok {
			info.Agent = model
		}

		if maxIter, ok := envVars["MAX_ITERATIONS"]; ok {
			_, _ = fmt.Sscanf(maxIter, "%d", &info.MaxIterations)
		}

		// Read state file from host for execution state
		p.enrichFromStateFile(&info, name)

		instances = append(instances, info)
	}

	return instances, nil
}

// enrichFromStateFile reads the state.json from the host-mounted state directory
// and populates execution state fields on the InstanceInfo.
func (p *Provider) enrichFromStateFile(info *provider.InstanceInfo, containerName string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	statePath := filepath.Join(homeDir, ".spinner", containerName, "state", "state.json")

	data, err := os.ReadFile(statePath)
	if err != nil {
		return
	}

	provider.EnrichFromStateData(info, data)
}

// dockerInspectField retrieves a single field from docker inspect for a container.
func dockerInspectField(containerName, format string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := fmt.Sprintf("docker inspect --format=%s %s", format, containerName)

	result, err := execCommand(ctx, cmd)
	if err != nil {
		return ""
	}

	return result
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
