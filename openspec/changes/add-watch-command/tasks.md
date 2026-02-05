# Implementation Tasks

## 1.0 Implement log file monitoring and parsing

- [x] 1.1 Add `fsnotify` dependency to go.mod
- [x] 1.2 Create `internal/docker/logs.go` with LogWatcher type and file watching logic
- [x] 1.3 Implement JSON log parsing function with fallback for non-JSON lines
- [x] 1.4 Implement channel-based log streaming architecture
- [x] 1.5 Add unit tests in `internal/docker/logs_test.go` for parsing and file watching
- [x] 1.6 Verify tests pass with `go test ./internal/docker/...`

## 2.0 Implement metrics abstraction and Docker provider

- [x] 2.1 Add Docker SDK client dependency to go.mod (`github.com/docker/docker/client`)
- [x] 2.2 Create `internal/metrics/types.go` with ContainerMetrics and ContainerState types
- [x] 2.3 Create `internal/metrics/provider.go` with MetricsProvider interface definition
- [x] 2.4 Create `internal/metrics/docker.go` with Docker-specific MetricsProvider implementation
- [x] 2.5 Implement StreamMetrics method that polls Docker stats every 1-2 seconds
- [x] 2.6 Implement CPU and memory metric extraction from Docker API and send to channel
- [x] 2.7 Implement container state detection (running/stopped/exited) and send to channel
- [x] 2.8 Add unit tests in `internal/metrics/docker_test.go` with mocked Docker client
- [x] 2.9 Verify tests pass with `go test ./internal/metrics/...`

## 3.0 Implement TUI rendering with tview

- [x] 3.1 Add `tview` dependency to go.mod (`github.com/rivo/tview`)
- [x] 3.2 Create `internal/tui/watch.go` with TUI rendering logic
- [x] 3.3 Implement split-pane layout (header + log view)
- [x] 3.4 Implement header rendering for container metadata and metrics
- [x] 3.5 Implement log view with auto-scrolling
- [x] 3.6 Implement keyboard handlers (q and Ctrl+C to quit)
- [x] 3.7 Implement channel consumers for logs and metrics updates (from MetricsProvider)
- [x] 3.8 Add graceful shutdown logic with context cancellation
- [x] 3.9 Build and manually test TUI rendering: `go build -o dist/spinner && ./dist/spinner watch <test-container>`

## 4.0 Implement standalone watch command

- [x] 4.1 Create `cmd/watch.go` with NewWatchCommand constructor
- [x] 4.2 Implement command argument validation (require container name)
- [x] 4.3 Implement container existence check using Docker client
- [x] 4.4 Implement log directory existence check
- [x] 4.5 Create Docker MetricsProvider instance for the container
- [x] 4.6 Integrate LogWatcher, MetricsProvider, and TUI components with channels
- [x] 4.7 Add watch command to root command in `cmd/watch.go` init function
- [x] 4.8 Update `cmd/constructors.go` if needed for watch command constructor
- [x] 4.9 Build and test standalone watch: `go build -o dist/spinner && ./dist/spinner watch <test-container>`

## 5.0 Implement --watch flag for spin command

- [x] 5.1 Add `--watch` boolean flag to spin command in `cmd/constructors.go`
- [x] 5.2 Implement conditional watch mode entry after container creation/reuse
- [x] 5.3 Share watch implementation between standalone and spin --watch
- [x] 5.4 Update spin command help text to document --watch flag
- [x] 5.5 Build and test spin with watch: `go build -o dist/spinner && ./dist/spinner spin --image test --repo <url> --watch`

## 6.0 Add integration tests

- [x] 6.1 Create `tests/integration/watch_test.go`
- [x] 6.2 Add integration test for standalone watch command with running container
- [x] 6.3 Add integration test for spin --watch flag
- [x] 6.4 Add integration test for watch with non-existent container
- [x] 6.5 Add integration test for watch with stopped container
- [x] 6.6 Verify all tests pass: `go test ./tests/integration/...`

## 7.0 Documentation and final verification

- [x] 7.1 Update main README.md with watch command documentation
- [x] 7.2 Add watch command examples to docs/usage.md
- [x] 7.3 Run full test suite: `npm test`
- [x] 7.4 Build final binary: `go build -o dist/spinner`
- [x] 7.5 Manual end-to-end test: create container, watch it, verify all features work
