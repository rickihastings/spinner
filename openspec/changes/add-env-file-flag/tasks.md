# Tasks: add-env-file-flag

## 1.0 Add --env-file flag to CLI and provider interface

- [x] 1.1 Add `EnvFile` field to `provider.CreateConfig` in `internal/provider/provider.go`
- [x] 1.2 Add `--env-file` flag to spin command in `cmd/spin.go`: validate file exists with `os.Stat`, set on
  `CreateConfig.EnvFile`
- [x] 1.3 Add unit tests in `cmd/spin_test.go`: flag wiring, file not found error

## 2.0 Docker backend: env file passthrough and workspace placement

- [x] 2.1 Update `buildDockerRunCommand` in `internal/backend/docker/run.go`: add `EnvFile` to `spinConfig`,
  if set add second `--env-file` arg and `-v <path>:/tmp/.env:ro` mount
- [x] 2.2 Update `templates/scripts/startup.sh`: after repo clone, if `/tmp/.env` exists copy to workspace root
- [x] 2.3 Add unit tests in `internal/backend/docker/run_test.go` for docker args with env file

## 3.0 GCP backend: env file metadata and runtime script

- [ ] 3.1 Update `Create()` in `internal/backend/gcp/gcp_provider.go`: if `config.EnvFile` is set, read file,
  base64-encode content, add as `SPINNER_ENV_FILE` metadata
- [ ] 3.2 Update `templates/scripts/gcp_runtime.sh`: read `SPINNER_ENV_FILE` metadata, base64-decode, write to
  workspace `.env`, source it
- [ ] 3.3 Verify build succeeds and all tests pass
