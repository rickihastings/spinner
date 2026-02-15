# Proposal: Add Rich Watch Formatter

## Summary

Add a `RichFormatter` that renders Claude CLI's `--output stream-json` events into visually rich
terminal output in the watch TUI, approximating the native Claude CLI experience. Tool calls display
as `Bash(brew install tmux)`, agent text renders with full markdown formatting (bullet points, bold,
headings, code blocks with syntax highlighting), and tool results show line counts or error
indicators.

## Motivation

The previous `Formatter` produced minimal output: timestamps + "Assistant:" prefixes with truncated
plain text (200 char limit). Tool use events were silently hidden. Tool results (user messages) were
entirely skipped. This meant users watching a long-running agent saw almost nothing of what it was
doing — no tool calls, no tool output, no markdown formatting.

The Claude CLI itself renders a rich experience: tool invocations as `ToolName(summary)`, markdown
with bullet points and code blocks, and tool output with line counts. Spinner's watch mode should
approximate this so users can meaningfully monitor agent progress without SSHing into the container.

Timestamps are removed from formatter output — the TUI header already shows timing context, and
per-line timestamps add visual noise without value. The old basic formatter is replaced entirely
by the rich formatter (consolidated into `formatter.go`).

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
- **MODIFIED** TUI Layout requirement: log section rendered without borders, title, or padding for a
  minimal Claude-Code-like appearance; header section uses a responsive layout that collapses to a
  compact 1–2 line format on narrow terminals (<80 columns); footer replaced with a solid
  vim/tmux-style status bar showing keyboard shortcuts

### Consolidated Formatter

- The old basic `Formatter` is replaced by the rich formatter, consolidated into `formatter.go`
- `Formatter` struct implements `agent.EventFormatter` — maintains tool_use_id → tool_name
  mapping for correlating tool results with their invocations
- Timestamps removed from all formatted output — no per-event timestamp prefix
- No new interfaces — uses existing `agent.EventFormatter`

### New Dependency

- `charmbracelet/glamour` — markdown → ANSI terminal rendering (brings `chroma` for syntax
  highlighting transitively)

## Impact

### Affected Specs

- `cli-watch` — three ADDED requirements (Rich Tool Call Display, Rich Tool Result Display, Rich
  Markdown Rendering), two MODIFIED requirements (Log Parsing — user messages no longer skipped;
  TUI Layout — borderless logs, responsive header, vim/tmux status bar)

### Affected Code

| Area                                           | Change Type                                                                                        |
|------------------------------------------------|----------------------------------------------------------------------------------------------------|
| `internal/agent/claude/formatter.go`           | **replace** — old basic formatter replaced with rich `Formatter` (no timestamps, markdown, tools)  |
| `internal/agent/claude/formatter_test.go`      | **replace** — old tests replaced with comprehensive rich formatter tests                           |
| `internal/tui/watch.go`                        | **modify** — remove log view border/title/padding; add responsive header with compact narrow mode  |
| `internal/tui/watch_test.go`                   | **modify** — add tests for responsive header rendering                                             |
| `cmd/watch.go`                                 | **modify** — uses `NewFormatter()` (unchanged call site, new implementation)                       |
| `go.mod` / `go.sum`                            | **modify** — add `charmbracelet/glamour` dependency                                                |

### Not Affected

- `internal/agent/claude/parser.go` — no parsing changes; all needed data already extracted
- `internal/agent/claude/types.go` — no type changes; tool_use blocks already have ID, Name, Input
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
- **Configurable formatter selection** — there is only one `Formatter`; if alternative formats are
  needed they can be added later
- **Custom themes or color schemes** — use glamour's default dark style; theming can be added later
- **Image or binary content rendering** — tool results containing non-text content are shown as
  `(binary content)` placeholder
