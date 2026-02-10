# Design: Add Rich Watch Formatter

## Technical Implementation Plan

### Component Map

| File                                           | Action     | Purpose                                                          |
|------------------------------------------------|------------|------------------------------------------------------------------|
| `internal/agent/claude/rich_formatter.go`      | **create** | `RichFormatter` struct + `FormatEvent` implementation            |
| `internal/agent/claude/rich_formatter_test.go` | **create** | Unit tests for all event type formatting                         |
| `cmd/watch.go`                                 | **modify** | Wire `NewRichFormatter()` instead of `NewFormatter()` (line 109) |
| `go.mod` / `go.sum`                            | **modify** | Add `charmbracelet/glamour` dependency                           |

### Approach

#### Architecture

The `RichFormatter` implements the existing `agent.EventFormatter` interface — the same single-method
interface (`FormatEvent(*Event) (string, bool)`) that the current `Formatter` implements. This means:

- No interface changes
- No TUI changes — `WatchUI` already consumes formatted strings via the interface
- Single wiring point: `cmd/watch.go:109` switches from `NewFormatter()` to `NewRichFormatter()`

The key architectural addition is **statefulness**: unlike the current `Formatter` which is
stateless, the `RichFormatter` maintains a `map[string]string` of `tool_use_id → tool_name`. When
an assistant message contains tool_use blocks, the formatter records each tool's ID and name. When a
subsequent user message contains a `tool_result` block referencing that ID, the formatter can display
the tool name alongside the result.

```
RichFormatter
├── toolNames map[string]string  // tool_use_id → tool_name (e.g., "toolu_abc" → "Bash")
├── glamourRenderer *glamour.TermRenderer  // reused across calls
└── FormatEvent(*Event) (string, bool)
    ├── system_init → same as current (model info line)
    ├── assistant_message → rich formatting:
    │   ├── text blocks → glamour markdown rendering → tview.TranslateANSI()
    │   └── tool_use blocks → "ToolName(param_summary)" format + record in toolNames
    ├── user_message → tool result rendering:
    │   └── tool_result blocks → look up tool name, show output with line count
    ├── result → same as current (success/error indicator)
    └── error → same as current (error message)
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
| New `RichFormatter` (not modify existing) | Clean separation; old formatter preserved as reference/fallback; no risk of breaking existing behavior                                       |
| Stateful formatter (tool ID map)          | Required to correlate tool results with their invocations; map cleaned up after each result                                                  |
| `glamour` for markdown rendering          | De facto standard in Go CLI tools (used by `gh`, `glow`); handles headings, bold, lists, code blocks with syntax highlighting out of the box |
| `tview.TranslateANSI()` for conversion    | Built-in tview function specifically designed for this purpose; avoids manual ANSI parsing                                                   |
| Truncate tool results by line count       | Prevents TUI flooding from large file reads or command outputs; shows `+N lines` summary                                                     |
| Extract tool summary from Input JSON      | Gives users actionable context (which file, which command) without showing full JSON                                                         |
| No `--formatter` CLI flag                 | YAGNI — there's no use case for choosing the old formatter; if needed, add later                                                             |
