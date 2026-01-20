# Change: Enable Deterministic Container Naming

## Why

Currently, `generateContainerName()` appends a timestamp to the repository name, creating a unique container for every `spin` invocation (e.g., `Hello-World-1705783456789`). This leads to container sprawl when developers repeatedly spin up the same repository, branch, and image combination.

Deterministic naming based on image + repo + branch enables container reuse. Developers can resume work in existing environments rather than creating duplicates, reducing resource usage and simplifying container management.

## What Changes

- **MODIFIED**: `generateContainerName()` generates deterministic names based on image name, repository name, and branch (when provided)
  - Format: `{image}-{repo}` or `{image}-{repo}-{branch}` (human-readable, sanitized)
  - Example: `spinner-default-myproject` or `spinner-default-myproject-feature-auth`
- **NEW**: Container reuse logic - check if deterministic name already exists before creating
  - If stopped: restart the container
  - If running: use as-is (output existing container info)
  - If doesn't exist: create new container
- **NEW**: `--recreate` flag to force removal and recreation of existing container
  - When provided, removes existing container and creates fresh one
  - Without flag, reuses existing container
- **MODIFIED**: CLI output distinguishes between "created", "reused (running)", and "restarted (stopped)"

## Impact

- Affected specs: `cli-spin`
- Affected code:
  - `src/utils/docker.ts` - modified `generateContainerName()`, new reuse logic functions
  - `src/commands/Spin.tsx` - updated to handle reuse scenarios and new flag
  - `src/App.tsx` - add `--recreate` flag definition
  - Tests in `tests/spin/` - updated to handle deterministic naming and test reuse scenarios
