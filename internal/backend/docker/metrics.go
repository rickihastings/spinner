package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	execpkg "github.com/rickihastings/spinner/internal/exec"
	"github.com/rickihastings/spinner/internal/provider"
)

// dockerStatsJSON represents the JSON output from docker stats --no-stream --format json.
type dockerStatsJSON struct {
	CPUPerc  string `json:"CPUPerc"`
	MemUsage string `json:"MemUsage"`
}

// dockerInspectState represents the State field from docker inspect JSON output.
type dockerInspectState struct {
	Running   bool `json:"Running"`
	Dead      bool `json:"Dead"`
	OOMKilled bool `json:"OOMKilled"`
	ExitCode  int  `json:"ExitCode"`
}

// streamMetrics streams real-time resource metrics for a container using CLI commands.
func streamMetrics(ctx context.Context, containerName string, ch chan<- provider.ContainerMetrics) error {
	// Send initial metrics immediately
	metrics := collectMetrics(ctx, containerName)

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
			metrics := collectMetrics(ctx, containerName)

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

// collectMetrics collects metrics for a single poll using Docker CLI commands.
func collectMetrics(ctx context.Context, containerName string) provider.ContainerMetrics {
	// First, inspect the container to get its state
	state, err := inspectContainerState(ctx, containerName)
	if err != nil {
		return provider.ContainerMetrics{
			State: provider.StateUnknown,
			Error: fmt.Errorf("failed to inspect container: %w", err),
		}
	}

	// Determine container state
	containerState := mapDockerStateToMetrics(state)

	// If container is not running, return state without stats
	if containerState != provider.StateRunning {
		return provider.ContainerMetrics{
			State: containerState,
		}
	}

	// Get container stats using docker stats CLI
	cpuPercent, memoryUsed, memoryLimit, err := getContainerStats(ctx, containerName)
	if err != nil {
		return provider.ContainerMetrics{
			State: containerState,
			Error: fmt.Errorf("failed to get container stats: %w", err),
		}
	}

	memoryPercent := 0.0
	if memoryLimit > 0 {
		memoryPercent = (float64(memoryUsed) / float64(memoryLimit)) * 100.0
	}

	metrics := provider.ContainerMetrics{
		State:         containerState,
		CPUPercent:    cpuPercent,
		MemoryUsed:    memoryUsed,
		MemoryLimit:   memoryLimit,
		MemoryPercent: memoryPercent,
	}

	// Read iteration count from state file
	metrics.Iteration = readIterationFromState(containerName)

	return metrics
}

// inspectContainerState gets the container's state using docker inspect.
func inspectContainerState(ctx context.Context, containerName string) (*dockerInspectState, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{json .State}}", containerName)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker inspect failed: %w", err)
	}

	var state dockerInspectState
	if err := json.Unmarshal(output, &state); err != nil {
		return nil, fmt.Errorf("failed to parse inspect state: %w", err)
	}

	return &state, nil
}

// getContainerStats gets CPU and memory stats using docker stats CLI.
// Returns cpuPercent, memoryUsed (bytes), memoryLimit (bytes), error.
func getContainerStats(ctx context.Context, containerName string) (float64, uint64, uint64, error) {
	cmd := exec.CommandContext(ctx, "docker", "stats", "--no-stream", "--format", "json", containerName)

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("docker stats failed: %w", err)
	}

	var stats dockerStatsJSON
	if err := json.Unmarshal(output, &stats); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse stats: %w", err)
	}

	cpuPercent := parseCPUPercent(stats.CPUPerc)
	memUsed, memLimit := parseMemUsage(stats.MemUsage)

	return cpuPercent, memUsed, memLimit, nil
}

// parseCPUPercent parses the CPU percentage string from docker stats (e.g. "1.23%").
func parseCPUPercent(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0
	}

	return val
}

// parseMemUsage parses the memory usage string from docker stats (e.g. "100MiB / 1.94GiB").
// Returns memoryUsed and memoryLimit in bytes.
func parseMemUsage(s string) (uint64, uint64) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0, 0
	}

	used := parseMemoryValue(strings.TrimSpace(parts[0]))
	limit := parseMemoryValue(strings.TrimSpace(parts[1]))

	return used, limit
}

// parseMemoryValue parses a memory value string like "100MiB", "1.94GiB", "512KiB", "1024B".
func parseMemoryValue(s string) uint64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	// Try binary suffixes (Docker uses these)
	suffixes := []struct {
		suffix     string
		multiplier uint64
	}{
		{"GiB", 1024 * 1024 * 1024},
		{"MiB", 1024 * 1024},
		{"KiB", 1024},
		{"B", 1},
	}

	for _, sf := range suffixes {
		if strings.HasSuffix(s, sf.suffix) {
			numStr := strings.TrimSuffix(s, sf.suffix)

			val, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0
			}

			return uint64(val * float64(sf.multiplier))
		}
	}

	return 0
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

// mapDockerStateToMetrics converts Docker container inspect state to provider.ContainerState type.
func mapDockerStateToMetrics(state *dockerInspectState) provider.ContainerState {
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
