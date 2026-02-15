# Design: add-model-flag

## Technical Implementation Plan

### Component Map

| File                                                    | Action | Purpose                                                                       |
|---------------------------------------------------------|--------|-------------------------------------------------------------------------------|
| `cmd/helpers.go`                                        | modify | Add `flagModel` constant                                                      |
| `cmd/spin.go`                                           | modify | Add `--model` flag, Viper binding, pass to CreateConfig, add to reserved list |
| `internal/provider/provider.go`                         | modify | Add `Model` field to `CreateConfig`                                           |
| `internal/backend/docker/run.go`                        | modify | Add `Model` to `spinConfig`, write `ANTHROPIC_MODEL` to env-file              |
| `internal/backend/docker/docker_provider.go`            | modify | Write `model.txt` override in `writeConfigOverrides()`                        |
| `internal/util/templates/scripts/startup.sh`            | modify | Read `model.txt` and export `ANTHROPIC_MODEL`                                 |
| `internal/exec/config.go`                               | modify | Add `Model` field, read from `ANTHROPIC_MODEL` env var                        |
| `internal/exec/loop.go`                                 | modify | Pass model to executor factory                                                |
| `internal/agent/claude/executor.go`                     | modify | Accept model in config, append `--model` to Claude CLI args                   |
| `internal/backend/gcp/gcp_provider.go`                  | modify | Add `ANTHROPIC_MODEL` to initial metadata and `updateMetadata()`              |
| `internal/backend/gcp/templates/scripts/gcp_runtime.sh` | modify | Read `ANTHROPIC_MODEL` from metadata and export                               |

### Approach

#### 1. CLI Flag and Config (cmd/spin.go, cmd/helpers.go)

Add `flagModel = "model"` constant. Add `--model` string flag to the spin command. Bind to Viper so
`.spinner.json` `"model"` key works. Read via `viper.GetString(flagModel)` and set on `CreateConfig.Model`.

Add `"ANTHROPIC_MODEL"` to the reserved env var map so `--env ANTHROPIC_MODEL=x` is rejected with a clear
error directing users to `--model`.

#### 2. Provider CreateConfig (internal/provider/provider.go)

Add `Model string` to `CreateConfig`. This is the universal carrier — both backends read it.

#### 3. Docker: Creation (internal/backend/docker/run.go)

Add `Model` field to `spinConfig`. In `buildDockerRunCommand()`, if `config.Model != ""`, write
`ANTHROPIC_MODEL={model}` to the env-file alongside other built-in vars.

#### 4. Docker: Restart Override (docker_provider.go, startup.sh)

In `writeConfigOverrides()`, write `model.txt` to the state directory (same pattern as `prompt.txt` and
`max-iterations.txt`).

In `startup.sh`, add a block after the existing override reads:
```bash
if [ -f "/state/model.txt" ]; then
  OVERRIDE_MODEL=$(cat /state/model.txt)
  if [ -n "$OVERRIDE_MODEL" ]; then
    ANTHROPIC_MODEL="$OVERRIDE_MODEL"
    export ANTHROPIC_MODEL
  fi
fi
```

This ensures that when a stopped container is restarted with a different `--model`, the new model takes effect.

#### 5. Exec Config (internal/exec/config.go)

Add `Model string` to `Config`. Read from `ANTHROPIC_MODEL` env var (optional — empty string means use default).
No validation.

#### 6. Executor Integration (internal/exec/loop.go, internal/agent/claude/executor.go)

The executor factory in `loop.go` already creates a `claude.Executor`. Add `Model` to `ExecutorConfig`.

In `executor.go`, if `e.config.Model != ""` is set (read from `ANTHROPIC_MODEL` env var which Claude CLI reads
natively), there's actually **no need to pass `--model` as a CLI arg** — Claude CLI reads `ANTHROPIC_MODEL`
from the environment automatically. The env var is already set in the container environment.

This means the executor changes are minimal: just add the field to `ExecutorConfig` for documentation/future use,
but the env var does the actual work.

#### 7. GCP: Creation (gcp_provider.go)

Add `"ANTHROPIC_MODEL": config.Model` to the metadata map in `Create()`. The value may be empty string, which
is fine — GCP metadata handles empty values.

#### 8. GCP: Restart (gcp_provider.go)

Add `"ANTHROPIC_MODEL": config.Model` to the `updates` map in `updateMetadata()`. This ensures the model is
updated when a stopped VM is restarted with a different `--model`.

#### 9. GCP Runtime Script (gcp_runtime.sh)

Add a line to read `ANTHROPIC_MODEL` from instance metadata and export it, alongside the other metadata reads:
```bash
ANTHROPIC_MODEL=$(curl -sf -H "$META_HEADER" "$META_URL/ANTHROPIC_MODEL" || echo "")
export ANTHROPIC_MODEL
```

### Key Decisions

- **No `--model` CLI arg to Claude**: Since Claude CLI reads `ANTHROPIC_MODEL` from the environment natively,
  we don't need to append `--model` to the command arguments. Setting the env var in the container is sufficient
  and simpler. This avoids changes to the executor's command building logic.

- **Same override pattern as prompt/max-iterations**: Docker can't update env vars after container creation,
  so we use the file-based override in `/state/`. GCP can update metadata directly. Both patterns are
  well-established and tested.

- **Empty string = use default**: If `--model` is not specified, the env var is either not set or empty,
  and Claude CLI uses its default model. No sentinel values needed.

### Risks / Trade-offs

- **Startup.sh is embedded**: The startup script is embedded via `go:embed`. Changes require rebuilding
  the binary and rebuilding Docker images (`spinner setup`). Existing images won't read `model.txt` until
  rebuilt. This is acceptable — `--model` is a new feature that requires the new binary anyway.
