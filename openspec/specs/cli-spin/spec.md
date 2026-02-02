# cli-spin Specification

## Purpose
TBD - created by archiving change add-spin-command. Update Purpose after archive.
## Requirements
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
- **THEN** the CLI SHALL start the container and execute `spinner exec` (Go implementation) with the provided prompt on
  the specified branch

#### Scenario: Prompt provided without branch

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --prompt "implement X"` without --branch flag
- **THEN** the CLI SHALL start the container and execute `spinner exec` (Go implementation) with the provided prompt on
  the default branch

#### Scenario: Prompt not provided

- **WHEN** user runs `spinner spin --image <image> --repo <repo>` without --prompt flag
- **THEN** the CLI SHALL start the container without executing `spinner exec`

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
The CLI SHALL create persistent containers that run in detached mode, implemented using Go's exec.Command for docker run. The CLI SHALL assign a deterministic name to the container based on the image name, repository name, and optionally the branch name.

#### Scenario: Container runs in background
- **WHEN** the CLI creates a container
- **THEN** docker run SHALL be executed with -d flag for detached mode

#### Scenario: Deterministic container naming without branch
- **WHEN** user spins up a container with `--image spinner:default --repo git@github.com:user/my-project.git`
- **THEN** the container is named `spinner-default-my-project`
- **AND** the container name is displayed to the user
- **AND** the naming logic SHALL be implemented in Go using string manipulation

#### Scenario: Deterministic container naming with branch
- **WHEN** user spins up a container with `--image spinner:default --repo git@github.com:user/my-project.git --branch feature/auth-v2`
- **THEN** the container is named `spinner-default-my-project-feature-auth-v2`
- **AND** the container name is displayed to the user
- **AND** the Go implementation SHALL append the sanitized branch name to the container name

#### Scenario: Container name sanitization
- **WHEN** the image is `spinner:my-env`, repo is `git@github.com:user/my.project.git`, and branch is `feature/auth-v2`
- **THEN** the container name is `spinner-my-env-my-project-feature-auth-v2`
- **AND** special characters (`:`, `/`, `.`) are replaced with hyphens
- **AND** the sanitization SHALL be implemented in Go using regex or strings package

#### Scenario: Container persists after exit
- **WHEN** the CLI exits
- **THEN** the container SHALL continue running in the background

#### Scenario: User can exec into container
- **WHEN** the CLI displays management instructions
- **THEN** the instructions SHALL include `docker exec -it <container-name> /bin/bash` for manual access
- **AND** the working directory is /workspace

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

### Requirement: Container Reuse
The CLI SHALL check if a container with the deterministic name already exists before creating a new one, implemented using Go's exec.Command to invoke docker inspect. If a container with the same name exists and is running, the CLI SHALL reuse it. If a container with the same name exists but is stopped, the CLI SHALL restart it. The CLI output SHALL clearly indicate whether a container was created, reused, or restarted.

#### Scenario: Reuse running container
- **WHEN** user runs `spin --image spinner:default --repo git@github.com:user/my-project.git`
- **AND** a container named `spinner-default-my-project` is already running
- **THEN** the CLI does not create a new container
- **AND** the CLI displays "Reusing running container: spinner-default-my-project"
- **AND** the CLI displays the standard management instructions
- **AND** the check SHALL be implemented in Go using `docker inspect -f '{{.State.Status}}' <container-name>`

#### Scenario: Restart stopped container
- **WHEN** user runs `spin --image spinner:default --repo git@github.com:user/my-project.git`
- **AND** a container named `spinner-default-my-project` exists but is stopped
- **THEN** the CLI restarts the existing container with `docker start`
- **AND** the CLI displays "Restarted container: spinner-default-my-project"
- **AND** the CLI displays the standard management instructions
- **AND** the restart SHALL be implemented in Go using exec.Command

#### Scenario: Create new container when none exists
- **WHEN** user runs `spin --image spinner:default --repo git@github.com:user/my-project.git`
- **AND** no container named `spinner-default-my-project` exists
- **THEN** the CLI creates a new container with that name
- **AND** the CLI displays "Container created successfully: spinner-default-my-project"
- **AND** the CLI displays the standard management instructions
- **AND** the creation SHALL follow the standard docker run logic implemented in Go

### Requirement: Container Recreation Flag
The CLI SHALL accept an optional `--recreate` boolean flag, implemented using Cobra's BoolP flag type. When provided, the CLI SHALL remove any existing container with the deterministic name and create a fresh container. This allows users to force a clean slate when reuse is not desired.

#### Scenario: Recreate removes running container
- **WHEN** user runs `spin --image spinner:default --repo git@github.com:user/my-project.git --recreate`
- **AND** a container named `spinner-default-my-project` is currently running
- **THEN** the CLI stops and removes the existing container using `docker rm -f`
- **AND** the CLI creates a new container with the same name
- **AND** the CLI displays "Container recreated: spinner-default-my-project"
- **AND** the removal SHALL be implemented in Go using exec.Command

#### Scenario: Recreate removes stopped container
- **WHEN** user runs `spin --image spinner:default --repo git@github.com:user/my-project.git --recreate`
- **AND** a container named `spinner-default-my-project` exists but is stopped
- **THEN** the CLI removes the existing container using `docker rm`
- **AND** the CLI creates a new container with the same name
- **AND** the CLI displays "Container recreated: spinner-default-my-project"

#### Scenario: Recreate creates when no container exists
- **WHEN** user runs `spin --image spinner:default --repo git@github.com:user/my-project.git --recreate`
- **AND** no container named `spinner-default-my-project` exists
- **THEN** the CLI creates a new container with that name
- **AND** the CLI displays "Container created successfully: spinner-default-my-project"
- **AND** the behavior is identical to running without `--recreate`

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
- **THEN** all flags (--image, --repo, --prompt, --branch, --max-iterations, --recreate) SHALL be registered with Cobra

#### Scenario: Required flag enforcement
- **WHEN** user omits required flags
- **THEN** Cobra SHALL display error messages and usage information automatically

#### Scenario: Optional flag defaults
- **WHEN** user omits optional flags (--prompt, --branch, --max-iterations, --recreate)
- **THEN** Cobra SHALL provide default values (empty string for prompt/branch, 100 for max-iterations, false for recreate)

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

### Requirement: Unit Test Coverage for Spin Command Flags
The spin command SHALL have comprehensive unit tests that validate all flag combinations and validation logic without requiring Docker operations.

#### Scenario: Test missing image flag validation
- **GIVEN** spin command is invoked without --image flag
- **WHEN** the command is executed in a test
- **THEN** the test SHALL verify an error is returned with message containing "--image flag is required"

#### Scenario: Test missing repo flag validation
- **GIVEN** spin command is invoked without --repo flag
- **WHEN** the command is executed in a test
- **THEN** the test SHALL verify an error is returned with message containing "--repo flag is required"

#### Scenario: Test prompt flag parsing
- **GIVEN** spin command is invoked with --prompt flag
- **WHEN** the command arguments are parsed
- **THEN** the test SHALL verify the prompt value is correctly extracted

#### Scenario: Test branch flag parsing
- **GIVEN** spin command is invoked with --branch flag
- **WHEN** the command arguments are parsed
- **THEN** the test SHALL verify the branch value is correctly extracted

#### Scenario: Test max-iterations flag parsing
- **GIVEN** spin command is invoked with --max-iterations flag
- **WHEN** the command arguments are parsed
- **THEN** the test SHALL verify the max-iterations value is correctly extracted

#### Scenario: Test recreate flag parsing
- **GIVEN** spin command is invoked with --recreate flag
- **WHEN** the command arguments are parsed
- **THEN** the test SHALL verify the recreate boolean is correctly set

#### Scenario: Test max-iterations default value
- **GIVEN** spin command is invoked without --max-iterations flag
- **WHEN** the command arguments are parsed
- **THEN** the test SHALL verify max-iterations defaults to 30

#### Scenario: Test setup flag parsing
- **GIVEN** spin command is invoked with --setup flag
- **WHEN** the command arguments are parsed
- **THEN** the test SHALL verify the setup flag is correctly set

#### Scenario: Test setup flag with base-image requires setup
- **GIVEN** spin command is invoked with --base-image but without --setup flag
- **WHEN** the command is validated
- **THEN** the test SHALL verify an error is returned indicating --base-image requires --setup

#### Scenario: Test setup flag with mutually exclusive options
- **GIVEN** spin command is invoked with --setup, --base-image, and --dockerfile flags
- **WHEN** the command is validated
- **THEN** the test SHALL verify an error is returned indicating flags are mutually exclusive

### Requirement: Unit Test Coverage for Container Operations
The Docker container operations SHALL have unit tests with mocked Docker client to verify run logic without actual Docker calls.

#### Scenario: Test container creation with mocked Docker
- **GIVEN** a mocked Docker client that returns success
- **WHEN** RunContainer is called with valid configuration
- **THEN** the test SHALL verify the container is created without error

#### Scenario: Test container naming logic
- **GIVEN** image, repo, and branch parameters
- **WHEN** container name is generated
- **THEN** the test SHALL verify the name follows the deterministic format

#### Scenario: Test container name sanitization
- **GIVEN** repo URL with special characters
- **WHEN** container name is generated
- **THEN** the test SHALL verify special characters are replaced with hyphens

#### Scenario: Test container reuse logic
- **GIVEN** a container with the same name already exists
- **WHEN** spin command is executed without --recreate flag
- **THEN** the test SHALL verify the existing container is reused

#### Scenario: Test container recreation logic
- **GIVEN** a container with the same name already exists
- **WHEN** spin command is executed with --recreate flag
- **THEN** the test SHALL verify the existing container is removed and recreated

### Requirement: Unit Test Coverage for Prerequisites
The prerequisite validation SHALL have unit tests that verify all required tokens and tools are checked properly.

#### Scenario: Test GitHub token validation
- **GIVEN** GITHUB_TOKEN environment variable is not set
- **WHEN** prerequisites are checked
- **THEN** the test SHALL verify an error is returned indicating missing GitHub token

#### Scenario: Test Claude token validation
- **GIVEN** CLAUDE_CODE_OAUTH_TOKEN environment variable is not set
- **WHEN** prerequisites are checked
- **THEN** the test SHALL verify an error is returned indicating missing Anthropic API key

#### Scenario: Test Docker availability check
- **GIVEN** Docker is not available on the system
- **WHEN** prerequisites are checked
- **THEN** the test SHALL verify an error is returned indicating Docker is required

### Requirement: Integration Test Coverage for Spin Command
The spin command SHALL have integration tests that verify end-to-end behavior with real Docker operations.

#### Scenario: Integration test for successful container creation
- **GIVEN** Docker is running and valid image exists
- **WHEN** spin command is executed with valid arguments
- **THEN** the test SHALL verify a Docker container is created and running

#### Scenario: Integration test for container naming
- **GIVEN** spin command is executed with image, repo, and branch
- **WHEN** docker ps is queried
- **THEN** the test SHALL verify container name matches expected format

#### Scenario: Integration test for repository cloning
- **GIVEN** spin command is executed with valid repo URL
- **WHEN** container filesystem is inspected
- **THEN** the test SHALL verify repository files exist in /workspace

#### Scenario: Integration test for branch checkout
- **GIVEN** spin command is executed with --branch flag
- **WHEN** container git status is checked
- **THEN** the test SHALL verify the specified branch is checked out

#### Scenario: Integration test for prompt execution
- **GIVEN** spin command is executed with --prompt flag
- **WHEN** container processes are inspected
- **THEN** the test SHALL verify ralph-loop is running with the prompt

#### Scenario: Integration test for container reuse
- **GIVEN** a container from previous spin command exists
- **WHEN** spin command is executed again without --recreate
- **THEN** the test SHALL verify the same container is reused

#### Scenario: Integration test for container recreation
- **GIVEN** a container from previous spin command exists
- **WHEN** spin command is executed with --recreate flag
- **THEN** the test SHALL verify old container is removed and new one is created

#### Scenario: Integration test for non-existent image
- **GIVEN** spin command is executed with non-existent image name
- **WHEN** Docker operations are attempted
- **THEN** the test SHALL verify appropriate error message is returned

#### Scenario: Integration test for max-iterations parameter
- **GIVEN** spin command is executed with --max-iterations flag
- **WHEN** container environment is inspected
- **THEN** the test SHALL verify MAX_ITERATIONS environment variable is set correctly

#### Scenario: Integration test cleanup
- **GIVEN** an integration test has created Docker containers
- **WHEN** the test completes (success or failure)
- **THEN** the test SHALL clean up created containers to prevent resource leaks

#### Scenario: Integration test for setup flag with base-image
- **GIVEN** spin command is executed with --setup, --image, and --base-image flags
- **WHEN** the command completes
- **THEN** the test SHALL verify the Docker image is built with the specified base image and container is created

#### Scenario: Integration test for setup flag with dockerfile
- **GIVEN** spin command is executed with --setup, --image, and --dockerfile flags
- **WHEN** the command completes
- **THEN** the test SHALL verify the Docker image is built using the custom Dockerfile and container is created

#### Scenario: Integration test for setup rebuilds existing image
- **GIVEN** an image with the same name already exists
- **WHEN** spin command is executed with --setup flag
- **THEN** the test SHALL verify the existing image is rebuilt and container is created

### Requirement: Table-Driven Test Coverage
Complex validation scenarios SHALL use table-driven tests to comprehensively cover all edge cases and variations.

#### Scenario: Table-driven tests for container naming
- **GIVEN** various combinations of repo URLs and branch names
- **WHEN** container names are generated
- **THEN** the test SHALL verify correct sanitization and formatting for all cases

#### Scenario: Table-driven tests for flag combinations
- **GIVEN** various combinations of flags (prompt, branch, max-iterations, recreate)
- **WHEN** spin command is executed
- **THEN** the test SHALL verify correct behavior for all valid combinations

### Requirement: State Directory Mounting

The CLI SHALL mount a state directory from the host into containers for state persistence.

#### Scenario: State directory created on host

- **WHEN** `spinner spin` creates a container
- **THEN** it SHALL create `~/.spinner/{CONTAINER_NAME}/state` directory on the host if it doesn't exist

#### Scenario: State directory mounted in container

- **WHEN** `spinner spin` runs the container
- **THEN** it SHALL mount `~/.spinner/{CONTAINER_NAME}/state` to `/state` in the container

#### Scenario: State persists across container recreations

- **WHEN** user runs `spinner spin --recreate`
- **THEN** the state directory SHALL be preserved and mounted into the new container

### Requirement: CLI Binary in Docker Image

The Docker image build process SHALL include the spinner CLI binary.

#### Scenario: CLI binary compiled for Linux

- **WHEN** `spinner setup` builds a Docker image
- **THEN** the Dockerfile SHALL compile the CLI with GOOS=linux GOARCH=amd64

#### Scenario: CLI binary available in container

- **WHEN** container starts
- **THEN** the spinner CLI SHALL be available at `/usr/local/bin/spinner`

#### Scenario: Startup script calls spinner exec

- **WHEN** container starts with PROMPT env var set
- **THEN** the startup script SHALL call `spinner exec` instead of ralph-loop.sh

