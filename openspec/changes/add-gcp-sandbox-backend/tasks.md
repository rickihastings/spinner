# Tasks: Add GCP Sandbox Backend

## ~~0.0 Prerequisite: GitHub Release Pipeline~~ (DONE — landed on main)

Already implemented on main (`a675ee6`):
- [x] `.goreleaser.yaml` — multi-platform builds (linux/darwin/windows, amd64/arm64)
- [x] `.github/workflows/release.yaml` — triggers GoReleaser on `v*` tag push
- [x] `internal/version/` — version injection via ldflags
- [x] `cmd/update.go` — self-update command using GitHub Releases
- [ ] 0.1 Tag and publish initial release (`v0.1.0`) to make binaries downloadable — **only remaining step**

Archive naming: `spinner_{version}_{os}_{arch}.tar.gz` (e.g., `spinner_0.1.0_linux_amd64.tar.gz`)

## 1.0 Provider Factory, Config File & Backend Selection

- [ ] 1.1 Create `internal/provider/factory.go` with `Factory` struct, `Register()`, `Create()`, `Available()` methods
- [ ] 1.2 Create `internal/provider/factory_test.go` with unit tests for factory registration, creation, and error cases
- [ ] 1.3 Modify `cmd/root.go` — add `.spinner.json` config file loading via Viper (alongside existing `.env` support)
- [ ] 1.4 Modify `cmd/constructors.go` — change `NewSetupCommand` and `NewSpinCommand` to accept `*provider.Factory`; add `--backend` flag (default: `"docker"`)
- [ ] 1.5 Add GCP-specific flags to setup and spin commands (`--project`, `--zone`, `--machine-type`, `--disk-size`, `--state-bucket`) organized into flag groups
- [ ] 1.6 Implement conditional flag validation in `RunE` — hard error when backend-specific CLI flags are used with the wrong `--backend`; config file values silently ignored cross-backend
- [ ] 1.7 Modify `cmd/constructors_watch.go` and `cmd/watch.go` — add `--backend` flag to standalone watch command
- [ ] 1.8 Update command tests to use factory instead of direct provider injection
- [ ] 1.9 Verify build and all existing tests pass (no regressions)

## 2.0 GCP Client Interface & Authentication

- [ ] 2.1 Create `internal/gcp/types.go` — GCP-specific types: `InstanceConfig`, `ImageConfig`, `SerialPortOutput`, `MetricsQuery`, `MetricPoint`, `VMStatus`
- [ ] 2.2 Create `internal/gcp/client.go` — `Client` interface with all GCP operations; `RealGCPClient` struct with SDK client fields
- [ ] 2.3 Implement `NewRealGCPClient(ctx, project)` — initialize all SDK clients with ADC authentication
- [ ] 2.4 Create `internal/gcp/mock_client.go` — testify mock implementing `Client` interface
- [ ] 2.5 Create `internal/gcp/client_test.go` — unit tests for client initialization and error handling
- [ ] 2.6 Add GCP SDK dependencies to `go.mod`: `cloud.google.com/go/compute`, `cloud.google.com/go/storage`, `cloud.google.com/go/logging`, `cloud.google.com/go/monitoring`
- [ ] 2.7 Verify build succeeds with new dependencies

## 3.0 GCP Setup — Image Baking

- [ ] 3.1 Create `templates/scripts/gcp_bake.sh` — startup script that installs git, gh, claude-code, downloads spinner from GitHub Releases (tar.gz), then shuts down
- [ ] 3.2 Create `internal/gcp/startup.go` — Go template rendering for bake and runtime startup scripts
- [ ] 3.3 Create `internal/gcp/image.go` — image baking logic: create temp VM, wait for shutdown, create image, cleanup
- [ ] 3.4 Implement `Provider.Setup()` — orchestrate image baking flow using client interface
- [ ] 3.5 Create `internal/gcp/image_test.go` — unit tests with mock client for full bake flow
- [ ] 3.6 Create `internal/gcp/startup_test.go` — test startup script template rendering
- [ ] 3.7 Verify build and tests pass

## 4.0 GCP Instance Lifecycle

- [ ] 4.1 Create `templates/scripts/gcp_runtime.sh` — runtime startup script that reads metadata, sets env vars, delegates to `startup.sh`
- [ ] 4.2 Create `internal/gcp/instance.go` — VM instance operations: create, start, stop, reset, delete, get status
- [ ] 4.3 Implement `Provider.Create()` — launch VM from baked image with metadata, wait for RUNNING state
- [ ] 4.4 Implement `Provider.Start()`, `Provider.Stop()`, `Provider.Restart()`, `Provider.Remove()` — VM lifecycle via client
- [ ] 4.5 Implement `Provider.Status()` — map GCP VM status to `provider.InstanceStatus`
- [ ] 4.6 Implement `Provider.InstanceName()` — deterministic name generation for GCP (max 63 chars, lowercase)
- [ ] 4.7 Create `internal/gcp/gcp_provider.go` — `Provider` struct, `NewGCPProvider()` constructor
- [ ] 4.8 Create `internal/gcp/instance_test.go` — unit tests for all lifecycle operations with mock client
- [ ] 4.9 Create `internal/gcp/gcp_provider_test.go` — provider-level tests
- [ ] 4.10 Verify build and tests pass

## 5.0 GCS Log Streaming

- [ ] 5.1 Create `internal/logs/gcs_sink.go` — `GCSSink` implementing `io.Writer`, buffers and flushes to GCS every 2s
- [ ] 5.2 Create `internal/logs/gcs_sink_test.go` — unit tests for GCS sink
- [ ] 5.3 Modify `internal/agent/claude/executor.go` — add `AdditionalWriter` field to executor config, use `io.MultiWriter` in TeeReader when set
- [ ] 5.4 Modify `internal/exec/loop.go` — detect `SPINNER_LOG_BUCKET` env var, create GCSSink, pass to executor config as `AdditionalWriter`
- [ ] 5.5 Create `internal/gcp/logs.go` — GCS-based log reader: `Logs()` downloads full object, `WatchLogs()` polls with byte offset + Range reads
- [ ] 5.6 Create `internal/gcp/logs_test.go` — unit tests for GCS log reader with mock client
- [ ] 5.7 Verify build and all tests pass (Docker unchanged, new GCS paths)

## 5.5 GCP Metrics

- [ ] 5.51 Create `internal/gcp/metrics.go` — Cloud Monitoring query for CPU utilization
- [ ] 5.52 Implement `Provider.WatchMetrics()` — poll Cloud Monitoring at 60s interval, map to `ContainerMetrics`, send to channel
- [ ] 5.53 Create `internal/gcp/metrics_test.go` — unit tests for metrics with mock client
- [ ] 5.54 Verify build and tests pass

## 6.0 GCP State Persistence & Integration

- [ ] 6.1 Create `internal/gcp/state.go` — GCS-based state read/write operations
- [ ] 6.2 Integrate state download into runtime startup script (read from GCS before `spinner exec`)
- [ ] 6.3 Integrate state upload into exec completion hook or Provider.Stop()
- [ ] 6.4 Create `internal/gcp/state_test.go` — unit tests for state persistence with mock client
- [ ] 6.5 Register GCP provider in factory wiring (`cmd/setup.go`, `cmd/spin.go`)
- [ ] 6.6 Update documentation: `docs/system-design.md` (architecture), `docs/usage.md` (GCP commands, `.spinner.json` config)
- [ ] 6.7 Full build + unit test suite verification

## 7.0 GCP Integration Tests

Mirrors the existing Docker integration tests in `tests/integration/`. Skipped when GCP is not available
(no credentials / no project), same pattern as Docker tests skipping when Docker daemon is unavailable.

- [ ] 7.1 Create `tests/testutil/gcp.go` — GCP test helpers: `SkipIfGCPNotAvailable(t)`, `GenerateTestInstanceName(t)`, VM cleanup helpers, GCS cleanup helpers
- [ ] 7.2 Create `tests/integration/gcp_setup_test.go` — setup command with `--backend gcp`: image bake, custom machine type, custom disk size, bake failure cleanup
- [ ] 7.3 Create `tests/integration/gcp_spin_test.go` — spin command with `--backend gcp`: VM creation, deterministic naming, container reuse/restart, recreate flag, repo cloning, branch handling
- [ ] 7.4 Create `tests/integration/gcp_watch_test.go` — watch command with `--backend gcp`: GCS log streaming, metrics polling, log object polling before VM writes
- [ ] 7.5 Create `tests/integration/gcp_lifecycle_test.go` — full lifecycle: setup → spin → watch → stop → start → remove; verify state persistence across stop/start via GCS
- [ ] 7.6 Create `tests/integration/gcp_flags_test.go` — conditional flag validation: wrong-backend flags error, `.spinner.json` config loading, precedence chain (CLI > env > config > defaults)
- [ ] 7.7 Create `tests/integration/gcp_cleanup_test.go` — resource cleanup: VM + disk deletion on remove, labels applied correctly, GCS state preserved after remove
- [ ] 7.8 Verify all integration tests pass with `go test ./tests/integration/... -run GCP` and existing Docker tests still pass
