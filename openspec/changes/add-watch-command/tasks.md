# Implementation Tasks

## 1.0 Implement log file monitoring and parsing

- [ ] 1.1 Add `fsnotify` dependency to go.mod
- [ ] 1.2 Create `internal/docker/logs.go` with LogWatcher type and file watching logic
- [ ] 1.3 Implement JSON log parsing function with fallback for non-JSON lines
- [ ] 1.4 Implement channel-based log streaming architecture
- [ ] 1.5 Add unit tests in `internal/docker/logs_test.go` for parsing and file watching
- [ ] 1.6 Verify tests pass with `go test ./internal/docker/...`

## 2.0 Implement metrics abstraction and Docker provider

- [ ] 2.1 Add Docker SDK client dependency to go.mod (`github.com/docker/docker/client`)
- [ ] 2.2 Create `internal/metrics/types.go` with ContainerMetrics and ContainerState types
- [ ] 2.3 Create `internal/metrics/provider.go` with MetricsProvider interface definition
- [ ] 2.4 Create `internal/metrics/docker.go` with Docker-specific MetricsProvider implementation
- [ ] 2.5 Implement StreamMetrics method that polls Docker stats every 1-2 seconds
- [ ] 2.6 Implement CPU and memory metric extraction from Docker API and send to channel
- [ ] 2.7 Implement container state detection (running/stopped/exited) and send to channel
- [ ] 2.8 Add unit tests in `internal/metrics/docker_test.go` with mocked Docker client
- [ ] 2.9 Verify tests pass with `go test ./internal/metrics/...`

## 3.0 Implement TUI rendering with tview

- [ ] 3.1 Add `tview` dependency to go.mod (`github.com/rivo/tview`)
- [ ] 3.2 Create `internal/tui/watch.go` with TUI rendering logic
- [ ] 3.3 Implement split-pane layout (header + log view)
- [ ] 3.4 Implement header rendering for container metadata and metrics
- [ ] 3.5 Implement log view with auto-scrolling
- [ ] 3.6 Implement keyboard handlers (q and Ctrl+C to quit)
- [ ] 3.7 Implement channel consumers for logs and metrics updates (from MetricsProvider)
- [ ] 3.8 Add graceful shutdown logic with context cancellation
- [ ] 3.9 Build and manually test TUI rendering: `go build -o dist/spinner && ./dist/spinner watch <test-container>`

## 4.0 Implement standalone watch command

- [ ] 4.1 Create `cmd/watch.go` with NewWatchCommand constructor
- [ ] 4.2 Implement command argument validation (require container name)
- [ ] 4.3 Implement container existence check using Docker client
- [ ] 4.4 Implement log directory existence check
- [ ] 4.5 Create Docker MetricsProvider instance for the container
- [ ] 4.6 Integrate LogWatcher, MetricsProvider, and TUI components with channels
- [ ] 4.7 Add watch command to root command in `cmd/watch.go` init function
- [ ] 4.8 Update `cmd/constructors.go` if needed for watch command constructor
- [ ] 4.9 Build and test standalone watch: `go build -o dist/spinner && ./dist/spinner watch <test-container>`

## 5.0 Implement --watch flag for spin command

- [ ] 5.1 Add `--watch` boolean flag to spin command in `cmd/constructors.go`
- [ ] 5.2 Implement conditional watch mode entry after container creation/reuse
- [ ] 5.3 Share watch implementation between standalone and spin --watch
- [ ] 5.4 Update spin command help text to document --watch flag
- [ ] 5.5 Build and test spin with watch: `go build -o dist/spinner && ./dist/spinner spin --image test --repo <url> --watch`

## 6.0 Add integration tests

- [ ] 6.1 Create `tests/integration/watch_test.go`
- [ ] 6.2 Add integration test for standalone watch command with running container
- [ ] 6.3 Add integration test for spin --watch flag
- [ ] 6.4 Add integration test for watch with non-existent container
- [ ] 6.5 Add integration test for watch with stopped container
- [ ] 6.6 Verify all tests pass: `go test ./tests/integration/...`

## 7.0 Documentation and final verification

- [ ] 7.1 Update main README.md with watch command documentation
- [ ] 7.2 Add watch command examples to docs/usage.md
- [ ] 7.3 Run full test suite: `npm test`
- [ ] 7.4 Build final binary: `go build -o dist/spinner`
- [ ] 7.5 Manual end-to-end test: create container, watch it, verify all features work
