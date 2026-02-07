package claude

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rickihastings/spinner/internal/agent"
)

func TestParser_ParseSystemInit(t *testing.T) {
	jsonStr := `{"type":"system","subtype":"init","model":"claude-sonnet-4-20250514","session_id":"abc123","cwd":"/home/test","max_turns":10}`

	parser := NewParser()
	event := parser.ParseLine(jsonStr)

	if event == nil {
		t.Fatal("expected event, got nil")
	}

	if event.Type != EventTypeSystemInit {
		t.Errorf("expected type %s, got %s", EventTypeSystemInit, event.Type)
	}

	data, ok := event.Data.(SystemInitData)
	if !ok {
		t.Fatal("expected SystemInitData")
	}

	if data.Model != "claude-sonnet-4-20250514" {
		t.Errorf("expected model claude-sonnet-4-20250514, got %s", data.Model)
	}

	if data.SessionID != "abc123" {
		t.Errorf("expected session_id abc123, got %s", data.SessionID)
	}

	if data.CWD != "/home/test" {
		t.Errorf("expected cwd /home/test, got %s", data.CWD)
	}
}

func TestParser_ParseAssistantMessage(t *testing.T) {
	jsonStr := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Hello, world!"}],"stop_reason":"end_turn"}}`

	parser := NewParser()
	event := parser.ParseLine(jsonStr)

	if event == nil {
		t.Fatal("expected event, got nil")
	}

	if event.Type != eventTypeAssistantMessage {
		t.Errorf("expected type %s, got %s", eventTypeAssistantMessage, event.Type)
	}

	data, ok := event.Data.(assistantMessageData)
	if !ok {
		t.Fatal("expected assistantMessageData")
	}

	if data.Role != "assistant" {
		t.Errorf("expected role assistant, got %s", data.Role)
	}

	if len(data.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(data.Content))
	}

	if data.Content[0].Type != "text" {
		t.Errorf("expected content type text, got %s", data.Content[0].Type)
	}

	if data.Content[0].Text != "Hello, world!" {
		t.Errorf("expected text 'Hello, world!', got %s", data.Content[0].Text)
	}

	if data.StopReason != "end_turn" {
		t.Errorf("expected stop_reason end_turn, got %s", data.StopReason)
	}
}

func TestParser_ParseMessageType(t *testing.T) {
	// "message" type is an alias for "assistant"
	jsonStr := `{"type":"message","message":{"role":"assistant","content":[{"type":"text","text":"Test"}],"stop_reason":"end_turn"}}`

	parser := NewParser()
	event := parser.ParseLine(jsonStr)

	if event == nil {
		t.Fatal("expected event, got nil")
	}

	if event.Type != eventTypeAssistantMessage {
		t.Errorf("expected type %s, got %s", eventTypeAssistantMessage, event.Type)
	}
}

func TestParser_ParseUserMessage(t *testing.T) {
	jsonStr := `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool123","content":"file contents here"}]}}`

	parser := NewParser()
	event := parser.ParseLine(jsonStr)

	if event == nil {
		t.Fatal("expected event, got nil")
	}

	if event.Type != eventTypeUserMessage {
		t.Errorf("expected type %s, got %s", eventTypeUserMessage, event.Type)
	}

	data, ok := event.Data.(userMessageData)
	if !ok {
		t.Fatal("expected userMessageData")
	}

	if data.Role != "user" {
		t.Errorf("expected role user, got %s", data.Role)
	}

	if len(data.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(data.Content))
	}

	if data.Content[0].Type != "tool_result" {
		t.Errorf("expected content type tool_result, got %s", data.Content[0].Type)
	}
}

func TestParser_ParseToolUse(t *testing.T) {
	jsonStr := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tool123","name":"Read","input":{"file_path":"/test/file.txt"}}],"stop_reason":"tool_use"}}`

	parser := NewParser()
	event := parser.ParseLine(jsonStr)

	if event == nil {
		t.Fatal("expected event, got nil")
	}

	data, ok := event.Data.(assistantMessageData)
	if !ok {
		t.Fatal("expected assistantMessageData")
	}

	if len(data.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(data.Content))
	}

	block := data.Content[0]
	if block.Type != "tool_use" {
		t.Errorf("expected type tool_use, got %s", block.Type)
	}

	if block.ID != "tool123" {
		t.Errorf("expected id tool123, got %s", block.ID)
	}

	if block.Name != "Read" {
		t.Errorf("expected name Read, got %s", block.Name)
	}

	// Verify input can be parsed
	var input map[string]interface{}
	if err := json.Unmarshal(block.Input, &input); err != nil {
		t.Fatalf("failed to parse input: %v", err)
	}

	if input["file_path"] != "/test/file.txt" {
		t.Errorf("expected file_path /test/file.txt, got %v", input["file_path"])
	}
}

func TestParser_ParseErrorMessage(t *testing.T) {
	jsonStr := `{"type":"error","error":{"type":"rate_limit_error","message":"Rate limit exceeded"}}`

	parser := NewParser()
	event := parser.ParseLine(jsonStr)

	if event == nil {
		t.Fatal("expected event, got nil")
	}

	if event.Type != eventTypeError {
		t.Errorf("expected type %s, got %s", eventTypeError, event.Type)
	}

	data, ok := event.Data.(errorData)
	if !ok {
		t.Fatal("expected errorData")
	}

	if data.Type != "rate_limit_error" {
		t.Errorf("expected type rate_limit_error, got %s", data.Type)
	}

	if data.Message != "Rate limit exceeded" {
		t.Errorf("expected message 'Rate limit exceeded', got %s", data.Message)
	}
}

func TestParser_ParseResultMessage(t *testing.T) {
	jsonStr := `{"type":"result","subtype":"success","cost_usd":0.05,"input_tokens":1000,"output_tokens":500,"duration":"10s","num_turns":3,"session_id":"abc123"}`

	parser := NewParser()
	event := parser.ParseLine(jsonStr)

	if event == nil {
		t.Fatal("expected event, got nil")
	}

	if event.Type != eventTypeResult {
		t.Errorf("expected type %s, got %s", eventTypeResult, event.Type)
	}

	data, ok := event.Data.(resultData)
	if !ok {
		t.Fatal("expected resultData")
	}

	if data.Subtype != "success" {
		t.Errorf("expected subtype success, got %s", data.Subtype)
	}

	if data.InputTokens != 1000 {
		t.Errorf("expected input_tokens 1000, got %d", data.InputTokens)
	}

	if data.OutputTokens != 500 {
		t.Errorf("expected output_tokens 500, got %d", data.OutputTokens)
	}
}

func TestParser_ParseSkipsUnknownTypes(t *testing.T) {
	// stream_event types should be skipped
	jsonStr := `{"type":"stream_event","event":{"type":"content_block_start"}}`

	parser := NewParser()
	event := parser.ParseLine(jsonStr)

	if event != nil {
		t.Error("expected nil for stream_event, got event")
	}
}

func TestParser_ParseSkipsNonJSON(t *testing.T) {
	parser := NewParser()
	event := parser.ParseLine("not json")

	if event != nil {
		t.Error("expected nil for non-JSON, got event")
	}
}

func TestParser_ParseSkipsEmptyLines(t *testing.T) {
	parser := NewParser()
	event := parser.ParseLine("")

	if event != nil {
		t.Error("expected nil for empty line, got event")
	}
}

func TestParser_IncludeRaw(t *testing.T) {
	jsonStr := `{"type":"system","subtype":"init","model":"test"}`

	parser := NewParser()
	parser.includeRaw = true
	event := parser.ParseLine(jsonStr)

	if event == nil {
		t.Fatal("expected event, got nil")
	}

	if event.Raw != jsonStr {
		t.Errorf("expected raw '%s', got '%s'", jsonStr, event.Raw)
	}
}

func TestParser_Parse_Stream(t *testing.T) {
	input := `{"type":"system","subtype":"init","model":"test"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Hello"}],"stop_reason":"end_turn"}}
{"type":"result","subtype":"success"}`

	parser := NewParser()
	ctx := context.Background()
	events := parser.Parse(ctx, strings.NewReader(input))

	var collected []agent.Event

	for event := range events {
		collected = append(collected, event)
	}

	if len(collected) != 3 {
		t.Fatalf("expected 3 events, got %d", len(collected))
	}

	if collected[0].Type != EventTypeSystemInit {
		t.Errorf("expected first event type %s, got %s", EventTypeSystemInit, collected[0].Type)
	}

	if collected[1].Type != eventTypeAssistantMessage {
		t.Errorf("expected second event type %s, got %s", eventTypeAssistantMessage, collected[1].Type)
	}

	if collected[2].Type != eventTypeResult {
		t.Errorf("expected third event type %s, got %s", eventTypeResult, collected[2].Type)
	}
}

func TestParser_Parse_ContextCancellation(t *testing.T) {
	// Create a slow reader that blocks
	input := `{"type":"system","subtype":"init","model":"test"}`
	ctx, cancel := context.WithCancel(context.Background())

	parser := NewParser()
	events := parser.Parse(ctx, strings.NewReader(input))

	// Read one event
	<-events

	// Cancel context
	cancel()

	// Channel should close after cancellation
	select {
	case <-events:
		// May receive more events, but should eventually close
	case <-time.After(time.Second):
		t.Error("channel did not close after cancellation")
	}
}

func TestExtractText(t *testing.T) {
	event := &agent.Event{
		Type: eventTypeAssistantMessage,
		Data: assistantMessageData{
			Content: []contentBlock{
				{Type: "text", Text: "First"},
				{Type: "tool_use", Name: "Read"},
				{Type: "text", Text: "Second"},
			},
		},
	}

	text := extractText(event)
	expected := "First\nSecond"

	if text != expected {
		t.Errorf("expected '%s', got '%s'", expected, text)
	}
}

func TestExtractText_NonAssistant(t *testing.T) {
	event := &agent.Event{
		Type: eventTypeUserMessage,
		Data: userMessageData{},
	}

	text := extractText(event)
	if text != "" {
		t.Errorf("expected empty string, got '%s'", text)
	}
}

func TestExtractToolUses(t *testing.T) {
	event := &agent.Event{
		Type: eventTypeAssistantMessage,
		Data: assistantMessageData{
			Content: []contentBlock{
				{Type: "text", Text: "Hello"},
				{Type: "tool_use", ID: "tool1", Name: "Read", Input: json.RawMessage(`{"path":"/a"}`)},
				{Type: "tool_use", ID: "tool2", Name: "Write", Input: json.RawMessage(`{"path":"/b"}`)},
			},
		},
	}

	tools := extractToolUses(event)

	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	if tools[0].Name != "Read" {
		t.Errorf("expected first tool name Read, got %s", tools[0].Name)
	}

	if tools[1].Name != "Write" {
		t.Errorf("expected second tool name Write, got %s", tools[1].Name)
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name     string
		event    agent.Event
		expected bool
	}{
		{
			name: "rate_limit_error",
			event: agent.Event{
				Type: eventTypeError,
				Data: errorData{Type: "rate_limit_error", Message: "Rate limit exceeded"},
			},
			expected: true,
		},
		{
			name: "api_rate_limit",
			event: agent.Event{
				Type: eventTypeError,
				Data: errorData{Type: "api_rate_limit", Message: "Too many requests"},
			},
			expected: true,
		},
		{
			name: "other_error",
			event: agent.Event{
				Type: eventTypeError,
				Data: errorData{Type: "api_error", Message: "Something wrong"},
			},
			expected: false,
		},
		{
			name: "non_error_event",
			event: agent.Event{
				Type: eventTypeAssistantMessage,
				Data: assistantMessageData{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRateLimitError(&tt.event)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		event    agent.Event
		expected bool
	}{
		{
			name: "authentication_error",
			event: agent.Event{
				Type: eventTypeError,
				Data: errorData{Type: "authentication_error", Message: "Invalid API key"},
			},
			expected: true,
		},
		{
			name: "other_error",
			event: agent.Event{
				Type: eventTypeError,
				Data: errorData{Type: "api_error", Message: "Something wrong"},
			},
			expected: false,
		},
		{
			name: "non_error_event",
			event: agent.Event{
				Type: eventTypeAssistantMessage,
				Data: assistantMessageData{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAuthError(&tt.event)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestContainsText(t *testing.T) {
	event := &agent.Event{
		Type: eventTypeAssistantMessage,
		Data: assistantMessageData{
			Content: []contentBlock{
				{Type: "text", Text: "Task completed ~~ FEATURE_COMPLETED ~~ done"},
			},
		},
	}

	if !containsText(event, "~~ FEATURE_COMPLETED ~~") {
		t.Error("expected to contain completion signal")
	}

	if containsText(event, "not present") {
		t.Error("expected not to contain 'not present'")
	}
}

func TestParser_ParseMultipleContentBlocks(t *testing.T) {
	jsonStr := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Let me read that file"},{"type":"tool_use","id":"tool1","name":"Read","input":{"file_path":"/test.txt"}},{"type":"text","text":"Now let me write"}],"stop_reason":"tool_use"}}`

	parser := NewParser()
	event := parser.ParseLine(jsonStr)

	if event == nil {
		t.Fatal("expected event, got nil")
	}

	data, ok := event.Data.(assistantMessageData)
	if !ok {
		t.Fatal("expected assistantMessageData")
	}

	if len(data.Content) != 3 {
		t.Errorf("expected 3 content blocks, got %d", len(data.Content))
	}

	if data.Content[0].Type != "text" {
		t.Errorf("expected first block type text, got %s", data.Content[0].Type)
	}

	if data.Content[1].Type != "tool_use" {
		t.Errorf("expected second block type tool_use, got %s", data.Content[1].Type)
	}

	if data.Content[2].Type != "text" {
		t.Errorf("expected third block type text, got %s", data.Content[2].Type)
	}
}
