package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rickihastings/spinner/internal/provider"
)

const (
	// metricsPollInterval is how often WatchMetrics polls Cloud Monitoring.
	// Cloud Monitoring has a minimum granularity of 60 seconds for most metrics.
	metricsPollInterval = 60 * time.Second

	// statePollInterval is how often to poll GCS for state updates (iteration count).
	// This is much faster than Cloud Monitoring since GCS reads are cheap and
	// the iteration count changes frequently.
	statePollInterval = 5 * time.Second

	// cpuMetricType is the fully-qualified Cloud Monitoring metric type for CPU utilization.
	// Returns a value between 0.0 and 1.0.
	cpuMetricType = "compute.googleapis.com/instance/cpu/utilization"

	// memoryPercentMetricType is the Ops Agent metric for memory usage percentage.
	// Requires the Ops Agent to be installed on the VM.
	// Returns percentage used (0-100) aggregated across all memory states.
	memoryPercentMetricType = "agent.googleapis.com/memory/percent_used"

	// metricsQueryIntervalSeconds is how far back to query for the latest data point.
	// Using 5 minutes to ensure we capture at least one data point given Cloud Monitoring's
	// write delay (typically 1-3 minutes).
	metricsQueryIntervalSeconds = 300
)

// collectGCPMetrics collects a single snapshot of VM metrics.
// It queries the VM's status, CPU utilization, and memory utilization from Cloud Monitoring.
// It also reads the current iteration from the state file in GCS.
// Note: Memory metrics require the Ops Agent to be installed. If the agent is not installed,
// memory fields will be zero and the UI will display "N/A".
func collectGCPMetrics(ctx context.Context, client Client, project, zone, instanceName, bucket string) provider.ContainerMetrics {
	// Get VM state first
	state, err := getVMState(ctx, client, project, zone, instanceName)
	if err != nil {
		return provider.ContainerMetrics{
			State: provider.StateUnknown,
			Error: fmt.Errorf("failed to get VM state: %w", err),
		}
	}

	// If VM is not running, return state without querying metrics
	if state != provider.StateRunning {
		return provider.ContainerMetrics{
			State: state,
		}
	}

	// Query CPU utilization
	// CPU query errors are non-fatal - Cloud Monitoring may have no data yet for
	// recently-created VMs. We return 0% instead of setting Error to avoid showing
	// "Error" in the UI for both CPU and Memory fields.
	cpuPercent, _ := queryCPUUtilization(ctx, client, project, zone, instanceName)

	// Query memory utilization
	// Memory metrics require the Ops Agent. If not installed, this returns 0.
	memoryPercent, _ := queryMemoryUtilization(ctx, client, project, zone, instanceName)

	metrics := provider.ContainerMetrics{
		State:         state,
		CPUPercent:    cpuPercent,
		MemoryPercent: memoryPercent,
		// MemoryUsed and MemoryLimit are not available from GCP metrics.
		// The Ops Agent only provides memory percentage, not absolute bytes.
		// We populate MemoryPercent which is sufficient for the watch UI.
		MemoryUsed:  0,
		MemoryLimit: 0,
	}

	// Read iteration count from GCS state file
	metrics.Iteration = readIterationFromGCS(ctx, client, bucket, instanceName)

	return metrics
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

	vmStatus := vmStatus(instance.GetStatus())

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

// queryCPUUtilization queries Cloud Monitoring for the latest CPU utilization
// value and returns it as a percentage (0-100).
func queryCPUUtilization(ctx context.Context, client Client, project, zone, instanceName string) (float64, error) {
	points, err := client.QueryTimeSeries(ctx, project, metricsQuery{
		MetricType:      cpuMetricType,
		InstanceName:    instanceName,
		Zone:            zone,
		IntervalSeconds: metricsQueryIntervalSeconds,
	})
	if err != nil {
		return 0, err
	}

	if len(points) == 0 {
		return 0, nil
	}

	// Use the most recent data point (last in the list).
	// Cloud Monitoring returns points in reverse chronological order,
	// but the SDK may vary; use the last point for recency.
	latest := points[len(points)-1]

	// CPU utilization from Cloud Monitoring is 0.0-1.0; scale to 0-100.
	return latest.Value * 100.0, nil
}

// queryMemoryUtilization queries Cloud Monitoring for the latest memory utilization
// from the Ops Agent and returns it as a percentage (0-100).
// Returns 0 if the Ops Agent is not installed or no data is available.
func queryMemoryUtilization(ctx context.Context, client Client, project, zone, instanceName string) (float64, error) {
	points, err := client.QueryTimeSeries(ctx, project, metricsQuery{
		MetricType:      memoryPercentMetricType,
		InstanceName:    instanceName,
		Zone:            zone,
		IntervalSeconds: metricsQueryIntervalSeconds,
	})
	if err != nil {
		// Memory metrics unavailable (Ops Agent not installed or not yet reporting)
		return 0, nil
	}

	if len(points) == 0 {
		return 0, nil
	}

	// Use the most recent data point
	latest := points[len(points)-1]

	// Memory percent from Ops Agent is already 0-100
	return latest.Value, nil
}

// readIterationFromGCS reads the current iteration count from the instance's
// state file in GCS. Returns 0 if the state cannot be read.
func readIterationFromGCS(ctx context.Context, client Client, bucket, instanceName string) int {
	if bucket == "" {
		return 0
	}

	data, err := readState(ctx, client, bucket, instanceName)
	if err != nil || data == nil {
		return 0
	}

	// Parse just the iteration field from the state JSON
	var state struct {
		Iteration int `json:"iteration"`
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return 0
	}

	return state.Iteration
}

// streamGCPMetrics polls Cloud Monitoring and GCS state, sending metrics to the channel.
// It uses two poll intervals: a fast one (5s) for GCS state (iteration count) and a
// slow one (60s) for Cloud Monitoring (CPU/memory). On fast ticks, only the iteration
// is updated. On slow ticks, a full metrics collection is performed.
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

	// Two tickers: fast for state/iteration, slow for Cloud Monitoring metrics
	stateTicker := time.NewTicker(statePollInterval)
	defer stateTicker.Stop()

	metricsTicker := time.NewTicker(metricsPollInterval)
	defer metricsTicker.Stop()

	// Track latest full metrics so state-only updates can merge
	lastMetrics := metrics

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-metricsTicker.C:
			// Full metrics collection (Cloud Monitoring + GCS state)
			lastMetrics = collectGCPMetrics(ctx, client, project, zone, instanceName, bucket)

			select {
			case ch <- lastMetrics:
			case <-ctx.Done():
				return ctx.Err()
			}

			// Fatal errors stop streaming
			if lastMetrics.Error != nil && lastMetrics.State == provider.StateUnknown {
				return lastMetrics.Error
			}

			// VM stopped or exited — done
			if lastMetrics.State == provider.StateStopped || lastMetrics.State == provider.StateExited {
				return nil
			}
		case <-stateTicker.C:
			// Fast state-only update: read iteration from GCS and send
			// updated metrics without querying Cloud Monitoring
			iteration := readIterationFromGCS(ctx, client, bucket, instanceName)
			if iteration != lastMetrics.Iteration {
				lastMetrics.Iteration = iteration

				select {
				case ch <- lastMetrics:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
}
