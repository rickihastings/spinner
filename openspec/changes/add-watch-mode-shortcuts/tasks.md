# Tasks: Add Watch Mode Shortcuts

## 1.0 Log Scrolling with Auto-Scroll Management

Add keyboard-driven log scrolling and smart auto-scroll pause/resume.

- [ ] 1.1 Add `userScrolled bool` field to `WatchUI` struct
- [ ] 1.2 Modify `SetChangedFunc` callback to check `userScrolled` before calling `ScrollToEnd()`
- [ ] 1.3 Add scroll key handlers (Up, Down, PgUp, PgDn, Home, End) to `setupKeyboardHandlers()`
- [ ] 1.4 Implement page-height calculation using `logView.GetInnerRect()` for PgUp/PgDn
- [ ] 1.5 Implement bottom-detection logic: after Down/PgDn, check scroll offset to determine if at bottom and clear `userScrolled`
- [ ] 1.6 Update footer text to show `SCROLLED` indicator when `userScrolled` is true
- [ ] 1.7 Add unit tests for scroll state transitions (up pauses, bottom resumes, End resumes)
- [ ] 1.8 Add unit tests for footer text content based on scroll state
- [ ] 1.9 Verify build passes and all tests pass

## 2.0 Header Toggle

Add `h` key to toggle the header panel visibility.

- [ ] 2.1 Add `headerVisible bool` field (default `true`) to `WatchUI` struct
- [ ] 2.2 Add `h` key handler to `setupKeyboardHandlers()` that calls a `toggleHeader()` method
- [ ] 2.3 Implement `toggleHeader()`: clear flex layout, rebuild with or without header, re-set app root
- [ ] 2.4 Add unit tests for header toggle state transitions
- [ ] 2.5 Verify build passes and all tests pass

## 3.0 Help Overlay

Add `?` key to show a centered help overlay listing all shortcuts.

- [ ] 3.1 Switch app root from `tview.Flex` to `tview.Pages` containing the main layout and a help page
- [ ] 3.2 Create help overlay content as a bordered, centered `tview.TextView` listing all shortcuts
- [ ] 3.3 Add `helpVisible bool` field and `?` key handler to toggle overlay page visibility
- [ ] 3.4 Add dismiss-on-any-key logic: when `helpVisible` is true, any keypress hides the overlay
- [ ] 3.5 Add unit tests for help overlay state transitions (show, dismiss)
- [ ] 3.6 Verify build passes and all tests pass
