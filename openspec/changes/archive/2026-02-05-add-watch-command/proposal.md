# Change: Add Watch Command for Real-Time Container Monitoring

## Why

Users need visibility into running containers to monitor task progress, debug issues, and track resource usage.
Currently, users must manually use `docker logs` or `docker stats` to monitor containers. A dedicated watch command
provides an integrated, user-friendly interface for real-time monitoring with structured log parsing and resource
metrics.

## What Changes

- **New `watch` command** - Terminal UI for monitoring container logs and metrics in real-time
- **Container log streaming** - Real-time parsing of JSON logs from `.spinner/{container-name}/logs` directory
- **Resource monitoring** - CPU and memory usage display via Docker API
- **Container state tracking** - Real-time container status (running/stopped/exited)
- **Interactive TUI** - Split-pane interface using tview library showing metrics at top and logs below
- **`--watch` flag for spin** - Optional flag to automatically enter watch mode after container creation

## Impact

- Affected specs: `cli-spin` (MODIFIED), new `cli-watch` capability (ADDED)
- Affected code:
    - `cmd/watch.go` - new file for watch command implementation
    - `cmd/spin.go` - add `--watch` flag support
    - `internal/docker/logs.go` - new file for log parsing and streaming
    - `internal/metrics/provider.go` - new interface for metrics abstraction
    - `internal/metrics/docker.go` - Docker-specific metrics provider implementation
    - `internal/metrics/types.go` - shared metric types
    - `internal/tui/watch.go` - new file for terminal UI rendering
