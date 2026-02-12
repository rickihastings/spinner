# Proposal: add-env-file-flag

## Summary

Add a `--env-file <path>` flag to the `spin` command that places the user's env file into the instance workspace
and makes its variables available in the runtime environment. The file is passed through as-is without parsing or
validation on the CLI side.

## Motivation

1. Users frequently need to pass many environment variables (database URLs, API keys, service credentials) to their
   sandboxed instances. Specifying each one individually via `--env KEY=VALUE` is tedious.
2. Teams already maintain `.env` files for local development. Being able to pass them directly to spinner avoids
   manual transcription.
3. The Claude agent inside the instance may need to run the application to verify its work. Having the `.env` file
   in the workspace means the app can load it via standard dotenv conventions.

## What Changes

- **Modified capability**: `cli-spin` — add `--env-file` flag
- **Modified capability**: `gcp-sandbox` — pass env file content via metadata, runtime script writes to workspace
  and sources it
- **Docker backend**: pass user's env file as a second `--env-file` to `docker run` and mount it into the workspace
- **GCP backend**: base64-encode file content into metadata, runtime script writes to workspace and sources
- **Provider interface**: add `EnvFile` field to `CreateConfig` to carry the file path
- **No breaking changes**: purely additive

## Decisions

1. **No CLI-side parsing**: The file is passed through as-is. No validation of contents, no reserved-var checking,
   no merging with `--env` flags. The runtime environment handles the file.
2. **Docker delivery**: Pass the user's file as a second `--env-file` to `docker run` (sets env vars in the
   container process). Mount the file into a staging path so `startup.sh` can copy it to the workspace.
3. **GCP delivery**: Base64-encode the file content and pass as metadata value `SPINNER_ENV_FILE`. The runtime
   script decodes it, writes to workspace as `.env`, and sources it.
4. **File existence**: The CLI validates the file exists and is readable before proceeding. That's the only
   validation performed.

## Impact

- **Affected specs**: `cli-spin` (modified), `gcp-sandbox` (modified)
- **Affected code**: `cmd/spin.go`, `cmd/spin_test.go`, `internal/backend/docker/run.go`,
  `internal/backend/docker/run_test.go`, `internal/backend/gcp/gcp_provider.go`, `internal/provider/provider.go`,
  `templates/scripts/startup.sh`, `templates/scripts/gcp_runtime.sh`
- **Risk**: Low — builds on existing infrastructure, Docker already uses `--env-file` internally
