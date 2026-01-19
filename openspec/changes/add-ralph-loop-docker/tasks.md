## 1. CLI Changes

- [x] 1.1 Add optional `--prompt` flag to Spin command (string value)
- [x] 1.2 Add optional `--branch` flag to Spin command
- [x] 1.3 Add optional `--max-iterations` flag with default 100
- [x] 1.4 Remove validation requiring both `--prompt` and `--branch`
- [x] 1.5 Pass `PROMPT`, `BRANCH` (if provided), `MAX_ITERATIONS` environment variables to container

## 2. Ralph Loop Script

- [x] 2.1 Create `templates/scripts/ralph-loop.sh` script
- [x] 2.2 Implement prompt reading from `$PROMPT` environment variable
- [x] 2.3 Implement iteration counter with `$MAX_ITERATIONS` limit
- [x] 2.4 Implement main loop: pipe prompt to `claude --dangerously-skip-permissions`
- [x] 2.5 Implement output capture and `~~ FEATURE_COMPLETED ~~` detection
- [x] 2.6 Implement exit messages for completion vs max iterations reached

## 3. Startup Script Integration

- [x] 3.1 Modify `templates/scripts/startup.sh` to check if `PROMPT` is set
- [x] 3.2 Conditionally checkout branch from `$BRANCH` if provided
- [x] 3.3 Stay on default branch if `BRANCH` not set but `PROMPT` is set
- [x] 3.4 Create branch if it doesn't exist (from default branch)
- [x] 3.5 Execute ralph-loop.sh after branch handling (when PROMPT is set)
- [x] 3.6 Stay idle with `tail -f /dev/null` when PROMPT not set

## 4. Dockerfile Updates

- [x] 4.1 Add `COPY ralph-loop.sh /usr/local/bin/ralph-loop.sh` to Dockerfile template
- [x] 4.2 Ensure ralph-loop.sh has execute permissions

## 5. Docker Build Integration

- [x] 5.1 Update `src/utils/docker.ts` to copy ralph-loop.sh to build context

## 6. Testing

- [x] 6.1 Update test for `--prompt` without `--branch` (should succeed)
- [x] 6.2 Update test for `--branch` without `--prompt` (should create idle container)
- [x] 6.3 Update test for `--max-iterations` flag (optional, default 100)
- [ ] 6.4 Manual integration test: verify loop runs and detects completion signal

## 7. SOLID Refactoring

- [x] 7.1 Create `validatePrerequisites()` utility function
- [x] 7.2 Create `generateContainerName()` utility function
- [x] 7.3 Create `buildDockerRunCommand()` utility function
- [x] 7.4 Create `executeDockerRun()` utility function
- [x] 7.5 Create `verifyContainerStatus()` utility function
- [x] 7.6 Define TypeScript interfaces for SpinConfig, ValidationResult, ContainerResult
- [x] 7.7 Refactor Spin.tsx to use utility functions
- [x] 7.8 Update CLAUDE.md with SOLID principles and coding standards

## 8. Documentation

- [x] 8.1 Update proposal.md to reflect optional flags and SOLID refactoring
- [x] 8.2 Update design.md to include SOLID approach
- [x] 8.3 Update spec to allow Ralph loop without branch
- [x] 8.4 Update App.tsx help text with new behavior
- [x] 8.5 Add examples showing prompt without branch
