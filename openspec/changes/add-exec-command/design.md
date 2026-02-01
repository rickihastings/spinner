# Design: Add `exec` Command

## Context

The ralph-loop bash script is fragile and difficult to maintain. It handles:

- Iteration loops with max iteration limits
- Claude CLI execution with streaming JSON output
- Rate limit detection and wait periods
- Authentication error handling
- Git push automation
- Feature completion detection

Moving to Go provides better structure, error handling, and testability.

## Goals / Non-Goals

**Goals:**

- Replace bash ralph-loop with Go implementation maintaining identical behavior
- Track iteration state in persistent JSON file
- Bootstrap CLI binary into Docker images
- Support both autonomous (with prompt) and interactive (without prompt) modes
- Maintain backward compatibility with existing container startup flow

**Non-Goals:**

- Changing the Claude CLI integration (still uses `claude -p --output-format=stream-json`)
- Modifying the completion signal (`~~ FEATURE_COMPLETED ~~`)
- Adding new ralph-loop features beyond current bash behavior

## Technical Implementation Plan

### Component Map

**New files:**

- `cmd/exec.go` - CLI command entry point (create)
- `cmd/constructors_exec.go` - Command constructor with dependency injection (create)
- `internal/exec/config.go` - Configuration struct and loading from env vars (create)
- `internal/exec/state.go` - JSON state file management (create)
- `internal/exec/loop.go` - Main iteration loop logic (create)
- `internal/exec/claude.go` - Claude CLI execution and output parsing (create)
- `internal/exec/git.go` - Simplified git push (try push, fallback to push -u) (create)

**Modified files:**

- `templates/docker/extending.template` - Add CLI binary copy to Dockerfile (modify)
- `templates/scripts/startup.sh` - Replace ralph-loop.sh call with `spinner exec` (modify)

**Removed files:**

- `templates/scripts/ralph-loop.sh` - Replaced by Go implementation (delete)

### Approach

1. **Create `internal/exec` package** with:
    - State management: Load/save JSON state file
    - Config loading: Parse environment variables (PROMPT, MAX_ITERATIONS, BRANCH, LOG_DIR)
    - Iteration loop: Main control flow matching bash logic
    - Claude integration: Execute command, stream output, parse JSON
    - Git automation: Simple push (try `git push`, fallback to `git push -u origin <branch>`)

2. **Create `cmd/exec.go`** command:
    - No flags (all config from env vars)
    - Loads config from environment
    - Initializes state file at `~/.spinner/{CONTAINER_NAME}/state.json`
    - Runs iteration loop
    - Handles Ctrl+C gracefully

3. **Update Docker templates**:
    - Copy compiled CLI binary into image at `/usr/local/bin/spinner`
    - Modify startup.sh to call `spinner exec` instead of ralph-loop.sh

4. **Testing strategy**:
    - Unit tests for state management (load/save JSON)
    - Unit tests for config loading
    - Unit tests for Claude output parsing
    - Integration test using mock exec.Command for claude CLI
    - Test state persistence across runs

### Patterns to Follow

- Use `exec.Command("claude", ...)` similar to existing Git commands
- Follow error handling pattern from `internal/docker/client.go`
- Use `os.Getenv()` for config like existing startup.sh
- State file format: JSON with clear fields matching bash tracking needs
- Signal handling: Use `signal.Notify()` for graceful shutdown

### Key Decisions

**Decision 1: Environment variable config (no CLI flags)**

- Rationale: Exec runs inside container with env vars already set by docker run command. No need for flag parsing
  overhead. Matches current bash behavior.

**Decision 2: State file location at `~/.spinner/{CONTAINER_NAME}/state.json`**

- Rationale: Container name is unique per image+repo+branch. Mount this directory from host to persist state across
  container recreations. Home directory is `/home/spinner` inside container.

**Decision 3: Keep Claude CLI interface unchanged**

- Rationale: Current streaming JSON format works well. No need to change Anthropic API integration. Just improve parsing
  with Go's json.Decoder.

**Decision 4: Bootstrap CLI binary into image at build time**

- Rationale: Ensures correct version of CLI matches the image. Simpler than mounting or downloading at runtime.

**Decision 5: Support both modes (with/without prompt)**

- Rationale: Without prompt, container is interactive workspace. With prompt, runs autonomous loop. This matches current
  bash behavior in startup.sh.

## Risks / Trade-offs

**Risk 1: State file corruption**

- Mitigation: Write to temp file and atomic rename. Add validation on load.

**Risk 2: Breaking change for existing containers**

- Mitigation: Document in proposal. Users must rebuild images with new templates.

**Risk 3: Cross-compilation for Docker (linux/amd64)**

- Mitigation: Use `GOOS=linux GOARCH=amd64 go build` during image build process.

## Open Questions

None - user confirmed exec runs inside container, state persists, and both modes supported.
