package gcp

import (
	"context"
	"fmt"
	"time"

	"github.com/rickihastings/spinner/internal/provider"
)

const (
	// metricsPollInterval is how often WatchMetrics polls Cloud Monitoring.
	// Cloud Monitoring has a minimum granularity of 60 seconds for most metrics.
	metricsPollInterval = 60 * time.Second

	// cpuMetricType is the fully-qualified Cloud Monitoring metric type for CPU utilization.
	// Returns a value between 0.0 and 1.0.
	cpuMetricType = "compute.googleapis.com/instance/cpu/utilization"

	// metricsQueryIntervalSeconds is how far back to query for the latest data point.
	// Using 5 minutes to ensure we capture at least one data point given Cloud Monitoring's
	// write delay (typically 1-3 minutes).
	metricsQueryIntervalSeconds = 300
)

// collectGCPMetrics collects a single snapshot of VM metrics.
// It queries the VM's status and CPU utilization from Cloud Monitoring.
func collectGCPMetrics(ctx context.Context, client Client, project, zone, instanceName string) provider.ContainerMetrics {
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
	cpuPercent, cpuErr := queryCPUUtilization(ctx, client, project, zone, instanceName)

	metrics := provider.ContainerMetrics{
		State:      state,
		CPUPercent: cpuPercent,
	}

	if cpuErr != nil {
		// CPU query errors are non-fatal; report in the metrics but don't fail streaming.
		// Cloud Monitoring may have no data yet for recently-created VMs.
		metrics.Error = fmt.Errorf("failed to query CPU metrics: %w", cpuErr)
	}

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

	vmStatus := VMStatus(instance.GetStatus())

	switch vmStatus {
	case VMStatusRunning:
		return provider.StateRunning, nil
	case VMStatusTerminated, VMStatusStopped, VMStatusSuspended:
		return provider.StateStopped, nil
	case VMStatusProvisioning, VMStatusStaging:
		return provider.StateRunning, nil
	case VMStatusStopping, VMStatusSuspending:
		return provider.StateStopped, nil
	default:
		return provider.StateUnknown, nil
	}
}

// queryCPUUtilization queries Cloud Monitoring for the latest CPU utilization
// value and returns it as a percentage (0-100).
func queryCPUUtilization(ctx context.Context, client Client, project, zone, instanceName string) (float64, error) {
	points, err := client.QueryTimeSeries(ctx, project, MetricsQuery{
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

// streamGCPMetrics polls Cloud Monitoring and sends metrics to the channel.
// It follows the same pattern as the Docker backend: send initial metrics
// immediately, then poll at regular intervals. Stops when the VM is stopped/exited
// or the context is cancelled.
func streamGCPMetrics(ctx context.Context, client Client, project, zone, instanceName string, ch chan<- provider.ContainerMetrics) error {
	// Send initial metrics immediately
	metrics := collectGCPMetrics(ctx, client, project, zone, instanceName)

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

	ticker := time.NewTicker(metricsPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			metrics := collectGCPMetrics(ctx, client, project, zone, instanceName)

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
