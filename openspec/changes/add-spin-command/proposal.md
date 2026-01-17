# Change: Add spin command for development environment containers

## Why

Developers need a streamlined way to spin up isolated development environments from pre-built sandbox images. The
existing `setup` command creates base images with development tools, but there's no command to launch running containers
with project repositories cloned and ready for work. The `spin` command fills this gap by creating persistent containers
with SSH agent forwarding and npm configuration, enabling immediate development work.

## What Changes

- Add `spin` command with flags: --image (required), --repo (required)
- Create persistent Docker containers (not auto-removed)
- Mount SSH agent socket for git authentication (SSH agent forwarding)
- Mount .npmrc from ~/.npmrc for npm registry authentication
- Clone git repository (via SSH) into /workspace on container startup
- Container runs in background; user can exec into it multiple times
- Container lifecycle managed by user (manual stop/remove)

## Impact

- Affected specs: `cli-spin` (new capability)
- Affected code: New command implementation, Docker container orchestration
- Dependencies: SSH agent must be running on host system, valid .npmrc in home directory
