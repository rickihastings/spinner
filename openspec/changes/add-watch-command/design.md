# Design: Watch Command for Container Monitoring

## Context

The spinner tool creates persistent Docker containers that run tasks asynchronously. Users need real-time visibility
into:

- Container execution logs (currently written to `.spinner/{container-name}/logs/`)
- Container resource usage (CPU, memory)
- Container state (running, stopped, exited)
- Task progress (iteration count, branch info)

The watch command provides a unified TUI for monitoring without requiring users to manually invoke Docker commands or
tail log files.

## Goals / Non-Goals

**Goals:**

- Real-time log streaming with human-readable formatting
- Display container resource metrics (CPU, memory)
- Show container metadata (branch, iteration number)
- Support both standalone `watch` command and integrated `--watch` flag
- Handle log file rotation and container state changes gracefully
- Use channels for concurrent log watching and stats polling

**Non-Goals:**

- Log searching or filtering (future enhancement)
- Multi-container monitoring in single view
- Log persistence or export features
- Container control (start/stop/restart) from watch UI

## Technical Implementation Plan

### Component Map

- `cmd/watch.go` - watch command implementation (create)
- `cmd/constructors.go` - watch command constructor (modify)
- `cmd/spin.go` - add --watch flag (modify)
- `internal/docker/logs.go` - log file monitoring and parsing (create)
- `internal/metrics/provider.go` - MetricsProvider interface abstraction (create)
- `internal/metrics/docker.go` - Docker-specific MetricsProvider implementation (create)
- `internal/metrics/types.go` - shared metric types (ContainerMetrics, ContainerState) (create)
- `internal/tui/watch.go` - tview TUI implementation (create)
- `tests/integration/watch_test.go` - integration tests (create)
- `internal/docker/logs_test.go` - unit tests for log parsing (create)
- `internal/metrics/docker_test.go` - unit tests for Docker metrics provider (create)

### Approach

**Phase 1: Log Infrastructure**

1. Implement file watching using `fsnotify` library for `.spinner/{container-name}/logs/` directory
2. Create JSON log parser to extract structured log data (timestamp, level, message, iteration)
3. Use channels to stream parsed log entries

**Phase 2: Metrics Abstraction and Stats Integration**

1. Define `MetricsProvider` interface in `internal/metrics/provider.go`:
   ```go
   type MetricsProvider interface {
       // StreamMetrics streams real-time metrics for a container to the provided channel
       StreamMetrics(ctx context.Context, containerName string, metricsCh chan<- ContainerMetrics) error
   }
   ```
2. Define shared metric types in `internal/metrics/types.go` (ContainerMetrics, ContainerState)
3. Implement Docker-specific provider in `internal/metrics/docker.go` using Docker SDK
4. Docker implementation polls stats every 1-2 seconds and sends to channel
5. Future cloud providers (ECS, Cloud Run, etc.) can implement same interface

**Phase 3: TUI Implementation**

1. Create split-pane layout using `tview` library
    - Top section: container metadata + resource metrics
    - Bottom section: scrolling log view
2. Handle terminal resizing and keyboard input (q to quit)
3. Auto-scroll logs with option to pause

**Phase 4: Command Integration**

1. Create standalone `watch` command accepting container name
2. Add `--watch` flag to `spin` command to transition to watch mode after creation
3. Share TUI implementation between both entry points

### Patterns to Follow

- **Command structure**: Follow existing pattern in `cmd/spin.go` and `cmd/setup.go` with constructor-based approach for
  testability
- **Docker client**: Use `internal/docker/client.go` pattern with interface abstraction for mocking
- **Error handling**: Match existing error reporting style with `fmt.Fprintf(os.Stderr, "✗ Error: %s\n", err.Error())`
- **Context usage**: Pass `context.Context` through all operations like in `cmd/constructors.go`

### Key Decisions

**Decision**: Use `fsnotify` instead of `tail -f` subprocess
**Rationale**: Native Go library provides better error handling, cross-platform support, and integrates cleanly with
channels. Avoids subprocess management complexity.

**Decision**: Abstract metrics collection behind `MetricsProvider` interface
**Rationale**: Enables future support for cloud-based container platforms (ECS, Cloud Run, Kubernetes) without changing TUI or command code. Docker is just one implementation. Interface uses channels for real-time streaming, making it easy to swap providers.

**Decision**: Use `github.com/docker/docker/client` for Docker implementation
**Rationale**: Go SDK provides structured data, better error handling, and consistent with future API integrations.

**Decision**: Implement TUI with `tview` library
**Rationale**: Suggested in requirements, mature library with good examples, handles terminal complexity.

**Decision**: Poll stats every 1-2 seconds
**Rationale**: Balances responsiveness with API overhead. Docker stats stream would be more efficient but adds
complexity.

**Decision**: Share TUI implementation between standalone watch and spin --watch
**Rationale**: Reduces duplication, ensures consistent UX. Both paths call same `RunWatch` function with container name.

**Decision**: Make --watch flag mutually compatible with all spin flags
**Rationale**: Users may want to watch when creating/recreating containers or reusing existing ones.

**Decision**: Use channel-based streaming from MetricsProvider
**Rationale**: Channels provide clean concurrency model in Go. TUI consumes from metrics channel in real-time. Provider implementation controls polling/streaming logic. Shutdown via context cancellation.

## Risks / Trade-offs

**Risk**: Log file doesn't exist yet when container starts
**Mitigation**: Wait for file creation with timeout, display friendly message while waiting

**Risk**: Large log files causing memory issues
**Mitigation**: Only load recent N lines on startup, stream new lines incrementally

**Risk**: Container stops/exits during watch
**Mitigation**: Detect state change via stats polling, display message and exit gracefully

**Trade-off**: Polling stats vs streaming stats API
**Choice**: Polling for simplicity in v1, can optimize with streaming later

**Trade-off**: fsnotify complexity with file rotation
**Choice**: Accept potential edge case issues, document limitation if needed

## Open Questions

None - requirements are clear from prompt.md
