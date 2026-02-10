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

	lw := &logWatcher{
		containerName: "test-container",
		logsDir:       logsDir,
		parser:        mock,
	}

	events, err := lw.tailExistingLogs(context.Background(), 10)
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

	lw := &logWatcher{
		containerName: "test-container",
		logsDir:       logsDir,
		parser:        mock,
	}

	ctx := context.Background()
	events, err := lw.tailExistingLogs(ctx, 10)

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

	lw := &logWatcher{
		containerName: "test-container",
		logsDir:       logsDir,
		parser:        mock,
	}

	ctx := context.Background()
	events, err := lw.tailExistingLogs(ctx, 10)

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

	lw := &logWatcher{
		containerName: "test-container",
		logsDir:       logsDir,
		parser:        mock,
	}

	ctx := context.Background()
	events, err := lw.tailExistingLogs(ctx, 100)
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

	// Write 10 lines, request only last 2. Head lines (first 5) should also be included.
	allLines := []string{
		"line-0", "line-1", "line-2", "line-3", "line-4",
		"line-5", "line-6", "line-7", "line-8", "line-9",
	}
	content := strings.Join(allLines, "\n") + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(logsDir, "raw.log"), []byte(content), 0644))

	mock := &mockLineParser{
		responses: map[string]*agent.Event{
			"line-0": {Type: "event-0", Timestamp: time.Now()},
			"line-1": {Type: "event-1", Timestamp: time.Now()},
			"line-2": {Type: "event-2", Timestamp: time.Now()},
			"line-3": {Type: "event-3", Timestamp: time.Now()},
			"line-4": {Type: "event-4", Timestamp: time.Now()},
			"line-8": {Type: "event-8", Timestamp: time.Now()},
			"line-9": {Type: "event-9", Timestamp: time.Now()},
		},
	}

	lw := &logWatcher{
		containerName: "test",
		logsDir:       logsDir,
		parser:        mock,
	}

	events, err := lw.tailExistingLogs(context.Background(), 2)
	require.NoError(t, err)

	// Should include head lines (first 5) + tail lines (last 2) = 7 events
	assert.Len(t, events, 7)
	assert.Equal(t, "event-0", events[0].Type)
	assert.Equal(t, "event-4", events[4].Type)
	assert.Equal(t, "event-8", events[5].Type)
	assert.Equal(t, "event-9", events[6].Type)
}

func TestWatchLines_ResetsOnTruncation(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")
	require.NoError(t, os.MkdirAll(logsDir, 0755))

	logFilePath := filepath.Join(logsDir, "raw.log")

	// Write initial content so the file exists and has data
	require.NoError(t, os.WriteFile(logFilePath, []byte("old-line-1\nold-line-2\n"), 0644))

	lw := &logWatcher{
		containerName: "test",
		logsDir:       logsDir,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lineCh := make(chan string, 100)

	// Start watchLines in background
	errCh := make(chan error, 1)

	go func() {
		errCh <- lw.watchLines(ctx, lineCh)
	}()

	// Give watcher time to start and seek to end
	time.Sleep(100 * time.Millisecond)

	// Append a line — watcher should see it
	f, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_APPEND, 0644)
	require.NoError(t, err)
	_, err = f.WriteString("pre-truncate\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	select {
	case line := <-lineCh:
		assert.Equal(t, "pre-truncate", line)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for pre-truncate line")
	}

	// Truncate and write new content (simulates new iteration with os.Create)
	f, err = os.Create(logFilePath)
	require.NoError(t, err)
	_, err = f.WriteString("post-truncate\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	// Watcher should detect truncation and read from start
	select {
	case line := <-lineCh:
		assert.Equal(t, "post-truncate", line)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for post-truncate line — watcher stuck after truncation")
	}

	cancel()
}

func TestTailExistingLines_HeadOverlapWithTail(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")
	require.NoError(t, os.MkdirAll(logsDir, 0755))

	// Write 4 lines, request last 3. Head (first 5) fully overlaps with tail,
	// so only tail should be returned (no duplicates).
	allLines := []string{"line-a", "line-b", "line-c", "line-d"}
	content := strings.Join(allLines, "\n") + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(logsDir, "raw.log"), []byte(content), 0644))

	lw := &logWatcher{
		containerName: "test",
		logsDir:       logsDir,
	}

	lines, err := lw.tailExistingLines(context.Background(), 3)
	require.NoError(t, err)

	// Total lines (4) <= numLines (3) is false, but head (4 lines) all overlap with tail start (4-3=1).
	// Head lines at index 0 don't overlap (0 < 1), so we get head[0] + tail.
	assert.Equal(t, []string{"line-a", "line-b", "line-c", "line-d"}, lines)
}
