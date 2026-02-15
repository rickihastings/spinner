package claude

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/rickihastings/spinner/internal/agent"
	"github.com/rivo/tview"
)

const (
	// maxToolSummaryLen is the maximum length of a tool parameter summary before truncation.
	maxToolSummaryLen = 80
)

// RichFormatter implements agent.EventFormatter with rich tool call display.
// Unlike the stateless Formatter, RichFormatter maintains a mapping of tool_use_id
// to tool_name so that tool results can be correlated with their invocations.
type RichFormatter struct {
	toolNames map[string]string          // tool_use_id → tool_name
	renderer  *glamour.TermRenderer      // reused across calls
}

// NewRichFormatter creates a new RichFormatter.
func NewRichFormatter() *RichFormatter {
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)

	return &RichFormatter{
		toolNames: make(map[string]string),
		renderer:  renderer,
	}
}

// FormatEvent converts an agent.Event to a rich display string.
func (f *RichFormatter) FormatEvent(event *agent.Event) (string, bool) {
	if event == nil {
		return "", false
	}

	timestamp := event.Timestamp.Format("15:04:05")
	if event.Timestamp.IsZero() {
		timestamp = time.Now().Format("15:04:05")
	}

	switch event.Type {
	case EventTypeSystemInit:
		return formatSystemInitEvent(timestamp, event), true
	case eventTypeAssistantMessage:
		return f.formatAssistantRich(timestamp, event)
	case eventTypeUserMessage:
		// Tool results — handled in slice 3.0; skip for now
		return "", false
	case eventTypeResult:
		return formatResultEvent(timestamp, event), true
	case eventTypeError:
		return formatErrorEvent(timestamp, event), true
	default:
		return "", false
	}
}

// formatAssistantRich renders assistant messages with rich formatting.
// Text blocks are rendered through glamour for markdown formatting.
// Tool use blocks are rendered as "⏺ ToolName(param_summary)".
// Mixed messages render text first, then tool calls.
func (f *RichFormatter) formatAssistantRich(timestamp string, event *agent.Event) (string, bool) {
	data, ok := event.Data.(assistantMessageData)
	if !ok {
		return "", false
	}

	var textParts []string
	var toolParts []string

	for _, block := range data.Content {
		switch block.Type {
		case contentBlockTypeText:
			if block.Text != "" {
				textParts = append(textParts, block.Text)
			}
		case contentBlockTypeToolUse:
			toolParts = append(toolParts, f.formatToolUse(block))
		}
	}

	if len(textParts) == 0 && len(toolParts) == 0 {
		return "", false
	}

	var parts []string

	if len(textParts) > 0 {
		rendered := f.renderMarkdown(strings.Join(textParts, "\n\n"))
		parts = append(parts, rendered)
	}

	parts = append(parts, toolParts...)

	return fmt.Sprintf("[darkgray]%s[-] %s", timestamp, strings.Join(parts, "\n")), true
}

// renderMarkdown renders text through glamour and converts ANSI to tview tags.
func (f *RichFormatter) renderMarkdown(text string) string {
	if f.renderer == nil {
		return text
	}

	rendered, err := f.renderer.Render(text)
	if err != nil {
		return text
	}

	// glamour adds trailing newlines; trim them
	rendered = strings.TrimRight(rendered, "\n")

	// Convert ANSI escape codes to tview color tags
	return tview.TranslateANSI(rendered)
}

// formatToolUse renders a single tool_use block and records the ID→name mapping.
func (f *RichFormatter) formatToolUse(block contentBlock) string {
	// Record for later correlation with tool results
	if block.ID != "" && block.Name != "" {
		f.toolNames[block.ID] = block.Name
	}

	summary := extractToolSummary(block.Name, block.Input)
	if summary != "" {
		return fmt.Sprintf("  [lightblue]⏺[-] [cyan]%s[-](%s)", block.Name, summary)
	}

	return fmt.Sprintf("  [lightblue]⏺[-] [cyan]%s[-]", block.Name)
}

// extractToolSummary extracts a human-readable parameter summary from tool input JSON.
func extractToolSummary(toolName string, input json.RawMessage) string {
	if len(input) == 0 {
		return ""
	}

	var params map[string]interface{}
	if err := json.Unmarshal(input, &params); err != nil {
		return ""
	}

	var summary string

	switch toolName {
	case "Bash":
		summary = stringField(params, "command")
	case "Read":
		summary = stringField(params, "file_path")
	case "Edit":
		summary = stringField(params, "file_path")
	case "Write":
		summary = stringField(params, "file_path")
	case "Glob":
		summary = stringField(params, "pattern")
	case "Grep":
		summary = stringField(params, "pattern")
	default:
		summary = firstStringField(params)
	}

	if len(summary) > maxToolSummaryLen {
		summary = summary[:maxToolSummaryLen-3] + "..."
	}

	return summary
}

// stringField extracts a string field from a map.
func stringField(params map[string]interface{}, key string) string {
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}

	return ""
}

// firstStringField returns the first short string value from a map.
func firstStringField(params map[string]interface{}) string {
	for _, v := range params {
		if s, ok := v.(string); ok && len(s) > 0 {
			return s
		}
	}

	return ""
}
