# Design: add-destroy-command

## Technical Implementation Plan

### Component Map

| File                    | Action | Purpose                                    |
| ----------------------- | ------ | ------------------------------------------ |
| `cmd/destroy.go`        | create | New Cobra command with factory injection    |
| `cmd/destroy_test.go`   | create | Unit tests with mock provider               |

### Approach

The `destroy` command follows the exact same pattern as the `watch` command:

1. **Positional argument**: `spinner destroy <instance-name>` (like `watch <instance-name>`)
2. **Factory injection**: `NewDestroyCommand(f *provider.Factory)` constructor
3. **Backend resolution**: `resolveAndValidateBackend(cmd)` for multi-backend support
4. **Provider call**: `p.Status()` to verify instance exists, then `p.Remove()` to destroy it

### Command Structure

```
spinner destroy <instance-name> [--backend docker|gcp] [--gcp-flags...]
```

- Takes exactly one positional argument (instance name)
- Supports `--backend` flag for backend selection (defaults to Docker)
- Supports GCP flags (`--project`, `--zone`, `--state-bucket`) when using GCP backend
- No confirmation prompt (consistent with `spin --recreate` pattern)
- Force-removes regardless of instance state (running, stopped, exited)

### Execution Flow

```
1. Resolve backend via resolveAndValidateBackend()
2. Create provider via factory
3. Check instance status via p.Status()
4. If InstanceStatusNone → error "instance not found"
5. Call p.Remove() → force-destroy the instance
6. Print success message
```

### Key Decisions

- **No confirmation prompt**: Matches existing codebase patterns (`spin --recreate` removes without asking).
  The command name `destroy` is explicit enough to convey intent.
- **Single instance per invocation**: Keeps the command simple and predictable. Multiple-instance support
  can be added later if needed.
- **No state cleanup by default**: The host-side state directory (`~/.spinner/<name>/state/`) is preserved.
  This is consistent with how `spin --recreate` works today — it removes the container but the state mount
  persists for the newly created container.
- **Reuses existing Provider.Remove()**: No provider interface changes needed. Docker already uses `Force=true`,
  GCP already deletes the VM directly.

### Error Handling

- Instance not found → clear error message with suggestion to check instance name and backend
- Provider.Remove() failure → propagate error with context
- Backend validation failure → standard validation errors from `resolveAndValidateBackend()`
