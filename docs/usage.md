# Development Setup & Workflow

## Build System

This project uses **Make** for build automation and **Go modules** for dependency management.

### Common Commands

```bash
make build          # Build the spinner binary to dist/spinner
make test           # Build and run all tests
make lint           # Run go vet and golangci-lint
make format         # Format code with go fmt
make format-check   # Check formatting (used by pre-commit)
make install-hooks  # Install git pre-commit hooks
make clean          # Remove build artifacts
make snapshot       # Test release build locally (no tag required)
make release        # Create a release (requires git tag)
```

### Pre-commit Hooks

Install git hooks to run format, lint, and test before each commit:

```bash
make install-hooks
```

## Environment Variable Configuration

Spinner uses Viper to support environment variable configuration. All command-line flags can be overridden using environment variables with the `SPINNER_` prefix.

### Supported Environment Variables

**Setup Command:**
- `SPINNER_NAME` - Override `--name` flag
- `SPINNER_BASE_IMAGE` - Override `--base-image` flag
- `SPINNER_DOCKERFILE` - Override `--dockerfile` flag

**Spin Command:**
- `SPINNER_IMAGE` - Override `--image` flag
- `SPINNER_REPO` - Override `--repo` flag
- `SPINNER_PROMPT` - Override `--prompt` flag
- `SPINNER_BRANCH` - Override `--branch` flag
- `SPINNER_MAX_ITERATIONS` - Override `--max-iterations` flag
- `SPINNER_RECREATE` - Override `--recreate` flag (set to `true` or `false`)
- `SPINNER_WATCH` - Override `--watch` flag (set to `true` or `false`)

### Usage Examples

```bash
# Set default image via environment variable
export SPINNER_IMAGE=spinner:default
./dist/spinner spin --repo https://github.com/user/repo --prompt "task"

# Set multiple configuration values
export SPINNER_IMAGE=spinner:default
export SPINNER_MAX_ITERATIONS=50
./dist/spinner spin --repo https://github.com/user/repo --prompt "task"

# Command-line flags take precedence over environment variables
export SPINNER_IMAGE=spinner:default
./dist/spinner spin --image spinner:custom --repo https://github.com/user/repo
# Uses spinner:custom, not spinner:default
```

### Configuration Precedence

Configuration values are applied in this order (highest to lowest priority):
1. Command-line flags
2. Environment variables
3. Default values

## Development Workflow

### Build and Test

Always build before testing CLI commands:

```bash
# 1. Build
make build

# 2. Setup (required before first spin)
./dist/spinner setup --name default

# 3. Test
./dist/spinner spin --image default --repo . --prompt "your test prompt"
```

### Running Tests

```bash
# Run all tests
make test

# Run tests for a specific package
go test ./internal/docker/...
```

### Debugging Checklist

- Docker is running: `docker ps` should execute without errors
- You're in a git repository with proper configuration
- The `--image` matches a previous `--name` from setup

## Working Command Examples

These are tested, working examples for future reference:

### Setup Command Examples

```bash
# Basic setup with default base image
./dist/spinner setup --name spinner:default

# Setup with custom base image
./dist/spinner setup --name ubuntu --base-image ubuntu:22.04

# Setup with custom Dockerfile
./dist/spinner setup --name custom --dockerfile ./path/to/Dockerfile
```

### Spin Command Examples

```bash
# Basic spin with prompt only (uses current branch)
./dist/spinner spin \
  --image spinner:default \
  --repo https://github.com/anthropics/anthropic-quickstarts \
  --prompt "add a readme explaining how to run the customer support agent"

# Spin with specific branch
./dist/spinner spin \
  --image spinner:default \
  --repo https://github.com/anthropics/anthropic-quickstarts \
  --branch main \
  --prompt "fix any linting errors"

# Spin with max iterations limit
./dist/spinner spin \
  --image spinner:default \
  --repo https://github.com/anthropics/anthropic-quickstarts \
  --prompt "refactor the authentication module" \
  --max-iterations 10

# Spin without prompt (interactive mode on current branch)
./dist/spinner spin \
  --image spinner:default \
  --repo https://github.com/user/repo \
  --branch feature/new-feature

# Spin and immediately enter watch mode
./dist/spinner spin \
  --image spinner:default \
  --repo https://github.com/user/repo \
  --prompt "implement feature X" \
  --watch
```

### Watch Command Examples

```bash
# Watch a running container
./dist/spinner watch spinner-default-abc123

# Watch mode displays:
# - Container status (running/stopped/exited)
# - CPU and memory usage
# - Real-time streaming logs
#
# Press 'q' or Ctrl+C to exit
```

### Important Notes

- The `--image` parameter must match a `--name` from a previous setup
- Repository must be a valid git URL (https://, http://, or git@)
- Either `--prompt` or `--branch` (or both) must be provided
- Default `max-iterations` is 30 if not specified
- The `--watch` flag can be combined with any spin flags

## State Management

Spinner maintains persistent state for each running container to track iteration progress and status. This state survives container restarts and allows you to resume work after interruptions.

### State File Location

State is stored in `${STATE_DIR}/state.json` where `STATE_DIR` defaults to `/state` inside the container. This directory is mounted from the host at `~/.spinner/<container-name>/state/` to ensure persistence.

### State File Format

The state file is JSON with the following structure:

```json
{
  "branch": "feature-branch",
  "iteration": 5,
  "status": "running",
  "last_updated": "2026-02-02T10:35:00Z",
  "started_at": "2026-02-02T10:30:00Z",
  "completed_at": "2026-02-02T10:40:00Z",
  "error_message": ""
}
```

**Fields:**

- `branch` - The git branch being worked on
- `iteration` - Current iteration count (increments after each Claude execution)
- `status` - Current execution status (values: `running`, `completed`, `rate_limited`, `error`, `auth_error`)
- `last_updated` - ISO 8601 timestamp of last state update
- `started_at` - ISO 8601 timestamp when execution started
- `completed_at` - ISO 8601 timestamp when execution completed (omitted if not completed)
- `error_message` - Error message if status is `error` or `auth_error` (omitted if no error)

### Status Values

- `running` - Agent is actively working through iterations
- `completed` - Agent detected completion signal (`~~ FEATURE_COMPLETED ~~`) and finished successfully
- `rate_limited` - Hit Claude API rate limit, waiting before retry
- `error` - General execution error occurred
- `auth_error` - Claude authentication failed (check `CLAUDE_CODE_OAUTH_TOKEN`)

### Accessing State

View state while container is running:

```bash
# From host
cat ~/.spinner/<container-name>/state/state.json

# Inside container
docker exec -it <container-name> cat /state/state.json
```

### Resetting State

To start fresh with a new state, either:

1. Use `--recreate` flag which removes the container and its state directory
2. Manually remove the state directory: `rm -rf ~/.spinner/<container-name>/state/`
