# Spec Delta: CLI Spin - Golang Migration

## MODIFIED Requirements

### Requirement: Spin Command Flags
The CLI SHALL accept required and optional flags for the spin command, validated by Cobra's flag parsing system.

#### Scenario: All required flags provided
- **WHEN** user runs `spinner spin --image <image> --repo <repo>`
- **THEN** the CLI SHALL proceed with container creation using Cobra's flag parsing

#### Scenario: Missing image flag
- **WHEN** user runs `spinner spin` without --image flag
- **THEN** the CLI SHALL print an error message and exit with non-zero status (enforced by Cobra's MarkFlagRequired)

#### Scenario: Missing repo flag
- **WHEN** user runs `spinner spin` without --repo flag
- **THEN** the CLI SHALL print an error message and exit with non-zero status (enforced by Cobra's MarkFlagRequired)

#### Scenario: Invalid image name
- **WHEN** user provides a non-existent image name
- **THEN** docker operations SHALL fail with appropriate error messages from docker CLI

### Requirement: Prompt Flag for Ralph Loop
The CLI SHALL support the --prompt flag for autonomous implementation, implemented in Go with identical behavior.

#### Scenario: Prompt provided with branch
- **WHEN** user runs `spinner spin --image <image> --repo <repo> --prompt "implement X" --branch feature`
- **THEN** the CLI SHALL start the container and execute ralph-loop with the provided prompt on the specified branch

#### Scenario: Prompt provided without branch
- **WHEN** user runs `spinner spin --image <image> --repo <repo> --prompt "implement X"` without --branch flag
- **THEN** the CLI SHALL start the container and execute ralph-loop with the provided prompt on the default branch

#### Scenario: Prompt not provided
- **WHEN** user runs `spinner spin --image <image> --repo <repo>` without --prompt flag
- **THEN** the CLI SHALL start the container without executing ralph-loop

### Requirement: Branch Flag
The CLI SHALL support the --branch flag for specifying which git branch to use, implemented in Go.

#### Scenario: Branch provided
- **WHEN** user runs `spinner spin --image <image> --repo <repo> --branch feature-x`
- **THEN** the container startup script SHALL check out the specified branch after cloning

#### Scenario: Branch does not exist
- **WHEN** user provides a non-existent branch name
- **THEN** the git checkout operation SHALL fail inside the container with an error message

#### Scenario: Branch not provided but prompt provided
- **WHEN** user runs `spinner spin --image <image> --repo <repo> --prompt "task"`
- **THEN** the CLI SHALL use the repository's default branch

### Requirement: Max Iterations Flag
The CLI SHALL support the --max-iterations flag for controlling ralph-loop execution, implemented in Go.

#### Scenario: Max iterations provided
- **WHEN** user runs `spinner spin --image <image> --repo <repo> --prompt "task" --max-iterations 50`
- **THEN** the CLI SHALL pass RALPH_MAX_ITERATIONS=50 environment variable to the container

#### Scenario: Max iterations not provided
- **WHEN** user runs `spinner spin` without --max-iterations flag
- **THEN** the CLI SHALL default to 100 iterations (RALPH_MAX_ITERATIONS=100)

#### Scenario: Max iterations reached
- **WHEN** ralph-loop reaches the max iteration limit
- **THEN** ralph-loop SHALL stop and the container SHALL remain running for manual inspection

### Requirement: Ralph Loop Execution
The CLI SHALL execute ralph-loop inside the container when a prompt is provided, implemented using Go's exec.Command.

#### Scenario: Ralph loop iteration
- **WHEN** ralph-loop is running
- **THEN** the CLI SHALL stream ralph-loop output to the host terminal in real-time via docker logs -f or equivalent

### Requirement: Feature Completion Detection
The CLI SHALL detect when ralph-loop signals feature completion, implemented in Go by monitoring container output.

#### Scenario: Completion signal detected
- **WHEN** ralph-loop outputs `~~ FEATURE_COMPLETED ~~`
- **THEN** the CLI SHALL stop following logs and display completion message

#### Scenario: No completion signal
- **WHEN** ralph-loop completes without outputting the completion signal
- **THEN** the CLI SHALL continue following logs until max iterations or manual interrupt

### Requirement: NPM Configuration Mount
The CLI SHALL mount the user's .npmrc file if present, implemented using Go's os.Stat for file existence checks.

#### Scenario: npmrc file exists
- **WHEN** ~/.npmrc exists on the host
- **THEN** the CLI SHALL mount it into the container at /root/.npmrc using docker run -v flag

#### Scenario: npmrc file missing
- **WHEN** ~/.npmrc does not exist on the host
- **THEN** the CLI SHALL not attempt to mount .npmrc (checked via os.Stat in Go)

### Requirement: Repository Cloning
The CLI SHALL support repository cloning via the container startup script, with SSH agent forwarding configured by Go.

#### Scenario: Successful repository clone
- **WHEN** the container starts with a valid repo URL
- **THEN** the startup script SHALL clone the repository into /workspace and the CLI SHALL verify success

#### Scenario: Clone failure due to authentication
- **WHEN** the SSH agent is not running or keys are not available
- **THEN** the git clone operation SHALL fail with an authentication error message

#### Scenario: Clone failure due to invalid URL
- **WHEN** user provides an invalid git repository URL
- **THEN** the git clone operation SHALL fail with a URL error message

### Requirement: Persistent Container
The CLI SHALL create persistent containers that run in detached mode, implemented using Go's exec.Command for docker run.

#### Scenario: Container runs in background
- **WHEN** the CLI creates a container
- **THEN** docker run SHALL be executed with -d flag for detached mode

#### Scenario: Container naming
- **WHEN** the CLI generates a container name
- **THEN** the name SHALL be derived from the repository URL using a deterministic naming function implemented in Go

#### Scenario: Container persists after exit
- **WHEN** the CLI exits
- **THEN** the container SHALL continue running in the background

#### Scenario: User can exec into container
- **WHEN** the CLI displays management instructions
- **THEN** the instructions SHALL include `docker exec -it <container-name> /bin/bash` for manual access

### Requirement: Container Lifecycle Management
The CLI SHALL display container management instructions after creation, implemented using Go's fmt package.

#### Scenario: Display management instructions
- **WHEN** the container is created successfully
- **THEN** the CLI SHALL print instructions for accessing the container (docker exec) and cleaning up (docker stop, docker rm)

#### Scenario: Container cleanup
- **WHEN** user runs suggested cleanup commands
- **THEN** the container SHALL be stopped and removed (manual operation by user)

### Requirement: Container Startup Command
The CLI SHALL execute the appropriate startup command based on flags, implemented using Go's string building or text/template.

#### Scenario: Container startup sequence with branch
- **WHEN** user provides --prompt and --branch flags
- **THEN** the container SHALL execute: git clone → git checkout <branch> → ralph-loop with prompt

#### Scenario: Container startup sequence without branch
- **WHEN** user provides --prompt but not --branch flag
- **THEN** the container SHALL execute: git clone → ralph-loop with prompt (using default branch)

#### Scenario: Container startup without prompt
- **WHEN** user does not provide --prompt flag
- **THEN** the container SHALL execute: git clone → tail -f /dev/null (keep running)

#### Scenario: Clone failure
- **WHEN** git clone fails in the startup script
- **THEN** the container SHALL exit and the CLI SHALL report the failure

### Requirement: GitHub Token Environment Variable
The CLI SHALL forward the GITHUB_TOKEN environment variable to the container, implemented using Go's os.Getenv and docker run -e flags.

#### Scenario: GITHUB_TOKEN is set
- **WHEN** GITHUB_TOKEN environment variable is set on the host
- **THEN** the CLI SHALL pass it to the container using docker run -e GITHUB_TOKEN=$GITHUB_TOKEN

#### Scenario: GITHUB_TOKEN is not set
- **WHEN** GITHUB_TOKEN environment variable is not set on the host
- **THEN** the CLI SHALL print an error message and exit with non-zero status (validated in Go using os.Getenv)

#### Scenario: Token is not exposed in logs
- **WHEN** the CLI executes docker commands
- **THEN** the token value SHALL NOT appear in docker command output or logs

### Requirement: GitHub CLI Installation
The CLI SHALL ensure gh CLI is available in the container image, with Dockerfile generation implemented in Go.

#### Scenario: gh CLI is available in container
- **WHEN** the container is created from a setup image
- **THEN** gh CLI SHALL be installed and accessible via PATH

#### Scenario: Dockerfile template includes gh installation
- **WHEN** the CLI generates a Dockerfile via Go code
- **THEN** the Dockerfile SHALL include gh CLI installation steps

### Requirement: Git Credential Configuration
The CLI SHALL configure git to use the GITHUB_TOKEN for HTTPS authentication, with configuration generated by Go.

#### Scenario: Git credential helper is configured
- **WHEN** the container starts
- **THEN** git config SHALL be set to use the GITHUB_TOKEN as a credential helper

#### Scenario: Token authentication succeeds
- **WHEN** git operations require authentication
- **THEN** git SHALL use the GITHUB_TOKEN successfully

#### Scenario: Token authentication fails
- **WHEN** the GITHUB_TOKEN is invalid or expired
- **THEN** git operations SHALL fail with authentication error messages

### Requirement: Repository URL Format
The CLI SHALL support both HTTPS and SSH repository URLs, with URL handling implemented in Go.

#### Scenario: HTTPS URL provided
- **WHEN** user provides a repository URL starting with https://
- **THEN** the CLI SHALL use HTTPS cloning with GITHUB_TOKEN authentication

#### Scenario: SSH URL provided
- **WHEN** user provides a repository URL starting with git@
- **THEN** the CLI SHALL use SSH cloning with SSH agent forwarding

#### Scenario: URL conversion handles edge cases
- **WHEN** user provides URLs with or without .git suffix
- **THEN** the CLI SHALL handle both formats correctly (using Go's strings package for normalization)

## ADDED Requirements

### Requirement: Go Binary Execution
The spin command SHALL be executed from a standalone Go binary without requiring Node.js runtime.

#### Scenario: Binary invocation
- **WHEN** user runs `./dist/spinner spin --image test --repo <url>`
- **THEN** the Go binary SHALL execute the spin command using Cobra

#### Scenario: Cross-platform compatibility
- **WHEN** the binary is compiled for different platforms (Linux, macOS, Windows)
- **THEN** the spin command SHALL work identically across platforms

### Requirement: Cobra Flag Parsing for Spin
The spin command SHALL use Cobra for flag definition and validation.

#### Scenario: Flag registration
- **WHEN** the spin command initializes
- **THEN** all flags (--image, --repo, --prompt, --branch, --max-iterations) SHALL be registered with Cobra

#### Scenario: Required flag enforcement
- **WHEN** user omits required flags
- **THEN** Cobra SHALL display error messages and usage information automatically

#### Scenario: Optional flag defaults
- **WHEN** user omits optional flags (--prompt, --branch, --max-iterations)
- **THEN** Cobra SHALL provide default values (empty string for prompt/branch, 100 for max-iterations)

### Requirement: Viper Environment Variable Support
The spin command SHALL support environment variable overrides via Viper (future-proofing).

#### Scenario: Environment variable binding
- **WHEN** Viper is initialized for spin command
- **THEN** flags like SPINNER_IMAGE, SPINNER_REPO SHALL be bindable to environment variables

#### Scenario: Flag precedence
- **WHEN** both CLI flags and environment variables are set
- **THEN** CLI flags SHALL take precedence (Cobra + Viper default behavior)

### Requirement: Real-time Output Streaming
The CLI SHALL stream docker logs in real-time using Go's exec.Command with stdout/stderr pipes.

#### Scenario: Log following
- **WHEN** ralph-loop is running
- **THEN** the CLI SHALL use `docker logs -f <container>` to stream output to the terminal

#### Scenario: Output buffering
- **WHEN** streaming logs
- **THEN** the CLI SHALL use line-buffered or unbuffered output to ensure real-time display (using bufio.Scanner in Go)

#### Scenario: Signal handling
- **WHEN** user presses Ctrl+C
- **THEN** the CLI SHALL stop following logs but leave the container running (using os.Signal and signal.Notify in Go)
