package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rickihastings/spinner/internal/agent"
	"github.com/rickihastings/spinner/internal/agent/claude"
	"github.com/rickihastings/spinner/internal/provider"
	"github.com/rickihastings/spinner/internal/tui"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: go run ./tools/tuipreview <path-to-raw.log>\n")
		os.Exit(1)
	}

	logPath := os.Args[1]

	file, err := os.Open(logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	containerName := extractContainerName(logPath)

	parser := claude.NewParser()
	formatter := claude.NewFormatter()

	logCh := make(chan agent.Event, 100)
	metricsCh := make(chan provider.ContainerMetrics)

	wctx := tui.WatchContext{
		Agent:         "claude",
		Environment:   "preview",
		HeaderVisible: true,
	}

	ui := tui.NewWatchUI(containerName, formatter, wctx)

	// Feed log lines in a goroutine
	go func() {
		defer close(logCh)
		defer close(metricsCh)

		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if event := parser.ParseLine(line); event != nil {
				select {
				case logCh <- *event:
				case <-ui.Context().Done():
					return
				}
			}
		}
	}()

	if err := ui.Run(logCh, metricsCh); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// extractContainerName derives a display name from the log file path.
// For paths like ~/.spinner/<name>/logs/raw.log, extracts <name>.
func extractContainerName(logPath string) string {
	absPath, err := filepath.Abs(logPath)
	if err != nil {
		return "preview"
	}

	parts := strings.Split(absPath, string(filepath.Separator))
	for i, part := range parts {
		if part == ".spinner" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	return filepath.Base(filepath.Dir(filepath.Dir(logPath)))
}
