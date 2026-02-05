# cli-watch Specification

## Purpose
TBD - created by archiving change add-watch-command. Update Purpose after archive.
## Requirements
### Requirement: Watch Command Invocation

The CLI SHALL provide a `watch` command that accepts a container name and displays real-time monitoring information.

#### Scenario: Watch with valid container name

- **WHEN** user runs `spinner watch <container-name>`
- **THEN** the CLI SHALL enter TUI mode and display container logs and metrics

#### Scenario: Watch with missing container name

- **WHEN** user runs `spinner watch` without a container name argument
- **THEN** the CLI SHALL print an error message and exit with non-zero status

#### Scenario: Watch with non-existent container

- **WHEN** user runs `spinner watch <non-existent-container>`
- **THEN** the CLI SHALL print an error message indicating the container does not exist and exit

### Requirement: TUI Layout

The watch command SHALL display a split-pane terminal UI with container metadata and metrics at the top and logs at the
bottom.

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

### Requirement: Container Metadata Display

The CLI SHALL display container metadata in the TUI header section.

#### Scenario: Display branch name

- **WHEN** the container was created with a --branch flag
- **THEN** the header SHALL display "Branch: <branch-name>"

#### Scenario: Display iteration number

- **WHEN** ralph-loop is running and logs contain iteration information
- **THEN** the header SHALL display "Iteration: <current>/<max>"

#### Scenario: No branch specified

- **WHEN** the container was created without a --branch flag
- **THEN** the header SHALL display "Branch: default" or omit the field

### Requirement: Resource Metrics Display

The CLI SHALL consume metrics from the MetricsProvider and display CPU and memory usage in the TUI header.

#### Scenario: Display CPU usage

- **WHEN** metrics are received from the provider
- **THEN** the header SHALL display "CPU: <percentage>%"

#### Scenario: Display memory usage

- **WHEN** metrics are received from the provider
- **THEN** the header SHALL display "Memory: <used>/<limit> (<percentage>%)"

#### Scenario: Metrics update frequency

- **WHEN** watch mode is active
- **THEN** the TUI SHALL update metrics display as new data arrives on the metrics channel

#### Scenario: Container stopped during watch

- **WHEN** the provider sends a stopped/exited state via metrics channel
- **THEN** the CLI SHALL detect the state change, display "Container stopped" message, and exit watch mode

### Requirement: Log File Monitoring

The CLI SHALL monitor the container's log file at `.spinner/<container-name>/logs/` and stream new entries to the TUI.

#### Scenario: Tail existing logs on startup

- **WHEN** watch mode starts and log file exists
- **THEN** the CLI SHALL display the last N lines of existing logs (e.g., 50 lines)

#### Scenario: Stream new log entries

- **WHEN** new log entries are written to the log file
- **THEN** the CLI SHALL display them in real-time in the log section

#### Scenario: Log file does not exist yet

- **WHEN** watch mode starts but log file does not exist
- **THEN** the CLI SHALL display "Waiting for logs..." and poll for file creation

#### Scenario: Log file appears after wait

- **WHEN** log file is created while watch mode is waiting
- **THEN** the CLI SHALL begin streaming logs immediately

### Requirement: Log Parsing

The CLI SHALL parse JSON-formatted log entries and display them in human-readable format.

#### Scenario: Parse JSON log entry

- **WHEN** a log entry is in JSON format with timestamp, level, and message fields
- **THEN** the CLI SHALL format it as "[timestamp] [level] message"

#### Scenario: Parse non-JSON log entry

- **WHEN** a log entry is not valid JSON
- **THEN** the CLI SHALL display it as-is without parsing

#### Scenario: Extract iteration from logs

- **WHEN** a log entry contains iteration information (e.g., "Iteration 5/100")
- **THEN** the CLI SHALL update the header iteration counter

### Requirement: File Watching Implementation

The CLI SHALL use the fsnotify library to watch for log file changes.

#### Scenario: Detect new log lines with fsnotify

- **WHEN** the log file is modified
- **THEN** fsnotify SHALL trigger a watch event and the CLI SHALL read new lines

#### Scenario: Handle fsnotify errors

- **WHEN** fsnotify encounters an error (e.g., file deleted)
- **THEN** the CLI SHALL handle the error gracefully and exit watch mode with a message

### Requirement: Keyboard Controls

The CLI SHALL support keyboard input for controlling the watch interface.

#### Scenario: Quit with 'q' key

- **WHEN** user presses 'q' key
- **THEN** the CLI SHALL exit watch mode and return to normal terminal

#### Scenario: Quit with Ctrl+C

- **WHEN** user presses Ctrl+C
- **THEN** the CLI SHALL exit watch mode and return to normal terminal

#### Scenario: Scroll logs (future enhancement)

- **WHEN** user presses arrow keys or page up/down
- **THEN** the CLI SHALL scroll the log view (not required for initial implementation)

### Requirement: Auto-Scroll Behavior

The CLI SHALL auto-scroll the log view as new entries arrive.

#### Scenario: Auto-scroll enabled by default

- **WHEN** new log entries arrive
- **THEN** the log view SHALL automatically scroll to show the latest entry

#### Scenario: Auto-scroll pauses when user scrolls up

- **WHEN** user manually scrolls up in the log view
- **THEN** auto-scroll SHALL pause until user returns to bottom (future enhancement, not required for v1)

### Requirement: Channel-Based Architecture

The CLI SHALL use Go channels for concurrent log watching and stats polling.

#### Scenario: Log entries sent via channel

- **WHEN** fsnotify detects a log file change
- **THEN** parsed log entries SHALL be sent to a channel consumed by the TUI renderer

#### Scenario: Stats sent via channel

- **WHEN** stats polling retrieves new data
- **THEN** stats data SHALL be sent to a channel consumed by the TUI renderer

#### Scenario: Graceful shutdown via channel

- **WHEN** user quits watch mode
- **THEN** a shutdown signal SHALL be sent via channel to stop all goroutines

### Requirement: MetricsProvider Abstraction

The CLI SHALL define a MetricsProvider interface to abstract container metrics collection from the underlying infrastructure (Docker, cloud platforms, etc.).

#### Scenario: MetricsProvider interface definition

- **WHEN** the metrics provider is initialized
- **THEN** it SHALL implement a StreamMetrics method that accepts context, container name, and metrics channel

#### Scenario: Channel-based metrics streaming

- **WHEN** StreamMetrics is called
- **THEN** the provider SHALL send ContainerMetrics to the provided channel in real-time

#### Scenario: Provider shutdown via context

- **WHEN** the context is cancelled
- **THEN** the provider SHALL stop streaming metrics and clean up resources

### Requirement: Docker MetricsProvider Implementation

The CLI SHALL provide a Docker-specific implementation of the MetricsProvider interface using the Docker SDK.

#### Scenario: Docker provider initialization

- **WHEN** watch mode starts with a Docker container
- **THEN** the CLI SHALL create a Docker SDK client using `github.com/docker/docker/client`

#### Scenario: Docker provider polls stats

- **WHEN** Docker metrics provider is streaming
- **THEN** it SHALL poll ContainerStats API every 1-2 seconds and send results to metrics channel

#### Scenario: Docker provider handles API errors

- **WHEN** Docker stats API returns an error (e.g., container not found)
- **THEN** the provider SHALL send an error via the metrics channel and stop streaming

### Requirement: Error Handling

The CLI SHALL handle errors gracefully and provide clear error messages.

#### Scenario: Missing log directory

- **WHEN** `.spinner/<container-name>/logs/` directory does not exist
- **THEN** the CLI SHALL display an error message and exit

#### Scenario: Permission errors on log file

- **WHEN** log file exists but is not readable
- **THEN** the CLI SHALL display a permission error and exit

#### Scenario: Docker daemon not available

- **WHEN** Docker daemon is not running or not accessible
- **THEN** the CLI SHALL display "Docker daemon not available" and exit

### Requirement: Testability

The watch command SHALL have unit tests for log parsing and integration tests for end-to-end watch functionality.

#### Scenario: Unit test for JSON log parsing

- **GIVEN** a JSON log entry string
- **WHEN** the parser processes it
- **THEN** the test SHALL verify correct extraction of timestamp, level, and message

#### Scenario: Unit test for non-JSON log handling

- **GIVEN** a non-JSON log entry string
- **WHEN** the parser processes it
- **THEN** the test SHALL verify it is returned as-is

#### Scenario: Integration test for watch command

- **GIVEN** a running container with log output
- **WHEN** `spinner watch <container-name>` is executed in test mode
- **THEN** the test SHALL verify logs are displayed (possibly using headless mode or output capture)

