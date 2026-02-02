# Implementation Tasks

## 1.0 Implement state management

- [x] 1.1 Create `internal/exec/state.go` with State struct (branch, iteration count, status, timestamps)
- [x] 1.2 Implement LoadState function with JSON unmarshaling and validation
- [x] 1.3 Implement SaveState function with atomic write (temp file + rename)
- [x] 1.4 Add unit tests for state load/save with various scenarios
- [x] 1.5 Verify tests pass

## 2.0 Implement configuration loading

- [x] 2.1 Create `internal/exec/config.go` with Config struct matching env vars
- [x] 2.2 Implement LoadConfig function reading PROMPT, MAX_ITERATIONS, BRANCH, LOG_DIR, CONTAINER_NAME
- [x] 2.3 Add validation for required fields (PROMPT, MAX_ITERATIONS)
- [x] 2.4 Add unit tests for config loading with missing/present env vars
- [x] 2.5 Verify tests pass

## 3.0 Implement Claude CLI integration

- [x] 3.1 Create `internal/exec/claude.go` with ClaudeMessage struct for JSON parsing
- [x] 3.2 Implement RunClaude function using exec.Command with streaming JSON output
- [x] 3.3 Add JSON parsing with error detection (rate_limit_error, auth_error)
- [x] 3.4 Add completion signal detection in message text
- [x] 3.5 Implement log file writing with TeeReader pattern
- [x] 3.6 Add unit tests for JSON parsing various Claude output formats
- [x] 3.7 Verify tests pass

## 4.0 Implement Git push automation

- [x] 4.1 Create `internal/exec/git.go` with PushChanges function
- [x] 4.2 Implement simple push logic: try `git push`, if fails try `git push -u origin <branch>`
- [x] 4.3 Add error handling that continues on failure (non-blocking)
- [x] 4.4 Add unit tests for push function with mock exec.Command
- [x] 4.5 Verify tests pass

## 5.0 Implement main iteration loop

- [x] 5.1 Create `internal/exec/loop.go` with main Run function
- [x] 5.2 Implement iteration loop matching bash logic (for loop up to MAX_ITERATIONS)
- [x] 5.3 Add rate limit wait function with countdown timer
- [x] 5.4 Integrate Claude execution, git push, state updates per iteration
- [x] 5.5 Handle completion signal, rate limits, auth errors with proper exit codes
- [x] 5.6 Add integration tests for full loop execution with mocks
- [x] 5.7 Verify tests pass

## 6.0 Implement exec CLI command

- [x] 6.1 Create `cmd/exec.go` registering command with root
- [x] 6.2 Create `cmd/constructors_exec.go` with NewExecCommand constructor
- [x] 6.3 Add signal handling for Ctrl+C (exit 130)
- [x] 6.4 Wire up config loading, state initialization, loop execution
- [x] 6.5 Add help text and command description
- [x] 6.6 Add unit tests for command construction
- [x] 6.7 Verify tests pass and `spinner exec --help` works

## 7.0 Update Docker templates

- [x] 7.1 Modify `templates/docker/extending.template` to add build stage copying CLI binary
- [x] 7.2 Update Dockerfile to compile CLI for linux/amd64 and copy to /usr/local/bin/spinner
- [x] 7.3 Modify `templates/scripts/startup.sh` to call `spinner exec` instead of ralph-loop.sh
- [x] 7.4 Remove call to /usr/local/bin/ralph-loop.sh
- [x] 7.5 Test template rendering with `spinner setup` command
- [x] 7.6 Verify generated Dockerfile contains CLI binary copy
- [x] 7.7 Verify startup.sh calls `spinner exec`

## 8.0 Remove old bash implementation

- [x] 8.1 Delete `templates/scripts/ralph-loop.sh`
- [x] 8.2 Update any references in documentation
- [x] 8.3 Verify no remaining references with grep
- [x] 8.4 Commit removal

## 9.0 Integration testing

- [x] 9.1 Build test image with `spinner setup --name exec-test`
- [x] 9.2 Run `spinner spin` with test repo and verify state file creation
- [x] 9.3 Test iteration loop with mock prompt (1 iteration)
- [x] 9.4 Test state persistence across container restarts
- [x] 9.5 Test completion signal detection
- [x] 9.6 Test rate limit handling (mock rate limit error)
- [x] 9.7 Verify all edge cases work correctly

## 10.0 Documentation updates

- [ ] 10.1 Update README.md with exec command description
- [ ] 10.2 Update docs/usage.md with state file location and format
- [ ] 10.3 Add migration notes for breaking change
- [ ] 10.4 Update examples to mention new Go-based loop
- [ ] 10.5 Verify documentation is complete and accurate
