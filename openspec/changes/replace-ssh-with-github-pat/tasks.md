# Implementation Tasks: Replace SSH with GitHub PAT Token Authentication

## 1. Update Docker Image Template

- [ ] 1.1 Add `gh` CLI installation to `templates/docker/extending.template`
- [ ] 1.2 Verify `gh` is available in PATH after installation

## 2. Update Startup Script

- [ ] 2.1 Add `GITHUB_TOKEN` validation check in `templates/scripts/startup.sh`
- [ ] 2.2 Configure `gh auth login --with-token` using `$GITHUB_TOKEN`
- [ ] 2.3 Run `gh auth setup-git` to configure git credential helper
- [ ] 2.4 Configure git credential cache with 1-year timeout
- [ ] 2.5 Ensure startup script fails gracefully if token is invalid or missing

## 3. Update CLI Code

- [ ] 3.1 Remove SSH agent socket validation from `src/commands/Spin.tsx`
- [ ] 3.2 Add `GITHUB_TOKEN` environment variable validation in `src/commands/Spin.tsx`
- [ ] 3.3 Remove SSH socket mount from `src/utils/docker.ts:executeDockerRun()`
- [ ] 3.4 Add `GITHUB_TOKEN` to container environment variables in `src/utils/docker.ts:executeDockerRun()`
- [ ] 3.5 Update error messages to reference `GITHUB_TOKEN` instead of SSH

## 4. Update Tests

- [ ] 4.1 Remove SSH-related test validations from existing tests
- [ ] 4.2 Add test for missing `GITHUB_TOKEN` environment variable
- [ ] 4.3 Add test for git clone with PAT authentication (integration test)
- [ ] 4.4 Verify all existing tests pass with new authentication method

## 5. Documentation

- [ ] 5.1 Update README with `GITHUB_TOKEN` setup instructions
- [ ] 5.2 Document required GitHub token scopes (`repo` for private repos)
- [ ] 5.3 Add example of setting `GITHUB_TOKEN` in shell environment
- [ ] 5.4 Remove SSH agent setup documentation

## 6. Validation

- [ ] 6.1 Build updated Docker image with `gh` CLI installed
- [ ] 6.2 Test git clone of public repository with token
- [ ] 6.3 Test git clone of private repository with token
- [ ] 6.4 Verify token is not exposed in container logs or bash history
- [ ] 6.5 Run full test suite and ensure all tests pass
