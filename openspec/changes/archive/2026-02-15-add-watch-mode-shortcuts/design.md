# Design: Add Watch Mode Shortcuts

## Technical Implementation Plan

### Component Map

| File | Action | Purpose |
|------|--------|---------|
| `internal/tui/watch.go` | modify | Add scrolling state, keyboard handlers, header toggle, help overlay, footer updates |
| `internal/tui/watch_test.go` | modify | Add tests for new keyboard handling, scroll state, header toggle, overlay logic |
| `cmd/watch.go` | modify | Read `watch-header` from Viper and pass to `NewWatchUI` |

### Approach

All changes are scoped to `internal/tui/watch.go`. No new files, interfaces, or dependencies are needed. The
implementation builds on existing `tview` capabilities (`SetInputCapture`, `ScrollTo`, flex layout manipulation).

#### 1. Auto-Scroll State Management

Add a `userScrolled bool` field to `WatchUI`. This tracks whether the user has scrolled away from the bottom.

- **Current behavior**: `logView.SetChangedFunc` unconditionally calls `ScrollToEnd()` on every new log line.
- **New behavior**: The `SetChangedFunc` callback checks `userScrolled`. If `true`, it skips `ScrollToEnd()` so the
  user's scroll position is preserved. New log lines still accumulate in the buffer — the user just isn't yanked to the
  bottom.
- **Re-engage**: When the user presses `End` (or scrolls to the bottom via `Page Down`), set `userScrolled = false` so
  auto-scroll resumes.
- **Detecting bottom**: After a `Page Down` or `Down` key, check if the scroll offset has reached the end of the
  content. If so, clear `userScrolled`.

#### 2. Keyboard Handlers

Extend `setupKeyboardHandlers()` to handle additional keys via `SetInputCapture`:

| Key | `tcell` constant / rune | Action |
|-----|------------------------|--------|
| Up Arrow | `tcell.KeyUp` | Scroll log view up 1 line, set `userScrolled = true` |
| Down Arrow | `tcell.KeyDown` | Scroll log view down 1 line, clear `userScrolled` if at bottom |
| Page Up | `tcell.KeyPgUp` | Scroll log view up 1 page, set `userScrolled = true` |
| Page Down | `tcell.KeyPgDn` | Scroll log view down 1 page, clear `userScrolled` if at bottom |
| Home | `tcell.KeyHome` | Scroll to top of log view, set `userScrolled = true` |
| End | `tcell.KeyEnd` | Scroll to bottom, set `userScrolled = false` |
| `h` | `event.Rune() == 'h'` | Toggle header visibility |
| `?` | `event.Rune() == '?'` | Toggle help overlay |

**Scroll implementation**: `tview.TextView` provides `ScrollTo(row, col)`, `ScrollToEnd()`, `ScrollToBeginning()`, and
`GetScrollOffset() (row, col)`. For line/page scrolling, read current offset with `GetScrollOffset()`, compute the new
row (±1 for line, ±page height for page), and call `ScrollTo(newRow, 0)`.

**Page height calculation**: Use `logView.GetInnerRect()` to get the visible height of the log view, then scroll by
that many rows for page up/down.

#### 3. Header Default Configuration

The initial value of `headerVisible` is driven by configuration rather than being hardcoded to `true`.

- **Viper key**: `watch-header` (accessed via `viper.GetBool("watch-header")`)
- **Env var**: `SPINNER_WATCH_HEADER` (automatic via Viper's `SPINNER_` prefix + `AutomaticEnv()`)
- **`.spinner.json`**: `{ "watch-header": false }`
- **Default**: `true` (set via `viper.SetDefault("watch-header", true)` in `cmd/root.go`)
- **No CLI flag**: This is intentionally not exposed as a `--watch-header` flag. It's a user/team preference setting,
  not a per-invocation option. The Viper key is registered with `SetDefault` only, no flag binding.

In `cmd/watch.go`, the `performWatch` function reads `viper.GetBool("watch-header")` and passes it to `NewWatchUI`
(or via a config/options struct). The TUI initializes `headerVisible` to this value.

#### 4. Header Toggle

Add a `headerVisible bool` field to `WatchUI`, initialized from configuration (see §3 above).

- **Toggle on `h`**: Toggle `headerVisible` and remove or re-add the header from the flex layout.
  - Hide: `ui.layout.RemoveItem(ui.header)`
  - Show: Use `tview.Flex.AddItemAtIndex` or rebuild the layout with the header re-inserted at index 0.
  - Since `tview.Flex` doesn't have `AddItemAtIndex`, the simplest approach is to clear and rebuild the flex:
    ```
    ui.layout.Clear()
    if headerVisible: ui.layout.AddItem(header, 5, 0, false)
    ui.layout.AddItem(logView, 0, 1, true)
    ui.layout.AddItem(footer, 1, 0, false)
    ```
- **Footer text update**: When header is hidden, update footer to show `h: show header` instead of `h: hide header`.

#### 5. Help Overlay

Add a `helpVisible bool` field and a `helpOverlay *tview.TextView` component.

- **Show on `?`**: Create a centered `tview.TextView` with border, listing all shortcuts. Use `tview.Pages` to layer it
  over the main layout, or use a `tview.Flex` centered approach.
- **Approach**: Switch the app root to a `tview.Pages` component at initialization:
  - Page "main": the existing flex layout
  - Page "help": a modal-style overlay using `tview.Flex` centering (horizontal and vertical flex with empty spacers)
- **Dismiss**: Any keypress while the overlay is visible dismisses it. The `SetInputCapture` handler checks
  `helpVisible` first — if true, hide the overlay and consume the key.

**Help overlay content:**
```
 Keyboard Shortcuts

 ↑/↓         Scroll line
 PgUp/PgDn   Scroll page
 Home/End     Top/Bottom
 h            Toggle header
 ?            This help
 q            Quit
```

#### 6. Footer Text Updates

The footer currently shows a static `Press q to quit`. Update it dynamically based on state:

- **Default**: `[darkgray]↑↓ scroll · h header · ? help · q quit[-]`
- **When scrolled**: `[yellow]SCROLLED[-] [darkgray]· ↑↓ scroll · h header · ? help · q quit[-]`

This keeps the footer to a single line and provides both discoverability and scroll-state feedback.

### Key Decisions

1. **Standard terminal keys over vim keys**: The user explicitly requested standard terminal-style bindings (Arrow,
   PgUp/PgDn, Home/End). Vim-style keys (j/k/g/G) are not included to keep scope minimal.
2. **`tview.Pages` for overlay**: Using `tview.Pages` is the idiomatic way to layer views in tview. It avoids complex
   z-ordering or custom draw logic.
3. **Rebuild flex on toggle**: Since `tview.Flex` lacks `InsertItem`, rebuilding the flex with 2-3 items on toggle is
   the simplest and most readable approach. The cost is negligible (3 `AddItem` calls).
4. **No separate scroll-lock key**: Auto-scroll state is managed implicitly by directional keys rather than an explicit
   toggle. Scrolling up pauses auto-scroll; reaching the bottom resumes it. This matches common terminal emulator
   behavior (e.g., iTerm2, Windows Terminal).
5. **Config-only header default, no CLI flag**: The header visibility default is a user/team preference (e.g., "I always
   want headers off") rather than a per-invocation option. Exposing it via `.spinner.json` and `SPINNER_WATCH_HEADER`
   fits the existing config precedence model. A `--watch-header` flag would clutter the command signature for something
   that rarely changes between runs. The `h` key remains the primary interaction for toggling during a session.
