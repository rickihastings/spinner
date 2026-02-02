package exec

import (
	"strings"
	"testing"

	"github.com/rickihastings/spinner/internal/agent/claude"
)

func TestParser_ParseNormalMessage(t *testing.T) {
	jsonStr := `{"type":"message","message":{"role":"assistant","content":[{"type":"text","text":"Hello"}],"stop_reason":"end_turn"}}`

	parser := claude.NewParser()
	event := parser.ParseLine(jsonStr)

	if event == nil {
		t.Fatal("expected event, got nil")
	}

	if event.Type != claude.EventTypeAssistantMessage {
		t.Errorf("expected type %s, got %s", claude.EventTypeAssistantMessage, event.Type)
	}

	data, ok := event.Data.(claude.AssistantMessageData)
	if !ok {
		t.Fatal("expected AssistantMessageData")
	}

	if len(data.Content) != 1 {
		t.Errorf("expected 1 content item, got %d", len(data.Content))
	}

	if data.Content[0].Text != "Hello" {
		t.Errorf("expected text 'Hello', got %s", data.Content[0].Text)
	}
}

func TestParser_ParseCompletionSignal(t *testing.T) {
	jsonStr := `{"type":"message","message":{"role":"assistant","content":[{"type":"text","text":"Task completed\n\n~~ FEATURE_COMPLETED ~~\n"}],"stop_reason":"end_turn"}}`

	parser := claude.NewParser()
	event := parser.ParseLine(jsonStr)

	if event == nil {
		t.Fatal("expected event, got nil")
	}

	// Check if completion signal is present using the helper function
	if !claude.ContainsText(event, CompletionSignal) {
		t.Error("expected completion signal to be present")
	}
}

func TestParser_ParseRateLimitError(t *testing.T) {
	jsonStr := `{"type":"error","error":{"type":"rate_limit_error","message":"Rate limit exceeded"}}`

	parser := claude.NewParser()
	event := parser.ParseLine(jsonStr)

	if event == nil {
		t.Fatal("expected event, got nil")
	}

	if event.Type != claude.EventTypeError {
		t.Errorf("expected type %s, got %s", claude.EventTypeError, event.Type)
	}

	if !claude.IsRateLimitError(event) {
		t.Error("expected IsRateLimitError to return true")
	}
}

func TestParser_ParseAuthError(t *testing.T) {
	jsonStr := `{"type":"error","error":{"type":"authentication_error","message":"Invalid API key"}}`

	parser := claude.NewParser()
	event := parser.ParseLine(jsonStr)

	if event == nil {
		t.Fatal("expected event, got nil")
	}

	if event.Type != claude.EventTypeError {
		t.Errorf("expected type %s, got %s", claude.EventTypeError, event.Type)
	}

	if !claude.IsAuthError(event) {
		t.Error("expected IsAuthError to return true")
	}
}

func TestParser_ParseGenericError(t *testing.T) {
	jsonStr := `{"type":"error","error":{"type":"api_error","message":"Something went wrong"}}`

	parser := claude.NewParser()
	event := parser.ParseLine(jsonStr)

	if event == nil {
		t.Fatal("expected event, got nil")
	}

	data, ok := event.Data.(claude.ErrorData)
	if !ok {
		t.Fatal("expected ErrorData")
	}

	if data.Type != "api_error" {
		t.Errorf("expected error type 'api_error', got %s", data.Type)
	}

	if data.Message != "Something went wrong" {
		t.Errorf("expected error message 'Something went wrong', got %s", data.Message)
	}
}

func TestParser_MultipleContentBlocks(t *testing.T) {
	jsonStr := `{"type":"message","message":{"role":"assistant","content":[{"type":"text","text":"First part"},{"type":"text","text":"Second part"}],"stop_reason":"end_turn"}}`

	parser := claude.NewParser()
	event := parser.ParseLine(jsonStr)

	if event == nil {
		t.Fatal("expected event, got nil")
	}

	data, ok := event.Data.(claude.AssistantMessageData)
	if !ok {
		t.Fatal("expected AssistantMessageData")
	}

	if len(data.Content) != 2 {
		t.Errorf("expected 2 content items, got %d", len(data.Content))
	}

	if data.Content[0].Text != "First part" {
		t.Errorf("expected first text 'First part', got %s", data.Content[0].Text)
	}

	if data.Content[1].Text != "Second part" {
		t.Errorf("expected second text 'Second part', got %s", data.Content[1].Text)
	}
}

func TestParser_EmptyContent(t *testing.T) {
	jsonStr := `{"type":"message","message":{"role":"assistant","content":[],"stop_reason":"end_turn"}}`

	parser := claude.NewParser()
	event := parser.ParseLine(jsonStr)

	if event == nil {
		t.Fatal("expected event, got nil")
	}

	data, ok := event.Data.(claude.AssistantMessageData)
	if !ok {
		t.Fatal("expected AssistantMessageData")
	}

	if len(data.Content) != 0 {
		t.Errorf("expected 0 content items, got %d", len(data.Content))
	}
}

func TestExtractText(t *testing.T) {
	jsonStr := `{"type":"message","message":{"role":"assistant","content":[{"type":"text","text":"First part"},{"type":"text","text":"Second part"}],"stop_reason":"end_turn"}}`

	parser := claude.NewParser()
	event := parser.ParseLine(jsonStr)

	text := claude.ExtractText(event)

	if !strings.Contains(text, "First part") {
		t.Error("expected text to contain 'First part'")
	}

	if !strings.Contains(text, "Second part") {
		t.Error("expected text to contain 'Second part'")
	}
}

func TestCompletionSignal(t *testing.T) {
	if CompletionSignal != "~~ FEATURE_COMPLETED ~~" {
		t.Errorf("unexpected completion signal: %s", CompletionSignal)
	}
}
