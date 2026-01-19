## 1. CLI Changes

- [ ] 1.1 Add required `--prompt` flag to Spin command (string value)
- [ ] 1.2 Add required `--branch` flag to Spin command
- [ ] 1.3 Add optional `--max-iterations` flag with default 100
- [ ] 1.4 Validate that `--prompt` and `--branch` are provided
- [ ] 1.5 Pass `PROMPT`, `BRANCH`, `MAX_ITERATIONS` environment variables to container

## 2. Ralph Loop Script

- [ ] 2.1 Create `templates/ralph-loop.sh` script
- [ ] 2.2 Implement prompt reading from `$PROMPT` environment variable
- [ ] 2.3 Implement iteration counter with `$MAX_ITERATIONS` limit
- [ ] 2.4 Implement main loop: pipe prompt to `claude --dangerously-skip-permissions`
- [ ] 2.5 Implement output capture and `~~ FEATURE_COMPLETED ~~` detection
- [ ] 2.6 Implement exit messages for completion vs max iterations reached

## 3. Startup Script Integration

- [ ] 3.1 Modify `templates/startup.sh` to checkout branch from `$BRANCH`
- [ ] 3.2 Create branch if it doesn't exist (from default branch)
- [ ] 3.3 Execute ralph-loop.sh after branch checkout

## 4. Dockerfile Updates

- [ ] 4.1 Add `COPY ralph-loop.sh /usr/local/bin/ralph-loop.sh` to Dockerfile template
- [ ] 4.2 Ensure ralph-loop.sh has execute permissions

## 5. Docker Build Integration

- [ ] 5.1 Update `src/utils/docker.ts` to copy ralph-loop.sh to build context

## 6. Testing

- [ ] 6.1 Add test for `--prompt` flag (required, string value)
- [ ] 6.2 Add test for `--branch` flag (required)
- [ ] 6.3 Add test for `--max-iterations` flag (optional, default 100)
- [ ] 6.4 Add test for error when `--prompt` missing
- [ ] 6.5 Add test for error when `--branch` missing
- [ ] 6.6 Manual integration test: verify loop runs and detects completion signal
