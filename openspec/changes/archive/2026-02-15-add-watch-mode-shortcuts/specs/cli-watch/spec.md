# cli-watch Specification Delta

## MODIFIED Requirements

### Requirement: Keyboard Controls

The CLI SHALL support keyboard input for controlling the watch interface.

#### Scenario: Quit with 'q' key

- **WHEN** user presses 'q' key
- **THEN** the CLI SHALL exit watch mode and return to normal terminal

#### Scenario: Quit with Ctrl+C

- **WHEN** user presses Ctrl+C
- **THEN** the CLI SHALL exit watch mode and return to normal terminal

#### Scenario: Scroll log view up by one line

- **WHEN** user presses the Up Arrow key
- **THEN** the log view SHALL scroll up by one line
- **AND** auto-scroll SHALL be paused

#### Scenario: Scroll log view down by one line

- **WHEN** user presses the Down Arrow key
- **THEN** the log view SHALL scroll down by one line
- **AND** if the log view has reached the bottom, auto-scroll SHALL resume

#### Scenario: Scroll log view up by one page

- **WHEN** user presses the Page Up key
- **THEN** the log view SHALL scroll up by one page (the visible height of the log panel)
- **AND** auto-scroll SHALL be paused

#### Scenario: Scroll log view down by one page

- **WHEN** user presses the Page Down key
- **THEN** the log view SHALL scroll down by one page (the visible height of the log panel)
- **AND** if the log view has reached the bottom, auto-scroll SHALL resume

#### Scenario: Scroll to top of logs

- **WHEN** user presses the Home key
- **THEN** the log view SHALL scroll to the very first log line
- **AND** auto-scroll SHALL be paused

#### Scenario: Scroll to bottom of logs

- **WHEN** user presses the End key
- **THEN** the log view SHALL scroll to the latest log line
- **AND** auto-scroll SHALL resume

#### Scenario: Toggle header visibility

- **WHEN** user presses 'h' key
- **THEN** the header panel SHALL toggle between visible and hidden
- **AND** the log view SHALL expand or contract to fill the available space

#### Scenario: Header default from configuration

- **WHEN** `watch-header` is set to `false` in `.spinner.json` or `SPINNER_WATCH_HEADER` is set to `false`
- **THEN** the header panel SHALL be hidden when watch mode starts
- **AND** the user SHALL still be able to toggle it with 'h' key

#### Scenario: Header default without configuration

- **WHEN** no `watch-header` configuration is present
- **THEN** the header panel SHALL be visible when watch mode starts (default: `true`)

#### Scenario: Show help overlay

- **WHEN** user presses '?' key and the help overlay is not visible
- **THEN** a help overlay SHALL appear centered over the log view listing all keyboard shortcuts

#### Scenario: Dismiss help overlay

- **WHEN** the help overlay is visible and user presses any key
- **THEN** the help overlay SHALL be dismissed and the key SHALL be consumed

### Requirement: Auto-Scroll Behavior

The CLI SHALL auto-scroll the log view as new entries arrive, with the ability to pause when the user scrolls.

#### Scenario: Auto-scroll enabled by default

- **WHEN** new log entries arrive and the user has not scrolled away from the bottom
- **THEN** the log view SHALL automatically scroll to show the latest entry

#### Scenario: Auto-scroll pauses when user scrolls up

- **WHEN** user scrolls up via Up Arrow, Page Up, or Home key
- **THEN** auto-scroll SHALL pause and new log entries SHALL accumulate without moving the viewport

#### Scenario: Auto-scroll resumes when user returns to bottom

- **WHEN** user presses End key or scrolls down to the bottom via Down Arrow or Page Down
- **THEN** auto-scroll SHALL resume and the log view SHALL jump to the latest entry

#### Scenario: Scroll state indicator in footer

- **WHEN** auto-scroll is paused because the user has scrolled up
- **THEN** the footer SHALL display a visible `SCROLLED` indicator

#### Scenario: Scroll state indicator clears on resume

- **WHEN** auto-scroll resumes after the user returns to the bottom
- **THEN** the `SCROLLED` indicator SHALL be removed from the footer

### Requirement: TUI Layout

The watch command SHALL display a split-pane terminal UI with container metadata and metrics at the top and logs at the
bottom. The header panel SHALL be toggleable. The initial header visibility SHALL be configurable via `.spinner.json`
(`watch-header`) or environment variable (`SPINNER_WATCH_HEADER`), defaulting to visible.

#### Scenario: TUI renders on terminal

- **WHEN** watch mode is active
- **THEN** the terminal SHALL display a header section with container name, branch, iteration count, CPU usage, and
  memory usage
- **AND** the terminal SHALL display a scrolling log section below the header

#### Scenario: TUI handles terminal resize

- **WHEN** the terminal window is resized during watch mode
- **THEN** the TUI SHALL automatically adjust layout to fit new dimensions

#### Scenario: TUI handles small terminal

- **WHEN** the terminal is too small to display both sections
- **THEN** the TUI SHALL prioritize log display and truncate or hide header elements

#### Scenario: Header hidden by user

- **WHEN** the user has toggled the header off with 'h' key
- **THEN** the log view SHALL occupy the full terminal height (minus footer)
- **AND** the header SHALL not be rendered

#### Scenario: Header restored by user

- **WHEN** the user presses 'h' key while the header is hidden
- **THEN** the header SHALL reappear at its original fixed height
- **AND** the log view SHALL contract to accommodate it

## ADDED Requirements

### Requirement: Header Default Configuration

The CLI SHALL allow the default header visibility to be configured via `.spinner.json` or environment variable, without
a CLI flag.

#### Scenario: Configure via .spinner.json

- **WHEN** `.spinner.json` contains `"watch-header": false`
- **THEN** the header panel SHALL default to hidden when watch mode starts

#### Scenario: Configure via environment variable

- **WHEN** `SPINNER_WATCH_HEADER` is set to `false`
- **THEN** the header panel SHALL default to hidden when watch mode starts

#### Scenario: Environment variable overrides .spinner.json

- **WHEN** `.spinner.json` contains `"watch-header": true` and `SPINNER_WATCH_HEADER` is set to `false`
- **THEN** the header panel SHALL default to hidden (env var takes precedence)

#### Scenario: No configuration present

- **WHEN** neither `.spinner.json` nor `SPINNER_WATCH_HEADER` specifies a value
- **THEN** the header panel SHALL default to visible (`true`)

#### Scenario: Runtime toggle unaffected by default

- **WHEN** the header default is configured to hidden
- **AND** the user presses 'h' key
- **THEN** the header panel SHALL become visible (toggle still works normally)

### Requirement: Help Overlay

The CLI SHALL provide a help overlay that displays all available keyboard shortcuts.

#### Scenario: Help overlay content

- **WHEN** the help overlay is displayed
- **THEN** it SHALL list all keyboard shortcuts with their key bindings and descriptions

#### Scenario: Help overlay positioning

- **WHEN** the help overlay is displayed
- **THEN** it SHALL appear centered over the main layout as a bordered modal

#### Scenario: Help overlay does not consume log data

- **WHEN** the help overlay is visible
- **THEN** log entries and metrics SHALL continue to be consumed and buffered in the background

### Requirement: Footer Help Text

The CLI SHALL display contextual help text and state indicators in the footer.

#### Scenario: Default footer text

- **WHEN** watch mode is active and auto-scroll is engaged
- **THEN** the footer SHALL display a summary of available keyboard shortcuts

#### Scenario: Footer shows scroll indicator

- **WHEN** auto-scroll is paused
- **THEN** the footer SHALL display a `SCROLLED` indicator alongside the shortcut summary
