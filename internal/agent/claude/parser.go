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

const (
	// scannerBufSize is the initial buffer size for the JSON line scanner.
	scannerBufSize = 64 * 1024

	// scannerMaxSize is the maximum line size the scanner will handle (1MB).
	scannerMaxSize = 1024 * 1024
)

// Parser parses streaming JSON output from the Claude CLI and emits structured events.
type Parser struct {
	// includeRaw includes the raw JSON line in emitted events when true.
	includeRaw bool
}

// NewParser creates a new Parser with default settings.
func NewParser() *Parser {
	return &Parser{
		includeRaw: false,
	}
}

// Parse reads lines from the reader, parses JSON messages, and emits events on the returned channel.
// The channel is closed when the reader is exhausted or the context is cancelled.
// Errors during parsing are emitted as eventTypeError events rather than stopping the parser.
func (p *Parser) Parse(ctx context.Context, reader io.Reader) <-chan agent.Event {
	events := make(chan agent.Event, 100)

	go func() {
		defer close(events)

		scanner := bufio.NewScanner(reader)
		// Increase buffer size for potentially large JSON lines
		buf := make([]byte, 0, scannerBufSize)
		scanner.Buffer(buf, scannerMaxSize)

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
				Type:      eventTypeError,
				Timestamp: time.Now(),
				Data: errorData{
					Type:    errorTypeScanner,
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
	var raw rawMessage
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		// Skip non-JSON lines silently
		return nil
	}

	event := &agent.Event{
		Timestamp: time.Now(),
	}

	if p.includeRaw {
		event.Raw = line
	}

	switch raw.Type {
	case rawMessageTypeSystem:
		return p.parseSystemMessage(event, &raw)
	case rawMessageTypeAssistant, rawMessageTypeMessage:
		return p.parseAssistantMessage(event, &raw)
	case rawMessageTypeUser:
		return p.parseUserMessage(event, &raw)
	case rawMessageTypeResult:
		return p.parseResultMessage(event, &raw)
	case rawMessageTypeError:
		return p.parseErrorMessage(event, &raw)
	default:
		// Unknown or streaming message types, skip
		return nil
	}
}

func (p *Parser) parseSystemMessage(event *agent.Event, raw *rawMessage) *agent.Event {
	if raw.Subtype == subtypeInit {
		event.Type = EventTypeSystemInit
		event.Data = SystemInitData{
			Model:       raw.Model,
			SessionID:   raw.SessionID,
			CWD:         raw.CWD,
			ClaudeEnv:   raw.ClaudeEnv,
			ModelID:     raw.ModelID,
			ProjectPath: raw.ProjectPath,
		}

		return event
	}

	return nil
}

func (p *Parser) parseAssistantMessage(event *agent.Event, raw *rawMessage) *agent.Event {
	if raw.Message == nil {
		return nil
	}

	var msg messageContent
	if err := json.Unmarshal(raw.Message, &msg); err != nil {
		return nil
	}

	event.Type = eventTypeAssistantMessage
	event.Data = assistantMessageData(msg)

	return event
}

func (p *Parser) parseUserMessage(event *agent.Event, raw *rawMessage) *agent.Event {
	if raw.Message == nil {
		return nil
	}

	var msg messageContent
	if err := json.Unmarshal(raw.Message, &msg); err != nil {
		return nil
	}

	event.Type = eventTypeUserMessage
	event.Data = userMessageData{
		Role:    msg.Role,
		Content: msg.Content,
	}

	return event
}

func (p *Parser) parseResultMessage(event *agent.Event, raw *rawMessage) *agent.Event {
	event.Type = eventTypeResult
	event.Data = resultData{
		Subtype:      raw.Subtype,
		InputTokens:  raw.InputTokens,
		OutputTokens: raw.OutputTokens,
		Duration:     raw.Duration,
		SessionID:    raw.SessionID,
		Result:       raw.Result,
		IsError:      raw.IsError,
	}

	return event
}

func (p *Parser) parseErrorMessage(event *agent.Event, raw *rawMessage) *agent.Event {
	if raw.Error == nil {
		return nil
	}

	var errContent errorContent
	if err := json.Unmarshal(raw.Error, &errContent); err != nil {
		return nil
	}

	event.Type = eventTypeError
	event.Data = errorData(errContent)

	return event
}

// extractText extracts all text content from a Claude assistant message event.
func extractText(event *agent.Event) string {
	if event.Type != eventTypeAssistantMessage {
		return ""
	}

	data, ok := event.Data.(assistantMessageData)
	if !ok {
		return ""
	}

	var texts []string

	for _, block := range data.Content {
		if block.Type == contentBlockTypeText && block.Text != "" {
			texts = append(texts, block.Text)
		}
	}

	return strings.Join(texts, "\n")
}

// extractToolUses extracts all tool use content blocks from a Claude assistant message event.
func extractToolUses(event *agent.Event) []toolUseData {
	if event.Type != eventTypeAssistantMessage {
		return nil
	}

	data, ok := event.Data.(assistantMessageData)
	if !ok {
		return nil
	}

	var tools []toolUseData

	for _, block := range data.Content {
		if block.Type == contentBlockTypeToolUse {
			tools = append(tools, toolUseData{
				ID:    block.ID,
				Name:  block.Name,
				Input: block.Input,
			})
		}
	}

	return tools
}

// isRateLimitError checks if an event is a rate limit error.
func isRateLimitError(event *agent.Event) bool {
	if event.Type != eventTypeError {
		return false
	}

	data, ok := event.Data.(errorData)
	if !ok {
		return false
	}

	return strings.Contains(data.Type, errorTypeRateLimit)
}

// isAuthError checks if an event is an authentication error.
func isAuthError(event *agent.Event) bool {
	if event.Type != eventTypeError {
		return false
	}

	data, ok := event.Data.(errorData)
	if !ok {
		return false
	}

	return strings.Contains(data.Type, errorTypeAuthentication)
}

// containsText checks if a Claude assistant message contains the specified text.
func containsText(event *agent.Event, text string) bool {
	return strings.Contains(extractText(event), text)
}
