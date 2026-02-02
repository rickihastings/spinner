package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/rickihastings/spinner/internal/agent"
)

// Parser parses streaming JSON output from the Claude CLI and emits structured events.
type Parser struct {
	// IncludeRaw includes the raw JSON line in emitted events when true.
	IncludeRaw bool
}

// NewParser creates a new Parser with default settings.
func NewParser() *Parser {
	return &Parser{
		IncludeRaw: false,
	}
}

// Parse reads lines from the reader, parses JSON messages, and emits events on the returned channel.
// The channel is closed when the reader is exhausted or the context is cancelled.
// Errors during parsing are emitted as EventTypeError events rather than stopping the parser.
func (p *Parser) Parse(ctx context.Context, reader io.Reader) <-chan agent.Event {
	events := make(chan agent.Event, 100)

	go func() {
		defer close(events)

		scanner := bufio.NewScanner(reader)
		// Increase buffer size for potentially large JSON lines
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024) // 1MB max line size

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line := scanner.Text()
			if line == "" {
				continue
			}

			event := p.parseLine(line)
			if event != nil {
				select {
				case events <- *event:
				case <-ctx.Done():
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			event := &agent.Event{
				Type:      EventTypeError,
				Timestamp: time.Now(),
				Data: ErrorData{
					Type:    "scanner_error",
					Message: err.Error(),
				},
			}

			select {
			case events <- *event:
			case <-ctx.Done():
			}
		}
	}()

	return events
}

// ParseLine parses a single JSON line and returns the corresponding event.
// Returns nil if the line cannot be parsed or is not a recognized message type.
func (p *Parser) ParseLine(line string) *agent.Event {
	return p.parseLine(line)
}

func (p *Parser) parseLine(line string) *agent.Event {
	var raw RawMessage
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		// Skip non-JSON lines silently
		return nil
	}

	event := &agent.Event{
		Timestamp: time.Now(),
	}

	if p.IncludeRaw {
		event.Raw = line
	}

	switch raw.Type {
	case "system":
		return p.parseSystemMessage(event, &raw)
	case "assistant", "message":
		return p.parseAssistantMessage(event, &raw)
	case "user":
		return p.parseUserMessage(event, &raw)
	case "result":
		return p.parseResultMessage(event, &raw)
	case "error":
		return p.parseErrorMessage(event, &raw)
	default:
		// Unknown or streaming message types, skip
		return nil
	}
}

func (p *Parser) parseSystemMessage(event *agent.Event, raw *RawMessage) *agent.Event {
	if raw.Subtype == "init" {
		event.Type = EventTypeSystemInit
		event.Data = SystemInitData{
			Model:       raw.Model,
			SessionID:   raw.SessionID,
			CWD:         raw.CWD,
			ClaudeEnv:   raw.ClaudeEnv,
			ModelID:     raw.ModelID,
			MaxTurns:    raw.MaxTurns,
			ProjectPath: raw.ProjectPath,
		}

		return event
	}

	return nil
}

func (p *Parser) parseAssistantMessage(event *agent.Event, raw *RawMessage) *agent.Event {
	if raw.Message == nil {
		return nil
	}

	var msg MessageContent
	if err := json.Unmarshal(raw.Message, &msg); err != nil {
		return nil
	}

	event.Type = EventTypeAssistantMessage
	event.Data = AssistantMessageData{
		Role:       msg.Role,
		Content:    msg.Content,
		StopReason: msg.StopReason,
		Model:      msg.Model,
	}

	return event
}

func (p *Parser) parseUserMessage(event *agent.Event, raw *RawMessage) *agent.Event {
	if raw.Message == nil {
		return nil
	}

	var msg MessageContent
	if err := json.Unmarshal(raw.Message, &msg); err != nil {
		return nil
	}

	event.Type = EventTypeUserMessage
	event.Data = UserMessageData{
		Role:    msg.Role,
		Content: msg.Content,
	}

	return event
}

func (p *Parser) parseResultMessage(event *agent.Event, raw *RawMessage) *agent.Event {
	event.Type = EventTypeResult
	event.Data = ResultData{
		Subtype:      raw.Subtype,
		CostUSD:      raw.CostUSD,
		InputTokens:  raw.InputTokens,
		OutputTokens: raw.OutputTokens,
		Duration:     raw.Duration,
		NumTurns:     raw.NumTurns,
		SessionID:    raw.SessionID,
		Result:       raw.Result,
		IsError:      raw.IsError,
	}

	return event
}

func (p *Parser) parseErrorMessage(event *agent.Event, raw *RawMessage) *agent.Event {
	if raw.Error == nil {
		return nil
	}

	var errContent ErrorContent
	if err := json.Unmarshal(raw.Error, &errContent); err != nil {
		return nil
	}

	event.Type = EventTypeError
	event.Data = ErrorData(errContent)

	return event
}

// ExtractText extracts all text content from a Claude assistant message event.
func ExtractText(event *agent.Event) string {
	if event.Type != EventTypeAssistantMessage {
		return ""
	}

	data, ok := event.Data.(AssistantMessageData)
	if !ok {
		return ""
	}

	var texts []string

	for _, block := range data.Content {
		if block.Type == "text" && block.Text != "" {
			texts = append(texts, block.Text)
		}
	}

	return strings.Join(texts, "\n")
}

// ExtractToolUses extracts all tool use content blocks from a Claude assistant message event.
func ExtractToolUses(event *agent.Event) []ToolUseData {
	if event.Type != EventTypeAssistantMessage {
		return nil
	}

	data, ok := event.Data.(AssistantMessageData)
	if !ok {
		return nil
	}

	var tools []ToolUseData

	for _, block := range data.Content {
		if block.Type == "tool_use" {
			tools = append(tools, ToolUseData{
				ID:    block.ID,
				Name:  block.Name,
				Input: block.Input,
			})
		}
	}

	return tools
}

// IsRateLimitError checks if an event is a rate limit error.
func IsRateLimitError(event *agent.Event) bool {
	if event.Type != EventTypeError {
		return false
	}

	data, ok := event.Data.(ErrorData)
	if !ok {
		return false
	}

	return strings.Contains(data.Type, "rate_limit")
}

// IsAuthError checks if an event is an authentication error.
func IsAuthError(event *agent.Event) bool {
	if event.Type != EventTypeError {
		return false
	}

	data, ok := event.Data.(ErrorData)
	if !ok {
		return false
	}

	return strings.Contains(data.Type, "authentication")
}

// ContainsText checks if a Claude assistant message contains the specified text.
func ContainsText(event *agent.Event, text string) bool {
	return strings.Contains(ExtractText(event), text)
}
