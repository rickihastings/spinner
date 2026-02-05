package docker

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rickihastings/spinner/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLineParser implements agent.LineParser for testing.
// It records every line it sees and returns events from a preconfigured map.
type mockLineParser struct {
	// responses maps input lines to the events they should produce.
	// If a line is not in the map, ParseLine returns nil (mimics unrecognized input).
	responses map[string]*agent.Event
	// calls records every line passed to ParseLine, in order.
	calls []string
}

func (m *mockLineParser) ParseLine(line string) *agent.Event {
	m.calls = append(m.calls, line)
	if ev, ok := m.responses[line]; ok {
		return ev
	}

	return nil
}

func TestTailExistingLogs_ParsesLines(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")
	require.NoError(t, os.MkdirAll(logsDir, 0755))

	line1 := `{"type":"system","subtype":"init"}`
	line2 := `{"type":"assistant","message":{}}`
	line3 := `{"type":"result","subtype":"success"}`

	logContent := line1 + "\n" + line2 + "\n" + line3 + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(logsDir, "raw.log"), []byte(logContent), 0644))

	mock := &mockLineParser{
		responses: map[string]*agent.Event{
			line1: {Type: "system_init", Timestamp: time.Now()},
			line2: {Type: "assistant_message", Timestamp: time.Now()},
			line3: {Type: "result", Timestamp: time.Now()},
		},
	}

	lw := &LogWatcher{
		containerName: "test-container",
		logsDir:       logsDir,
		parser:        mock,
	}

	events, err := lw.TailExistingLogs(context.Background(), 10)
	require.NoError(t, err)
	assert.Len(t, events, 3)
	assert.Equal(t, "system_init", events[0].Type)
	assert.Equal(t, "assistant_message", events[1].Type)
	assert.Equal(t, "result", events[2].Type)
	// Verify all three lines were passed to the parser
	assert.Equal(t, []string{line1, line2, line3}, mock.calls)
}

func TestTailExistingLogs_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")
	err := os.MkdirAll(logsDir, 0755)
	require.NoError(t, err)

	mock := &mockLineParser{
		responses: map[string]*agent.Event{},
	}

	lw := &LogWatcher{
		containerName: "test-container",
		logsDir:       logsDir,
		parser:        mock,
	}

	ctx := context.Background()
	events, err := lw.TailExistingLogs(ctx, 10)

	// Should return nil events and no error when file doesn't exist
	assert.NoError(t, err)
	assert.Nil(t, events)
	// Parser should not have been called
	assert.Empty(t, mock.calls)
}

func TestTailExistingLogs_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")
	err := os.MkdirAll(logsDir, 0755)
	require.NoError(t, err)

	// Create an empty log file
	logFile := filepath.Join(logsDir, "raw.log")
	err = os.WriteFile(logFile, []byte(""), 0644)
	require.NoError(t, err)

	mock := &mockLineParser{
		responses: map[string]*agent.Event{},
	}

	lw := &LogWatcher{
		containerName: "test-container",
		logsDir:       logsDir,
		parser:        mock,
	}

	ctx := context.Background()
	events, err := lw.TailExistingLogs(ctx, 10)

	require.NoError(t, err)
	assert.Empty(t, events, "Should return empty events for empty file")
	// Parser should not have been called (no lines)
	assert.Empty(t, mock.calls)
}

func TestTailExistingLogs_MixedContent(t *testing.T) {
	// Test with mixed JSON and non-parseable content
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")
	err := os.MkdirAll(logsDir, 0755)
	require.NoError(t, err)

	line1 := `{"type":"system","subtype":"init"}`
	line2 := "This is plain text that won't parse"
	line3 := `{"type":"assistant","message":{}}`
	line4 := "Another plain line"
	line5 := `{"type":"result","is_error":false}`

	// Some lines will parse, some won't
	logContent := line1 + "\n" + line2 + "\n" + line3 + "\n" + line4 + "\n" + line5 + "\n"
	logFile := filepath.Join(logsDir, "raw.log")
	err = os.WriteFile(logFile, []byte(logContent), 0644)
	require.NoError(t, err)

	mock := &mockLineParser{
		responses: map[string]*agent.Event{
			line1: {Type: "system_init", Timestamp: time.Now()},
			line3: {Type: "assistant_message", Timestamp: time.Now()},
			line5: {Type: "result", Timestamp: time.Now()},
			// line2 and line4 are not in the map, so they return nil
		},
	}

	lw := &LogWatcher{
		containerName: "test-container",
		logsDir:       logsDir,
		parser:        mock,
	}

	ctx := context.Background()
	events, err := lw.TailExistingLogs(ctx, 100)
	require.NoError(t, err)

	// Should only return the 3 parseable events, not the plain text lines
	assert.Len(t, events, 3, "Should return only parseable events")
	assert.Equal(t, "system_init", events[0].Type)
	assert.Equal(t, "assistant_message", events[1].Type)
	assert.Equal(t, "result", events[2].Type)

	// Verify all 5 lines were passed to the parser
	assert.Equal(t, []string{line1, line2, line3, line4, line5}, mock.calls)
}

func TestTailExistingLogs_RingBufferKeepsLastN(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")
	require.NoError(t, os.MkdirAll(logsDir, 0755))

	// Write 5 lines, request only last 2
	lines := []string{"line-a", "line-b", "line-c", "line-d", "line-e"}
	content := strings.Join(lines, "\n") + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(logsDir, "raw.log"), []byte(content), 0644))

	mock := &mockLineParser{
		responses: map[string]*agent.Event{
			"line-d": {Type: "event-d", Timestamp: time.Now()},
			"line-e": {Type: "event-e", Timestamp: time.Now()},
		},
	}

	lw := &LogWatcher{
		containerName: "test",
		logsDir:       logsDir,
		parser:        mock,
	}

	events, err := lw.TailExistingLogs(context.Background(), 2)
	require.NoError(t, err)

	// Only last 2 lines were parsed
	assert.Len(t, events, 2)
	assert.Equal(t, "event-d", events[0].Type)
	assert.Equal(t, "event-e", events[1].Type)
	// Verify the mock only saw the last 2 lines
	assert.Equal(t, []string{"line-d", "line-e"}, mock.calls)
}
