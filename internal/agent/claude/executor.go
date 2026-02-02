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
}

// Executor executes Claude CLI commands and implements agent.Executor.
type Executor struct {
	config *ExecutorConfig
	parser *Parser

	// CompletionSignal is the text that indicates task completion.
	CompletionSignal string

	// CommandArgs are additional arguments to pass to the Claude CLI.
	CommandArgs []string
}

// NewExecutor creates a new Claude Executor with the given configuration.
func NewExecutor(config *ExecutorConfig) *Executor {
	if config == nil {
		config = &ExecutorConfig{}
	}

	parser := NewParser()
	parser.IncludeRaw = config.IncludeRaw

	return &Executor{
		config:           config,
		parser:           parser,
		CompletionSignal: "~~ FEATURE_COMPLETED ~~",
		CommandArgs:      []string{},
	}
}

// Execute runs Claude CLI with the given prompt and returns a channel of events.
func (e *Executor) Execute(ctx context.Context, prompt string) (<-chan agent.Event, error) {
	args := append([]string{
		"-p", prompt,
		"--dangerously-skip-permissions",
		"--output-format=stream-json",
		"--verbose",
	}, e.CommandArgs...)

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

		// Create a reader that also writes to log file if configured
		var reader io.Reader = stdout
		if logFile != nil {
			reader = io.TeeReader(stdout, logFile)
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
					Type: EventTypeError,
					Data: ErrorData{
						Type:    "command_error",
						Message: err.Error(),
					},
				}
			}
		}
	}()

	return events, nil
}

// ExecuteAndCollect runs Claude CLI and collects results into an ExecutionResult.
// This provides a simpler interface when you don't need to process events in real-time.
func (e *Executor) ExecuteAndCollect(ctx context.Context, prompt string) (*ExecutionResult, error) {
	events, err := e.Execute(ctx, prompt)
	if err != nil {
		return nil, err
	}

	result := &ExecutionResult{}

	for event := range events {
		result.TotalEvents++

		switch event.Type {
		case EventTypeError:
			data, ok := event.Data.(ErrorData)
			if !ok {
				continue
			}

			result.ErrorMessage = data.Message

			if IsRateLimitError(&event) {
				result.RateLimited = true
			} else if IsAuthError(&event) {
				result.AuthError = true
			} else if data.Type != "command_error" {
				result.Error = fmt.Errorf("claude error: %s", data.Message)
			}

		case EventTypeAssistantMessage:
			if ContainsText(&event, e.CompletionSignal) {
				result.Completed = true
			}
		}
	}

	return result, nil
}

// EventCollector collects events from a channel for later processing.
type EventCollector struct {
	events []agent.Event
	mu     sync.Mutex
}

// NewEventCollector creates a new EventCollector.
func NewEventCollector() *EventCollector {
	return &EventCollector{
		events: make([]agent.Event, 0),
	}
}

// Collect reads all events from the channel and stores them.
// Returns when the channel is closed.
func (c *EventCollector) Collect(events <-chan agent.Event) {
	for event := range events {
		c.mu.Lock()
		c.events = append(c.events, event)
		c.mu.Unlock()
	}
}

// Events returns all collected events.
func (c *EventCollector) Events() []agent.Event {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([]agent.Event, len(c.events))
	copy(result, c.events)

	return result
}

// Filter returns events matching the given types.
func (c *EventCollector) Filter(types ...string) []agent.Event {
	c.mu.Lock()
	defer c.mu.Unlock()

	typeSet := make(map[string]bool)

	for _, t := range types {
		typeSet[t] = true
	}

	var result []agent.Event

	for _, event := range c.events {
		if typeSet[event.Type] {
			result = append(result, event)
		}
	}

	return result
}

// HasError returns true if any error events were collected.
func (c *EventCollector) HasError() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, event := range c.events {
		if event.Type == EventTypeError {
			return true
		}
	}

	return false
}

// GetAllText returns all text content from Claude assistant messages.
func (c *EventCollector) GetAllText() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	var texts []string

	for _, event := range c.events {
		if event.Type == EventTypeAssistantMessage {
			text := ExtractText(&event)
			if text != "" {
				texts = append(texts, text)
			}
		}
	}

	return strings.Join(texts, "\n")
}

// GetAllToolUses returns all tool uses from Claude assistant messages.
func (c *EventCollector) GetAllToolUses() []ToolUseData {
	c.mu.Lock()
	defer c.mu.Unlock()

	var tools []ToolUseData

	for _, event := range c.events {
		if event.Type == EventTypeAssistantMessage {
			tools = append(tools, ExtractToolUses(&event)...)
		}
	}

	return tools
}
