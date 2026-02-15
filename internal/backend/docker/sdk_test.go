package docker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultLogStreamOptions(t *testing.T) {
	opts := defaultLogStreamOptions()

	assert.False(t, opts.Follow)
	assert.False(t, opts.Timestamps)
	assert.Equal(t, "all", opts.Tail)
	assert.True(t, opts.Stdout)
	assert.True(t, opts.Stderr)
}

func TestBuildEvent_IsError(t *testing.T) {
	tests := []struct {
		name     string
		event    buildEvent
		expected bool
	}{
		{
			name:     "no error",
			event:    buildEvent{Stream: "Step 1/3 : FROM ubuntu"},
			expected: false,
		},
		{
			name:     "error string set",
			event:    buildEvent{Error: "build failed"},
			expected: true,
		},
		{
			name:     "error detail set",
			event:    buildEvent{ErrorDetail: &buildErrorDetail{Code: 1, Message: "failed"}},
			expected: true,
		},
		{
			name:     "both error and detail set",
			event:    buildEvent{Error: "error", ErrorDetail: &buildErrorDetail{Code: 1}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.event.isError())
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

	opts := defaultLogStreamOptions()
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

func TestContainerListEntry_Fields(t *testing.T) {
	entry := ContainerListEntry{
		ID:     "abc123",
		Names:  []string{"test-container"},
		Image:  "spinner:test",
		State:  "running",
		Labels: map[string]string{"spinner-managed": "true"},
	}

	assert.Equal(t, "abc123", entry.ID)
	assert.Equal(t, []string{"test-container"}, entry.Names)
	assert.Equal(t, "spinner:test", entry.Image)
	assert.Equal(t, "running", entry.State)
	assert.Equal(t, "true", entry.Labels["spinner-managed"])
}

func TestParseContainerListOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []ContainerListEntry
		wantErr  bool
	}{
		{
			name:     "empty output",
			input:    "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			input:    "  \n  ",
			expected: nil,
		},
		{
			name:  "single container",
			input: `{"ID":"abc123","Names":"my-container","Image":"spinner:test","State":"running","Labels":"spinner-managed=true"}`,
			expected: []ContainerListEntry{
				{
					ID:     "abc123",
					Names:  []string{"my-container"},
					Image:  "spinner:test",
					State:  "running",
					Labels: map[string]string{"spinner-managed": "true"},
				},
			},
		},
		{
			name: "multiple containers",
			input: `{"ID":"abc123","Names":"container-1","Image":"spinner:test","State":"running","Labels":"spinner-managed=true"}
{"ID":"def456","Names":"container-2","Image":"spinner:test","State":"exited","Labels":"spinner-managed=true"}`,
			expected: []ContainerListEntry{
				{
					ID:     "abc123",
					Names:  []string{"container-1"},
					Image:  "spinner:test",
					State:  "running",
					Labels: map[string]string{"spinner-managed": "true"},
				},
				{
					ID:     "def456",
					Names:  []string{"container-2"},
					Image:  "spinner:test",
					State:  "exited",
					Labels: map[string]string{"spinner-managed": "true"},
				},
			},
		},
		{
			name:  "multiple labels",
			input: `{"ID":"abc","Names":"c1","Image":"img","State":"running","Labels":"key1=val1,key2=val2"}`,
			expected: []ContainerListEntry{
				{
					ID:     "abc",
					Names:  []string{"c1"},
					Image:  "img",
					State:  "running",
					Labels: map[string]string{"key1": "val1", "key2": "val2"},
				},
			},
		},
		{
			name:  "empty labels",
			input: `{"ID":"abc","Names":"c1","Image":"img","State":"running","Labels":""}`,
			expected: []ContainerListEntry{
				{
					ID:     "abc",
					Names:  []string{"c1"},
					Image:  "img",
					State:  "running",
					Labels: map[string]string{},
				},
			},
		},
		{
			name:    "invalid json",
			input:   `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseContainerListOutput(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestListContainers_MockClient(t *testing.T) {
	mockClient := new(MockDockerClient)
	ctx := context.Background()

	expected := []ContainerListEntry{
		{
			ID:     "abc123",
			Names:  []string{"spinner-test"},
			Image:  "spinner:test",
			State:  "running",
			Labels: map[string]string{"spinner-managed": "true"},
		},
	}

	mockClient.On("ListContainers", ctx, map[string]string{"spinner-managed": "true"}).Return(expected, nil)

	result, err := mockClient.ListContainers(ctx, map[string]string{"spinner-managed": "true"})
	assert.NoError(t, err)
	assert.Equal(t, expected, result)

	mockClient.AssertExpectations(t)
}

func TestListContainers_MockClient_Error(t *testing.T) {
	mockClient := new(MockDockerClient)
	ctx := context.Background()

	expectedErr := errors.New("docker unavailable")
	mockClient.On("ListContainers", ctx, map[string]string{"spinner-managed": "true"}).Return(nil, expectedErr)

	result, err := mockClient.ListContainers(ctx, map[string]string{"spinner-managed": "true"})
	assert.Error(t, err)
	assert.Nil(t, result)

	mockClient.AssertExpectations(t)
}

// Ensure MockDockerClient implements Client interface
var _ Client = (*MockDockerClient)(nil)
