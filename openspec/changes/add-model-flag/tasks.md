# Tasks: add-model-flag

## 1.0 Model flag on spin command and Docker creation

- [x] 1.1 Add `flagModel` constant to `cmd/helpers.go` and `Model` field to `CreateConfig` in
  `internal/provider/provider.go`
- [x] 1.2 Add `--model` flag to spin command in `cmd/spin.go` with Viper binding for `.spinner.json` support,
  add `ANTHROPIC_MODEL` to reserved env var list, wire into `CreateConfig`
- [x] 1.3 Add `Model` field to Docker `spinConfig` in `internal/backend/docker/run.go`, write `ANTHROPIC_MODEL`
  to env-file in `buildDockerRunCommand()` when model is set
- [x] 1.4 Add unit tests for model flag: flag parsing, reserved var rejection, config file default, env-file
  inclusion
- [x] 1.5 Verify build succeeds and all tests pass

## 2.0 Docker restart override and startup.sh

- [x] 2.1 Write `model.txt` in `writeConfigOverrides()` in `internal/backend/docker/docker_provider.go`
- [x] 2.2 Read `model.txt` and export `ANTHROPIC_MODEL` in `internal/util/templates/scripts/startup.sh`
- [x] 2.3 Add unit tests for model override file writing
- [x] 2.4 Verify build succeeds and all tests pass

## 3.0 GCP metadata and runtime script

- [ ] 3.1 Add `ANTHROPIC_MODEL` to initial metadata map in `Create()` in `internal/backend/gcp/gcp_provider.go`
- [ ] 3.2 Add `ANTHROPIC_MODEL` to `updates` map in `updateMetadata()` for restart override
- [ ] 3.3 Read and export `ANTHROPIC_MODEL` from metadata in
  `internal/backend/gcp/templates/scripts/gcp_runtime.sh`
- [ ] 3.4 Add unit tests for GCP metadata inclusion and update
- [ ] 3.5 Verify build succeeds and all tests pass

## 4.0 Docker integration tests

- [ ] 4.1 Add integration test: spin with `--model`, verify `ANTHROPIC_MODEL` is set in container environment
- [ ] 4.2 Add integration test: spin without `--model`, verify `ANTHROPIC_MODEL` is not set
- [ ] 4.3 Verify build succeeds and all integration tests pass

## 5.0 GCP integration tests

- [ ] 5.1 Add integration test: spin with `--model` on GCP, verify `ANTHROPIC_MODEL` in instance metadata
- [ ] 5.2 Add integration test: restart with different `--model`, verify metadata updated
- [ ] 5.3 Verify build succeeds and all integration tests pass
