# Change: Migrate GCP Backend from SDK to CLI

## Why

The GCP backend currently uses the official Go SDK packages (`cloud.google.com/go/compute/apiv1`,
`cloud.google.com/go/storage`, `cloud.google.com/go/compute/metadata`) for all Compute Engine and Cloud Storage
operations. These SDK packages pull in 29+ direct and indirect dependencies in `go.mod`, including gRPC, protobuf,
OpenTelemetry, Google Cloud monitoring, and envoy control plane libraries.

**Problems with SDK-based approach:**

1. **Binary bloat** — the GCP SDK and its transitive dependencies (gRPC, protobuf, OpenTelemetry, envoy) add
   significant weight to the compiled binary
2. **Multi-cloud scaling** — adding AWS and Azure backends via their respective SDKs would exponentially increase
   dependency count and binary size. Each cloud SDK brings its own protobuf/gRPC/telemetry stack
3. **Dependency maintenance burden** — 29+ indirect dependencies require ongoing vulnerability monitoring, version
   pinning, and compatibility management
4. **Inconsistent patterns** — the Docker backend already uses CLI-based execution (`docker` commands via
   `os/exec`). Having GCP use SDK while Docker uses CLI creates architectural inconsistency

**CLI-based approach benefits:**

- **Lean binary** — no cloud SDK dependencies; `gcloud` handles its own dependency management
- **Consistent multi-cloud pattern** — all cloud backends (`gcloud`, `aws`, `az`, `docker`) use the same
  CLI + JSON parsing pattern
- **Simpler error handling** — exit codes + stderr instead of SDK-specific error types and gRPC status codes
- **Auth delegation** — `gcloud` handles ADC, service accounts, and credential refresh natively

## What Changes

- **MODIFIED** `Client` interface — returns plain Go structs instead of `*computepb.Instance`, `*computepb.Image`,
  `*computepb.Metadata` protobuf types
- **MODIFIED** `RealGCPClient` — implements all operations via `gcloud compute` and `gcloud storage` CLI commands
  with `--format=json` output parsing, replacing SDK client calls
- **MODIFIED** `MockGCPClient` — updated to use new plain Go types instead of protobuf types
- **MODIFIED** `gcp_provider.go` — updated to consume plain Go types instead of protobuf accessor methods
  (`.GetName()`, `.GetStatus()`, `.GetLabels()`, `.Disks[0].GetSource()`, etc.)
- **MODIFIED** `exec_hooks.go` — replaces `metadata.OnGCE()` with direct HTTP call to GCE metadata server
- **MODIFIED** `object_writer.go` — replaces `storage.Client` with CLI-based writes via Client interface
- **MODIFIED** all test files — updated to construct plain Go structs instead of protobuf types
- **MODIFIED** `go.mod` — removes `cloud.google.com/go/compute`, `cloud.google.com/go/storage`,
  `cloud.google.com/go/compute/metadata` and their transitive dependencies
- **ADDED** `gcloud` prerequisite check — verifies `gcloud` CLI is installed before GCP operations
- **ADDED** new Go struct types for Instance, Image, Metadata — replaces `computepb` protobuf types
- **ADDED** `cli_runner.go` — shared `runGcloud`/`runGcloudJSON` helper for executing gcloud commands

## Impact

### Affected Specs

- `gcp-sandbox` — requirement "SDK-Based GCP Operations" **MODIFIED** to "CLI-Based GCP Operations"

### Affected Code

| Area | Change Type |
|---|---|
| `internal/backend/gcp/types.go` | **modify** — add GCPInstance, GCPImage, GCPMetadata, GCPMetadataItem Go types |
| `internal/backend/gcp/client.go` | **modify** — change Client interface returns; rewrite RealGCPClient |
| `internal/backend/gcp/cli_runner.go` | **create** — shared gcloud command runner |
| `internal/backend/gcp/gcp_provider.go` | **modify** — consume plain Go types instead of protobuf accessors |
| `internal/backend/gcp/mock_client.go` | **modify** — use plain Go types |
| `internal/backend/gcp/exec_hooks.go` | **modify** — replace metadata.OnGCE() with HTTP check |
| `internal/backend/gcp/object_writer.go` | **modify** — replace storage.Client with Client interface |
| `internal/backend/gcp/gcp_provider_test.go` | **modify** — use plain Go types in test fixtures |
| `internal/backend/gcp/client_test.go` | **modify** — use plain Go types in test fixtures |
| `internal/backend/gcp/image_test.go` | **modify** — use plain Go types in test fixtures |
| `internal/backend/gcp/metrics_test.go` | **modify** — use plain Go types in test fixtures |
| `go.mod` / `go.sum` | **modify** — remove GCP SDK dependencies |

### Not Affected

- Provider interface (`internal/provider/provider.go`) — no changes
- Command layer (`cmd/`) — no changes
- Docker backend — no changes
- Shell scripts (`gcp_bake.sh`, `gcp_runtime.sh`) — no changes
- `internal/backend/gcp/instance.go` — no SDK imports, uses plain types already
- `internal/backend/gcp/startup.go` — no SDK imports
- `internal/backend/gcp/logs.go` — uses Client interface only
- `internal/backend/gcp/state.go` — uses Client interface only
- `internal/backend/gcp/metrics.go` — uses Client interface only

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| `gcloud` not installed on user machines | High | Add prerequisite check with clear error message and install instructions |
| `gcloud` JSON output format changes | Low | Pin to well-documented stable output fields; add integration test |
| CLI invocation latency vs SDK | Low | Each `gcloud` call is ~200ms; same as Docker CLI pattern already in use |
| `gcloud` auth not configured | Medium | Error messages from `gcloud` are clear; document `gcloud auth login` |
| Async operation wait semantics differ | Medium | `gcloud` with `--quiet` waits for operations by default; verify per command |
| Metadata fingerprint handling via CLI | Medium | `gcloud compute instances add-metadata` handles fingerprints internally |
