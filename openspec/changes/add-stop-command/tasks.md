# Tasks: add-stop-command

## 1.0 Stop command — core implementation

- [ ] 1.1 Create `cmd/stop.go` with `NewStopCommand(f *provider.Factory)` following `destroy.go` pattern
- [ ] 1.2 Handle already-stopped and not-found cases gracefully (skip, not error)
- [ ] 1.3 Register command in `cmd/stop.go` init() via `rootCmd.AddCommand`
- [ ] 1.4 Update `cmd/spin.go` Docker "To stop:" hint to use `spinner stop <name>`
- [ ] 1.5 Update `cmd/spin.go` GCP "To stop:" hint to use `spinner stop <name> --backend gcp --project ... --zone ...`
- [ ] 1.6 Write unit tests in `cmd/stop_test.go` (mock factory): running→stopped, already-stopped, not-found, multi-instance, aggregate error
- [ ] 1.7 Verify: `go build ./...` passes and `go test ./cmd/...` passes
- [ ] 1.8 Commit

## 2.0 Documentation update

- [ ] 2.1 Add `spinner stop` section to `docs/usage.md`
- [ ] 2.2 Verify: `go build ./...` still passes
- [ ] 2.3 Commit

## 3.0 Docker integration test

- [ ] 3.1 Add `TestStop_DockerRunningInstance` to `tests/integration/` (or existing docker test file)
- [ ] 3.2 Verify integration test compiles and passes locally
- [ ] 3.3 Commit

## 4.0 GCP integration test

- [ ] 4.1 Add one GCP integration test: `TestStop_GCPRunningInstance` — spins up a GCP instance, calls `spinner stop`,
       verifies instance status becomes stopped
- [ ] 4.2 Verify test compiles (runtime execution is slow, mark as integration-gated)
- [ ] 4.3 Commit