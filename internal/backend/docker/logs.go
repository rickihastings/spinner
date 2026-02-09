package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rickihastings/spinner/internal/agent"
)

// logWatcher monitors container log files and streams parsed entries
type logWatcher struct {
	containerName string
	logsDir       string
	parser        agent.LineParser
}

// newLogWatcher creates a new logWatcher for the given container with an explicit LineParser.
// This keeps the docker package free of agent-implementation imports.
func newLogWatcher(containerName string, parser agent.LineParser) (*logWatcher, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	logsDir := filepath.Join(homeDir, ".spinner", containerName, "logs")
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("logs directory does not exist: %s", logsDir)
	}

	return &logWatcher{
		containerName: containerName,
		logsDir:       logsDir,
		parser:        parser,
	}, nil
}

// headLines is the number of lines captured from the start of the log file.
// These are always included alongside the tail to ensure early events like
// system_init (which contains the model name) are not lost.
const headLines = 5

// tailExistingLines reads the last N lines from the log file and returns them as raw strings.
// It also always includes the first few lines of the file so that early events
// (e.g. system_init) are captured even for long-running containers.
func (lw *logWatcher) tailExistingLines(_ context.Context, numLines int) ([]string, error) {
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

	// Also capture the first few lines for early events like system_init
	head := make([]string, 0, headLines)
	totalLines := 0

	scanner := bufio.NewScanner(file)
	// Increase buffer for large JSON lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 1MB max

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Capture head lines
		if totalLines < headLines {
			head = append(head, line)
		}

		totalLines++

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
	var tail []string

	start := 0
	if ringCount == numLines {
		start = ringIdx
	}

	for i := 0; i < ringCount; i++ {
		line := ring[(start+i)%numLines]
		tail = append(tail, line)
	}

	// If the file is short enough that head overlaps with tail, just return tail
	if totalLines <= numLines {
		return tail, nil
	}

	// Prepend head lines that aren't already in the tail window
	// Head lines are from positions 0..len(head)-1, tail starts at totalLines-numLines.
	// If head count < totalLines-numLines, none overlap.
	tailStart := totalLines - numLines

	var lines []string

	for i, line := range head {
		if i < tailStart {
			lines = append(lines, line)
		}
	}

	lines = append(lines, tail...)

	return lines, nil
}

// tailExistingLogs reads the last N lines from the log file and parses them into events.
func (lw *logWatcher) tailExistingLogs(_ context.Context, numLines int) ([]agent.Event, error) {
	lines, err := lw.tailExistingLines(context.TODO(), numLines)
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

// watchLines watches for new log entries and streams raw lines to the provided channel.
func (lw *logWatcher) watchLines(ctx context.Context, lineCh chan<- string) error {
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

	// Use bufio.Reader instead of bufio.Scanner to avoid the EOF-stuck bug:
	// bufio.Scanner sets an internal error state on io.EOF and never retries,
	// so new data appended to the file after EOF is never read.
	// bufio.Reader does not have this problem — ReadString will read new data
	// on subsequent calls even after returning io.EOF.
	reader := bufio.NewReaderSize(file, 1024*1024) // 1MB buffer for large JSON lines

	for {
		select {
		case fsEvent, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Only process write events
			if fsEvent.Op&fsnotify.Write == fsnotify.Write {
				// Read all available new lines
				for {
					line, err := reader.ReadString('\n')
					if line != "" {
						trimmed := strings.TrimRight(line, "\n\r")
						if trimmed != "" {
							select {
							case lineCh <- trimmed:
							case <-ctx.Done():
								return ctx.Err()
							}
						}
					}

					if err != nil {
						if err == io.EOF {
							break // No more data right now, wait for next write event
						}

						return fmt.Errorf("error reading log file: %w", err)
					}
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
