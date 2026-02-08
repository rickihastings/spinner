# Design: add-destroy-command

## Technical Implementation Plan

### Component Map

| File                                              | Action | Purpose                                            |
| ------------------------------------------------- | ------ | -------------------------------------------------- |
| `cmd/destroy.go`                                  | create | New Cobra command with factory injection            |
| `cmd/destroy_test.go`                             | create | Unit tests with mock provider                      |
| `cmd/spin.go`                                     | modify | Update management instructions to use `spinner destroy` |

### Approach

The `destroy` command follows the same pattern as the `watch` command, extended to support multiple positional
arguments and host-side state cleanup.

### Command Structure

```
spinner destroy <instance-name>... [--backend docker|gcp] [--gcp-flags...]
```

- Takes one or more positional arguments (instance names) via `cobra.MinimumNArgs(1)`
- Supports `--backend` flag for backend selection (defaults to Docker)
- Supports GCP flags (`--project`, `--zone`, `--state-bucket`) when using GCP backend
- No confirmation prompt (consistent with `spin --recreate` pattern)

### Execution Flow

```
1. Resolve backend via resolveAndValidateBackend()
2. Create provider via factory
3. For each instance name in args:
   a. Check instance status via p.Status()
   b. If InstanceStatusNone → print error, track failure, continue to next
   c. Call p.Remove() → force-destroy the instance
   d. Remove ~/.spinner/<instance-name>/ directory (state + logs)
   e. Print success message
4. If any instance failed → return error
```

### State Directory Cleanup

On successful instance removal, delete the entire `~/.spinner/<instance-name>/` directory which contains:
- `state/state.json` — iteration state
- `logs/` — log files

This uses `os.RemoveAll()` with the path derived from `os.UserHomeDir()` + `/.spinner/` + instance name.
If the directory doesn't exist, this is a no-op (not an error).

### Multi-Instance Error Handling

When multiple instance names are provided:
- Process all instances, don't stop on first failure
- Print per-instance success/failure messages
- Return an aggregate error if any instance failed (e.g., "failed to destroy 2 of 5 instances")

### Spin Command Output Change

Replace backend-specific "To remove:" lines in `cmd/spin.go` with a unified line:

```
To destroy: spinner destroy <instance-name>
```

This applies to both Docker and GCP output blocks. The "To access:" and "To stop:" lines remain
backend-specific since those operations are not (yet) abstracted into spinner commands.

### Key Decisions

- **No confirmation prompt**: Matches existing codebase patterns (`spin --recreate` removes without asking).
  The command name `destroy` is explicit enough to convey intent.
- **Always clean state**: Destroy means full cleanup — instance + host state. Users who want to preserve
  state can simply use `spin --recreate` instead.
- **Continue on failure**: When destroying multiple instances, process all and report failures at the end.
  This is more useful than failing fast when cleaning up several instances.
- **Reuses existing Provider.Remove()**: No provider interface changes needed.
