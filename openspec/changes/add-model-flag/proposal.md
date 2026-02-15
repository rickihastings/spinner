# Proposal: add-model-flag

## Summary

Add a `--model` flag to `spinner spin` that selects which Claude model the agent uses inside the container/VM. The
model is passed as the `ANTHROPIC_MODEL` environment variable (which Claude CLI reads natively). The value is stored
and overridable on restart — if a spinner is stopped and restarted with a different `--model`, the new value takes
effect. The flag is also defaultable via `.spinner.json`.

## Motivation

Currently, spinner always uses whichever Claude model the Claude CLI defaults to. Users have no way to select a
specific model (e.g., `claude-sonnet-4-5-20250929` for faster/cheaper runs, or `claude-opus-4-6` for harder tasks)
without resorting to `--env ANTHROPIC_MODEL=...`. A first-class `--model` flag makes this explicit, discoverable, and
consistent across backends.

The metadata display already reads `ANTHROPIC_MODEL` from containers/VMs (for the watch UI), but nothing sets it
today. This change closes that gap.

## What Changes

- **Modified capability**: `cli-spin` — add `--model` flag, add `ANTHROPIC_MODEL` to reserved env vars, add `model`
  key to `.spinner.json` support, write model override file for Docker restart, include in GCP metadata updates
- **Modified capability**: `cli-exec` — read `ANTHROPIC_MODEL` env var in exec config, pass `--model` to Claude CLI
  when set
- **Modified capability**: `gcp-sandbox` — include `ANTHROPIC_MODEL` in initial VM metadata and in metadata updates
  on restart
- **No breaking changes**: The flag is optional. Existing spinners without `--model` continue to use Claude's default.

## Decisions

1. **Env var name**: `ANTHROPIC_MODEL` — this is what Claude CLI natively reads, and what both backends already check
   for in metadata display. No mapping layer needed.

2. **Reserved variable**: `ANTHROPIC_MODEL` is added to the reserved env var list. Users must use `--model` instead of
   `--env ANTHROPIC_MODEL=...` for consistency and discoverability.

3. **Config file support**: `.spinner.json` supports `"model": "<model-name>"` as a defaultable key. CLI flag
   `--model` overrides the config file value, following existing precedence rules.

4. **No validation**: Model names are passed through as-is. No validation against known model IDs — this keeps
   spinner forward-compatible with new models without code changes.

5. **Override on restart**: Same pattern as prompt and max-iterations — Docker writes `model.txt` to the state
   directory, startup.sh reads and exports it. GCP updates the `ANTHROPIC_MODEL` metadata item via `updateMetadata()`.

## Impact

- **Affected specs**: `cli-spin` (modified), `cli-exec` (modified), `gcp-sandbox` (modified)
- **Affected code**: `cmd/spin.go`, `cmd/helpers.go`, `internal/provider/provider.go`,
  `internal/backend/docker/run.go`, `internal/backend/docker/docker_provider.go`,
  `internal/util/templates/scripts/startup.sh`, `internal/exec/config.go`, `internal/exec/loop.go`,
  `internal/agent/claude/executor.go`, `internal/backend/gcp/gcp_provider.go`,
  `internal/backend/gcp/templates/scripts/gcp_runtime.sh`
- **Risk**: Low — additive change with no breaking behavior. Touches existing override pattern that's well-tested.