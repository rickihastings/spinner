package claude

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rickihastings/spinner/internal/agent"
)

// completionSignal is the text that indicates task completion.
const completionSignal = "~~ FEATURE_COMPLETED ~~"

// ExecutorConfig contains configuration for creating a Claude executor.
type ExecutorConfig struct {
	// LogPath is the path to write raw log output (optional).
	LogPath string

	// WorkDir is the working directory for the command (optional).
	WorkDir string

	// Env contains additional environment variables (optional).
	Env []string

	// IncludeRaw includes raw JSON lines in events when true.
	IncludeRaw bool

	// AdditionalWriter receives a copy of the raw stdout stream alongside
	// the log file. Used by the GCP backend to stream logs to GCS.
	AdditionalWriter io.Writer
}

// Executor executes Claude CLI commands and implements agent.Executor.
type Executor struct {
	config *ExecutorConfig
	parser *Parser

	// completionSignal is the text that indicates task completion.
	completionSignal string

	// commandArgs are additional arguments to pass to the Claude CLI.
	commandArgs []string
}

// NewExecutor creates a new Claude Executor with the given configuration.
func NewExecutor(config *ExecutorConfig) *Executor {
	if config == nil {
		config = &ExecutorConfig{}
	}

	parser := NewParser()
	parser.includeRaw = config.IncludeRaw

	return &Executor{
		config:           config,
		parser:           parser,
		completionSignal: completionSignal,
		commandArgs:      []string{},
	}
}

// Execute runs Claude CLI with the given prompt and returns a channel of events.
func (e *Executor) Execute(ctx context.Context, prompt string) (<-chan agent.Event, error) {
	args := append([]string{
		"-p", prompt,
		"--dangerously-skip-permissions",
		"--output-format=stream-json",
		"--verbose",
	}, e.commandArgs...)

	cmd := exec.CommandContext(ctx, "claude", args...)

	if e.config.WorkDir != "" {
		cmd.Dir = e.config.WorkDir
	}

	if len(e.config.Env) > 0 {
		cmd.Env = append(os.Environ(), e.config.Env...)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start claude command: %w", err)
	}

	// Create log file if path is provided
	var logFile *os.File

	if e.config.LogPath != "" {
		logDir := filepath.Dir(e.config.LogPath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		logFile, err = os.Create(e.config.LogPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create log file: %w", err)
		}
	}

	events := make(chan agent.Event, 100)

	go func() {
		defer close(events)
		defer func() {
			if logFile != nil {
				_ = logFile.Close()
			}
		}()

		// Create a reader that also writes to log file and/or additional writer
		var reader io.Reader = stdout

		var writers []io.Writer
		if logFile != nil {
			writers = append(writers, logFile)
		}

		if e.config.AdditionalWriter != nil {
			writers = append(writers, e.config.AdditionalWriter)
		}

		if len(writers) > 0 {
			reader = io.TeeReader(stdout, io.MultiWriter(writers...))
		}

		// Parse stdout and forward events
		parserEvents := e.parser.Parse(ctx, reader)
		for event := range parserEvents {
			select {
			case events <- event:
			case <-ctx.Done():
				return
			}
		}

		// Capture stderr to log file
		if logFile != nil {
			stderrScanner := bufio.NewScanner(stderr)
			for stderrScanner.Scan() {
				_, _ = fmt.Fprintln(logFile, stderrScanner.Text())
			}
		} else {
			// Drain stderr even if not logging
			_, _ = io.Copy(io.Discard, stderr)
		}

		// Wait for command to finish
		if err := cmd.Wait(); err != nil {
			// Emit error event if command failed
			if ctx.Err() == nil { // Don't emit error if context was cancelled
				events <- agent.Event{
					Type: eventTypeError,
					Data: errorData{
						Type:    errorTypeCommand,
						Message: err.Error(),
					},
				}
			}
		}
	}()

	return events, nil
}

// ExecuteAndCollect runs Claude CLI and collects results into an agent.Result.
// This implements the agent.Executor interface.
func (e *Executor) ExecuteAndCollect(ctx context.Context, prompt string) (*agent.Result, error) {
	events, err := e.Execute(ctx, prompt)
	if err != nil {
		return nil, err
	}

	result := &agent.Result{}

	for event := range events {
		switch event.Type {
		case eventTypeError:
			data, ok := event.Data.(errorData)
			if !ok {
				continue
			}

			result.ErrorMessage = data.Message

			if isRateLimitError(&event) {
				result.RateLimited = true
			} else if isAuthError(&event) {
				result.AuthError = true
			} else if data.Type != errorTypeCommand {
				result.Error = fmt.Errorf("claude error: %s", data.Message)
			}

		case eventTypeAssistantMessage:
			if containsText(&event, e.completionSignal) {
				result.Completed = true
			}
		}
	}

	return result, nil
}

// eventCollector collects events from a channel for later processing.
type eventCollector struct {
	collected []agent.Event
	mu        sync.Mutex
}

// newEventCollector creates a new eventCollector.
func newEventCollector() *eventCollector {
	return &eventCollector{
		collected: make([]agent.Event, 0),
	}
}

// collect reads all events from the channel and stores them.
// Returns when the channel is closed.
func (c *eventCollector) collect(ch <-chan agent.Event) {
	for event := range ch {
		c.mu.Lock()
		c.collected = append(c.collected, event)
		c.mu.Unlock()
	}
}

// events returns all collected events.
func (c *eventCollector) events() []agent.Event {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([]agent.Event, len(c.collected))
	copy(result, c.collected)

	return result
}

// filter returns events matching the given types.
func (c *eventCollector) filter(types ...string) []agent.Event {
	c.mu.Lock()
	defer c.mu.Unlock()

	typeSet := make(map[string]bool)

	for _, t := range types {
		typeSet[t] = true
	}

	var result []agent.Event

	for _, event := range c.collected {
		if typeSet[event.Type] {
			result = append(result, event)
		}
	}

	return result
}

// hasError returns true if any error events were collected.
func (c *eventCollector) hasError() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, event := range c.collected {
		if event.Type == eventTypeError {
			return true
		}
	}

	return false
}

// getAllText returns all text content from Claude assistant messages.
func (c *eventCollector) getAllText() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	var texts []string

	for _, event := range c.collected {
		if event.Type == eventTypeAssistantMessage {
			text := extractText(&event)
			if text != "" {
				texts = append(texts, text)
			}
		}
	}

	return strings.Join(texts, "\n")
}

// getAllToolUses returns all tool uses from Claude assistant messages.
func (c *eventCollector) getAllToolUses() []toolUseData {
	c.mu.Lock()
	defer c.mu.Unlock()

	var tools []toolUseData

	for _, event := range c.collected {
		if event.Type == eventTypeAssistantMessage {
			tools = append(tools, extractToolUses(&event)...)
		}
	}

	return tools
}
