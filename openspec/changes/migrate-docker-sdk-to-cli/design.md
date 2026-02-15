# Technical Implementation Plan: Migrate Docker SDK to CLI

## Goals

- Replace all Docker Go SDK calls with `docker` CLI equivalents
- Remove SDK dependencies from `go.mod` to reduce binary size
- Maintain the existing `Client` interface contract — no changes to method signatures
- Keep the `Provider` interface and all consuming code untouched

## Non-Goals

- Changing the `Client` or `Provider` interface signatures
- Adding new Docker operations or capabilities
- Supporting alternative container runtimes (Podman, nerdctl) — that's a future concern
- Changing the mock/test infrastructure beyond adapting to new return types

## Component Map

| File | Action | Description |
|------|--------|-------------|
| `internal/backend/docker/sdk.go` | **delete** | Lazy SDK client wrapper; no longer needed |
| `internal/backend/docker/client.go` | **modify** | Rewrite `RealDockerClient` methods to use CLI; remove SDK imports; replace `container.Summary` with local struct |
| `internal/backend/docker/metrics.go` | **modify** | Replace `metricsAPIClient` interface and `createMetricsClient()` with CLI-based metrics; keep `detectDockerHost()` (it already uses CLI) |
| `internal/backend/docker/mock_client.go` | **modify** | Replace `container.Summary` with local `ContainerListEntry` struct |
| `internal/backend/docker/docker_provider.go` | **modify** | Update `List()` to work with new `ContainerListEntry` type instead of `container.Summary` |
| `internal/backend/docker/docker_provider_test.go` | **modify** | Update mock expectations to use new types |
| `internal/backend/docker/build.go` | **modify** | Keep `createBuildContextTar` (still needed for context), remove `.dockerignore` handling if unused |
| `internal/backend/docker/events.go` | **modify** | Remove SDK-specific `buildEvent`/`buildErrorDetail`/`buildAux` types if no longer needed |
| `go.mod` | **modify** | Remove `github.com/docker/docker`, `containerd/errdefs`, `moby/term`, and their transitive deps |

## Approach

### Phase 1: Core CLI Client (client.go rewrite)

Replace each SDK method with its CLI equivalent. The `Client` interface stays identical. Pattern for each method:

```go
func (c *RealDockerClient) ImageExists(ctx context.Context, imageName string) (bool, error) {
    cmd := exec.CommandContext(ctx, "docker", "image", "inspect", imageName)
    if err := cmd.Run(); err != nil {
        // Exit code != 0 means image doesn't exist
        return false, nil
    }
    return true, nil
}
```

Method-by-method mapping:

| Client Method | SDK Call | CLI Replacement |
|---|---|---|
| `ImageExists` | `cli.ImageInspect` | `docker image inspect <name>` (exit code) |
| `ContainerExists` | `cli.ContainerInspect` | `docker inspect --format '{{.State.Running}}' <name>` |
| `StartContainer` | `cli.ContainerStart` | `docker start <name>` |
| `StopContainer` | `cli.ContainerStop` | `docker stop -t 10 <name>` |
| `RemoveContainer` | `cli.ContainerRemove` | `docker rm -f <name>` |
| `LogsContainer` | `cli.ContainerLogs` | `docker logs <name>` |
| `VerifyContainerStatus` | `cli.ContainerInspect` | `docker inspect --format '{{.State.Running}}' <name>` + `docker logs --tail 100` on failure |
| `StreamContainerLogs` | `cli.ContainerLogs` (follow) | `docker logs --follow --timestamps <name>` with line-by-line reader |
| `ListContainers` | `cli.ContainerList` | `docker ps -a --filter label=... --format json` |
| `BuildImage` | `cli.ImageBuild` | `docker build -t <tag> -f <dockerfile> --build-arg ... <context>` |
| `RunContainer` | Already CLI | No change |
| `ContainerEnvVars` | Already CLI | No change |

### Phase 2: Metrics (metrics.go rewrite)

Replace `metricsAPIClient` (which uses SDK's `ContainerInspect` and `ContainerStats`) with CLI calls:

- **Container state**: `docker inspect --format '{{json .State}}' <name>` — parse JSON for Running/ExitCode/Dead/OOMKilled
- **Container stats**: `docker stats --no-stream --format json <name>` — parse JSON for CPU/Memory

The `docker stats` JSON output provides pre-calculated CPU percentage and memory values, which is simpler than the SDK's
raw counters that require manual delta calculation. The `cpuSnapshot` struct and `calculateCPUPercent` function can be
removed.

### Phase 3: ListContainers Return Type

Replace `container.Summary` (Docker SDK type) with a local struct:

```go
type ContainerListEntry struct {
    ID     string
    Names  []string
    Image  string
    State  string
    Labels map[string]string
}
```

Update `docker_provider.go` `List()` method to use the new type.

### Phase 4: Cleanup

- Delete `sdk.go`
- Remove SDK imports from all files
- Run `go mod tidy` to remove unused dependencies
- Verify build and tests pass

## Key Decisions

1. **Keep `Client` interface unchanged** — This is critical. The interface is the contract between the Docker backend
   and the provider layer. By keeping it stable, we avoid cascading changes.

2. **Use JSON output format for structured data** — `--format json` gives us parseable output for `docker ps`,
   `docker inspect`, and `docker stats`. This is more robust than template-based formatting for complex data.

3. **Simplify log streaming** — The SDK requires manual parsing of Docker's 8-byte multiplexed header format. CLI
   `docker logs` outputs plain text, so we can use simple line-by-line buffered reading instead.

4. **Keep `createBuildContextTar`** — Docker CLI `docker build` needs a context directory (which we already create),
   but doesn't need us to tar it. However, the tar helper may still be useful if we want to support piping context
   via stdin in the future.

5. **Use `exec.CommandContext` consistently** — All CLI calls use `exec.CommandContext` with the request context for
   proper cancellation support.

## Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| Docker CLI not on PATH | Already required; `prerequisites` package checks for Docker availability |
| Output format changes between Docker versions | Use `--format json` which is stable across versions |
| Performance of spawning processes vs SDK calls | Negligible for the operation frequency; container operations are already I/O-bound |
| Log streaming fidelity | CLI `docker logs` provides the same content; we lose only the stdout/stderr stream distinction from the multiplexed header, which we can recover via `2>&1` separation if needed |
