  # Tasks: add-list-command

## 1.0 Provider interface extension and Docker list with labels

- [x] 1.1 Add `InstanceInfo` struct and `List(ctx) ([]InstanceInfo, error)` to `Provider` interface in
  `internal/provider/provider.go`
- [x] 1.2 Add `--label spinner-managed=true` to Docker container creation in `internal/backend/docker/run.go`
- [x] 1.3 Add `ListContainers` method to Docker `Client` interface and implement using Docker SDK `ContainerList`
  in `internal/backend/docker/client.go`
- [x] 1.4 Implement `List()` on `DockerProvider` with label filter + state file reading
  in `internal/backend/docker/docker_provider.go`
- [x] 1.5 Update mock provider and Docker mock client with `List`/`ListContainers` stubs
- [x] 1.6 Add unit tests for Docker `List()`: containers with labels, state enrichment, no containers found,
  Docker unavailable
- [x] 1.7 Verify build succeeds and all tests pass

## 2.0 GCP instance listing

- [x] 2.1 Add `ListInstances` method to GCP `Client` interface and implement using Compute Engine `List` API
  in `internal/backend/gcp/client.go`
- [x] 2.2 Implement `List()` on `GCPProvider` with label filter + metadata/label extraction + GCS state reading
  in `internal/backend/gcp/gcp_provider.go`
- [x] 2.3 Update GCP mock client with `ListInstances` stub
- [x] 2.4 Add unit tests for GCP `List()`: VMs with labels, metadata extraction, state enrichment from GCS,
  no state bucket, no instances found
- [x] 2.5 Verify build succeeds and all tests pass

## 3.0 CLI list command with multi-backend orchestration

- [ ] 3.1 Create `cmd/list.go` with `NewListCommand(f *provider.Factory)`: iterate backends, collect InstanceInfo,
  render table output with BACKEND/NAME/STATUS/STATE/ITER/AGE/LAST UPDATE columns, stale warning indicator
- [ ] 3.2 Add GCP config flags (project/zone/state-bucket)
- [ ] 3.3 Register command in `cmd/root.go`
- [ ] 3.4 Create `cmd/list_test.go` with unit tests: multi-backend listing, backend unavailable warning,
  no instances, stale warning
- [ ] 3.5 Verify build succeeds and all tests pass

## 4.0 Docker integration tests

- [ ] 4.1 Create `tests/integration/list_test.go` with Docker integration tests: spin up a container via
  `spinner spin`, verify `spinner list` shows it with correct status/state/labels
- [ ] 4.2 Test label presence on newly created containers (`docker inspect` to confirm `spinner-managed=true`)
- [ ] 4.3 Verify build succeeds and all integration tests pass

## 5.0 GCP integration tests

- [ ] 5.1 Extend `TestGCPLifecycle_FullCycle` in `tests/integration/gcp_lifecycle_test.go` to verify
  `spinner list` shows the instance with correct status/metadata/labels after spin
- [ ] 5.2 Verify state enrichment from GCS: iteration count, agent status, and timestamps are populated
  from the GCS state file when `--state-bucket` is configured
- [ ] 5.3 Verify build succeeds and all integration tests pass
