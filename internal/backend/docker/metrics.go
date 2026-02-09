package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	execpkg "github.com/rickihastings/spinner/internal/exec"
	"github.com/rickihastings/spinner/internal/provider"
)

// metricsAPIClient defines the Docker API methods needed for metrics collection
// This interface enables mocking for tests
type metricsAPIClient interface {
	ContainerInspect(ctx context.Context, containerID string) (containertypes.InspectResponse, error)
	ContainerStats(ctx context.Context, containerID string, stream bool) (containertypes.StatsResponseReader, error)
	Close() error
}

// createMetricsClient creates a Docker client for metrics collection
func createMetricsClient() (metricsAPIClient, error) {
	// Detect the correct Docker host
	host, err := detectDockerHost()
	if err != nil {
		return nil, fmt.Errorf("failed to detect Docker host: %w", err)
	}

	// Build client options
	opts := []client.Opt{client.WithAPIVersionNegotiation()}
	if host != "" {
		opts = append(opts, client.WithHost(host))
	} else {
		opts = append(opts, client.FromEnv)
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return cli, nil
}

// detectDockerHost detects the correct Docker socket path by checking:
// 1. DOCKER_HOST environment variable
// 2. Current Docker context
// 3. Common socket locations for Docker Desktop on macOS
func detectDockerHost() (string, error) {
	// Check if DOCKER_HOST is already set
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		return "", nil // FromEnv will handle it
	}

	// Try to get the host from the current Docker context
	cmd := exec.Command("docker", "context", "inspect", "-f", "{{.Endpoints.docker.Host}}")

	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		host := strings.TrimSpace(string(output))
		if host != "" && host != "<no value>" {
			return host, nil
		}
	}

	// On macOS, check for Docker Desktop socket
	homeDir, err := os.UserHomeDir()
	if err == nil {
		macSocketPath := filepath.Join(homeDir, ".docker", "run", "docker.sock")
		if _, err := os.Stat(macSocketPath); err == nil {
			return "unix://" + macSocketPath, nil
		}
	}

	// Return empty to use default behavior
	return "", nil
}

// streamMetrics streams real-time resource metrics for a container
func streamMetrics(ctx context.Context, cli metricsAPIClient, containerName string, ch chan<- provider.ContainerMetrics) error {
	// Send initial metrics immediately
	metrics := collectMetrics(ctx, cli, containerName)
	select {
	case ch <- metrics:
	case <-ctx.Done():
		return ctx.Err()
	}

	// If there was a fatal error (container not found, etc.), stop streaming
	if metrics.Error != nil {
		return metrics.Error
	}

	// If container is stopped or exited, stop streaming
	if metrics.State == provider.StateStopped || metrics.State == provider.StateExited {
		return nil
	}

	ticker := time.NewTicker(1500 * time.Millisecond) // Poll every 1.5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			metrics := collectMetrics(ctx, cli, containerName)

			// Send metrics to channel
			select {
			case ch <- metrics:
			case <-ctx.Done():
				return ctx.Err()
			}

			// If there was a fatal error (container not found, etc.), stop streaming
			if metrics.Error != nil {
				return metrics.Error
			}

			// If container is stopped or exited, stop streaming
			if metrics.State == provider.StateStopped || metrics.State == provider.StateExited {
				return nil
			}
		}
	}
}

// collectMetrics collects metrics for a single poll
func collectMetrics(ctx context.Context, cli metricsAPIClient, containerName string) provider.ContainerMetrics {
	// First, inspect the container to get its state
	inspect, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return provider.ContainerMetrics{
			State: provider.StateUnknown,
			Error: fmt.Errorf("failed to inspect container: %w", err),
		}
	}

	// Determine container state
	state := mapDockerStateToMetrics(inspect.State)

	// If container is not running, return state without stats
	if state != provider.StateRunning {
		return provider.ContainerMetrics{
			State: state,
		}
	}

	// Get container stats
	statsResp, err := cli.ContainerStats(ctx, containerName, false) // stream=false for single read
	if err != nil {
		return provider.ContainerMetrics{
			State: state,
			Error: fmt.Errorf("failed to get container stats: %w", err),
		}
	}

	defer func() {
		_ = statsResp.Body.Close()
	}()

	// Parse stats response
	var stats containertypes.StatsResponse
	if err := json.NewDecoder(statsResp.Body).Decode(&stats); err != nil {
		// Read and discard remaining data to avoid resource leaks
		_, _ = io.Copy(io.Discard, statsResp.Body)

		return provider.ContainerMetrics{
			State: state,
			Error: fmt.Errorf("failed to decode stats: %w", err),
		}
	}

	// Calculate CPU percentage
	cpuPercent := calculateCPUPercent(&stats)

	// Extract memory metrics
	memoryUsed := stats.MemoryStats.Usage
	memoryLimit := stats.MemoryStats.Limit

	memoryPercent := 0.0
	if memoryLimit > 0 {
		memoryPercent = (float64(memoryUsed) / float64(memoryLimit)) * 100.0
	}

	metrics := provider.ContainerMetrics{
		State:         state,
		CPUPercent:    cpuPercent,
		MemoryUsed:    memoryUsed,
		MemoryLimit:   memoryLimit,
		MemoryPercent: memoryPercent,
	}

	// Read iteration count from state file
	metrics.Iteration = readIterationFromState(containerName)

	return metrics
}

// readIterationFromState reads the current iteration count from the container's state file.
func readIterationFromState(containerName string) int {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return 0
	}

	statePath := filepath.Join(homeDir, ".spinner", containerName, "state", "state.json")

	state, err := execpkg.LoadState(statePath)
	if err != nil {
		return 0
	}

	return state.Iteration
}

// mapDockerStateToMetrics converts Docker container state to provider.ContainerState type
func mapDockerStateToMetrics(state *containertypes.State) provider.ContainerState {
	if state == nil {
		return provider.StateUnknown
	}

	if state.Running {
		return provider.StateRunning
	}

	if state.ExitCode != 0 || state.Dead || state.OOMKilled {
		return provider.StateExited
	}

	return provider.StateStopped
}

// calculateCPUPercent calculates CPU usage percentage from Docker stats
// Algorithm based on Docker CLI implementation
func calculateCPUPercent(stats *containertypes.StatsResponse) float64 {
	// Get CPU delta
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage) - float64(stats.PreCPUStats.CPUUsage.TotalUsage)

	// Get system delta
	systemDelta := float64(stats.CPUStats.SystemUsage) - float64(stats.PreCPUStats.SystemUsage)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		// Number of CPUs
		numCPUs := float64(stats.CPUStats.OnlineCPUs)

		if numCPUs == 0 {
			numCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
		}

		if numCPUs == 0 {
			numCPUs = 1.0
		}

		return (cpuDelta / systemDelta) * numCPUs * 100.0
	}

	return 0.0
}
