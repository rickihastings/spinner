# Change: Add `exec` Command to Migrate Ralph Loop from Bash to Go

## Why

The current ralph-loop implementation in bash is fragile and unsuitable for the complexity of autonomous iteration
loops. Bash's limited error handling, JSON parsing difficulties, and lack of structured state management make it hard to
maintain and extend. Migrating to Go provides type safety, better error handling, robust JSON parsing, and easier
testing.

## What Changes

- **New CLI command**: `spinner exec` that runs inside Docker containers
- **State management**: JSON state file at `${STATE_DIR}/state.json` (defaults to `/state`, mounted from host) tracking:
    - Branch name
    - Iteration count
    - Status (running/completed/rate_limited/error/auth_error)
    - Timestamps and other metadata
- **Iteration loop**: Replaces bash ralph-loop.sh with Go implementation
- **Claude integration**: Streams Claude CLI output, parses JSON, handles errors
- **Git automation**: Pushes changes after each iteration
- **Completion detection**: Identifies `~~ FEATURE_COMPLETED ~~` signal
- **Container bootstrap**: CLI binary bundled into Docker image via updated Dockerfile template
- **Startup script**: Modified templates/scripts/startup.sh to call `spinner exec` instead of ralph-loop.sh

## Impact

- Affected specs: `cli-spin` (modified), `cli-exec` (added new)
- Affected code:
    - New: `cmd/exec.go`, `cmd/constructors_exec.go`
    - New: `internal/exec/` package (state management, iteration loop, Claude integration)
    - Modified: `templates/docker/extending.template` (add CLI binary to image)
    - Modified: `templates/scripts/startup.sh` (call `spinner exec`)
    - Removed: `templates/scripts/ralph-loop.sh` (replaced by Go)
- **BREAKING**: Containers built with old templates will not work with new spin command