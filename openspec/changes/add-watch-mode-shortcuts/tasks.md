# Tasks: Add Watch Mode Shortcuts

## 1.0 Log Scrolling with Auto-Scroll Management

Add keyboard-driven log scrolling and smart auto-scroll pause/resume.

- [x] 1.1 Add `userScrolled bool` field to `WatchUI` struct
- [x] 1.2 Modify `SetChangedFunc` callback to check `userScrolled` before calling `ScrollToEnd()`
- [x] 1.3 Add scroll key handlers (Up, Down, PgUp, PgDn, Home, End) to `setupKeyboardHandlers()`
- [x] 1.4 Implement page-height calculation using `logView.GetInnerRect()` for PgUp/PgDn
- [x] 1.5 Implement bottom-detection logic: after Down/PgDn, check scroll offset to determine if at bottom and clear `userScrolled`
- [x] 1.6 Update footer text to show `SCROLLED` indicator when `userScrolled` is true
- [x] 1.7 Add unit tests for scroll state transitions (up pauses, bottom resumes, End resumes)
- [x] 1.8 Add unit tests for footer text content based on scroll state
- [x] 1.9 Verify build passes and all tests pass

## 2.0 Header Toggle with Configurable Default

Add `h` key to toggle the header panel visibility, with the initial state driven by configuration.

- [x] 2.1 Register `viper.SetDefault("watch-header", true)` in `cmd/root.go`
- [x] 2.2 Read `viper.GetBool("watch-header")` in `cmd/watch.go` and pass to `NewWatchUI`
- [x] 2.3 Add `headerVisible bool` field to `WatchUI` struct, initialized from the config value
- [x] 2.4 Add `h` key handler to `setupKeyboardHandlers()` that calls a `toggleHeader()` method
- [x] 2.5 Implement `toggleHeader()`: clear flex layout, rebuild with or without header, re-set app root
- [x] 2.6 Add unit tests for header toggle state transitions (including starting hidden)
- [x] 2.7 Add `.env.example` entry for `SPINNER_WATCH_HEADER`
- [x] 2.8 Verify build passes and all tests pass

## 3.0 Help Overlay

Add `?` key to show a centered help overlay listing all shortcuts.

- [ ] 3.1 Switch app root from `tview.Flex` to `tview.Pages` containing the main layout and a help page
- [ ] 3.2 Create help overlay content as a bordered, centered `tview.TextView` listing all shortcuts
- [ ] 3.3 Add `helpVisible bool` field and `?` key handler to toggle overlay page visibility
- [ ] 3.4 Add dismiss-on-any-key logic: when `helpVisible` is true, any keypress hides the overlay
- [ ] 3.5 Add unit tests for help overlay state transitions (show, dismiss)
- [ ] 3.6 Verify build passes and all tests pass
