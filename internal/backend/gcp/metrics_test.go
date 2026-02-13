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

func TestCollectGCPMetrics_RunningWithGCSState(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	running := "RUNNING"
	client.On("GetInstance", ctx, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &running}, nil)

	// Mock GCS state file with CPU and memory metrics
	client.On("ObjectExists", ctx, "my-bucket", "vm-1/state.json").
		Return(true, nil)
	client.On("ReadObject", ctx, "my-bucket", "vm-1/state.json").
		Return([]byte(`{"iteration": 5, "status": "running", "cpu_percent": 75.0, "memory_percent": 60.0}`), nil)

	metrics := collectGCPMetrics(ctx, client, "proj", "zone", "vm-1", "my-bucket")
	assert.Equal(t, provider.StateRunning, metrics.State)
	assert.Equal(t, 5, metrics.Iteration)
	assert.InDelta(t, 75.0, metrics.CPUPercent, 0.01)
	assert.InDelta(t, 60.0, metrics.MemoryPercent, 0.01)
	assert.NoError(t, metrics.Error)
	client.AssertExpectations(t)
}

func TestCollectGCPMetrics_RunningNoGCSState(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	running := "RUNNING"
	client.On("GetInstance", ctx, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &running}, nil)

	// No GCS state file yet
	client.On("ObjectExists", ctx, "my-bucket", "vm-1/state.json").
		Return(false, nil)

	metrics := collectGCPMetrics(ctx, client, "proj", "zone", "vm-1", "my-bucket")
	assert.Equal(t, provider.StateRunning, metrics.State)
	assert.Equal(t, 0, metrics.Iteration)
	assert.Equal(t, 0.0, metrics.CPUPercent)
	assert.Equal(t, 0.0, metrics.MemoryPercent)
	assert.NoError(t, metrics.Error)
	client.AssertExpectations(t)
}

func TestCollectGCPMetrics_RunningNoBucket(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	running := "RUNNING"
	client.On("GetInstance", ctx, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &running}, nil)

	metrics := collectGCPMetrics(ctx, client, "proj", "zone", "vm-1", "")
	assert.Equal(t, provider.StateRunning, metrics.State)
	assert.Equal(t, 0, metrics.Iteration)
	assert.Equal(t, 0.0, metrics.CPUPercent)
	assert.Equal(t, 0.0, metrics.MemoryPercent)
	assert.NoError(t, metrics.Error)
	client.AssertExpectations(t)
}

func TestCollectGCPMetrics_Stopped(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	terminated := "TERMINATED"
	client.On("GetInstance", ctx, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &terminated}, nil)

	// Should not read GCS state for stopped VMs
	metrics := collectGCPMetrics(ctx, client, "proj", "zone", "vm-1", "")
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

	metrics := collectGCPMetrics(ctx, client, "proj", "zone", "vm-1", "")
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

	metrics := collectGCPMetrics(ctx, client, "proj", "zone", "vm-1", "")
	assert.Equal(t, provider.StateUnknown, metrics.State)
	assert.Error(t, metrics.Error)
	assert.Contains(t, metrics.Error.Error(), "failed to get VM state")
	client.AssertExpectations(t)
}

func TestReadStateFromGCS_NoBucket(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	state := readStateFromGCS(ctx, client, "", "vm-1")
	assert.Equal(t, gcsState{}, state)
}

func TestReadStateFromGCS_NoState(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	client.On("ObjectExists", ctx, "bucket", "vm-1/state.json").
		Return(false, nil)

	state := readStateFromGCS(ctx, client, "bucket", "vm-1")
	assert.Equal(t, gcsState{}, state)
	client.AssertExpectations(t)
}

func TestReadStateFromGCS_WithMetrics(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	client.On("ObjectExists", ctx, "bucket", "vm-1/state.json").
		Return(true, nil)
	client.On("ReadObject", ctx, "bucket", "vm-1/state.json").
		Return([]byte(`{"iteration": 3, "cpu_percent": 42.5, "memory_percent": 67.8}`), nil)

	state := readStateFromGCS(ctx, client, "bucket", "vm-1")
	assert.Equal(t, 3, state.Iteration)
	assert.InDelta(t, 42.5, state.CPUPercent, 0.01)
	assert.InDelta(t, 67.8, state.MemoryPercent, 0.01)
	client.AssertExpectations(t)
}

func TestReadStateFromGCS_InvalidJSON(t *testing.T) {
	client := new(MockGCPClient)
	ctx := context.Background()

	client.On("ObjectExists", ctx, "bucket", "vm-1/state.json").
		Return(true, nil)
	client.On("ReadObject", ctx, "bucket", "vm-1/state.json").
		Return([]byte(`not json`), nil)

	state := readStateFromGCS(ctx, client, "bucket", "vm-1")
	assert.Equal(t, gcsState{}, state)
	client.AssertExpectations(t)
}

func TestStreamGCPMetrics_InitialSendRunning(t *testing.T) {
	client := new(MockGCPClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	running := "RUNNING"
	client.On("GetInstance", mock.Anything, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &running}, nil).Once()

	// GCS state with metrics
	client.On("ObjectExists", mock.Anything, "my-bucket", "vm-1/state.json").
		Return(true, nil).Once()
	client.On("ReadObject", mock.Anything, "my-bucket", "vm-1/state.json").
		Return([]byte(`{"iteration": 2, "cpu_percent": 42.0, "memory_percent": 55.0}`), nil).Once()

	// Second call (ticker fires) — return stopped to exit the loop
	terminated := "TERMINATED"
	client.On("GetInstance", mock.Anything, "proj", "zone", "vm-1").
		Return(&computepb.Instance{Status: &terminated}, nil).Once()

	ch := make(chan provider.ContainerMetrics, 10)
	done := make(chan error, 1)

	go func() {
		done <- streamGCPMetrics(ctx, client, "proj", "zone", "vm-1", "my-bucket", ch)
	}()

	// Read initial metrics
	select {
	case m := <-ch:
		assert.Equal(t, provider.StateRunning, m.State)
		assert.InDelta(t, 42.0, m.CPUPercent, 0.01)
		assert.InDelta(t, 55.0, m.MemoryPercent, 0.01)
		assert.Equal(t, 2, m.Iteration)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for initial metrics")
	}

	// Cancel to stop the loop (don't wait for ticker)
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

	err := streamGCPMetrics(ctx, client, "proj", "zone", "vm-1", "", ch)
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

	err := streamGCPMetrics(ctx, client, "proj", "zone", "vm-1", "", ch)
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

	err := streamGCPMetrics(ctx, client, "proj", "zone", "vm-1", "", ch)
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
	client.On("ObjectExists", mock.Anything, "my-bucket", "vm-1/state.json").
		Return(true, nil)
	client.On("ReadObject", mock.Anything, "my-bucket", "vm-1/state.json").
		Return([]byte(`{"iteration": 1, "cpu_percent": 10.0, "memory_percent": 30.0}`), nil)

	ch := make(chan provider.ContainerMetrics, 10)
	done := make(chan error, 1)

	go func() {
		done <- streamGCPMetrics(ctx, client, "proj", "zone", "vm-1", "my-bucket", ch)
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
