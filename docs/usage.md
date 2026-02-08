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

## Environment Variable Configuration

All command-line flags can be overridden using environment variables with the `SPINNER_` prefix (e.g., `SPINNER_IMAGE`, `SPINNER_REPO`). Command-line flags take precedence over environment variables.

A `.env` file in the current directory is also loaded automatically.

## User Guides

For end-user documentation on running Spinner, see the [guides](guides/) directory:

- **[Docker Guide](guides/docker.md)** — Setting up sandboxes, running autonomous agents, and monitoring progress
