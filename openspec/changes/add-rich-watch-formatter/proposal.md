# Proposal: Add Rich Watch Formatter

## Summary

Add a `RichFormatter` that renders Claude CLI's `--output stream-json` events into visually rich
terminal output in the watch TUI, approximating the native Claude CLI experience. Tool calls display
as `Bash(brew install tmux)`, agent text renders with full markdown formatting (bullet points, bold,
headings, code blocks with syntax highlighting), and tool results show line counts or error
indicators.

## Motivation

The current `Formatter` produces minimal output: timestamps + "Assistant:" prefixes with truncated
plain text (200 char limit). Tool use events are silently hidden. Tool results (user messages) are
entirely skipped. This means users watching a long-running agent see almost nothing of what it's
doing — no tool calls, no tool output, no markdown formatting.

The Claude CLI itself renders a rich experience: tool invocations as `ToolName(summary)`, markdown
with bullet points and code blocks, and tool output with line counts. Spinner's watch mode should
approximate this so users can meaningfully monitor agent progress without SSHing into the container.

## What Changes

### Modified Capability: `cli-watch`

- **ADDED** Rich event formatting: tool calls displayed as `ToolName(parameter_summary)` with
  distinct visual style
- **ADDED** Tool result rendering: user messages (tool results) shown as indented output with line
  counts or error indicators
- **ADDED** Markdown rendering for agent text: headings, bold, bullet points, numbered lists, and
  code blocks with syntax highlighting via `glamour`
- **MODIFIED** Log Parsing requirement: user messages (tool results) are no longer unconditionally
  skipped — they are rendered when the rich formatter is active

### New Internal Types

- `RichFormatter` struct implementing `agent.EventFormatter` — maintains tool_use_id → tool_name
  mapping for correlating tool results with their invocations
- No new interfaces — uses existing `agent.EventFormatter`

### New Dependency

- `charmbracelet/glamour` — markdown → ANSI terminal rendering (brings `chroma` for syntax
  highlighting transitively)

## Impact

### Affected Specs

- `cli-watch` — three ADDED requirements (Rich Tool Call Display, Rich Tool Result Display, Rich
  Markdown Rendering), one MODIFIED requirement (Log Parsing — user messages no longer skipped)

### Affected Code

| Area                                           | Change Type                                                             |
|------------------------------------------------|-------------------------------------------------------------------------|
| `internal/agent/claude/rich_formatter.go`      | **create** — `RichFormatter` implementing `EventFormatter`              |
| `internal/agent/claude/rich_formatter_test.go` | **create** — unit tests for rich formatting                             |
| `cmd/watch.go`                                 | **modify** — switch `NewFormatter()` to `NewRichFormatter()` (line 109) |
| `go.mod` / `go.sum`                            | **modify** — add `charmbracelet/glamour` dependency                     |

### Not Affected

- `internal/agent/claude/formatter.go` — preserved as-is (fallback / reference)
- `internal/agent/claude/parser.go` — no parsing changes; all needed data already extracted
- `internal/agent/claude/types.go` — no type changes; tool_use blocks already have ID, Name, Input
- `internal/tui/watch.go` — no TUI changes; receives formatted strings via `EventFormatter` interface
- `internal/agent/agent.go` — `EventFormatter` interface unchanged
- All other commands (`spin`, `setup`, `exec`, `destroy`)

## Risks & Mitigations

| Risk                                  | Impact                              | Mitigation                                                                                             |
|---------------------------------------|-------------------------------------|--------------------------------------------------------------------------------------------------------|
| `glamour` adds dependency weight      | Larger binary, more transitive deps | glamour is well-maintained (charmbracelet ecosystem), already common in Go CLI tools                   |
| ANSI → tview tag conversion lossy     | Some formatting lost or garbled     | Use `tview.TranslateANSI()` which handles common ANSI codes; test with representative markdown         |
| Large tool outputs flood the log view | Scroll position lost, TUI sluggish  | Truncate tool result text to a configurable line limit (e.g., 50 lines) with `+N more lines` indicator |
| Tool ID tracking grows unbounded      | Memory leak in long sessions        | Clear map entries after tool result is received (each ID used exactly once)                            |

## Non-Goals

- **Streaming/partial rendering** — events arrive as complete JSON objects; no incremental token
  rendering needed
- **Configurable formatter selection** — always use `RichFormatter`; the old `Formatter` remains in
  code but is not wired as an option
- **Custom themes or color schemes** — use glamour's default dark style; theming can be added later
- **Image or binary content rendering** — tool results containing non-text content are shown as
  `(binary content)` placeholder
