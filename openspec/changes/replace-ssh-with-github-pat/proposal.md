# Change: Replace SSH with GitHub PAT Token Authentication

## Why

SSH agent forwarding to Docker containers is unreliable. We need a more robust authentication mechanism for git
operations in sandboxed environments.

## What Changes

- **BREAKING**: Remove SSH agent socket forwarding from container setup
- **BREAKING**: Replace SSH authentication with GitHub Personal Access Token (PAT) authentication
- Add `gh` CLI installation requirement to Docker images
- Configure git to use `gh` as credential helper with long-lived cache
- Accept GitHub token via `GITHUB_TOKEN` environment variable (not CLI flag to avoid bash history exposure)
- Configure `gh auth login` with token and `gh auth setup-git` for credential helper

## Impact

- **Affected specs**: `cli-spin`
- **Affected code**:
    - `src/commands/Spin.tsx` - Remove SSH agent socket mounting, add GITHUB_TOKEN env var handling
    - `src/utils/docker.ts` - Update container configuration
    - `templates/docker/extending.template` - Add `gh` CLI installation
    - `templates/scripts/startup.sh` - Replace SSH git operations with `gh` authentication setup
- **Breaking change**: Users must provide `GITHUB_TOKEN` environment variable instead of running `ssh-agent`
- **Backwards compatibility**: Not required per user request
