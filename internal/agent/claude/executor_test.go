package claude

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rickihastings/spinner/internal/agent"
)

func TestNewExecutor(t *testing.T) {
	executor := NewExecutor(nil)

	if executor == nil {
		t.Fatal("expected executor, got nil")
	}

	if executor.completionSignal != "~~ FEATURE_COMPLETED ~~" {
		t.Errorf("expected default completion signal, got %s", executor.completionSignal)
	}

	if executor.parser == nil {
		t.Error("expected parser to be initialized")
	}
}

func TestNewExecutor_WithConfig(t *testing.T) {
	config := &ExecutorConfig{
		LogPath:    "/tmp/test.log",
		WorkDir:    "/tmp",
		IncludeRaw: true,
	}

	executor := NewExecutor(config)

	if executor.config.LogPath != "/tmp/test.log" {
		t.Errorf("expected log path /tmp/test.log, got %s", executor.config.LogPath)
	}

	if executor.config.WorkDir != "/tmp" {
		t.Errorf("expected work dir /tmp, got %s", executor.config.WorkDir)
	}

	if !executor.parser.includeRaw {
		t.Error("expected parser.includeRaw to be true")
	}
}

func TestEventCollector_Collect(t *testing.T) {
	collector := newEventCollector()
	events := make(chan agent.Event, 3)

	events <- agent.Event{Type: EventTypeSystemInit}

	events <- agent.Event{Type: eventTypeAssistantMessage}

	events <- agent.Event{Type: eventTypeResult}

	close(events)
	collector.collect(events)

	collected := collector.events()
	if len(collected) != 3 {
		t.Fatalf("expected 3 events, got %d", len(collected))
	}
}

func TestEventCollector_Filter(t *testing.T) {
	collector := newEventCollector()
	events := make(chan agent.Event, 5)

	events <- agent.Event{Type: EventTypeSystemInit}

	events <- agent.Event{Type: eventTypeAssistantMessage}

	events <- agent.Event{Type: eventTypeError}

	events <- agent.Event{Type: eventTypeAssistantMessage}

	events <- agent.Event{Type: eventTypeResult}

	close(events)
	collector.collect(events)

	filtered := collector.filter(eventTypeAssistantMessage)
	if len(filtered) != 2 {
		t.Errorf("expected 2 assistant messages, got %d", len(filtered))
	}

	filtered = collector.filter(eventTypeError, eventTypeResult)
	if len(filtered) != 2 {
		t.Errorf("expected 2 error/result events, got %d", len(filtered))
	}
}

func TestEventCollector_HasError(t *testing.T) {
	tests := []struct {
		name     string
		events   []agent.Event
		expected bool
	}{
		{
			name: "with_error",
			events: []agent.Event{
				{Type: EventTypeSystemInit},
				{Type: eventTypeError, Data: errorData{Type: "api_error"}},
			},
			expected: true,
		},
		{
			name: "no_error",
			events: []agent.Event{
				{Type: EventTypeSystemInit},
				{Type: eventTypeAssistantMessage},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := newEventCollector()

			ch := make(chan agent.Event, len(tt.events))

			for _, e := range tt.events {
				ch <- e
			}

			close(ch)
			collector.collect(ch)

			if collector.hasError() != tt.expected {
				t.Errorf("expected hasError=%v, got %v", tt.expected, collector.hasError())
			}
		})
	}
}

func TestEventCollector_GetAllText(t *testing.T) {
	collector := newEventCollector()

	events := make(chan agent.Event, 3)
	events <- agent.Event{
		Type: eventTypeAssistantMessage,
		Data: assistantMessageData{
			Content: []contentBlock{
				{Type: "text", Text: "First message"},
			},
		},
	}

	events <- agent.Event{Type: eventTypeUserMessage}

	events <- agent.Event{
		Type: eventTypeAssistantMessage,
		Data: assistantMessageData{
			Content: []contentBlock{
				{Type: "text", Text: "Second message"},
			},
		},
	}

	close(events)

	collector.collect(events)

	text := collector.getAllText()

	if !strings.Contains(text, "First message") {
		t.Error("expected text to contain 'First message'")
	}

	if !strings.Contains(text, "Second message") {
		t.Error("expected text to contain 'Second message'")
	}
}

func TestEventCollector_GetAllToolUses(t *testing.T) {
	collector := newEventCollector()

	events := make(chan agent.Event, 2)
	events <- agent.Event{
		Type: eventTypeAssistantMessage,
		Data: assistantMessageData{
			Content: []contentBlock{
				{Type: "tool_use", ID: "tool1", Name: "Read", Input: json.RawMessage(`{}`)},
			},
		},
	}

	events <- agent.Event{
		Type: eventTypeAssistantMessage,
		Data: assistantMessageData{
			Content: []contentBlock{
				{Type: "tool_use", ID: "tool2", Name: "Write", Input: json.RawMessage(`{}`)},
				{Type: "tool_use", ID: "tool3", Name: "Bash", Input: json.RawMessage(`{}`)},
			},
		},
	}

	close(events)

	collector.collect(events)

	tools := collector.getAllToolUses()
	if len(tools) != 3 {
		t.Fatalf("expected 3 tool uses, got %d", len(tools))
	}

	names := make(map[string]bool)

	for _, tool := range tools {
		names[tool.Name] = true
	}

	if !names["Read"] || !names["Write"] || !names["Bash"] {
		t.Error("expected tools Read, Write, and Bash")
	}
}

func TestExecutorConfig_Defaults(t *testing.T) {
	config := &ExecutorConfig{}

	if config.LogPath != "" {
		t.Error("expected empty LogPath by default")
	}

	if config.WorkDir != "" {
		t.Error("expected empty WorkDir by default")
	}

	if config.IncludeRaw {
		t.Error("expected IncludeRaw false by default")
	}
}
