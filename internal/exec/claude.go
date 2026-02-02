package exec

import (
	"context"

	"github.com/rickihastings/spinner/internal/agent"
	"github.com/rickihastings/spinner/internal/agent/claude"
)

const (
	CompletionSignal = "~~ FEATURE_COMPLETED ~~"
)

// ClaudeResult represents the result of running Claude CLI.
type ClaudeResult struct {
	Completed    bool
	RateLimited  bool
	AuthError    bool
	Error        error
	ErrorMessage string
}

// RunClaude executes the Claude CLI with the given prompt and logs output.
// It uses the claude.Executor internally for parsing and execution.
func RunClaude(ctx context.Context, prompt string, logPath string) (*ClaudeResult, error) {
	config := &claude.ExecutorConfig{
		LogPath: logPath,
	}

	executor := claude.NewExecutor(config)
	executor.CompletionSignal = CompletionSignal

	execResult, err := executor.ExecuteAndCollect(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return &ClaudeResult{
		Completed:    execResult.Completed,
		RateLimited:  execResult.RateLimited,
		AuthError:    execResult.AuthError,
		Error:        execResult.Error,
		ErrorMessage: execResult.ErrorMessage,
	}, nil
}

// RunClaudeWithEvents executes the Claude CLI and returns a channel of events.
// This allows real-time processing of Claude's output for use cases like
// the watch command or progress monitoring.
func RunClaudeWithEvents(ctx context.Context, prompt string, logPath string) (<-chan agent.Event, error) {
	config := &claude.ExecutorConfig{
		LogPath: logPath,
	}

	executor := claude.NewExecutor(config)
	executor.CompletionSignal = CompletionSignal

	return executor.Execute(ctx, prompt)
}
