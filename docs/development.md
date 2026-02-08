# Local Development Guide

This guide is for developers working on Spinner itself who want to test changes without creating a release.

## Quick Start

```bash
# 1. Make your code changes
vim cmd/setup.go

# 2. Run the dev setup script
./scripts/dev-setup.sh

# 3. Run setup normally (local binary auto-detected)
./dist/spinner setup --name test

# For GCP
./dist/spinner setup --backend gcp --name test \
    --state-bucket my-dev-bucket \
    --project my-project \
    --zone us-central1-a
```

**Note:** The setup command automatically detects if you're running from source (checks for `dist/spinner-linux-amd64`)
and uses the local binary. No need to set `LOCAL_BUILD` environment variable manually.

## How It Works

### Docker Backend

1. `dev-setup.sh` builds a linux/amd64 binary to `dist/spinner-linux-amd64`
2. When you run `setup`:
    - Setup checks if `dist/spinner-linux-amd64` exists
    - If found, automatically copies it into the build context
    - Sets `LOCAL_BUILD=true` for the Dockerfile
    - Dockerfile uses the local binary instead of downloading from GitHub

### GCP Backend

1. `dev-setup.sh` builds and creates `dist/spinner-dev-linux-amd64.tar.gz`
2. When you run `setup`:
    - Setup checks if `dist/spinner-dev-linux-amd64.tar.gz` exists
    - If found, automatically uploads to `gs://{state-bucket}/local-dev/`
    - Sets `LOCAL_BUILD=true` for the bake VM metadata
    - Bake VM downloads from state bucket instead of GitHub releases

## Production Users

Production users (who download the `spinner` binary from GitHub releases) don't have the source code or the dev script.
When they run `setup`:

1. `LOCAL_BUILD` is not set
2. Docker downloads the latest release from GitHub
3. GCP downloads the latest release from GitHub

Everything works normally without any dev infrastructure.

## Troubleshooting

### "LOCAL_BUILD=true but dist/spinner-linux-amd64 not found"

You need to run `./scripts/dev-setup.sh` first to build the binary.

### "failed to upload local binary"

For GCP, ensure:

- You have `gcloud auth application-default login` configured
- The `--state-bucket` exists and you have write access
- You ran `./scripts/dev-setup.sh` to create the tarball

### Changes not reflected in container/VM

Make sure you:

1. Ran `./scripts/dev-setup.sh` after making changes
2. Rebuilt the image with `setup` (not reusing an old image)
3. For Docker: use `--recreate` flag if the image already exists

## Environment Variables

**`LOCAL_BUILD`:**

- Auto-detected when running from source (if `dist/spinner-linux-amd64` exists)
- Can also be set manually: `LOCAL_BUILD=true ./dist/spinner setup ...`
- When `"true"`: Use local binary instead of downloading from GitHub
- When unset or auto-detection fails: Download from GitHub releases (production behavior)

**Production users** (who download the released `spinner` binary) never have the local dev files, so auto-detection
doesn't trigger.

## Development Workflow

### Testing Docker Changes

```bash
# Make changes to code
vim internal/backend/docker/client.go

# Build and setup
./scripts/dev-setup.sh
./dist/spinner setup --name test-docker

# Test the container
./dist/spinner spin --image test-docker --repo https://github.com/user/repo --prompt "test task"
```

### Testing GCP Changes

```bash
# Make changes to code
vim internal/backend/gcp/gcp_provider.go

# Build and setup
./scripts/dev-setup.sh
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

- ✅ **No release needed for local development**
- ✅ **Works for both developers and production users**
- ✅ **Consistent across Docker and GCP backends**
- ✅ **Automatic for GCP** — upload handled by setup command
- ✅ **Uses existing infrastructure** — state-bucket is already required
- ✅ **Simple one-script workflow** — `./scripts/dev-setup.sh` does everything

## Architecture

The implementation uses environment variable detection to choose between development and production mode:

**Development Mode** (`LOCAL_BUILD=true`):

- Docker: Copy binary from `dist/spinner-linux-amd64` to build context
- GCP: Upload to `gs://{state-bucket}/local-dev/`, download during bake

**Production Mode** (`LOCAL_BUILD` unset):

- Docker: Download from GitHub releases during image build
- GCP: Download from GitHub releases during bake

This separation ensures production users never accidentally use dev infrastructure, and developers have a clear,
explicit workflow.

### Shared Installation Logic

To avoid duplication, both Docker and GCP use a shared `install_spinner.sh` script (
`templates/scripts/install_spinner.sh`) that handles:

- Detecting `LOCAL_BUILD` environment variable
- Downloading from GitHub releases (production mode)
- Installing from local build (development mode)

**Docker** copies this script into the build context and runs it during image build.

**GCP** passes this script as metadata (`spinner-install-script`) and the bake VM downloads and executes it.
