# cli-exec Specification

## Purpose
TBD - created by archiving change add-exec-command. Update Purpose after archive.
## Requirements
### Requirement: Exec Command Entry Point

The CLI SHALL provide an `exec` command that runs inside Docker containers to execute autonomous iteration loops.

#### Scenario: Exec command runs with valid environment

- **WHEN** `spinner exec` is called inside a container with PROMPT and MAX_ITERATIONS env vars set
- **THEN** the command SHALL load configuration, initialize state, and start the iteration loop

#### Scenario: Exec command missing required env vars

- **WHEN** `spinner exec` is called without PROMPT env var
- **THEN** the command SHALL print an error message and exit with non-zero status

#### Scenario: Ctrl+C signal during exec

- **WHEN** user presses Ctrl+C while exec is running
- **THEN** the command SHALL trap the signal, print interruption message, and exit with status 130

### Requirement: Configuration Loading

The exec command SHALL load configuration from environment variables.

#### Scenario: All environment variables present

- **WHEN** PROMPT, MAX_ITERATIONS, LOG_DIR, BRANCH, STATE_DIR are set
- **THEN** configuration SHALL be loaded with all values

#### Scenario: Optional environment variables missing

- **WHEN** PROMPT and MAX_ITERATIONS are set but LOG_DIR, STATE_DIR, and ANTHROPIC_MODEL are not
- **THEN** configuration SHALL use default STATE_DIR value of "/state" and empty model (Claude CLI default)

#### Scenario: Required environment variable missing

- **WHEN** PROMPT is not set
- **THEN** configuration loading SHALL fail with error "PROMPT environment variable is not set"

#### Scenario: ANTHROPIC_MODEL environment variable present

- **WHEN** ANTHROPIC_MODEL is set in the environment
- **THEN** configuration SHALL include the model value
- **AND** the Claude CLI SHALL use the specified model (via environment variable, no CLI arg needed)

### Requirement: State File Management

The exec command SHALL maintain iteration state in a JSON file at `${STATE_DIR}/state.json` where STATE_DIR defaults to `/state`.

#### Scenario: State file does not exist

- **WHEN** exec runs for the first time
- **THEN** a new state file SHALL be created with initial values (iteration=0, status="running")

#### Scenario: State file exists from previous run

- **WHEN** exec runs and state file exists
- **THEN** the state SHALL be loaded and iteration count SHALL continue from previous value

#### Scenario: State updated after each iteration

- **WHEN** an iteration completes
- **THEN** the state file SHALL be updated with new iteration count, status, and timestamp

#### Scenario: State file write uses atomic operation

- **WHEN** state is saved
- **THEN** it SHALL write to a temporary file and atomically rename to prevent corruption

### Requirement: Iteration Loop Execution

The exec command SHALL run Claude CLI in a loop up to MAX_ITERATIONS times.

#### Scenario: Successful iteration completion

- **WHEN** an iteration runs successfully
- **THEN** the iteration count SHALL increment and loop SHALL continue

#### Scenario: Max iterations reached

- **WHEN** iteration count reaches MAX_ITERATIONS without completion
- **THEN** exec SHALL print warning message and exit with status 1

#### Scenario: Rate limit detected during iteration

- **WHEN** Claude CLI returns rate_limit_error or overloaded_error
- **THEN** exec SHALL wait 61 minutes, decrement iteration counter, and retry

#### Scenario: Authentication error detected

- **WHEN** Claude CLI returns authentication_error
- **THEN** exec SHALL print error message and exit with status 1

#### Scenario: Feature completion signal detected

- **WHEN** Claude output contains "~~ FEATURE_COMPLETED ~~" on its own line
- **THEN** exec SHALL print success message and exit with status 0

### Requirement: Claude CLI Integration

The exec command SHALL execute the Claude CLI with streaming JSON output and parse results.

#### Scenario: Claude command executed with correct arguments

- **WHEN** iteration runs
- **THEN** exec SHALL run `claude -p --dangerously-skip-permissions --output-format=stream-json --verbose`

#### Scenario: Claude output streamed and logged

- **WHEN** Claude produces output
- **THEN** exec SHALL write output to log file and parse JSON simultaneously

#### Scenario: Claude JSON message parsing

- **WHEN** Claude outputs JSON line `{"type":"assistant","message":{"content":[{"type":"text","text":"Hello"}]}}`
- **THEN** exec SHALL parse and display "Hello"

#### Scenario: Claude error detection in JSON

- **WHEN** Claude outputs `{"error":{"type":"rate_limit_error","message":"..."}}`
- **THEN** exec SHALL detect rate limit error and trigger wait period

### Requirement: Git Push Automation

The exec command SHALL attempt to push changes to remote repository after each iteration.

#### Scenario: Push with tracking branch

- **WHEN** iteration completes
- **THEN** exec SHALL run `git push` (git handles "nothing to push" case automatically)

#### Scenario: No tracking branch exists

- **WHEN** `git push` fails due to no upstream
- **THEN** exec SHALL run `git push -u origin <branch>` to set tracking

#### Scenario: Push fails for other reasons

- **WHEN** git push returns non-zero exit code
- **THEN** exec SHALL print warning but continue to next iteration

### Requirement: Rate Limit Handling

The exec command SHALL wait 61 minutes when rate limit is detected.

#### Scenario: Wait with countdown timer

- **WHEN** rate limit wait starts
- **THEN** exec SHALL display countdown showing remaining time in seconds

#### Scenario: Wait completes

- **WHEN** 61 minute wait finishes
- **THEN** exec SHALL print completion message and retry iteration

#### Scenario: Iteration counter decremented

- **WHEN** rate limit causes retry
- **THEN** iteration counter SHALL be decremented so retry uses same iteration number

### Requirement: Log File Management

The exec command SHALL write Claude output to a log file.

#### Scenario: Log directory created

- **WHEN** exec starts and LOG_DIR does not exist
- **THEN** exec SHALL create the directory with 0755 permissions

#### Scenario: Log file appended

- **WHEN** Claude produces output
- **THEN** exec SHALL append to LOG_DIR/raw.log file

#### Scenario: Log file permissions

- **WHEN** log file is created
- **THEN** it SHALL have 0644 permissions (readable by owner, group, others)

