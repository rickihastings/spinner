# Tasks: Add Custom Environment Variables / Secrets

## 1.0 Core `--env` Flag and Provider Plumbing

- [x] 1.1 Add `flagEnv` constant to `cmd/helpers.go` and `EnvVars map[string]string` field to `provider.CreateConfig`
- [x] 1.2 Add `--env` flag (StringSliceVar) to `cmd/spin.go`, parse `KEY=VALUE` pairs into map, validate format and reserved vars
- [x] 1.3 Pass parsed `EnvVars` map from spin command into `CreateConfig`
- [x] 1.4 Add unit tests for `--env` flag parsing: valid pairs, multiple vars, equals-in-value, empty value, invalid format, empty key, reserved var rejection
- [x] 1.5 Verify build and all existing tests pass

## 2.0 Docker Backend: `--env-file` Implementation

- [x] 2.1 Add `EnvVars map[string]string` to `docker.SpinConfig`
- [x] 2.2 Modify `BuildDockerRunCommand` to write all env vars (built-in + custom) to a temp file with `0600` permissions and use `--env-file` instead of individual `-e` flags
- [x] 2.3 Return temp file path from `BuildDockerRunCommand` so caller can clean up after `RunContainer`
- [x] 2.4 Update `docker_provider.go` `Create()` to pass `EnvVars` from `CreateConfig` to `SpinConfig` and handle temp file cleanup
- [x] 2.5 Update `run_test.go` with tests: env-file generation, file permissions, built-in vars included, custom vars included, cleanup after use
- [x] 2.6 Update `docker_provider_test.go` to verify `EnvVars` flow through `Create()`
- [x] 2.7 Verify build and all tests pass

## 3.0 GCP Backend: Metadata Prefix Implementation

- [x] 3.1 Modify `gcp_provider.go` `Create()` to merge custom env vars into instance metadata with `SPINNER_ENV_` prefix
- [x] 3.2 Update `templates/scripts/gcp_runtime.sh` to read all `SPINNER_ENV_*` metadata keys, strip prefix, and export as env vars
- [x] 3.3 Add unit tests for GCP metadata merging: custom vars prefixed, no collision with internal keys, empty env vars map
- [x] 3.4 Verify build and all tests pass
