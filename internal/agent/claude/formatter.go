package claude

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/rickihastings/spinner/internal/agent"
	"github.com/rivo/tview"
)

const (
	// maxToolSummaryLen is the maximum length of a tool parameter summary before truncation.
	maxToolSummaryLen = 80
	// maxToolResultLines is the maximum number of lines to show from a tool result.
	maxToolResultLines = 50
)

// Formatter implements agent.EventFormatter with rich tool call display.
// It maintains a mapping of tool_use_id to tool_name so that tool results
// can be correlated with their invocations.
type Formatter struct {
	toolNames map[string]string     // tool_use_id → tool_name
	renderer  *glamour.TermRenderer // reused across calls
}

// NewFormatter creates a new RichFormatter.
func NewFormatter() *Formatter {
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)

	return &Formatter{
		toolNames: make(map[string]string),
		renderer:  renderer,
	}
}

// FormatEvent converts an agent.Event to a rich display string.
func (f *Formatter) FormatEvent(event *agent.Event) (string, bool) {
	if event == nil {
		return "", false
	}

	switch event.Type {
	case EventTypeSystemInit:
		return formatSystemInitEvent(event), true
	case eventTypeAssistantMessage:
		return f.formatAssistantRich(event)
	case eventTypeUserMessage:
		return f.formatToolResult(event)
	case eventTypeResult:
		return formatResultEvent(event), true
	case eventTypeError:
		return formatErrorEvent(event), true
	default:
		return "", false
	}
}

// formatAssistantRich renders assistant messages with rich formatting.
// Text blocks are rendered through glamour for markdown formatting.
// Tool use blocks are rendered as "⏺ ToolName(param_summary)".
// Mixed messages render text first, then tool calls.
func (f *Formatter) formatAssistantRich(event *agent.Event) (string, bool) {
	data, ok := event.Data.(assistantMessageData)
	if !ok {
		return "", false
	}

	var (
		textParts []string
		toolParts []string
	)

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

	return strings.Join(parts, "\n"), true
}

// formatToolResult renders user_message events containing tool_result blocks.
// Each tool_result is correlated with its tool invocation via tool_use_id.
func (f *Formatter) formatToolResult(event *agent.Event) (string, bool) {
	data, ok := event.Data.(userMessageData)
	if !ok {
		return "", false
	}

	var parts []string

	for _, block := range data.Content {
		if block.Type != "tool_result" {
			continue
		}

		// Look up the tool name from the recorded mapping
		toolName := "Tool"

		if block.ToolUseID != "" {
			if name, exists := f.toolNames[block.ToolUseID]; exists {
				toolName = name

				delete(f.toolNames, block.ToolUseID)
			}
		}

		// Build the header line: "⏺ ToolName ⎯⎯⎯⎯⎯⎯"
		separator := strings.Repeat("⎯", 30)
		header := fmt.Sprintf("[lightblue]⏺[-] [cyan]%s[-] [darkgray]%s[-]", toolName, separator)

		// Extract text content and build summary line
		text := extractToolResultText(block)
		if block.IsError {
			summary := "  [red]Error[-]"

			if text != "" {
				lines := strings.Split(text, "\n")
				// Show first line of error
				errLine := lines[0]
				if len(errLine) > 100 {
					errLine = errLine[:97] + "..."
				}

				summary = fmt.Sprintf("  [red]Error[-] %s", errLine)
			}

			parts = append(parts, header+"\n"+summary)
		} else {
			lines := strings.Split(text, "\n")

			lineCount := len(lines)
			if text == "" {
				lineCount = 0
			}

			if lineCount > maxToolResultLines {
				summary := fmt.Sprintf("  +%d lines [darkgray](%d more lines)[-]", lineCount, lineCount-maxToolResultLines)
				parts = append(parts, header+"\n"+summary)
			} else if lineCount > 0 {
				summary := fmt.Sprintf("  +%d lines", lineCount)
				parts = append(parts, header+"\n"+summary)
			} else {
				parts = append(parts, header+"\n  [darkgray](empty)[-]")
			}
		}
	}

	if len(parts) == 0 {
		return "", false
	}

	return strings.Join(parts, "\n"), true
}

// extractToolResultText extracts text from a tool_result content block.
// The Content field can be a string or a list of content blocks.
func extractToolResultText(block contentBlock) string {
	if block.Content == nil {
		return ""
	}

	// Try as string first
	if s, ok := block.Content.(string); ok {
		return s
	}

	// Try as array of content blocks ([]interface{})
	if arr, ok := block.Content.([]interface{}); ok {
		var texts []string

		for _, item := range arr {
			if m, ok := item.(map[string]interface{}); ok {
				if t, ok := m["text"].(string); ok {
					texts = append(texts, t)
				}
			}
		}

		return strings.Join(texts, "\n")
	}

	return ""
}

// renderMarkdown renders text through glamour and converts ANSI to tview tags.
func (f *Formatter) renderMarkdown(text string) string {
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
func (f *Formatter) formatToolUse(block contentBlock) string {
	// Record for later correlation with tool results
	if block.ID != "" && block.Name != "" {
		f.toolNames[block.ID] = block.Name
	}

	summary := extractToolSummary(block.Name, block.Input)
	if summary != "" {
		return fmt.Sprintf("[lightblue]⏺[-] [cyan]%s[-](%s)", block.Name, summary)
	}

	return fmt.Sprintf("[lightblue]⏺[-] [cyan]%s[-]", block.Name)
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

func formatSystemInitEvent(event *agent.Event) string {
	data, ok := event.Data.(SystemInitData)
	if !ok {
		return "[gray]System initialized[-]"
	}

	if data.Model != "" {
		return fmt.Sprintf("[gray]System initialized (model: %s)[-]", data.Model)
	}

	return "[gray]System initialized[-]"
}

func formatResultEvent(event *agent.Event) string {
	data, ok := event.Data.(resultData)
	if !ok {
		return "[cyan]Result:[-] [gray](unknown)[-]"
	}

	status := "[green]✓ Success[-]"
	if data.IsError {
		status = "[red]✗ Error[-]"
	}

	return fmt.Sprintf("[cyan]Result:[-] %s", status)
}

func formatErrorEvent(event *agent.Event) string {
	data, ok := event.Data.(errorData)
	if !ok {
		return "[red]Error:[-] [gray](unknown error)[-]"
	}

	msg := data.Message
	if len(msg) > 100 {
		msg = msg[:97] + "..."
	}

	return fmt.Sprintf("[red]Error:[-] %s", msg)
}
