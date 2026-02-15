# Tasks: Migrate Docker SDK to CLI

## 1.0 Core Client Migration (inspection, lifecycle, logs)

Migrate the core `RealDockerClient` methods from SDK to CLI. This covers container inspection,
lifecycle management (start/stop/remove), log retrieval, and status verification. After this slice,
all container operations except build, list, streaming logs, and metrics use CLI.

- [ ] 1.1 Replace `ImageExists` — use `docker image inspect` exit code
- [ ] 1.2 Replace `ContainerExists` — use `docker inspect --format` for state
- [ ] 1.3 Replace `StartContainer` — use `docker start`
- [ ] 1.4 Replace `StopContainer` — use `docker stop -t 10`
- [ ] 1.5 Replace `RemoveContainer` — use `docker rm -f`
- [ ] 1.6 Replace `LogsContainer` — use `docker logs`
- [ ] 1.7 Replace `VerifyContainerStatus` — use `docker inspect` + `docker logs --tail 100`
- [ ] 1.8 Remove `getSDKClient` method and `sdk` field from `RealDockerClient`
- [ ] 1.9 Update unit tests for changed methods
- [ ] 1.10 Verify build passes and existing tests pass

## 2.0 Image Building Migration

Replace SDK-based image building with `docker build` CLI. This removes the tar-streaming build,
JSON message parsing, and `moby/term` terminal detection.

- [ ] 2.1 Replace `buildImageWithOptions` with `docker build` CLI invocation
- [ ] 2.2 Replace `buildUserDockerfile` with `docker build` CLI invocation
- [ ] 2.3 Update `BuildImage` to use CLI build methods
- [ ] 2.4 Remove `processBuildOutput` (SDK JSON message stream parser)
- [ ] 2.5 Remove `createBuildContextTar` if no longer needed, or keep if context dir is sufficient
- [ ] 2.6 Update tests for build changes
- [ ] 2.7 Verify build passes and existing tests pass

## 3.0 Log Streaming and Container Listing Migration

Replace SDK-based log streaming (`StreamContainerLogs`) and container listing (`ListContainers`)
with CLI equivalents. Introduce local `ContainerListEntry` type to replace `container.Summary`.

- [ ] 3.1 Replace `StreamContainerLogs` — use `docker logs --follow` with line reader
- [ ] 3.2 Remove `streamLogs` (8-byte Docker header parser, no longer needed)
- [ ] 3.3 Define local `ContainerListEntry` struct replacing `container.Summary`
- [ ] 3.4 Replace `ListContainers` — use `docker ps -a --filter --format json`
- [ ] 3.5 Update `mock_client.go` for new `ContainerListEntry` type
- [ ] 3.6 Update `docker_provider.go` `List()` for new `ContainerListEntry` type
- [ ] 3.7 Update tests for streaming and listing changes
- [ ] 3.8 Verify build passes and existing tests pass

## 4.0 Metrics Migration

Replace SDK-based metrics collection with CLI-based collection using `docker stats` and
`docker inspect`.

- [ ] 4.1 Replace `metricsAPIClient` interface and `createMetricsClient` with CLI-based collection
- [ ] 4.2 Replace `collectMetrics` — use `docker stats --no-stream --format json` and `docker inspect`
- [ ] 4.3 Remove `cpuSnapshot` struct and `calculateCPUPercent` (CLI provides pre-calculated values)
- [ ] 4.4 Remove `mapDockerStateToMetrics` or update to work with CLI inspect output
- [ ] 4.5 Update `streamMetrics` to use new CLI-based collection
- [ ] 4.6 Update metrics tests
- [ ] 4.7 Verify build passes and existing tests pass

## 5.0 SDK Removal and Dependency Cleanup

Delete SDK wrapper, remove all Docker SDK imports, and clean up dependencies.

- [ ] 5.1 Delete `internal/backend/docker/sdk.go`
- [ ] 5.2 Remove unused SDK-specific types from `events.go` (`buildEvent`, `buildErrorDetail`, `buildAux`)
- [ ] 5.3 Remove all `github.com/docker/docker` imports from remaining files
- [ ] 5.4 Remove `containerd/errdefs` import (used for SDK not-found detection)
- [ ] 5.5 Run `go mod tidy` to remove unused dependencies
- [ ] 5.6 Verify binary size reduction
- [ ] 5.7 Run full test suite and verify everything passes
- [ ] 5.8 Update docker-client spec to reflect CLI-based implementation
