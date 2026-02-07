package gcp

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/rickihastings/spinner/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetVMState_Running(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	running := "RUNNING"
	client.On("GetInstance", ctx, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &running}, nil)

	state, err := getVMState(ctx, client, "proj", "zone", "vm-1")
	assert.NoError(t, err)
	assert.Equal(t, provider.StateRunning, state)
	client.AssertExpectations(t)
}

func TestGetVMState_Terminated(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	terminated := "TERMINATED"
	client.On("GetInstance", ctx, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &terminated}, nil)

	state, err := getVMState(ctx, client, "proj", "zone", "vm-1")
	assert.NoError(t, err)
	assert.Equal(t, provider.StateStopped, state)
	client.AssertExpectations(t)
}

func TestGetVMState_Stopped(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	stopped := "STOPPED"
	client.On("GetInstance", ctx, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &stopped}, nil)

	state, err := getVMState(ctx, client, "proj", "zone", "vm-1")
	assert.NoError(t, err)
	assert.Equal(t, provider.StateStopped, state)
	client.AssertExpectations(t)
}

func TestGetVMState_Provisioning(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	provisioning := "PROVISIONING"
	client.On("GetInstance", ctx, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &provisioning}, nil)

	state, err := getVMState(ctx, client, "proj", "zone", "vm-1")
	assert.NoError(t, err)
	assert.Equal(t, provider.StateRunning, state)
	client.AssertExpectations(t)
}

func TestGetVMState_Staging(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	staging := "STAGING"
	client.On("GetInstance", ctx, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &staging}, nil)

	state, err := getVMState(ctx, client, "proj", "zone", "vm-1")
	assert.NoError(t, err)
	assert.Equal(t, provider.StateRunning, state)
	client.AssertExpectations(t)
}

func TestGetVMState_Stopping(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	stopping := "STOPPING"
	client.On("GetInstance", ctx, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &stopping}, nil)

	state, err := getVMState(ctx, client, "proj", "zone", "vm-1")
	assert.NoError(t, err)
	assert.Equal(t, provider.StateStopped, state)
	client.AssertExpectations(t)
}

func TestGetVMState_NotFound(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	client.On("GetInstance", ctx, "proj", "zone", "vm-1").
		Return(nil, fmt.Errorf("googleapi: Error 404: The resource was not found"))

	state, err := getVMState(ctx, client, "proj", "zone", "vm-1")
	assert.NoError(t, err)
	assert.Equal(t, provider.StateExited, state)
	client.AssertExpectations(t)
}

func TestGetVMState_APIError(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	client.On("GetInstance", ctx, "proj", "zone", "vm-1").
		Return(nil, fmt.Errorf("permission denied"))

	state, err := getVMState(ctx, client, "proj", "zone", "vm-1")
	assert.Error(t, err)
	assert.Equal(t, provider.StateUnknown, state)
	client.AssertExpectations(t)
}

func TestGetVMState_UnknownStatus(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	unknown := "REPAIRING"
	client.On("GetInstance", ctx, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &unknown}, nil)

	state, err := getVMState(ctx, client, "proj", "zone", "vm-1")
	assert.NoError(t, err)
	assert.Equal(t, provider.StateUnknown, state)
	client.AssertExpectations(t)
}

func TestQueryCPUUtilization_WithData(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	client.On("QueryTimeSeries", ctx, "proj", MetricsQuery{
		MetricType:      cpuMetricType,
		InstanceName:    "vm-1",
		Zone:            "zone",
		IntervalSeconds: metricsQueryIntervalSeconds,
	}).Return([]MetricPoint{
		{Value: 0.25},
		{Value: 0.50},
	}, nil)

	cpuPercent, err := queryCPUUtilization(ctx, client, "proj", "zone", "vm-1")
	assert.NoError(t, err)
	// Last point (0.50) scaled to 50.0%
	assert.InDelta(t, 50.0, cpuPercent, 0.01)
	client.AssertExpectations(t)
}

func TestQueryCPUUtilization_NoData(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	client.On("QueryTimeSeries", ctx, "proj", MetricsQuery{
		MetricType:      cpuMetricType,
		InstanceName:    "vm-1",
		Zone:            "zone",
		IntervalSeconds: metricsQueryIntervalSeconds,
	}).Return([]MetricPoint{}, nil)

	cpuPercent, err := queryCPUUtilization(ctx, client, "proj", "zone", "vm-1")
	assert.NoError(t, err)
	assert.Equal(t, 0.0, cpuPercent)
	client.AssertExpectations(t)
}

func TestQueryCPUUtilization_Error(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	client.On("QueryTimeSeries", ctx, "proj", MetricsQuery{
		MetricType:      cpuMetricType,
		InstanceName:    "vm-1",
		Zone:            "zone",
		IntervalSeconds: metricsQueryIntervalSeconds,
	}).Return(nil, fmt.Errorf("monitoring API error"))

	cpuPercent, err := queryCPUUtilization(ctx, client, "proj", "zone", "vm-1")
	assert.Error(t, err)
	assert.Equal(t, 0.0, cpuPercent)
	client.AssertExpectations(t)
}

func TestCollectGCPMetrics_RunningWithCPU(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	running := "RUNNING"
	client.On("GetInstance", ctx, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &running}, nil)

	client.On("QueryTimeSeries", ctx, "proj", MetricsQuery{
		MetricType:      cpuMetricType,
		InstanceName:    "vm-1",
		Zone:            "zone",
		IntervalSeconds: metricsQueryIntervalSeconds,
	}).Return([]MetricPoint{{Value: 0.75}}, nil)

	metrics := collectGCPMetrics(ctx, client, "proj", "zone", "vm-1")
	assert.Equal(t, provider.StateRunning, metrics.State)
	assert.InDelta(t, 75.0, metrics.CPUPercent, 0.01)
	assert.NoError(t, metrics.Error)
	client.AssertExpectations(t)
}

func TestCollectGCPMetrics_RunningNoCPUData(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	running := "RUNNING"
	client.On("GetInstance", ctx, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &running}, nil)

	client.On("QueryTimeSeries", ctx, "proj", MetricsQuery{
		MetricType:      cpuMetricType,
		InstanceName:    "vm-1",
		Zone:            "zone",
		IntervalSeconds: metricsQueryIntervalSeconds,
	}).Return([]MetricPoint{}, nil)

	metrics := collectGCPMetrics(ctx, client, "proj", "zone", "vm-1")
	assert.Equal(t, provider.StateRunning, metrics.State)
	assert.Equal(t, 0.0, metrics.CPUPercent)
	assert.NoError(t, metrics.Error)
	client.AssertExpectations(t)
}

func TestCollectGCPMetrics_RunningCPUError(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	running := "RUNNING"
	client.On("GetInstance", ctx, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &running}, nil)

	client.On("QueryTimeSeries", ctx, "proj", MetricsQuery{
		MetricType:      cpuMetricType,
		InstanceName:    "vm-1",
		Zone:            "zone",
		IntervalSeconds: metricsQueryIntervalSeconds,
	}).Return(nil, fmt.Errorf("monitoring unavailable"))

	metrics := collectGCPMetrics(ctx, client, "proj", "zone", "vm-1")
	assert.Equal(t, provider.StateRunning, metrics.State)
	assert.Equal(t, 0.0, metrics.CPUPercent)
	// Error is set but non-fatal (state is Running, not Unknown)
	assert.Error(t, metrics.Error)
	assert.Contains(t, metrics.Error.Error(), "failed to query CPU metrics")
	client.AssertExpectations(t)
}

func TestCollectGCPMetrics_Stopped(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	terminated := "TERMINATED"
	client.On("GetInstance", ctx, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &terminated}, nil)

	// Should not query metrics for stopped VMs
	metrics := collectGCPMetrics(ctx, client, "proj", "zone", "vm-1")
	assert.Equal(t, provider.StateStopped, metrics.State)
	assert.Equal(t, 0.0, metrics.CPUPercent)
	assert.NoError(t, metrics.Error)
	client.AssertExpectations(t)
}

func TestCollectGCPMetrics_VMNotFound(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	client.On("GetInstance", ctx, "proj", "zone", "vm-1").
		Return(nil, fmt.Errorf("googleapi: Error 404: not found"))

	metrics := collectGCPMetrics(ctx, client, "proj", "zone", "vm-1")
	assert.Equal(t, provider.StateExited, metrics.State)
	assert.Equal(t, 0.0, metrics.CPUPercent)
	assert.NoError(t, metrics.Error)
	client.AssertExpectations(t)
}

func TestCollectGCPMetrics_APIError(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	client.On("GetInstance", ctx, "proj", "zone", "vm-1").
		Return(nil, fmt.Errorf("permission denied"))

	metrics := collectGCPMetrics(ctx, client, "proj", "zone", "vm-1")
	assert.Equal(t, provider.StateUnknown, metrics.State)
	assert.Error(t, metrics.Error)
	assert.Contains(t, metrics.Error.Error(), "failed to get VM state")
	client.AssertExpectations(t)
}

func TestStreamGCPMetrics_InitialSendRunning(t *testing.T) {
	client := new(MockGCPClient)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	running := "RUNNING"
	// First call for initial metrics
	client.On("GetInstance", mock.Anything, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &running}, nil).Once()
	client.On("QueryTimeSeries", mock.Anything, "proj", mock.Anything).
		Return([]MetricPoint{{Value: 0.42}}, nil).Once()

	// Second call (ticker fires) — return stopped to exit the loop
	terminated := "TERMINATED"
	client.On("GetInstance", mock.Anything, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &terminated}, nil).Once()

	ch := make(chan provider.ContainerMetrics, 10)
	done := make(chan error, 1)

	go func() {
		done <- streamGCPMetrics(ctx, client, "proj", "zone", "vm-1", ch)
	}()

	// Read initial metrics
	select {
	case m := <-ch:
		assert.Equal(t, provider.StateRunning, m.State)
		assert.InDelta(t, 42.0, m.CPUPercent, 0.01)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for initial metrics")
	}

	// Cancel to stop the loop (don't wait for 60s ticker)
	cancel()

	err := <-done
	assert.ErrorIs(t, err, context.Canceled)
}

func TestStreamGCPMetrics_StoppedVM(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	terminated := "TERMINATED"
	client.On("GetInstance", mock.Anything, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &terminated}, nil)

	ch := make(chan provider.ContainerMetrics, 10)

	err := streamGCPMetrics(ctx, client, "proj", "zone", "vm-1", ch)
	assert.NoError(t, err)

	// Should get one metrics message and then return
	select {
	case m := <-ch:
		assert.Equal(t, provider.StateStopped, m.State)
	default:
		t.Fatal("expected metrics in channel")
	}

	client.AssertExpectations(t)
}

func TestStreamGCPMetrics_ExitedVM(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	client.On("GetInstance", mock.Anything, "proj", "zone", "vm-1").
		Return(nil, fmt.Errorf("googleapi: Error 404: not found"))

	ch := make(chan provider.ContainerMetrics, 10)

	err := streamGCPMetrics(ctx, client, "proj", "zone", "vm-1", ch)
	assert.NoError(t, err)

	select {
	case m := <-ch:
		assert.Equal(t, provider.StateExited, m.State)
	default:
		t.Fatal("expected metrics in channel")
	}

	client.AssertExpectations(t)
}

func TestStreamGCPMetrics_FatalError(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	client.On("GetInstance", mock.Anything, "proj", "zone", "vm-1").
		Return(nil, fmt.Errorf("permission denied"))

	ch := make(chan provider.ContainerMetrics, 10)

	err := streamGCPMetrics(ctx, client, "proj", "zone", "vm-1", ch)
	// Fatal error: unknown state with error should return the error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get VM state")

	// Should still send the error metrics to the channel
	select {
	case m := <-ch:
		assert.Equal(t, provider.StateUnknown, m.State)
		assert.Error(t, m.Error)
	default:
		t.Fatal("expected metrics in channel")
	}

	client.AssertExpectations(t)
}

func TestStreamGCPMetrics_ContextCancellation(t *testing.T) {
	client := new(MockGCPClient)
	ctx, cancel := context.WithCancel(context.Background())

	running := "RUNNING"
	client.On("GetInstance", mock.Anything, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &running}, nil)
	client.On("QueryTimeSeries", mock.Anything, "proj", mock.Anything).
		Return([]MetricPoint{{Value: 0.10}}, nil)

	ch := make(chan provider.ContainerMetrics, 10)
	done := make(chan error, 1)

	go func() {
		done <- streamGCPMetrics(ctx, client, "proj", "zone", "vm-1", ch)
	}()

	// Read initial metrics
	select {
	case m := <-ch:
		assert.Equal(t, provider.StateRunning, m.State)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for initial metrics")
	}

	// Cancel context
	cancel()

	err := <-done
	assert.ErrorIs(t, err, context.Canceled)
}

func TestProviderWatchMetrics_Delegates(t *testing.T) {
	client := new(MockGCPClient)
	p := newTestProvider(client)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// VM is stopped — should send one metrics and return
	terminated := "TERMINATED"
	client.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(&computepb.Instance{Status: &terminated}, nil)

	ch := make(chan provider.ContainerMetrics, 10)

	err := p.WatchMetrics(ctx, "test-vm", ch)
	assert.NoError(t, err)

	select {
	case m := <-ch:
		assert.Equal(t, provider.StateStopped, m.State)
	default:
		t.Fatal("expected metrics in channel")
	}

	client.AssertExpectations(t)
}
