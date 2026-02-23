# Development Guide

This guide covers building, testing, and iterating on Spinner itself.

## Build System

This project uses **Make** for build automation and **Go modules** for dependency management.

### Common Commands

```bash
make build          # Build native + linux/amd64 binaries to dist/
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

## Running Tests

```bash
# Run all tests
make test

# Run tests for a specific package
go test ./internal/docker/...
```

## Debugging Checklist

- Docker is running: `docker ps` should execute without errors
- You're in a git repository with proper configuration
- The `--image` matches a previous `--name` from setup

## Local Dev Setup (Testing Your Changes End-to-End)

```bash
# 1. Make your code changes
vim cmd/setup.go

# 2. Build (produces both native and linux/amd64 binaries)
make build

# 3. Run setup normally (local binary auto-detected)
./dist/spinner setup --name test

# For GCP
./dist/spinner setup --backend gcp --name test \
    --state-bucket my-dev-bucket \
    --project my-project \
    --zone us-central1-a
```

**Note:** The setup command automatically detects if you're running from source (checks for `dist/spinner-linux-amd64`)
and uses the local binary. On Linux (e.g. a GCP VM), a dev build will also auto-detect and use the running binary
itself as a fallback when the source tree isn't available.

## How It Works

### Docker Backend

1. `make build` produces `dist/spinner-linux-amd64`
2. When you run `setup`:
    - Setup checks if `dist/spinner-linux-amd64` exists in the source tree
    - If found, automatically copies it into the build context
    - **Fallback:** if no source tree but running a dev build on Linux, uses the running binary itself
    - The `install_spinner.sh` script detects the non-empty `/tmp/spinner` file and uses it
    - If no local binary, `install_spinner.sh` downloads from GitHub releases

### GCP Backend

1. `make build` produces `dist/spinner-linux-amd64`
2. When you run `setup`:
    - Setup checks if a local binary is available (source tree or running binary on Linux)
    - Creates a tarball and uploads to `gs://{state-bucket}/local-dev/`
    - The `install_spinner.sh` script tries the state bucket path when `STATE_BUCKET` is set
    - If no dev tarball found, downloads from GitHub releases

## Production Users

Production users (who download the `spinner` binary from GitHub releases) don't have the source code or dev builds.
When they run `setup`:

1. No local binary exists in the Docker build context (empty placeholder)
2. No dev tarball exists in the GCS state bucket
3. `install_spinner.sh` falls through to downloading from GitHub releases

Everything works normally without any dev infrastructure.

## Troubleshooting

### "dist/spinner-linux-amd64 not found"

Run `make build` first to build the binary.

### "failed to upload local binary"

For GCP, ensure:

- You have `gcloud auth application-default login` configured
- The `--state-bucket` exists and you have write access
- You ran `make build` to create the linux binary

### Changes not reflected in container/VM

Make sure you:

1. Ran `make build` after making changes
2. Rebuilt the image with `setup` (not reusing an old image)
3. For Docker: use `--recreate` flag if the image already exists

## Development Workflow

### Testing Docker Changes

```bash
# Make changes to code
vim internal/backend/docker/client.go

# Build and setup
make build
./dist/spinner setup --name test-docker

# Test the container
./dist/spinner spin --image test-docker --repo https://github.com/user/repo --prompt "test task"
```

### Testing GCP Changes

```bash
# Make changes to code
vim internal/backend/gcp/gcp_provider.go

# Build and setup
make build
./dist/spinner setup --backend gcp --name test-gcp \
    --state-bucket my-dev-bucket \
    --project my-project \
    --zone us-central1-a

# Spin up VM
./dist/spinner spin --backend gcp --image test-gcp \
    --repo https://github.com/user/repo \
    --prompt "test task"
```

## Benefits

- **No release needed for local development**
- **Works for both developers and production users**
- **Consistent across Docker and GCP backends**
- **Auto-detects on Linux** — dev builds running on GCP VMs work without the source tree
- **Uses existing infrastructure** — state-bucket is already required

## Architecture

The implementation uses automatic binary detection to choose between development and production mode:

**Development Mode** (local binary detected):

- Docker: Copy binary from `dist/spinner-linux-amd64` to build context (or running binary on Linux); `install_spinner.sh` finds it at `/tmp/spinner`
- GCP: Upload to `gs://{state-bucket}/local-dev/`; `install_spinner.sh` downloads from state bucket

**Production Mode** (no local binary):

- Docker: Empty placeholder at `/tmp/spinner`; `install_spinner.sh` downloads from GitHub releases
- GCP: No dev tarball in state bucket; `install_spinner.sh` downloads from GitHub releases

### Shared Installation Logic

To avoid duplication, both Docker and GCP use a shared `install_spinner.sh` script (
`templates/scripts/install_spinner.sh`) that handles:

1. Checking for a non-empty local binary at `/tmp/spinner` (Docker dev)
2. Trying the state bucket dev tarball when `STATE_BUCKET` is set (GCP dev)
3. Downloading from GitHub releases (production mode)

**Docker** copies this script into the build context and runs it during image build.

**GCP** passes this script as metadata (`spinner-install-script`) and the bake VM downloads and executes it.
