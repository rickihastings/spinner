package exec

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestClaudeMessage_ParseNormalMessage(t *testing.T) {
	jsonStr := `{"type":"message","message":{"role":"assistant","content":[{"type":"text","text":"Hello"}],"stop_reason":"end_turn"}}`

	var msg ClaudeMessage
	if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if msg.Type != "message" {
		t.Errorf("expected type 'message', got %s", msg.Type)
	}

	if len(msg.Message.Content) != 1 {
		t.Errorf("expected 1 content item, got %d", len(msg.Message.Content))
	}

	if msg.Message.Content[0].Text != "Hello" {
		t.Errorf("expected text 'Hello', got %s", msg.Message.Content[0].Text)
	}
}

func TestClaudeMessage_ParseCompletionSignal(t *testing.T) {
	jsonStr := `{"type":"message","message":{"role":"assistant","content":[{"type":"text","text":"Task completed\n\n~~ FEATURE_COMPLETED ~~\n"}],"stop_reason":"end_turn"}}`

	var msg ClaudeMessage
	if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Check if completion signal is present
	hasSignal := false
	for _, content := range msg.Message.Content {
		if content.Type == "text" && strings.Contains(content.Text, CompletionSignal) {
			hasSignal = true
			break
		}
	}

	if !hasSignal {
		t.Error("expected completion signal to be present")
	}
}

func TestClaudeMessage_ParseRateLimitError(t *testing.T) {
	jsonStr := `{"type":"error","error":{"type":"rate_limit_error","message":"Rate limit exceeded"}}`

	var msg ClaudeMessage
	if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if msg.Error.Type != "rate_limit_error" {
		t.Errorf("expected error type 'rate_limit_error', got %s", msg.Error.Type)
	}

	if !strings.Contains(msg.Error.Type, "rate_limit") {
		t.Error("expected rate_limit in error type")
	}
}

func TestClaudeMessage_ParseAuthError(t *testing.T) {
	jsonStr := `{"type":"error","error":{"type":"authentication_error","message":"Invalid API key"}}`

	var msg ClaudeMessage
	if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if msg.Error.Type != "authentication_error" {
		t.Errorf("expected error type 'authentication_error', got %s", msg.Error.Type)
	}

	if !strings.Contains(msg.Error.Type, "authentication") {
		t.Error("expected authentication in error type")
	}
}

func TestClaudeMessage_ParseGenericError(t *testing.T) {
	jsonStr := `{"type":"error","error":{"type":"api_error","message":"Something went wrong"}}`

	var msg ClaudeMessage
	if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if msg.Error.Type != "api_error" {
		t.Errorf("expected error type 'api_error', got %s", msg.Error.Type)
	}

	if msg.Error.Message != "Something went wrong" {
		t.Errorf("expected error message 'Something went wrong', got %s", msg.Error.Message)
	}
}

func TestClaudeMessage_MultipleContentBlocks(t *testing.T) {
	jsonStr := `{"type":"message","message":{"role":"assistant","content":[{"type":"text","text":"First part"},{"type":"text","text":"Second part"}],"stop_reason":"end_turn"}}`

	var msg ClaudeMessage
	if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(msg.Message.Content) != 2 {
		t.Errorf("expected 2 content items, got %d", len(msg.Message.Content))
	}

	if msg.Message.Content[0].Text != "First part" {
		t.Errorf("expected first text 'First part', got %s", msg.Message.Content[0].Text)
	}

	if msg.Message.Content[1].Text != "Second part" {
		t.Errorf("expected second text 'Second part', got %s", msg.Message.Content[1].Text)
	}
}

func TestClaudeMessage_EmptyContent(t *testing.T) {
	jsonStr := `{"type":"message","message":{"role":"assistant","content":[],"stop_reason":"end_turn"}}`

	var msg ClaudeMessage
	if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(msg.Message.Content) != 0 {
		t.Errorf("expected 0 content items, got %d", len(msg.Message.Content))
	}
}

func TestCompletionSignalConstant(t *testing.T) {
	if CompletionSignal != "~~ FEATURE_COMPLETED ~~" {
		t.Errorf("expected completion signal '~~ FEATURE_COMPLETED ~~', got %s", CompletionSignal)
	}
}
