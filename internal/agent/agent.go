// Package agent provides interfaces for AI agent execution.
package agent

import (
	"context"
	"time"
)

// Event represents a parsed event from an agent's output stream.
// The Type field uses a string to allow each agent implementation
// to define its own event types.
type Event struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Raw       string      `json:"raw,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// Result represents the final result of agent execution.
type Result struct {
	Completed    bool
	RateLimited  bool
	AuthError    bool
	Error        error
	ErrorMessage string
}

// Executor defines the interface for running AI agent commands.
type Executor interface {
	// Execute runs the agent with the given prompt and returns a channel of events.
	// The channel is closed when execution completes or the context is cancelled.
	Execute(ctx context.Context, prompt string) (<-chan Event, error)

	// ExecuteAndCollect runs the agent and collects all events into a final result.
	ExecuteAndCollect(ctx context.Context, prompt string) (*Result, error)
}
