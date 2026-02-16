package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rickihastings/spinner/internal/provider"
)

const (
	// statePollInterval is how often to poll GCS for state updates.
	// GCS reads are cheap and the state file contains all metrics data.
	statePollInterval = 5 * time.Second
)

// gcsState holds the fields parsed from the GCS state file that are
// relevant for the watch UI metrics display.
type gcsState struct {
	Iteration     int     `json:"iteration"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
}

// readStateFromGCS reads and parses the instance's state file from GCS.
// Returns a zero-value gcsState if the state cannot be read.
func readStateFromGCS(ctx context.Context, client Client, bucket, instanceName string) gcsState {
	if bucket == "" {
		return gcsState{}
	}

	data, err := readState(ctx, client, bucket, instanceName)
	if err != nil || data == nil {
		return gcsState{}
	}

	var state gcsState
	if err := json.Unmarshal(data, &state); err != nil {
		return gcsState{}
	}

	return state
}

// collectGCPMetrics collects a single snapshot of VM metrics.
// It queries the VM's status and reads CPU/memory/iteration from the GCS state file.
func collectGCPMetrics(ctx context.Context, client Client, project, zone, instanceName, bucket string) provider.ContainerMetrics {
	// Get VM state first
	state, err := getVMState(ctx, client, project, zone, instanceName)
	if err != nil {
		return provider.ContainerMetrics{
			State: provider.StateUnknown,
			Error: fmt.Errorf("failed to get VM state: %w", err),
		}
	}

	// If VM is not running, return state without reading metrics
	if state != provider.StateRunning {
		return provider.ContainerMetrics{
			State: state,
		}
	}

	// Read metrics from GCS state file
	gcsData := readStateFromGCS(ctx, client, bucket, instanceName)

	return provider.ContainerMetrics{
		State:         state,
		CPUPercent:    gcsData.CPUPercent,
		MemoryPercent: gcsData.MemoryPercent,
		Iteration:     gcsData.Iteration,
	}
}

// getVMState queries the VM's current status and maps it to a provider.ContainerState.
func getVMState(ctx context.Context, client Client, project, zone, name string) (provider.ContainerState, error) {
	instance, err := client.GetInstance(ctx, project, zone, name)
	if err != nil {
		if isNotFoundError(err) {
			return provider.StateExited, nil
		}

		return provider.StateUnknown, err
	}

	vmStatus := vmStatus(instance.Status)

	switch vmStatus {
	case vmStatusRunning:
		return provider.StateRunning, nil
	case vmStatusTerminated, vmStatusStopped, vmStatusSuspended:
		return provider.StateStopped, nil
	case vmStatusProvisioning, vmStatusStaging:
		return provider.StateRunning, nil
	case vmStatusStopping, vmStatusSuspending:
		return provider.StateStopped, nil
	default:
		return provider.StateUnknown, nil
	}
}

// streamGCPMetrics polls GCS state at a fixed interval, sending metrics to the channel.
// All metrics (CPU, memory, iteration) come from the GCS state file which is updated
// by the exec loop inside the VM.
func streamGCPMetrics(ctx context.Context, client Client, project, zone, instanceName, bucket string, ch chan<- provider.ContainerMetrics) error {
	// Send initial metrics immediately
	metrics := collectGCPMetrics(ctx, client, project, zone, instanceName, bucket)

	select {
	case ch <- metrics:
	case <-ctx.Done():
		return ctx.Err()
	}

	// If there was a fatal error (VM not found, etc.), stop streaming
	if metrics.Error != nil && metrics.State == provider.StateUnknown {
		return metrics.Error
	}

	// If VM is stopped or exited, stop streaming
	if metrics.State == provider.StateStopped || metrics.State == provider.StateExited {
		return nil
	}

	ticker := time.NewTicker(statePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			metrics = collectGCPMetrics(ctx, client, project, zone, instanceName, bucket)

			select {
			case ch <- metrics:
			case <-ctx.Done():
				return ctx.Err()
			}

			// Fatal errors stop streaming
			if metrics.Error != nil && metrics.State == provider.StateUnknown {
				return metrics.Error
			}

			// VM stopped or exited — done
			if metrics.State == provider.StateStopped || metrics.State == provider.StateExited {
				return nil
			}
		}
	}
}
