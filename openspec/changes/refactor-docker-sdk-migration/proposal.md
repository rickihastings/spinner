# Change: Migrate Docker Client from CLI to SDK

## Why

The current Docker implementation uses `os/exec` to invoke CLI commands (`docker build`, `docker run`, etc.). This approach has several limitations:

1. **No real-time event streaming** - Cannot consume Docker events for container lifecycle, logs, or build progress without complex log parsing
2. **Shell overhead** - Each operation spawns a new shell process
3. **Error handling** - Relies on exit codes and string parsing of error messages
4. **Platform inconsistencies** - CLI behavior may differ across platforms
5. **Missing capabilities** - No access to advanced Docker API features (health checks, signals, event subscriptions)

The Docker SDK (`github.com/docker/docker/client`) provides native Go access to all Docker operations with type-safe APIs and event streaming capabilities needed for upcoming work.

## What Changes

- Replace `RealDockerClient` implementation to use Docker SDK instead of CLI commands
- Add new interface methods for event streaming (container logs, build progress)
- Update `BuildImage` to use SDK's `ImageBuild` with progress streaming
- Update `RunContainer` to use SDK's `ContainerCreate` + `ContainerStart`
- Update all inspection/removal methods to use SDK equivalents
- Add new `StreamContainerLogs` method for real-time log consumption
- Maintain existing mock implementations for testing
- All existing tests continue to pass (behavior unchanged)

## Impact

- Affected code: `internal/docker/` (all changes isolated to this package)
- Affected files:
  - `internal/docker/client.go` - Major rewrite of `RealDockerClient`
  - `internal/docker/mock_client.go` - Add new mock methods
  - `internal/docker/events.go` - New file for event types and streaming
- No changes outside `internal/docker/` required (validates abstraction is clean)
- New dependency: `github.com/docker/docker` SDK

## Success Criteria

1. All integration tests pass without modification
2. Unit test coverage remains equivalent (updated for new implementation)
3. Build and container operations work identically from user perspective
4. New event streaming capabilities available for future work
