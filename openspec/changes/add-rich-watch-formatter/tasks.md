# Tasks: Add Rich Watch Formatter

## 1.0 Core RichFormatter with Tool Call Display

- [x] 1.1 Add `charmbracelet/glamour` dependency to `go.mod`
- [x] 1.2 Create `internal/agent/claude/rich_formatter.go` with `RichFormatter` struct, constructor, tool_use_id tracking map, and glamour renderer initialization
- [x] 1.3 Implement `FormatEvent` for `system_init`, `result`, and `error` events (same logic as current formatter)
- [x] 1.4 Implement `FormatEvent` for `assistant_message` tool_use blocks: render as `⏺ ToolName(param_summary)` with parameter extraction per tool type
- [x] 1.5 Add unit tests for tool call formatting: Bash command summary, Read file path, Edit file path, Glob pattern, Grep pattern, unknown tool fallback, long parameter truncation
- [x] 1.6 Verify build and all tests pass

## 2.0 Markdown Rendering for Agent Text ✓

- [x] 2.1 Implement glamour markdown rendering for assistant_message text blocks: render through glamour, convert ANSI to tview tags via `tview.TranslateANSI()`
- [x] 2.2 Handle mixed content (text + tool_use blocks in same message): render text first, then tool calls
- [x] 2.3 Add unit tests for markdown rendering: plain text, bold, bullet points, code blocks, headings, mixed text+tool messages
- [x] 2.4 Verify build and all tests pass

## 3.0 Tool Result Rendering and Wiring

- [ ] 3.1 Implement `FormatEvent` for `user_message` tool_result blocks: correlate with tool_use_id, show tool name header, line count, error indicator, and clean up map entry
- [ ] 3.2 Modify `cmd/watch.go` line 109 to use `NewRichFormatter()` instead of `NewFormatter()`
- [ ] 3.3 Add unit tests for tool result formatting: success with line count, error result, unknown tool_use_id fallback, map cleanup after result
- [ ] 3.4 Verify build and all tests pass

## 4.0 TUI Simplification — Borderless Logs, Responsive Header, Status Bar

- [ ] 4.1 Remove border, title, and padding from `logView` in `internal/tui/watch.go`
- [ ] 4.2 Extract current `renderHeader` into `renderHeaderWide` for ≥80 column terminals
- [ ] 4.3 Implement `renderHeaderCompact` for <80 column terminals: single-line format showing status, iteration, branch, CPU, memory
- [ ] 4.4 Implement `renderHeaderMinimal` for <40 column terminals: minimal single-line status
- [ ] 4.5 Add width-based dispatch in `renderHeader` to select wide/compact/minimal mode
- [ ] 4.6 Replace footer with solid vim/tmux-style status bar: inverted background, left-aligned keyboard shortcuts
- [ ] 4.7 Add unit tests for responsive header rendering at different widths and status bar content
- [ ] 4.8 Verify build and all tests pass
