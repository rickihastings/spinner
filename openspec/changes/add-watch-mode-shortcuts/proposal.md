# Proposal: Add Watch Mode Shortcuts

## Summary

Add keyboard shortcuts to the watch mode TUI for log scrolling, header toggle, and a help overlay. Currently the only
keyboard controls are `q`/`Q` and `Ctrl+C` to quit. Users cannot scroll back through logs or reclaim screen space by
hiding the header panel.

## Motivation

When monitoring long-running agent sessions, users need to:

1. **Review earlier log output** — auto-scroll currently forces the view to the bottom with no way to scroll back.
2. **Maximize log visibility** — the 5-line header panel is useful for status checks but wastes space when the user
   just wants to watch logs.
3. **Discover available controls** — there is no discoverability mechanism for keyboard shortcuts.

## Changes

### 1. Log Scrolling with Auto-Scroll Management

Add standard terminal-style navigation keys (Arrow Up/Down, Page Up/Down, Home/End) for scrolling the log view.
Auto-scroll (which currently always fires via `ScrollToEnd()`) will disengage when the user scrolls up and re-engage
when they return to the bottom. A `[SCROLLED]` indicator in the footer signals when auto-scroll is paused.

### 2. Header Toggle

Add `h` key to toggle the header panel visibility. When hidden, the log view expands to fill the freed space. The
footer help text updates to reflect the toggle state.

### 3. Help Overlay

Add `?` key to show/dismiss a small overlay listing all available keyboard shortcuts. The overlay appears centered over
the log view and dismisses on any keypress.

## Impact

- **Specs affected**: `cli-watch` (Keyboard Controls, Auto-Scroll Behavior, TUI Layout)
- **Code affected**: `internal/tui/watch.go` (keyboard handlers, layout management, new overlay component)
- **No breaking changes** — all new behavior is additive; default experience (auto-scroll, header visible) is unchanged.
