package docker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSDKClient(t *testing.T) {
	sdk := newSDKClient()
	assert.NotNil(t, sdk)
	assert.Nil(t, sdk.cli)
}

func TestSDKClient_Close_NilClient(t *testing.T) {
	sdk := newSDKClient()

	// Closing a nil client should not error
	err := sdk.close()
	assert.NoError(t, err)
}

func TestDefaultLogStreamOptions(t *testing.T) {
	opts := DefaultLogStreamOptions()

	assert.False(t, opts.Follow)
	assert.False(t, opts.Timestamps)
	assert.Equal(t, "all", opts.Tail)
	assert.True(t, opts.Stdout)
	assert.True(t, opts.Stderr)
}

func TestBuildEvent_IsError(t *testing.T) {
	tests := []struct {
		name     string
		event    BuildEvent
		expected bool
	}{
		{
			name:     "no error",
			event:    BuildEvent{Stream: "Step 1/3 : FROM ubuntu"},
			expected: false,
		},
		{
			name:     "error string set",
			event:    BuildEvent{Error: "build failed"},
			expected: true,
		},
		{
			name:     "error detail set",
			event:    BuildEvent{ErrorDetail: &BuildErrorDetail{Code: 1, Message: "failed"}},
			expected: true,
		},
		{
			name:     "both error and detail set",
			event:    BuildEvent{Error: "error", ErrorDetail: &BuildErrorDetail{Code: 1}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.event.IsError())
		})
	}
}

func TestLogEvent_Fields(t *testing.T) {
	event := LogEvent{
		Stream:  "stdout",
		Message: "test message",
	}

	assert.Equal(t, "stdout", event.Stream)
	assert.Equal(t, "test message", event.Message)
	assert.Nil(t, event.Error)
}

func TestLogEvent_WithError(t *testing.T) {
	testErr := errors.New("stream error")
	event := LogEvent{
		Timestamp: time.Now(),
		Error:     testErr,
	}

	assert.NotNil(t, event.Error)
	assert.Equal(t, "stream error", event.Error.Error())
}

func TestLogEvent_WithTimestamp(t *testing.T) {
	now := time.Now()
	event := LogEvent{
		Timestamp: now,
		Stream:    "stderr",
		Message:   "error output",
	}

	assert.Equal(t, now, event.Timestamp)
	assert.Equal(t, "stderr", event.Stream)
	assert.Equal(t, "error output", event.Message)
}

func TestStreamContainerLogs_MockClient(t *testing.T) {
	mockClient := new(MockDockerClient)
	ctx := context.Background()

	// Create a channel with some test events
	events := make(chan LogEvent, 3)
	events <- LogEvent{Stream: "stdout", Message: "line 1\n"}

	events <- LogEvent{Stream: "stdout", Message: "line 2\n"}

	events <- LogEvent{Stream: "stderr", Message: "error line\n"}

	close(events)

	opts := DefaultLogStreamOptions()
	mockClient.On("StreamContainerLogs", ctx, "test-container", opts).Return((<-chan LogEvent)(events), nil)

	resultChan, err := mockClient.StreamContainerLogs(ctx, "test-container", opts)
	assert.NoError(t, err)
	assert.NotNil(t, resultChan)

	// Collect all events
	var receivedEvents []LogEvent
	for event := range resultChan {
		receivedEvents = append(receivedEvents, event)
	}

	assert.Len(t, receivedEvents, 3)
	assert.Equal(t, "stdout", receivedEvents[0].Stream)
	assert.Equal(t, "line 1\n", receivedEvents[0].Message)
	assert.Equal(t, "stderr", receivedEvents[2].Stream)

	mockClient.AssertExpectations(t)
}

func TestStreamContainerLogs_MockClient_Error(t *testing.T) {
	mockClient := new(MockDockerClient)
	ctx := context.Background()

	opts := LogStreamOptions{Follow: true, Stdout: true, Stderr: true}
	expectedErr := errors.New("container not found")
	mockClient.On("StreamContainerLogs", ctx, "nonexistent", opts).Return(nil, expectedErr)

	resultChan, err := mockClient.StreamContainerLogs(ctx, "nonexistent", opts)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, resultChan)

	mockClient.AssertExpectations(t)
}

func TestStreamContainerLogs_MockClient_WithFollow(t *testing.T) {
	mockClient := new(MockDockerClient)
	ctx := context.Background()

	events := make(chan LogEvent, 1)
	events <- LogEvent{Stream: "stdout", Message: "streaming output\n"}

	close(events)

	opts := LogStreamOptions{
		Follow:     true,
		Stdout:     true,
		Stderr:     true,
		Timestamps: true,
		Tail:       "100",
	}
	mockClient.On("StreamContainerLogs", ctx, "streaming-container", opts).Return((<-chan LogEvent)(events), nil)

	resultChan, err := mockClient.StreamContainerLogs(ctx, "streaming-container", opts)
	assert.NoError(t, err)

	event := <-resultChan
	assert.Equal(t, "stdout", event.Stream)
	assert.Equal(t, "streaming output\n", event.Message)

	mockClient.AssertExpectations(t)
}

func TestLogStreamOptions_CustomValues(t *testing.T) {
	opts := LogStreamOptions{
		Follow:     true,
		Timestamps: true,
		Tail:       "50",
		Since:      "2024-01-01T00:00:00Z",
		Until:      "2024-12-31T23:59:59Z",
		Stdout:     true,
		Stderr:     false,
	}

	assert.True(t, opts.Follow)
	assert.True(t, opts.Timestamps)
	assert.Equal(t, "50", opts.Tail)
	assert.Equal(t, "2024-01-01T00:00:00Z", opts.Since)
	assert.Equal(t, "2024-12-31T23:59:59Z", opts.Until)
	assert.True(t, opts.Stdout)
	assert.False(t, opts.Stderr)
}

// Ensure MockDockerClient implements Client interface
var _ Client = (*MockDockerClient)(nil)
