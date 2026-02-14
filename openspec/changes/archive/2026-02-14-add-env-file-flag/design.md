# Design: add-env-file-flag

## Technical Implementation Plan

### Component Map

| File                                     | Action | Purpose                                                          |
| ---------------------------------------- | ------ | ---------------------------------------------------------------- |
| `internal/provider/provider.go`          | modify | Add `EnvFile` field to `CreateConfig`                            |
| `cmd/spin.go`                            | modify | Add `--env-file` flag, validate file exists, set on CreateConfig |
| `cmd/spin_test.go`                       | modify | Unit tests for flag wiring                                       |
| `internal/backend/docker/run.go`         | modify | Pass as second `--env-file`, mount to staging path               |
| `internal/backend/docker/run_test.go`    | modify | Tests for docker args with env file                              |
| `templates/scripts/startup.sh`           | modify | Copy env file from staging path to workspace after clone         |
| `internal/backend/gcp/gcp_provider.go`   | modify | Base64-encode file content into metadata                         |
| `templates/scripts/gcp_runtime.sh`       | modify | Decode metadata, write to workspace, source                     |

### Approach

The file is passed through as-is — no parsing, no validation of contents, no merging with `--env`. The only
CLI-side check is that the file exists and is readable. Each backend handles placing the file in the workspace
and making its vars available.

### Data Flow

```
User provides --env-file .env
            |
            v
    cmd/spin.go:
    - Validate file exists (os.Stat)
    - Set CreateConfig.EnvFile = path
            |
       +----+----+
       |         |
  Docker      GCP
       |         |
  1. --env-file   1. Read file, base64
     <path>         encode content
  2. -v <path>:  2. Set metadata key
     /tmp/.env:ro   SPINNER_ENV_FILE
  3. startup.sh  3. gcp_runtime.sh
     copies to      decodes, writes
     workspace/     workspace/.env,
     .env           sources it
```

### Docker Implementation

1. **Env var injection**: Add the user's file as a second `--env-file` flag to `docker run`. Docker supports
   multiple `--env-file` flags. This sets all `KEY=VALUE` lines from the file as environment variables.
2. **Workspace placement**: Mount the file read-only at `/tmp/.env` via `-v <path>:/tmp/.env:ro`.
3. **Startup copy**: `startup.sh` checks if `/tmp/.env` exists after cloning and copies it to the workspace root.

### GCP Implementation

1. **Metadata transport**: Read the file content, base64-encode it, and set as `SPINNER_ENV_FILE` metadata value.
   GCP metadata values support up to 256 KB — more than enough for any env file.
2. **Runtime placement**: `gcp_runtime.sh` reads the `SPINNER_ENV_FILE` metadata, base64-decodes it, and writes
   to `/home/spinner/workspace/.env`.
3. **Env var injection**: `gcp_runtime.sh` sources the file after writing, which exports the vars into the
   runtime environment for `startup.sh` and `spinner exec`.

### Key Decisions

- **No CLI-side parsing**: The file format is the runtime's concern, not the CLI's. This keeps the CLI simple
  and avoids reimplementing dotenv parsing rules (quoted values, multiline, escapes, etc.).
- **Second `--env-file` for Docker**: Docker natively supports this. Keeps the existing temp file logic untouched.
- **Mount + copy for Docker**: The mount ensures the file is available immediately. The copy in `startup.sh`
  places it in the workspace root where dotenv libraries expect it.
- **Base64 for GCP**: Handles any special characters in env file values without metadata escaping issues.
