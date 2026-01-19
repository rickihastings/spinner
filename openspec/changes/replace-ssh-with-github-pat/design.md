# Design: Replace SSH with GitHub PAT Token Authentication

## Context

SSH agent forwarding (`$SSH_AUTH_SOCK`) is unreliable when forwarding into Docker containers. This causes intermittent
authentication failures for git operations, blocking the Ralph loop and other automated workflows. GitHub Personal
Access Tokens (PATs) with the `gh` CLI provide a more robust authentication mechanism that works reliably in
containerized environments.

## Goals / Non-Goals

### Goals

- Replace SSH-based git authentication with GitHub PAT tokens
- Ensure git operations work reliably in Docker containers
- Keep tokens out of bash history by using environment variables
- Configure long-lived credential caching (1 year timeout)

### Non-Goals

- Supporting SSH authentication as a fallback
- Backwards compatibility with existing SSH-based workflows
- Supporting other git hosting providers (GitLab, Bitbucket) - GitHub only for now

## Technical Implementation Plan

### Component Map

- `src/commands/Spin.tsx` - Remove SSH socket validation and mounting logic, add GITHUB_TOKEN validation (modify)
- `src/utils/docker.ts` - Update `executeDockerRun()` to remove SSH socket mount and add GITHUB_TOKEN env var (modify)
- `templates/docker/extending.template` - Add `gh` CLI installation step (modify)
- `templates/scripts/startup.sh` - Add `gh auth login` and `gh auth setup-git` before git clone (modify)
- Tests:
    - Update existing SSH-related tests to validate GITHUB_TOKEN requirement (modify)
    - Add test for git clone with PAT authentication (create)

### Approach

**Phase 1: Update Docker image template**

1. Add `gh` CLI installation to `extending.template` using official installation method
2. Ensure `gh` is available in PATH before startup script runs

**Phase 2: Update startup script**

1. Check for `GITHUB_TOKEN` environment variable (fail fast if missing)
2. Configure `gh auth login --with-token` using the token from env var
3. Run `gh auth setup-git` to configure git credential helper
4. Configure git credential cache with 1-year timeout:
   `git config --global credential.helper 'cache --timeout=31536000'`
5. Proceed with existing git clone logic

**Phase 3: Update CLI code**

1. Remove SSH agent socket validation in `Spin.tsx`
2. Remove SSH socket mount from `executeDockerRun()` in `docker.ts`
3. Add `GITHUB_TOKEN` environment variable to container configuration
4. Read token from host environment variable (fail if not set)

**Phase 4: Update tests**

1. Remove SSH-related test assertions
2. Add GITHUB_TOKEN requirement validation test
3. Ensure git clone works with token authentication

### Patterns to Follow

- **Environment variable handling**: See how `REPO_URL`, `PROMPT`, `BRANCH` are passed in
  `src/utils/docker.ts:executeDockerRun()`
- **Validation pattern**: See `validatePrerequisites()` in `src/commands/Spin.tsx` for how to validate requirements
  before proceeding
- **Dockerfile template**: See `templates/docker/extending.template` for how to add installation steps
- **Startup script pattern**: See `templates/scripts/startup.sh` for how to add initialization logic

### Key Decisions

- **Token via environment variable only**: Prevents accidental exposure in bash history or logs. Users must set
  `GITHUB_TOKEN` in their shell environment before running `spin`.
- **Long-lived credential cache (1 year)**: Reduces authentication friction during long-running Ralph loops. The
  31536000 second timeout ensures tokens don't expire mid-task.
- **Fail fast on missing token**: If `GITHUB_TOKEN` is not set, fail immediately with clear error message rather than
  attempting git operations that will fail later.
- **Use `gh` CLI instead of direct git credential helper**: The `gh` CLI provides better integration with GitHub's
  authentication system and simplifies token management.

## Risks / Trade-offs

- **Security**: Tokens in environment variables are more secure than in CLI args (not in bash history) but less secure
  than SSH keys. Mitigation: Document best practices for token scope (minimum required permissions).
- **GitHub-only**: This approach works only for GitHub repositories. Mitigation: Acceptable for current use case; can
  add other providers in future changes if needed.
- **Token expiration**: PATs can expire or be revoked. Mitigation: Clear error messages when authentication fails;
  document token refresh process.

## Open Questions

- Should we support a fallback to read token from a file (e.g., `~/.github-token`)? **Decision: No, environment variable
  only for simplicity and security.**
- What permissions/scopes should the GitHub token have? **Decision: Document in README that token needs `repo` scope for
  private repositories.**
