// Package claude provides Claude CLI execution and log parsing.
package claude

import (
	"encoding/json"
)

// Event type constants for Claude CLI output.
const (
	EventTypeSystemInit       = "system_init"
	eventTypeAssistantMessage = "assistant_message"
	eventTypeUserMessage      = "user_message"
	eventTypeResult           = "result"
	eventTypeError            = "error"
)

// Raw message type constants (JSON "type" field values).
const (
	rawMessageTypeSystem    = "system"
	rawMessageTypeAssistant = "assistant"
	rawMessageTypeMessage   = "message" // Alias for assistant
	rawMessageTypeUser      = "user"
	rawMessageTypeResult    = "result"
	rawMessageTypeError     = "error"
)

// subtypeInit Message subtype constants.
const (
	subtypeInit = "init"
)

// Content block type constants.
const (
	contentBlockTypeText    = "text"
	contentBlockTypeToolUse = "tool_use"
)

// Error type constants.
const (
	errorTypeRateLimit      = "rate_limit"
	errorTypeAuthentication = "authentication"
	errorTypeCommand        = "command_error"
	errorTypeScanner        = "scanner_error"
)

// errorData contains data from a Claude error event.
type errorData struct {
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

// toolUseData contains data from a tool use content block.
type toolUseData struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// assistantMessageData contains data from a Claude assistant message.
type assistantMessageData struct {
	Role       string         `json:"role"`
	Content    []contentBlock `json:"content"`
	StopReason string         `json:"stop_reason,omitempty"`
	Model      string         `json:"model,omitempty"`
}

// userMessageData contains data from a user message (tool results).
type userMessageData struct {
	Role    string         `json:"role"`
	Content []contentBlock `json:"content"`
}

// contentBlock represents a single content block in a Claude message.
type contentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   interface{}     `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

// resultData contains data from a Claude result event.
type resultData struct {
	Subtype      string `json:"subtype"` // "success" or "error"
	InputTokens  int    `json:"input_tokens,omitempty"`
	OutputTokens int    `json:"output_tokens,omitempty"`
	Duration     string `json:"duration,omitempty"`
	SessionID    string `json:"session_id,omitempty"`
	Result       string `json:"result,omitempty"`
	IsError      bool   `json:"is_error"`
}

// rawMessage represents a raw JSON message from the Claude CLI stream.
// This is used for initial parsing before converting to typed events.
type rawMessage struct {
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

// messageContent represents the content field in Claude assistant/user messages.
type messageContent struct {
	Role       string         `json:"role"`
	Content    []contentBlock `json:"content"`
	StopReason string         `json:"stop_reason,omitempty"`
	Model      string         `json:"model,omitempty"`
}

// errorContent represents the error field in Claude error messages.
type errorContent struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
