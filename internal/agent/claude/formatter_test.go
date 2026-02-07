package claude

import (
	"testing"
	"time"

	"github.com/rickihastings/spinner/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatter_ColorCodesUseResetTag(t *testing.T) {
	f := NewFormatter()

	testEvents := []agent.Event{
		{
			Type:      EventTypeSystemInit,
			Timestamp: time.Now(),
			Data:      SystemInitData{Model: "claude-sonnet-4"},
		},
		{
			Type:      eventTypeAssistantMessage,
			Timestamp: time.Now(),
			Data: assistantMessageData{
				Content: []contentBlock{{Type: "text", Text: "Test"}},
			},
		},
		{
			Type:      eventTypeResult,
			Timestamp: time.Now(),
			Data: resultData{
				IsError: false,
			},
		},
	}

	for _, event := range testEvents {
		t.Run(event.Type, func(t *testing.T) {
			formatted, ok := f.FormatEvent(&event)
			require.True(t, ok)
			assert.Contains(t, formatted, "[-]", "all events must use [-] color reset")
			assert.NotContains(t, formatted, "[white]", "must not use [white] -- invisible on light backgrounds")
		})
	}
}

func TestFormatter_DoesNotExposeRawJSON(t *testing.T) {
	f := NewFormatter()

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Now(),
		Data: assistantMessageData{
			Content: []contentBlock{{Type: "text", Text: "Hello world"}},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)

	assert.Contains(t, formatted, "Assistant:")
	assert.Contains(t, formatted, "Hello world")
	assert.NotContains(t, formatted, `"type":"assistant"`)
	assert.NotContains(t, formatted, `"session_id"`)
}

func TestFormatter_HidesToolUseMessages(t *testing.T) {
	f := NewFormatter()

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Now(),
		Data: assistantMessageData{
			Content: []contentBlock{
				{Type: contentBlockTypeToolUse, ID: "tool_1", Name: "Read"},
			},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	assert.False(t, ok, "tool-use-only messages should be hidden")
	assert.Empty(t, formatted)
}

func TestFormatter_HidesToolResultMessages(t *testing.T) {
	f := NewFormatter()

	event := agent.Event{
		Type:      eventTypeUserMessage,
		Timestamp: time.Now(),
		Data: userMessageData{
			Role:    "user",
			Content: []contentBlock{{Type: "tool_result", Content: "file contents"}},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	assert.False(t, ok, "tool result messages should be hidden")
	assert.Empty(t, formatted)
}

func TestFormatter_ShowsAssistantMessageWithTextAndToolUse(t *testing.T) {
	f := NewFormatter()

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Now(),
		Data: assistantMessageData{
			Content: []contentBlock{
				{Type: contentBlockTypeText, Text: "Let me read the file"},
				{Type: contentBlockTypeToolUse, ID: "tool_1", Name: "Read"},
			},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	assert.True(t, ok, "messages with both text and tool use should be shown")
	assert.Contains(t, formatted, "Let me read the file")
	assert.NotContains(t, formatted, "(tool use)")
}
