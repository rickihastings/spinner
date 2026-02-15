# Migrate Docker SDK to CLI

## Why

The Docker backend was originally implemented using CLI commands, then refactored to use the Docker Go SDK
(`github.com/docker/docker/client`). This migration has proven to be the wrong direction for several reasons:

1. **Arg passthrough limitations** — The SDK requires decomposing Docker commands into API structs (`ContainerConfig`,
   `HostConfig`, `NetworkingConfig`), making it difficult to pass arbitrary Docker flags down from the CLI. The CLI
   approach (`docker run <args>`) naturally extends to any Docker option without code changes.

2. **Large binary size** — The Docker SDK and its transitive dependencies (`containerd/errdefs`,
   `docker/go-connections`, `docker/go-units`, `moby/term`, `moby/patternmatcher`, `distribution/reference`,
   `opencontainers/*`, etc.) significantly bloat the compiled binary. CLI invocation has zero dependency cost.

3. **Inconsistent approach** — The codebase already mixes SDK and CLI calls. `RunContainer` uses `exec.CommandContext`
   for `docker run`, `ContainerEnvVars` uses `docker inspect` via shell, `getDockerContainerID`/`getDockerImageID`
   use `docker inspect` via shell, and `detectDockerHost` uses `docker context inspect`. Standardizing on CLI
   eliminates this inconsistency.

4. **Portability** — CLI commands extend naturally to other container runtimes (Podman, nerdctl) that implement the
   Docker CLI interface. The SDK ties the project specifically to Docker's Go API.

5. **Simpler testing** — Mock interfaces stay the same (the `Client` interface doesn't change), but the real
   implementation becomes simpler: each method is a well-defined CLI invocation with predictable text/JSON output
   parsing instead of SDK struct handling.

## What Changes

- **BREAKING**: Remove `github.com/docker/docker` SDK dependency and all related transitive dependencies
- **BREAKING**: Remove `sdk.go` (lazy SDK client wrapper) entirely
- Replace all SDK-based methods in `client.go` with Docker CLI equivalents
- Replace SDK-based metrics collection in `metrics.go` with `docker stats` and `docker inspect` CLI calls
- Replace SDK-based image building in `client.go` with `docker build` CLI invocation
- Replace SDK-based log streaming in `client.go` with `docker logs --follow` CLI invocation
- Replace SDK-based container listing in `client.go` with `docker ps --filter` CLI invocation
- Update `mock_client.go` to remove SDK type references (replace `container.Summary` with a local struct)
- Remove `moby/term` dependency (used only for SDK build output rendering)
- Remove `containerd/errdefs` dependency (used only for SDK not-found error detection)

## Impact

### Affected Specs

- `docker-client` — All requirements change from SDK to CLI implementation

### Affected Code

- `internal/backend/docker/client.go` — Full rewrite of `RealDockerClient` methods
- `internal/backend/docker/sdk.go` — Deleted entirely
- `internal/backend/docker/metrics.go` — Rewrite metrics to use CLI
- `internal/backend/docker/mock_client.go` — Update mock return types
- `internal/backend/docker/docker_provider.go` — Minor: remove SDK-specific error handling
- `internal/backend/docker/docker_provider_test.go` — Update mock expectations
- `internal/backend/docker/events.go` — May simplify (remove SDK-specific stream types)
- `go.mod` / `go.sum` — Remove Docker SDK and related dependencies

### Risk

- **Low**: The `Client` interface and `Provider` interface remain unchanged. All commands already use these
  abstractions, so nothing outside `internal/backend/docker/` is affected.
- **Medium**: Log streaming via CLI (`docker logs -f`) delivers raw text instead of multiplexed binary frames from the
  SDK. The `streamLogs` function that manually parses 8-byte Docker headers can be replaced with simple line reading.
- **Low**: Build output from `docker build` goes directly to stdout/stderr, which is simpler than the SDK's JSON
  message stream parsing.
