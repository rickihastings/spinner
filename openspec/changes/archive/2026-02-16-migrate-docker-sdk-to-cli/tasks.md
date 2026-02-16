# Tasks: Migrate Docker SDK to CLI

## 1.0 Core Client Migration (inspection, lifecycle, logs)

Migrate the core `RealDockerClient` methods from SDK to CLI. This covers container inspection,
lifecycle management (start/stop/remove), log retrieval, and status verification. After this slice,
all container operations except build, list, streaming logs, and metrics use CLI.

- [x] 1.1 Replace `ImageExists` — use `docker image inspect` exit code
- [x] 1.2 Replace `ContainerExists` — use `docker inspect --format` for state
- [x] 1.3 Replace `StartContainer` — use `docker start`
- [x] 1.4 Replace `StopContainer` — use `docker stop -t 10`
- [x] 1.5 Replace `RemoveContainer` — use `docker rm -f`
- [x] 1.6 Replace `LogsContainer` — use `docker logs`
- [x] 1.7 Replace `VerifyContainerStatus` — use `docker inspect` + `docker logs --tail 100`
- [x] 1.8 ~~Remove `getSDKClient` method and `sdk` field from `RealDockerClient`~~ Completed in 3.0 — `StreamContainerLogs` and `ListContainers` migrated to CLI
- [x] 1.9 Update unit tests for changed methods
- [x] 1.10 Verify build passes and existing tests pass

## 2.0 Image Building Migration

Replace SDK-based image building with `docker build` CLI. This removes the tar-streaming build,
JSON message parsing, and `moby/term` terminal detection.

- [x] 2.1 Replace `buildImageWithOptions` with `docker build` CLI invocation
- [x] 2.2 Replace `buildUserDockerfile` with `docker build` CLI invocation
- [x] 2.3 Update `BuildImage` to use CLI build methods
- [x] 2.4 Remove `processBuildOutput` (SDK JSON message stream parser)
- [x] 2.5 Remove `createBuildContextTar` — CLI handles context directory and `.dockerignore` natively
- [x] 2.6 Update tests for build changes
- [x] 2.7 Verify build passes and existing tests pass

## 3.0 Log Streaming and Container Listing Migration

Replace SDK-based log streaming (`StreamContainerLogs`) and container listing (`ListContainers`)
with CLI equivalents. Introduce local `ContainerListEntry` type to replace `container.Summary`.

- [x] 3.1 Replace `StreamContainerLogs` — use `docker logs --follow` with line reader
- [x] 3.2 Remove `streamLogs` (8-byte Docker header parser, no longer needed)
- [x] 3.3 Define local `ContainerListEntry` struct replacing `container.Summary`
- [x] 3.4 Replace `ListContainers` — use `docker ps -a --filter --format json`
- [x] 3.5 Update `mock_client.go` for new `ContainerListEntry` type
- [x] 3.6 Update `docker_provider.go` `List()` for new `ContainerListEntry` type
- [x] 3.7 Update tests for streaming and listing changes
- [x] 3.8 Verify build passes and existing tests pass

## 4.0 Metrics Migration

Replace SDK-based metrics collection with CLI-based collection using `docker stats` and
`docker inspect`.

- [x] 4.1 Replace `metricsAPIClient` interface and `createMetricsClient` with CLI-based collection
- [x] 4.2 Replace `collectMetrics` — use `docker stats --no-stream --format json` and `docker inspect`
- [x] 4.3 Remove `cpuSnapshot` struct and `calculateCPUPercent` (CLI provides pre-calculated values)
- [x] 4.4 Update `mapDockerStateToMetrics` to work with local `dockerInspectState` struct from CLI inspect output
- [x] 4.5 Update `streamMetrics` to use new CLI-based collection (no longer requires `metricsAPIClient`)
- [x] 4.6 Add metrics tests for parsing and state mapping
- [x] 4.7 Verify build passes and existing tests pass

## 5.0 SDK Removal and Dependency Cleanup

Delete SDK wrapper, remove all Docker SDK imports, and clean up dependencies.

- [x] 5.1 Delete `internal/backend/docker/sdk.go` — completed in 3.0 (linter required removal of unused code)
- [x] 5.2 Remove unused SDK-specific types from `events.go` (`buildEvent`, `buildErrorDetail`, `buildAux`)
- [x] 5.3 Remove all `github.com/docker/docker` imports from remaining files — already removed in prior slices
- [x] 5.4 Remove `containerd/errdefs` import (used for SDK not-found detection) — removed from go.mod
- [x] 5.5 Run `go mod tidy` to remove unused dependencies
- [x] 5.6 Verify binary size reduction — binary unchanged (deps weren't compiled in since imports already removed)
- [x] 5.7 Run full test suite and verify everything passes — all unit tests pass
- [x] 5.8 Update docker-client spec to reflect CLI-based implementation
