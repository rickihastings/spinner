package exec

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	CompletionSignal = "~~ FEATURE_COMPLETED ~~"
)

// ClaudeMessage represents a message from Claude's streaming JSON output.
type ClaudeMessage struct {
	Type    string `json:"type"`
	Message struct {
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
	} `json:"message"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// ClaudeResult represents the result of running Claude CLI.
type ClaudeResult struct {
	Completed    bool
	RateLimited  bool
	AuthError    bool
	Error        error
	ErrorMessage string
}

// RunClaude executes the Claude CLI with the given prompt and logs output.
func RunClaude(ctx context.Context, prompt string, logPath string) (*ClaudeResult, error) {
	cmd := exec.CommandContext(ctx, "claude", "-p", prompt, "--dangerously-skip-permissions", "--output-format=stream-json", "--verbose")

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

	if logPath != "" {
		logDir := filepath.Dir(logPath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		logFile, err = os.Create(logPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create log file: %w", err)
		}

		defer func() {
			_ = logFile.Close()
		}()
	}

	result := &ClaudeResult{}

	// Process stdout with JSON streaming
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		// Write to log file
		if logFile != nil {
			_, _ = fmt.Fprintln(logFile, line)
		}

		// Parse JSON message
		var msg ClaudeMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// Skip non-JSON lines
			continue
		}

		// Check for errors
		if msg.Error.Type != "" {
			result.ErrorMessage = msg.Error.Message

			if strings.Contains(msg.Error.Type, "rate_limit") {
				result.RateLimited = true
			} else if strings.Contains(msg.Error.Type, "authentication") {
				result.AuthError = true
			} else {
				result.Error = fmt.Errorf("claude error: %s", msg.Error.Message)
			}
		}

		// Check for completion signal in message content
		for _, content := range msg.Message.Content {
			if content.Type == "text" && strings.Contains(content.Text, CompletionSignal) {
				result.Completed = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading stdout: %w", err)
	}

	// Also capture stderr to log file
	if logFile != nil {
		stderrScanner := bufio.NewScanner(stderr)
		for stderrScanner.Scan() {
			_, _ = fmt.Fprintln(logFile, stderrScanner.Text())
		}
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		// If we already detected a specific error, don't override it
		if result.Error == nil && !result.RateLimited && !result.AuthError {
			result.Error = fmt.Errorf("claude command failed: %w", err)
		}
	}

	return result, nil
}
