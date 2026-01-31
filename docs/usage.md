# Development Setup & Workflow

## Package Manager

- **Go modules for dependencies**: This project uses Go modules for managing Go dependencies
- **npm for OpenSpec only**: npm is used only for OpenSpec tooling (`@fission-ai/openspec` package)
- Use `go mod download` for Go dependencies
- Use `npm install` only when working with OpenSpec features

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
go build -o dist/spinner

# 2. Setup (required before first spin)
./dist/spinner setup --name default

# 3. Test
./dist/spinner spin --image default --repo . --prompt "your test prompt"
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
```

### Important Notes

- The `--image` parameter must match a `--name` from a previous setup
- Repository must be a valid git URL (https://, http://, or git@)
- Either `--prompt` or `--branch` (or both) must be provided
- Default `max-iterations` is 30 if not specified
