package claude

import (
	"fmt"
	"strings"
	"time"

	"github.com/rickihastings/spinner/internal/agent"
)

// Formatter implements agent.EventFormatter for Claude CLI output.
type Formatter struct{}

// NewFormatter creates a new Claude Formatter.
func NewFormatter() *Formatter {
	return &Formatter{}
}

// FormatEvent converts an agent.Event to a human-readable display string.
// Returns the formatted string and true if formatting was successful,
// or empty string and false if the event cannot be formatted.
func (f *Formatter) FormatEvent(event *agent.Event) (string, bool) {
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
	case EventTypeAssistantMessage:
		// Skip tool-use-only messages
		if isToolUseOnly(event) {
			return "", false
		}

		return formatAssistantEvent(timestamp, event), true
	case EventTypeUserMessage:
		// Skip tool result messages
		return "", false
	case EventTypeResult:
		return formatResultEvent(timestamp, event), true
	case EventTypeError:
		return formatErrorEvent(timestamp, event), true
	default:
		return "", false
	}
}

func formatSystemInitEvent(timestamp string, event *agent.Event) string {
	data, ok := event.Data.(SystemInitData)
	if !ok {
		return fmt.Sprintf("[darkgray]%s[-] [gray]System initialized[-]", timestamp)
	}

	if data.Model != "" {
		return fmt.Sprintf("[darkgray]%s[-] [gray]System initialized (model: %s)[-]", timestamp, data.Model)
	}

	return fmt.Sprintf("[darkgray]%s[-] [gray]System initialized[-]", timestamp)
}

func isToolUseOnly(event *agent.Event) bool {
	data, ok := event.Data.(AssistantMessageData)
	if !ok {
		return false
	}

	hasText := false
	hasToolUse := false

	for _, block := range data.Content {
		if block.Type == ContentBlockTypeText && block.Text != "" {
			hasText = true
		} else if block.Type == ContentBlockTypeToolUse {
			hasToolUse = true
		}
	}

	// Only skip if it has tool use but no text
	return hasToolUse && !hasText
}

func formatAssistantEvent(timestamp string, event *agent.Event) string {
	data, ok := event.Data.(AssistantMessageData)
	if !ok {
		return fmt.Sprintf("[darkgray]%s[-] [lightblue]Assistant:[-] [gray](message)[-]", timestamp)
	}

	// Extract text from content blocks
	var texts []string

	for _, block := range data.Content {
		if block.Type == ContentBlockTypeText && block.Text != "" {
			texts = append(texts, block.Text)
		}
	}

	if len(texts) > 0 {
		text := strings.Join(texts, "\n")
		// Truncate long messages for display
		if len(text) > 200 {
			text = text[:197] + "..."
		}

		// Calculate indent for multiline text to align with the first line
		// Indent should be: timestamp + one space
		indent := strings.Repeat(" ", len(timestamp)+1)
		lines := strings.Split(text, "\n")

		for i := 1; i < len(lines); i++ {
			lines[i] = indent + lines[i]
		}

		text = strings.Join(lines, "\n")

		return fmt.Sprintf("[darkgray]%s[-] [lightblue]Assistant:[-] %s", timestamp, text)
	}

	return fmt.Sprintf("[darkgray]%s[-] [lightblue]Assistant:[-] [gray](message)[-]", timestamp)
}

func formatResultEvent(timestamp string, event *agent.Event) string {
	data, ok := event.Data.(ResultData)
	if !ok {
		return fmt.Sprintf("[darkgray]%s[-] [cyan]Result:[-] [gray](unknown)[-]", timestamp)
	}

	status := "[green]✓ Success[-]"
	if data.IsError {
		status = "[red]✗ Error[-]"
	}

	return fmt.Sprintf("[darkgray]%s[-] [cyan]Result:[-] %s [darkgray][-]",
		timestamp, status)
}

func formatErrorEvent(timestamp string, event *agent.Event) string {
	data, ok := event.Data.(ErrorData)
	if !ok {
		return fmt.Sprintf("[darkgray]%s[-] [red]Error:[-] [gray](unknown error)[-]", timestamp)
	}

	msg := data.Message
	if len(msg) > 100 {
		msg = msg[:97] + "..."
	}

	return fmt.Sprintf("[darkgray]%s[-] [red]Error:[-] %s", timestamp, msg)
}
