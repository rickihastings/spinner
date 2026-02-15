# Design: Add Rich Watch Formatter

## Technical Implementation Plan

### Component Map

| File                                           | Action      | Purpose                                                                      |
|------------------------------------------------|-------------|------------------------------------------------------------------------------|
| `internal/agent/claude/formatter.go`           | **replace** | Rich `Formatter` replaces old basic formatter (no timestamps, markdown, tools)|
| `internal/agent/claude/formatter_test.go`      | **replace** | Comprehensive tests for rich formatting                                      |
| `internal/tui/watch.go`                        | **modify**  | Remove log view border/title/padding; add responsive header with breakpoints |
| `internal/tui/watch_test.go`                   | **modify**  | Add tests for responsive header layout                                       |
| `cmd/watch.go`                                 | **modify**  | Uses `NewFormatter()` (same call, new rich implementation)                   |
| `go.mod` / `go.sum`                            | **modify**  | Add `charmbracelet/glamour` dependency                                       |

### Approach

#### Architecture

The rich `Formatter` replaces the old basic formatter entirely — consolidated into `formatter.go`.
It implements the same `agent.EventFormatter` interface. This means:

- No interface changes
- No TUI changes — `WatchUI` already consumes formatted strings via the interface
- Same `NewFormatter()` call site in `cmd/watch.go`, new rich implementation behind it

Key design decisions:
- **No timestamps** — per-event timestamps removed; the TUI header provides timing context
- **No left padding** — output flows naturally without indentation for timestamp alignment
- **Stateful** — maintains a `map[string]string` of `tool_use_id → tool_name` to correlate
  tool results with their invocations

```
Formatter
├── toolNames map[string]string  // tool_use_id → tool_name (e.g., "toolu_abc" → "Bash")
├── renderer *glamour.TermRenderer  // reused across calls
└── FormatEvent(*Event) (string, bool)
    ├── system_init → model info line (no timestamp)
    ├── assistant_message → rich formatting:
    │   ├── text blocks → glamour markdown rendering → tview.TranslateANSI()
    │   └── tool_use blocks → "⏺ ToolName(param_summary)" format + record in toolNames
    ├── user_message → tool result rendering:
    │   └── tool_result blocks → look up tool name, show output with line count
    ├── result → success/error indicator
    └── error → error message
```

#### Event Rendering Details

**Assistant Messages — Text Blocks:**

Text content is rendered through glamour's markdown pipeline:

1. Extract text from content blocks (same as current)
2. Render through `glamour.RenderWithEnvironmentConfig()` using dark style
3. Convert ANSI escape codes to tview color tags via `tview.TranslateANSI()`
4. No truncation — full text displayed (the TUI scrolls)

**Assistant Messages — Tool Use Blocks:**

Each tool_use block is rendered as a single line showing the tool name and a parameter summary:

```
  ⏺ Bash(brew install tmux)
  ⏺ Read(/path/to/file.go)
  ⏺ Edit(internal/agent/claude/formatter.go)
  ⏺ Glob(**/*.go)
  ⏺ Write(/path/to/new_file.go)
```

The parameter summary is extracted from the tool's `Input` JSON:
- `Bash` → `command` field
- `Read` → `file_path` field
- `Edit` → `file_path` field
- `Write` → `file_path` field
- `Glob` → `pattern` field
- `Grep` → `pattern` field
- Other tools → first string field value, or tool name only if no suitable field

The tool_use_id and tool_name are recorded in the `toolNames` map for later correlation with tool
results.

**User Messages — Tool Result Blocks:**

Tool results (previously skipped entirely) are now rendered. Each `tool_result` block is correlated
with its tool invocation via `tool_use_id`:

```
  ⏺ Bash ⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯
    +84 lines (success)

  ⏺ Read ⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯
    +142 lines

  ⏺ Bash ⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯
    [red]Error[-] exit code 1
```

After rendering a tool result, its entry is removed from the `toolNames` map (each ID is used
exactly once).

**Result and Error Events:**

These use the same formatting as the current `Formatter` — no changes needed.

#### Glamour Integration

The `glamour.TermRenderer` is created once at `RichFormatter` construction time and reused:

```go
renderer, _ := glamour.NewTermRenderer(
    glamour.WithAutoStyle(),     // detect dark/light terminal
    glamour.WithWordWrap(100),   // wrap at reasonable width
)
```

The renderer produces ANSI escape sequences. Since tview uses its own color tag system
(`[red]text[-]`), the ANSI output must be converted using `tview.TranslateANSI()` before returning
from `FormatEvent`.

#### Parameter Summary Extraction

Tool input is stored as `json.RawMessage`. The formatter unmarshals it into `map[string]interface{}`
and extracts a summary based on tool name:

```go
func extractToolSummary(toolName string, input json.RawMessage) string {
    var params map[string]interface{}
    if err := json.Unmarshal(input, &params); err != nil {
        return ""
    }

    switch toolName {
    case "Bash":
        return stringField(params, "command")
    case "Read":
        return stringField(params, "file_path")
    case "Edit":
        return stringField(params, "file_path")
    case "Write":
        return stringField(params, "file_path")
    case "Glob":
        return stringField(params, "pattern")
    case "Grep":
        return stringField(params, "pattern")
    default:
        // Return first short string field value
        return firstStringField(params)
    }
}
```

Long parameter values are truncated to ~80 characters with `...` suffix.

### Key Decisions

| Decision                                  | Rationale                                                                                                                                    |
|-------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------|
| Replace old formatter (not keep both)     | Single formatter reduces complexity; no use case for the old basic output                                                                   |
| Remove timestamps from output             | TUI header provides timing context; per-event timestamps add visual noise without value                                                      |
| Remove left padding                       | Padding existed to align with timestamp prefix; without timestamps, it's unnecessary                                                         |
| Stateful formatter (tool ID map)          | Required to correlate tool results with their invocations; map cleaned up after each result                                                  |
| `glamour` for markdown rendering          | De facto standard in Go CLI tools (used by `gh`, `glow`); handles headings, bold, lists, code blocks with syntax highlighting out of the box |
| `tview.TranslateANSI()` for conversion    | Built-in tview function specifically designed for this purpose; avoids manual ANSI parsing                                                   |
| Truncate tool results by line count       | Prevents TUI flooding from large file reads or command outputs; shows `+N lines` summary                                                     |
| Extract tool summary from Input JSON      | Gives users actionable context (which file, which command) without showing full JSON                                                         |

### TUI Simplification

#### Log View — Borderless

The log view (`logView`) removes all chrome to feel like raw terminal output:

```go
logView := tview.NewTextView().
    SetDynamicColors(true).
    SetScrollable(true)
// No SetBorder, no SetTitle, no SetBorderPadding
```

This makes agent output flow naturally, matching Claude Code's own appearance where tool calls and
markdown text simply appear inline without being boxed.

#### Responsive Header — Width Breakpoints

The header switches between two rendering modes based on terminal width. The width is read from
`header.GetInnerRect()` (already used today) and checked on each `renderHeader()` call, meaning it
responds to terminal resizes automatically via tview's redraw cycle.

**Wide mode (≥80 columns):** The existing 3-column grid layout is preserved:

```
┌─ Container: my-container ──────────────────────────────────────────────┐
│ Branch:     main        │ Env:        production │ Agent:      opus-4  │
│ Status:     running     │ Memory:     1.2 GB     │ Container:  abc123  │
│ Iterations: 5/100       │ CPU:        12.3%      │ Image:      def456  │
└────────────────────────────────────────────────────────────────────────┘
```

**Compact mode (<80 columns):** Collapses to 1–2 lines showing only essential fields, with the
header box height reduced accordingly (2–3 rows instead of 5):

```
┌─ my-container ─────────────────────────────────┐
│ running │ iter 5/100 │ main │ 12.3% │ 1.2 GB   │
└────────────────────────────────────────────────┘
```

Fields hidden in compact mode: environment, container ID, image ID, agent name. These are
operational details rarely needed during active monitoring.

**Very narrow mode (<40 columns):** Single-line status, minimal header:

```
┌─ my-container ─────────────┐
│ running 5/100 12.3% 1.2GB  │
└────────────────────────────┘
```

#### Implementation Approach

The `renderHeader()` method already reads terminal width and branches on it. The change adds a
width check at the top:

```go
func (ui *WatchUI) renderHeader() {
    _, _, width, _ := ui.header.GetInnerRect()

    if width < 40 {
        ui.renderHeaderMinimal(width)
        return
    }
    if width < 80 {
        ui.renderHeaderCompact(width)
        return
    }
    ui.renderHeaderWide(width)
}
```

The header's `AddItem` height in the layout flex needs to be dynamic. One approach: set a fixed
max height (5 for wide) and let the compact renderers simply use fewer lines within that space.
Alternatively, the header height could be updated on resize, though this adds complexity. The
simpler fixed-height approach (always allocate 5 rows, compact modes just leave blank rows) is
recommended for the initial implementation.

#### Status Bar (Footer)

The current footer ("Press q to quit" in dark gray, right-aligned) is replaced with a solid
vim/tmux-style status bar. This is a full-width single-line bar with an inverted/highlighted
background color that displays keyboard shortcuts.

**Appearance:**

```
 q: quit
▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔
(entire line has a dark background, e.g., gray on black or similar inverted style)
```

**Implementation:** The footer `tview.TextView` uses a background color tag to create the solid
bar effect. In tview, this can be achieved with:

```go
footer := tview.NewTextView().
    SetDynamicColors(true).
    SetTextAlign(tview.AlignLeft)
footer.SetBackgroundColor(tcell.ColorDarkSlateGray)
footer.SetText(" [white]q[darkgray]: quit[-]")
```

The bar spans the full terminal width automatically since it's in a flex layout. The key hint
format uses a brighter color for the key and a dimmer color for the description, matching the
tmux status bar convention.

As more keyboard shortcuts are added (from the `add-watch-mode-shortcuts` change), they would
appear space-separated in this bar: `q: quit  ↑/↓: scroll  f: follow`.
