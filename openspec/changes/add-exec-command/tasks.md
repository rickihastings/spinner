# Implementation Tasks

## 1.0 Implement state management

- [ ] 1.1 Create `internal/exec/state.go` with State struct (branch, iteration count, status, timestamps)
- [ ] 1.2 Implement LoadState function with JSON unmarshaling and validation
- [ ] 1.3 Implement SaveState function with atomic write (temp file + rename)
- [ ] 1.4 Add unit tests for state load/save with various scenarios
- [ ] 1.5 Verify tests pass

## 2.0 Implement configuration loading

- [ ] 2.1 Create `internal/exec/config.go` with Config struct matching env vars
- [ ] 2.2 Implement LoadConfig function reading PROMPT, MAX_ITERATIONS, BRANCH, LOG_DIR, CONTAINER_NAME
- [ ] 2.3 Add validation for required fields (PROMPT, MAX_ITERATIONS)
- [ ] 2.4 Add unit tests for config loading with missing/present env vars
- [ ] 2.5 Verify tests pass

## 3.0 Implement Claude CLI integration

- [ ] 3.1 Create `internal/exec/claude.go` with ClaudeMessage struct for JSON parsing
- [ ] 3.2 Implement RunClaude function using exec.Command with streaming JSON output
- [ ] 3.3 Add JSON parsing with error detection (rate_limit_error, auth_error)
- [ ] 3.4 Add completion signal detection in message text
- [ ] 3.5 Implement log file writing with TeeReader pattern
- [ ] 3.6 Add unit tests for JSON parsing various Claude output formats
- [ ] 3.7 Verify tests pass

## 4.0 Implement Git push automation

- [ ] 4.1 Create `internal/exec/git.go` with PushChanges function
- [ ] 4.2 Implement simple push logic: try `git push`, if fails try `git push -u origin <branch>`
- [ ] 4.3 Add error handling that continues on failure (non-blocking)
- [ ] 4.4 Add unit tests for push function with mock exec.Command
- [ ] 4.5 Verify tests pass

## 5.0 Implement main iteration loop

- [ ] 5.1 Create `internal/exec/loop.go` with main Run function
- [ ] 5.2 Implement iteration loop matching bash logic (for loop up to MAX_ITERATIONS)
- [ ] 5.3 Add rate limit wait function with countdown timer
- [ ] 5.4 Integrate Claude execution, git push, state updates per iteration
- [ ] 5.5 Handle completion signal, rate limits, auth errors with proper exit codes
- [ ] 5.6 Add integration tests for full loop execution with mocks
- [ ] 5.7 Verify tests pass

## 6.0 Implement exec CLI command

- [ ] 6.1 Create `cmd/exec.go` registering command with root
- [ ] 6.2 Create `cmd/constructors_exec.go` with NewExecCommand constructor
- [ ] 6.3 Add signal handling for Ctrl+C (exit 130)
- [ ] 6.4 Wire up config loading, state initialization, loop execution
- [ ] 6.5 Add help text and command description
- [ ] 6.6 Add unit tests for command construction
- [ ] 6.7 Verify tests pass and `spinner exec --help` works

## 7.0 Update Docker templates

- [ ] 7.1 Modify `templates/docker/extending.template` to add build stage copying CLI binary
- [ ] 7.2 Update Dockerfile to compile CLI for linux/amd64 and copy to /usr/local/bin/spinner
- [ ] 7.3 Modify `templates/scripts/startup.sh` to call `spinner exec` instead of ralph-loop.sh
- [ ] 7.4 Remove call to /usr/local/bin/ralph-loop.sh
- [ ] 7.5 Test template rendering with `spinner setup` command
- [ ] 7.6 Verify generated Dockerfile contains CLI binary copy
- [ ] 7.7 Verify startup.sh calls `spinner exec`

## 8.0 Remove old bash implementation

- [ ] 8.1 Delete `templates/scripts/ralph-loop.sh`
- [ ] 8.2 Update any references in documentation
- [ ] 8.3 Verify no remaining references with grep
- [ ] 8.4 Commit removal

## 9.0 Integration testing

- [ ] 9.1 Build test image with `spinner setup --name exec-test`
- [ ] 9.2 Run `spinner spin` with test repo and verify state file creation
- [ ] 9.3 Test iteration loop with mock prompt (1 iteration)
- [ ] 9.4 Test state persistence across container restarts
- [ ] 9.5 Test completion signal detection
- [ ] 9.6 Test rate limit handling (mock rate limit error)
- [ ] 9.7 Verify all edge cases work correctly

## 10.0 Documentation updates

- [ ] 10.1 Update README.md with exec command description
- [ ] 10.2 Update docs/usage.md with state file location and format
- [ ] 10.3 Add migration notes for breaking change
- [ ] 10.4 Update examples to mention new Go-based loop
- [ ] 10.5 Verify documentation is complete and accurate
