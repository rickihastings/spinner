# Tasks: Add GCP Sandbox Backend

## 1.0 Provider Factory & Backend Selection

- [ ] 1.1 Create `internal/provider/factory.go` with `Factory` struct, `Register()`, `Create()`, `Available()` methods
- [ ] 1.2 Create `internal/provider/factory_test.go` with unit tests for factory registration, creation, and error cases
- [ ] 1.3 Modify `cmd/constructors.go` — change `NewSetupCommand` and `NewSpinCommand` to accept `*provider.Factory` and add `--backend` flag (default: `"docker"`)
- [ ] 1.4 Modify `cmd/setup.go` and `cmd/spin.go` — wire factory with Docker registered; add GCP-specific flags (`--project`, `--zone`, `--machine-type`)
- [ ] 1.5 Modify `cmd/constructors_watch.go` and `cmd/watch.go` — add `--backend` flag to standalone watch command
- [ ] 1.6 Update command tests to use factory instead of direct provider injection
- [ ] 1.7 Verify build and all existing tests pass (no regressions)

## 2.0 GCP Client Interface & Authentication

- [ ] 2.1 Create `internal/gcp/types.go` — GCP-specific types: `InstanceConfig`, `ImageConfig`, `SerialPortOutput`, `MetricsQuery`, `MetricPoint`, `VMStatus`
- [ ] 2.2 Create `internal/gcp/client.go` — `Client` interface with all GCP operations; `RealGCPClient` struct with SDK client fields
- [ ] 2.3 Implement `NewRealGCPClient(ctx, project)` — initialize all SDK clients with ADC authentication
- [ ] 2.4 Create `internal/gcp/mock_client.go` — testify mock implementing `Client` interface
- [ ] 2.5 Create `internal/gcp/client_test.go` — unit tests for client initialization and error handling
- [ ] 2.6 Add GCP SDK dependencies to `go.mod`: `cloud.google.com/go/compute`, `cloud.google.com/go/storage`, `cloud.google.com/go/logging`, `cloud.google.com/go/monitoring`
- [ ] 2.7 Verify build succeeds with new dependencies

## 3.0 GCP Setup — Image Baking

- [ ] 3.1 Create `templates/scripts/gcp_bake.sh` — startup script that installs git, gh, claude-code, spinner binary, then shuts down
- [ ] 3.2 Create `internal/gcp/startup.go` — Go template rendering for bake and runtime startup scripts
- [ ] 3.3 Create `internal/gcp/image.go` — image baking logic: upload binary to GCS, create temp VM, wait for shutdown, create image, cleanup
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

## 5.0 GCP Logs & Metrics

- [ ] 5.1 Create `internal/gcp/logs.go` — serial port output reading with byte-offset tracking
- [ ] 5.2 Implement `Provider.Logs()` — return full serial port output as `io.ReadCloser`
- [ ] 5.3 Implement `Provider.WatchLogs()` — poll serial port at 1s interval, send new lines to channel
- [ ] 5.4 Create `internal/gcp/metrics.go` — Cloud Monitoring query for CPU utilization
- [ ] 5.5 Implement `Provider.WatchMetrics()` — poll Cloud Monitoring, map to `ContainerMetrics`, send to channel
- [ ] 5.6 Create `internal/gcp/logs_test.go` — unit tests for log streaming with mock client
- [ ] 5.7 Create `internal/gcp/metrics_test.go` — unit tests for metrics with mock client
- [ ] 5.8 Verify build and tests pass

## 6.0 GCP State Persistence & Integration

- [ ] 6.1 Create `internal/gcp/state.go` — GCS-based state read/write operations
- [ ] 6.2 Integrate state download into runtime startup script (read from GCS before `spinner exec`)
- [ ] 6.3 Integrate state upload into exec completion hook or Provider.Stop()
- [ ] 6.4 Create `internal/gcp/state_test.go` — unit tests for state persistence with mock client
- [ ] 6.5 Register GCP provider in factory wiring (`cmd/setup.go`, `cmd/spin.go`)
- [ ] 6.6 Update documentation: `docs/system-design.md` (architecture), `docs/usage.md` (GCP commands)
- [ ] 6.7 Full build + test suite verification
- [ ] 6.8 Integration test smoke plan documented (manual verification steps for GCP)
