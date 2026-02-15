package claude

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rickihastings/spinner/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRichFormatter_SystemInit(t *testing.T) {
	f := NewRichFormatter()

	event := agent.Event{
		Type:      EventTypeSystemInit,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data:      SystemInitData{Model: "claude-sonnet-4"},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)
	assert.Contains(t, formatted, "System initialized")
	assert.Contains(t, formatted, "claude-sonnet-4")
}

func TestRichFormatter_Result(t *testing.T) {
	f := NewRichFormatter()

	event := agent.Event{
		Type:      eventTypeResult,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data:      resultData{IsError: false},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)
	assert.Contains(t, formatted, "Success")
}

func TestRichFormatter_Error(t *testing.T) {
	f := NewRichFormatter()

	event := agent.Event{
		Type:      eventTypeError,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data:      errorData{Message: "something went wrong"},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)
	assert.Contains(t, formatted, "Error")
	assert.Contains(t, formatted, "something went wrong")
}

func TestRichFormatter_NilEvent(t *testing.T) {
	f := NewRichFormatter()

	formatted, ok := f.FormatEvent(nil)
	assert.False(t, ok)
	assert.Empty(t, formatted)
}

func TestRichFormatter_BashToolCall(t *testing.T) {
	f := NewRichFormatter()

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: assistantMessageData{
			Content: []contentBlock{
				{
					Type:  contentBlockTypeToolUse,
					ID:    "toolu_abc",
					Name:  "Bash",
					Input: json.RawMessage(`{"command": "brew install tmux"}`),
				},
			},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)
	assert.Contains(t, formatted, "Bash")
	assert.Contains(t, formatted, "brew install tmux")
	assert.Contains(t, formatted, "⏺")
}

func TestRichFormatter_ReadToolCall(t *testing.T) {
	f := NewRichFormatter()

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: assistantMessageData{
			Content: []contentBlock{
				{
					Type:  contentBlockTypeToolUse,
					ID:    "toolu_read1",
					Name:  "Read",
					Input: json.RawMessage(`{"file_path": "/path/to/file.go"}`),
				},
			},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)
	assert.Contains(t, formatted, "Read")
	assert.Contains(t, formatted, "/path/to/file.go")
}

func TestRichFormatter_EditToolCall(t *testing.T) {
	f := NewRichFormatter()

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: assistantMessageData{
			Content: []contentBlock{
				{
					Type:  contentBlockTypeToolUse,
					ID:    "toolu_edit1",
					Name:  "Edit",
					Input: json.RawMessage(`{"file_path": "internal/agent/claude/formatter.go", "old_string": "foo", "new_string": "bar"}`),
				},
			},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)
	assert.Contains(t, formatted, "Edit")
	assert.Contains(t, formatted, "internal/agent/claude/formatter.go")
}

func TestRichFormatter_GlobToolCall(t *testing.T) {
	f := NewRichFormatter()

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: assistantMessageData{
			Content: []contentBlock{
				{
					Type:  contentBlockTypeToolUse,
					ID:    "toolu_glob1",
					Name:  "Glob",
					Input: json.RawMessage(`{"pattern": "**/*.go"}`),
				},
			},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)
	assert.Contains(t, formatted, "Glob")
	assert.Contains(t, formatted, "**/*.go")
}

func TestRichFormatter_GrepToolCall(t *testing.T) {
	f := NewRichFormatter()

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: assistantMessageData{
			Content: []contentBlock{
				{
					Type:  contentBlockTypeToolUse,
					ID:    "toolu_grep1",
					Name:  "Grep",
					Input: json.RawMessage(`{"pattern": "func main"}`),
				},
			},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)
	assert.Contains(t, formatted, "Grep")
	assert.Contains(t, formatted, "func main")
}

func TestRichFormatter_UnknownToolFallback(t *testing.T) {
	f := NewRichFormatter()

	t.Run("with string field", func(t *testing.T) {
		event := agent.Event{
			Type:      eventTypeAssistantMessage,
			Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			Data: assistantMessageData{
				Content: []contentBlock{
					{
						Type:  contentBlockTypeToolUse,
						ID:    "toolu_custom1",
						Name:  "CustomTool",
						Input: json.RawMessage(`{"url": "https://example.com"}`),
					},
				},
			},
		}

		formatted, ok := f.FormatEvent(&event)
		require.True(t, ok)
		assert.Contains(t, formatted, "CustomTool")
		assert.Contains(t, formatted, "https://example.com")
	})

	t.Run("with no string fields", func(t *testing.T) {
		event := agent.Event{
			Type:      eventTypeAssistantMessage,
			Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			Data: assistantMessageData{
				Content: []contentBlock{
					{
						Type:  contentBlockTypeToolUse,
						ID:    "toolu_custom2",
						Name:  "NumericTool",
						Input: json.RawMessage(`{"count": 42}`),
					},
				},
			},
		}

		formatted, ok := f.FormatEvent(&event)
		require.True(t, ok)
		assert.Contains(t, formatted, "NumericTool")
		// Should just show tool name without parentheses when no string field
		assert.NotContains(t, formatted, "(")
	})
}

func TestRichFormatter_LongParameterTruncation(t *testing.T) {
	f := NewRichFormatter()

	longCommand := strings.Repeat("a", 100)

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: assistantMessageData{
			Content: []contentBlock{
				{
					Type:  contentBlockTypeToolUse,
					ID:    "toolu_long1",
					Name:  "Bash",
					Input: json.RawMessage(`{"command": "` + longCommand + `"}`),
				},
			},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)
	assert.Contains(t, formatted, "...")
	// The summary (inside parens) should be truncated to maxToolSummaryLen
	// Extract the part between ( and )
	start := strings.Index(formatted, "(")
	end := strings.LastIndex(formatted, ")")
	require.True(t, start > 0 && end > start)
	summary := formatted[start+1 : end]
	assert.LessOrEqual(t, len(summary), maxToolSummaryLen)
}

func TestRichFormatter_ToolUseIDTracking(t *testing.T) {
	f := NewRichFormatter()

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: assistantMessageData{
			Content: []contentBlock{
				{
					Type:  contentBlockTypeToolUse,
					ID:    "toolu_track1",
					Name:  "Bash",
					Input: json.RawMessage(`{"command": "ls"}`),
				},
			},
		},
	}

	_, ok := f.FormatEvent(&event)
	require.True(t, ok)

	// Verify the ID was recorded
	assert.Equal(t, "Bash", f.toolNames["toolu_track1"])
}

func TestRichFormatter_MultipleToolUseBlocks(t *testing.T) {
	f := NewRichFormatter()

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: assistantMessageData{
			Content: []contentBlock{
				{
					Type:  contentBlockTypeToolUse,
					ID:    "toolu_multi1",
					Name:  "Read",
					Input: json.RawMessage(`{"file_path": "/a.go"}`),
				},
				{
					Type:  contentBlockTypeToolUse,
					ID:    "toolu_multi2",
					Name:  "Read",
					Input: json.RawMessage(`{"file_path": "/b.go"}`),
				},
			},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)
	assert.Contains(t, formatted, "/a.go")
	assert.Contains(t, formatted, "/b.go")

	// Both IDs should be tracked
	assert.Equal(t, "Read", f.toolNames["toolu_multi1"])
	assert.Equal(t, "Read", f.toolNames["toolu_multi2"])
}

func TestRichFormatter_MixedTextAndToolUse(t *testing.T) {
	f := NewRichFormatter()

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: assistantMessageData{
			Content: []contentBlock{
				{Type: contentBlockTypeText, Text: "Let me read the file"},
				{
					Type:  contentBlockTypeToolUse,
					ID:    "toolu_mix1",
					Name:  "Read",
					Input: json.RawMessage(`{"file_path": "/path/to/file.go"}`),
				},
			},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)
	assert.Contains(t, formatted, "Let me read the file")
	assert.Contains(t, formatted, "Read")
	assert.Contains(t, formatted, "/path/to/file.go")
}

func TestRichFormatter_UserMessageSkipped(t *testing.T) {
	f := NewRichFormatter()

	event := agent.Event{
		Type:      eventTypeUserMessage,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: userMessageData{
			Role:    "user",
			Content: []contentBlock{{Type: "tool_result", Content: "file contents"}},
		},
	}

	// In slice 1.0, user messages are still skipped (handled in slice 3.0)
	formatted, ok := f.FormatEvent(&event)
	assert.False(t, ok)
	assert.Empty(t, formatted)
}

func TestRichFormatter_EmptyToolInput(t *testing.T) {
	f := NewRichFormatter()

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: assistantMessageData{
			Content: []contentBlock{
				{
					Type: contentBlockTypeToolUse,
					ID:   "toolu_empty1",
					Name: "SomeTool",
				},
			},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)
	assert.Contains(t, formatted, "SomeTool")
	// No parens when no input
	assert.NotContains(t, formatted, "()")
}

func TestRichFormatter_MarkdownPlainText(t *testing.T) {
	f := NewRichFormatter()

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: assistantMessageData{
			Content: []contentBlock{
				{Type: contentBlockTypeText, Text: "This is plain text"},
			},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)
	assert.Contains(t, formatted, "This is plain text")
}

func TestRichFormatter_MarkdownBold(t *testing.T) {
	f := NewRichFormatter()

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: assistantMessageData{
			Content: []contentBlock{
				{Type: contentBlockTypeText, Text: "This is **bold** text"},
			},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)
	// The word "bold" should appear (glamour renders bold with ANSI, then translated to tview tags)
	assert.Contains(t, formatted, "bold")
	assert.Contains(t, formatted, "text")
}

func TestRichFormatter_MarkdownBulletPoints(t *testing.T) {
	f := NewRichFormatter()

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: assistantMessageData{
			Content: []contentBlock{
				{Type: contentBlockTypeText, Text: "Items:\n- First item\n- Second item\n- Third item"},
			},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)
	assert.Contains(t, formatted, "First item")
	assert.Contains(t, formatted, "Second item")
	assert.Contains(t, formatted, "Third item")
}

func TestRichFormatter_MarkdownCodeBlock(t *testing.T) {
	f := NewRichFormatter()

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: assistantMessageData{
			Content: []contentBlock{
				{Type: contentBlockTypeText, Text: "Example:\n```go\nfunc main() {}\n```"},
			},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)
	assert.Contains(t, formatted, "func main")
}

func TestRichFormatter_MarkdownHeading(t *testing.T) {
	f := NewRichFormatter()

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: assistantMessageData{
			Content: []contentBlock{
				{Type: contentBlockTypeText, Text: "# My Heading\nSome content"},
			},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)
	assert.Contains(t, formatted, "My Heading")
	assert.Contains(t, formatted, "Some content")
}

func TestRichFormatter_MixedTextAndToolUseSeparation(t *testing.T) {
	f := NewRichFormatter()

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: assistantMessageData{
			Content: []contentBlock{
				{Type: contentBlockTypeText, Text: "I'll read the file and edit it."},
				{
					Type:  contentBlockTypeToolUse,
					ID:    "toolu_mixed1",
					Name:  "Read",
					Input: json.RawMessage(`{"file_path": "/a.go"}`),
				},
				{
					Type:  contentBlockTypeToolUse,
					ID:    "toolu_mixed2",
					Name:  "Edit",
					Input: json.RawMessage(`{"file_path": "/b.go"}`),
				},
			},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)
	// Text should appear
	assert.Contains(t, formatted, "read the file and edit it")
	// Tool calls should appear
	assert.Contains(t, formatted, "Read")
	assert.Contains(t, formatted, "/a.go")
	assert.Contains(t, formatted, "Edit")
	assert.Contains(t, formatted, "/b.go")

	// Text should come before tool calls
	textIdx := strings.Index(formatted, "read the file")
	readIdx := strings.Index(formatted, "Read")
	assert.True(t, textIdx < readIdx, "text should appear before tool calls")
}

func TestRichFormatter_NilRendererFallback(t *testing.T) {
	// Test that renderMarkdown falls back to raw text when renderer is nil
	f := &RichFormatter{
		toolNames: make(map[string]string),
		renderer:  nil,
	}

	event := agent.Event{
		Type:      eventTypeAssistantMessage,
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: assistantMessageData{
			Content: []contentBlock{
				{Type: contentBlockTypeText, Text: "Plain text fallback"},
			},
		},
	}

	formatted, ok := f.FormatEvent(&event)
	require.True(t, ok)
	assert.Contains(t, formatted, "Plain text fallback")
}

func TestExtractToolSummary(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		input    string
		expected string
	}{
		{"Bash command", "Bash", `{"command": "go test ./..."}`, "go test ./..."},
		{"Read file", "Read", `{"file_path": "/foo/bar.go"}`, "/foo/bar.go"},
		{"Edit file", "Edit", `{"file_path": "/foo/bar.go", "old_string": "x"}`, "/foo/bar.go"},
		{"Write file", "Write", `{"file_path": "/new/file.go"}`, "/new/file.go"},
		{"Glob pattern", "Glob", `{"pattern": "**/*.go"}`, "**/*.go"},
		{"Grep pattern", "Grep", `{"pattern": "TODO"}`, "TODO"},
		{"Unknown with string", "Custom", `{"url": "https://example.com"}`, "https://example.com"},
		{"Empty input", "Bash", `{}`, ""},
		{"Invalid JSON", "Bash", `not json`, ""},
		{"Null input", "Bash", ``, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractToolSummary(tt.toolName, json.RawMessage(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}
