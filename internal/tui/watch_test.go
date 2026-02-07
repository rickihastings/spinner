package tui

import (
	"testing"

	"github.com/rickihastings/spinner/internal/agent"
	"github.com/rickihastings/spinner/internal/provider"
	"github.com/stretchr/testify/assert"
)

// Note: Tests for event formatting have been moved to internal/agent/claude/formatter_test.go
// This file now only tests TUI-specific formatting helpers (metrics, bytes)

func TestMetricsFormatting_InitialState(t *testing.T) {
	// Test that empty metrics don't show invalid state
	m := provider.ContainerMetrics{}

	// When State is empty string (zero value), it should not display as "container"
	assert.NotEqual(t, "container", string(m.State), "Empty state should not be 'container'")
	assert.Equal(t, "", string(m.State), "Empty state should be empty string")
}

func TestMetricsFormatting_ValidStates(t *testing.T) {
	// Test all valid container states
	states := []provider.ContainerState{
		provider.StateRunning,
		provider.StateStopped,
		provider.StateExited,
		provider.StateUnknown,
	}

	for _, state := range states {
		t.Run(string(state), func(t *testing.T) {
			assert.NotEmpty(t, string(state), "State should not be empty")
			assert.NotEqual(t, "container", string(state), "State should not be 'container'")
		})
	}
}

func TestFormatMemoryValue(t *testing.T) {
	tests := []struct {
		bytes    uint64
		expected string
	}{
		{100 * 1024, "[cyan]100.00 KB[-]"},
		{512 * 1024 * 1024, "[cyan]512.00 MB[-]"},
		{2 * 1024 * 1024 * 1024, "[cyan]2.00 GB[-]"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatMemoryValue(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWatchUI_TestMode(t *testing.T) {
	// Test that watch UI doesn't block when running in test mode
	logCh := make(chan agent.Event, 1)
	metricsCh := make(chan provider.ContainerMetrics, 1)

	// Create a watch UI instance with minimal context
	ui := NewWatchUI("test-container", &mockFormatter{}, WatchContext{})

	// Enable test mode to prevent TUI startup
	ui.setTestMode(true)

	// Close channels immediately to simulate end of data
	close(logCh)
	close(metricsCh)

	// This should not block in test mode
	err := ui.Run(logCh, metricsCh)

	// Verify it completed successfully
	assert.NoError(t, err)
}

// mockFormatter is a simple formatter for testing
type mockFormatter struct{}

func (m *mockFormatter) FormatEvent(_ *agent.Event) (string, bool) {
	return "", false
}
