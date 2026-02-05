// Package claude provides Claude CLI execution and log parsing.
package claude

import (
	"encoding/json"
)

// Event type constants for Claude CLI output.
const (
	EventTypeSystemInit       = "system_init"
	EventTypeAssistantMessage = "assistant_message"
	EventTypeUserMessage      = "user_message"
	EventTypeResult           = "result"
	EventTypeError            = "error"
)

// Raw message type constants (JSON "type" field values).
const (
	RawMessageTypeSystem    = "system"
	RawMessageTypeAssistant = "assistant"
	RawMessageTypeMessage   = "message" // Alias for assistant
	RawMessageTypeUser      = "user"
	RawMessageTypeResult    = "result"
	RawMessageTypeError     = "error"
)

// SubtypeInit Message subtype constants.
const (
	SubtypeInit = "init"
)

// Content block type constants.
const (
	ContentBlockTypeText    = "text"
	ContentBlockTypeToolUse = "tool_use"
)

// Error type constants.
const (
	ErrorTypeRateLimit      = "rate_limit"
	ErrorTypeAuthentication = "authentication"
	ErrorTypeCommand        = "command_error"
	ErrorTypeScanner        = "scanner_error"
)

// ErrorData contains data from a Claude error event.
type ErrorData struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// SystemInitData contains data from a Claude system init event.
type SystemInitData struct {
	Model       string `json:"model,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	CWD         string `json:"cwd,omitempty"`
	ClaudeEnv   string `json:"claude_env,omitempty"`
	ModelID     string `json:"model_id,omitempty"`
	ProjectPath string `json:"project_path,omitempty"`
}

// ToolUseData contains data from a tool use content block.
type ToolUseData struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// AssistantMessageData contains data from a Claude assistant message.
type AssistantMessageData struct {
	Role       string         `json:"role"`
	Content    []ContentBlock `json:"content"`
	StopReason string         `json:"stop_reason,omitempty"`
	Model      string         `json:"model,omitempty"`
}

// UserMessageData contains data from a user message (tool results).
type UserMessageData struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

// ContentBlock represents a single content block in a Claude message.
type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   interface{}     `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

// ResultData contains data from a Claude result event.
type ResultData struct {
	Subtype      string `json:"subtype"` // "success" or "error"
	InputTokens  int    `json:"input_tokens,omitempty"`
	OutputTokens int    `json:"output_tokens,omitempty"`
	Duration     string `json:"duration,omitempty"`
	SessionID    string `json:"session_id,omitempty"`
	Result       string `json:"result,omitempty"`
	IsError      bool   `json:"is_error"`
}

// RawMessage represents a raw JSON message from the Claude CLI stream.
// This is used for initial parsing before converting to typed events.
type RawMessage struct {
	Type    string          `json:"type"`
	Subtype string          `json:"subtype,omitempty"`
	Message json.RawMessage `json:"message,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
	Result  string          `json:"result,omitempty"`

	// System init fields
	Model       string `json:"model,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	CWD         string `json:"cwd,omitempty"`
	ClaudeEnv   string `json:"claude_env,omitempty"`
	ModelID     string `json:"model_id,omitempty"`
	ProjectPath string `json:"project_path,omitempty"`

	// Result fields
	InputTokens  int    `json:"input_tokens,omitempty"`
	OutputTokens int    `json:"output_tokens,omitempty"`
	Duration     string `json:"duration,omitempty"`
	IsError      bool   `json:"is_error,omitempty"`
}

// MessageContent represents the content field in Claude assistant/user messages.
type MessageContent struct {
	Role       string         `json:"role"`
	Content    []ContentBlock `json:"content"`
	StopReason string         `json:"stop_reason,omitempty"`
	Model      string         `json:"model,omitempty"`
}

// ErrorContent represents the error field in Claude error messages.
type ErrorContent struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
