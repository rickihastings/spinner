## Context

The spinner CLI creates Docker containers for isolated development environments. Currently, every `spin` invocation creates a new container with a timestamp-based name, even when spinning the same repo/branch/image combination. This creates container sprawl and prevents work resumption in existing environments.

Deterministic naming enables container reuse by generating consistent names based on the development context (image + repo + branch).

### Stakeholders
- Developers using spinner who repeatedly work on the same repos/branches
- CI/CD systems that may benefit from container caching
- Users managing container cleanup

### Constraints
- Names must be valid Docker container names (alphanumeric, hyphens, underscores)
- Names should be human-readable for easier identification
- Must handle repository URLs in both SSH and HTTPS formats
- Branch names may contain special characters requiring sanitization

## Goals / Non-Goals

### Goals
- Generate deterministic container names based on image + repo + branch (optional)
- Reuse existing containers when they match the current context
- Provide `--recreate` flag for forced recreation
- Update CLI output to clearly indicate reuse vs creation
- Sanitize inputs to create valid Docker container names
- Handle both running and stopped containers appropriately

### Non-Goals
- Automatic cleanup of old containers
- Container versioning or migration
- State preservation guarantees (users control git commits)
- Cross-machine container sharing

## Technical Implementation Plan

### Component Map

| File                      | Change                                               | Type   |
|---------------------------|------------------------------------------------------|--------|
| `src/utils/docker.ts`     | Modify `generateContainerName()`, add reuse logic    | modify |
| `src/commands/Spin.tsx`   | Handle container reuse scenarios, add `--recreate`   | modify |
| `src/App.tsx`             | Add `--recreate` flag definition                     | modify |
| `tests/spin/*.sh`         | Add tests for deterministic naming and reuse         | create |

### Approach

1. **Container name generation** (utils/docker.ts):
   - Extract components from inputs:
     - Image: `spinner:default` → `spinner-default`
     - Repo: `git@github.com:user/my-project.git` → `my-project`
     - Repo: `https://github.com/user/my-project.git` → `my-project`
     - Branch (optional): `feature/auth-v2` → `feature-auth-v2`
   - Sanitize each component:
     - Replace invalid chars (`:`, `/`, `.`, `@`) with hyphens
     - Remove consecutive hyphens
     - Lowercase the result
   - Combine: `{image}-{repo}` or `{image}-{repo}-{branch}`

2. **Container reuse logic** (utils/docker.ts):
   - New function: `checkContainerExists(name: string)` → returns status: 'running' | 'stopped' | 'none'
   - New function: `restartContainer(name: string)` → restarts stopped container
   - New function: `removeContainer(name: string)` → force-removes container
   - Modified function: `executeDockerRun()` → only creates if container doesn't exist

3. **Spin command flow** (Spin.tsx):
   - Generate deterministic name
   - Check if container exists
   - If `--recreate` flag set:
     - Remove existing container (if any)
     - Create new container
     - Output: "Container recreated: {name}"
   - Else if container running:
     - Skip creation
     - Output: "Reusing running container: {name}"
   - Else if container stopped:
     - Restart container
     - Output: "Restarted container: {name}"
   - Else:
     - Create new container
     - Output: "Created container: {name}"

4. **CLI flag** (App.tsx):
   - Add `--recreate` boolean flag (optional, default false)
   - Pass to Spin component

### Name Sanitization Algorithm

```typescript
function sanitizeComponent(input: string): string {
  return input
    .toLowerCase()
    .replace(/[^a-z0-9-_]/g, '-')  // Replace invalid chars with hyphen
    .replace(/-+/g, '-')            // Collapse consecutive hyphens
    .replace(/^-|-$/g, '');         // Trim leading/trailing hyphens
}

function extractRepoName(repoUrl: string): string {
  // Extract repo name from SSH or HTTPS URL
  // git@github.com:user/repo.git → repo
  // https://github.com/user/repo.git → repo
  const match = repoUrl.match(/([^/:]+)(\.git)?$/);
  return match ? match[1].replace(/\.git$/, '') : 'sandbox';
}

export function generateContainerName(config: SpinConfig): string {
  const imagePart = sanitizeComponent(config.image.replace(':', '-'));
  const repoPart = sanitizeComponent(extractRepoName(config.repo));
  const branchPart = config.branch ? sanitizeComponent(config.branch) : null;

  if (branchPart) {
    return `${imagePart}-${repoPart}-${branchPart}`;
  }
  return `${imagePart}-${repoPart}`;
}
```

### Patterns to Follow

- See `src/utils/docker.ts:158-166` for existing `generateContainerName()` pattern
- See `src/utils/docker.ts:253-284` for `verifyContainerStatus()` pattern using `docker inspect`
- Use `execSync()` with try/catch for Docker commands
- Return structured result objects with `success` and `error` fields

### Key Decisions

| Decision                                     | Rationale                                                                        |
|----------------------------------------------|----------------------------------------------------------------------------------|
| Human-readable names over hashes             | Easier debugging and container identification                                    |
| Sanitize with hyphens, not underscores       | Hyphens are more conventional in Docker container names                          |
| Include image name in container name         | Two projects with same repo name but different images should be separate         |
| Branch is optional in name                   | Matches CLI behavior where branch flag is optional                               |
| Default to reuse, require flag for recreate  | Safer default - users don't accidentally lose uncommitted work                   |
| Restart stopped containers automatically     | Most intuitive behavior - resume previous state                                  |
| No automatic cleanup of old containers       | User controls cleanup to avoid data loss                                         |

## Risks / Trade-offs

| Risk                                              | Mitigation                                         |
|---------------------------------------------------|----------------------------------------------------|
| Name collisions from aggressive sanitization     | Test with diverse repo/branch names                |
| Users confused by reuse behavior                  | Clear CLI output indicating reuse vs creation      |
| Uncommitted work lost with `--recreate`          | Require explicit flag, warn in help text           |
| Very long names from long repo/branch names      | Docker supports up to 255 chars; acceptable        |

## Open Questions

None - all questions resolved during requirements discussion.
