package docker

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rickihastings/spinner/internal/agent"
)

// LogWatcher monitors container log files and streams parsed entries
type LogWatcher struct {
	containerName string
	logsDir       string
	parser        agent.LineParser
}

// NewLogWatcher creates a new LogWatcher for the given container with an explicit LineParser.
// This keeps the docker package free of agent-implementation imports.
func NewLogWatcher(containerName string, parser agent.LineParser) (*LogWatcher, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	logsDir := filepath.Join(homeDir, ".spinner", containerName, "logs")
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("logs directory does not exist: %s", logsDir)
	}

	return &LogWatcher{
		containerName: containerName,
		logsDir:       logsDir,
		parser:        parser,
	}, nil
}

// TailExistingLines reads the last N lines from the log file and returns them as raw strings.
func (lw *LogWatcher) TailExistingLines(_ context.Context, numLines int) ([]string, error) {
	logFilePath := filepath.Join(lw.logsDir, "raw.log")

	file, err := os.Open(logFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No logs yet
		}

		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	defer func() {
		_ = file.Close()
	}()

	// Ring buffer: keeps only the last numLines lines without reading
	// the entire file into memory. Write index wraps around.
	ring := make([]string, numLines)
	ringIdx := 0
	ringCount := 0

	scanner := bufio.NewScanner(file)
	// Increase buffer for large JSON lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 1MB max

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		ring[ringIdx] = line

		ringIdx = (ringIdx + 1) % numLines
		if ringCount < numLines {
			ringCount++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	// Reconstruct the last N lines in order from the ring buffer.
	var lines []string

	start := 0
	if ringCount == numLines {
		start = ringIdx
	}

	for i := 0; i < ringCount; i++ {
		line := ring[(start+i)%numLines]
		lines = append(lines, line)
	}

	return lines, nil
}

// TailExistingLogs reads the last N lines from the log file and parses them into events.
func (lw *LogWatcher) TailExistingLogs(_ context.Context, numLines int) ([]agent.Event, error) {
	lines, err := lw.TailExistingLines(context.TODO(), numLines)
	if err != nil {
		return nil, err
	}

	var events []agent.Event

	for _, line := range lines {
		event := lw.parser.ParseLine(line)
		if event != nil {
			events = append(events, *event)
		}
	}

	return events, nil
}

// WatchLines watches for new log entries and streams raw lines to the provided channel.
func (lw *LogWatcher) WatchLines(ctx context.Context, lineCh chan<- string) error {
	logFilePath := filepath.Join(lw.logsDir, "raw.log")

	// Wait for log file to exist if it doesn't yet
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		// Poll for file creation with timeout
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		timeout := time.After(30 * time.Second)

		for {
			select {
			case <-ticker.C:
				if _, err := os.Stat(logFilePath); err == nil {
					goto FileExists
				}
			case <-timeout:
				return fmt.Errorf("timeout waiting for log file to be created")
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

FileExists:
	// Open the file for reading
	file, err := os.Open(logFilePath)

	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	defer func() {
		_ = file.Close()
	}()

	// Seek to end of file to only watch new entries
	if _, err := file.Seek(0, 2); err != nil {
		return fmt.Errorf("failed to seek to end of file: %w", err)
	}

	// Create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	defer func() {
		_ = watcher.Close()
	}()

	// Add log file to watcher
	if err := watcher.Add(logFilePath); err != nil {
		return fmt.Errorf("failed to watch log file: %w", err)
	}

	// Create scanner for reading new lines with larger buffer for JSON
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 1MB max

	for {
		select {
		case fsEvent, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Only process write events
			if fsEvent.Op&fsnotify.Write == fsnotify.Write {
				// Read new lines
				for scanner.Scan() {
					line := scanner.Text()
					if line == "" {
						continue
					}

					select {
					case lineCh <- line:
					case <-ctx.Done():
						return ctx.Err()
					}
				}

				if err := scanner.Err(); err != nil {
					return fmt.Errorf("error reading log file: %w", err)
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}

			return fmt.Errorf("file watcher error: %w", err)

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// WatchLogs watches for new log entries and streams parsed events to the provided channel.
func (lw *LogWatcher) WatchLogs(ctx context.Context, logCh chan<- agent.Event) error {
	// Create a channel for raw lines
	lineCh := make(chan string, 100)

	// Start WatchLines in a goroutine
	go func() {
		defer close(lineCh)

		_ = lw.WatchLines(ctx, lineCh)
	}()

	// Parse lines and send events
	for line := range lineCh {
		event := lw.parser.ParseLine(line)
		if event != nil {
			select {
			case logCh <- *event:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return nil
}
