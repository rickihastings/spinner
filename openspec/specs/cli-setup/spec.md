# cli-setup Specification

## Purpose
TBD - created by archiving change add-setup-command. Update Purpose after archive.
## Requirements
### Requirement: Prerequisite Verification
The CLI SHALL verify that required tools are installed on the host system before building Docker images. The verification SHALL be implemented in Go using os/exec for command execution.

#### Scenario: All prerequisites installed
- **WHEN** user runs `spinner setup --name <name>`
- **THEN** the CLI SHALL verify docker, git, and claude-code are available via Go's exec.LookPath or equivalent

#### Scenario: Docker not installed
- **WHEN** user runs `spinner setup` and docker is not installed
- **THEN** the CLI SHALL print an error message indicating docker is required and exit with non-zero status

#### Scenario: Git not installed
- **WHEN** user runs `spinner setup` and git is not installed
- **THEN** the CLI SHALL print an error message indicating git is required and exit with non-zero status

#### Scenario: Claude not installed
- **WHEN** user runs `spinner setup` and claude-code is not installed
- **THEN** the CLI SHALL print an error message indicating claude-code is required and exit with non-zero status

### Requirement: Docker Image Build
The Docker image build process SHALL be implemented using Go's os/exec package to execute docker build commands with identical behavior to the TypeScript implementation.

#### Scenario: Successful image build
- **WHEN** user provides valid build configuration
- **THEN** the CLI SHALL execute docker build using exec.Command and stream stdout/stderr to the user

#### Scenario: Git inclusion
- **WHEN** building the image
- **THEN** the CLI SHALL ensure git is installed in the final image via Dockerfile RUN commands

#### Scenario: Claude-code inclusion
- **WHEN** building the image
- **THEN** the CLI SHALL ensure claude-code is installed in the final image via Dockerfile RUN commands

#### Scenario: Base image tools preserved
- **WHEN** using a base image with existing tools
- **THEN** the CLI SHALL preserve existing tools during the build process

### Requirement: Startup Script Inclusion
The CLI SHALL generate and include a startup script in the Docker image, implemented in Go with identical template output to the TypeScript version.

#### Scenario: Startup script exists in image
- **WHEN** the image is built
- **THEN** the startup script SHALL be present at /startup.sh in the image

#### Scenario: Startup script clones repository
- **WHEN** the container starts with a repo URL
- **THEN** the startup script SHALL clone the repository into /workspace using git clone

#### Scenario: Startup script keeps container running
- **WHEN** the startup script completes
- **THEN** the container SHALL remain running via tail -f /dev/null or equivalent

### Requirement: No Secrets in Image
The Docker image SHALL NOT contain any secrets, tokens, or sensitive data. This requirement remains unchanged but is implemented in Go.

#### Scenario: Clean image
- **WHEN** inspecting the built image
- **THEN** no environment variables, files, or layers SHALL contain GitHub tokens, SSH keys, or other secrets

### Requirement: Base Image Input Options
The CLI SHALL accept either --base-image or --dockerfile flags, validated by Cobra's flag parsing system.

#### Scenario: Base image name provided
- **WHEN** user runs `spinner setup --name my-env --base-image ubuntu:22.04`
- **THEN** the CLI SHALL build using the specified base image

#### Scenario: Dockerfile path provided
- **WHEN** user runs `spinner setup --name my-env --dockerfile ./Dockerfile.custom`
- **THEN** the CLI SHALL build using the custom Dockerfile

#### Scenario: Both flags provided
- **WHEN** user provides both --base-image and --dockerfile flags
- **THEN** the CLI SHALL print an error message indicating they are mutually exclusive and exit with non-zero status

#### Scenario: Neither flag provided
- **WHEN** user provides neither --base-image nor --dockerfile flags
- **THEN** the CLI SHALL default to ubuntu:22.04 as the base image

#### Scenario: Missing name flag
- **WHEN** user runs `spinner setup` without --name flag
- **THEN** the CLI SHALL print an error message indicating --name is required and exit with non-zero status (enforced by Cobra's MarkFlagRequired)

### Requirement: Custom Dockerfile Build
The CLI SHALL support building from custom Dockerfiles using Go's exec.Command for docker build operations.

#### Scenario: Valid Dockerfile path
- **WHEN** user provides a valid Dockerfile path
- **THEN** the CLI SHALL build the custom Dockerfile first using docker build -f <path>, then use the result as the base image for additional setup

#### Scenario: Invalid Dockerfile path
- **WHEN** user provides a non-existent Dockerfile path
- **THEN** the CLI SHALL print an error message and exit with non-zero status (validated using os.Stat in Go)

#### Scenario: Dockerfile build failure
- **WHEN** the custom Dockerfile build fails
- **THEN** the CLI SHALL display the docker build error output and exit with non-zero status

### Requirement: Conditional Dependency Installation
The CLI SHALL conditionally install dependencies in the Docker image based on their presence, using Go-generated Dockerfile templates.

#### Scenario: Git already present
- **WHEN** the base image already has git installed
- **THEN** the CLI SHALL skip git installation (checked via `which git` in Dockerfile)

#### Scenario: Git missing
- **WHEN** the base image does not have git installed
- **THEN** the CLI SHALL install git via apt-get in the Dockerfile

#### Scenario: Claude-code already present
- **WHEN** the base image already has claude-code installed
- **THEN** the CLI SHALL skip claude-code installation

#### Scenario: Claude-code missing
- **WHEN** the base image does not have claude-code installed
- **THEN** the CLI SHALL install claude-code via npm or official installation method

### Requirement: Ubuntu/Debian Base Image Restriction
The CLI SHALL only support Ubuntu/Debian-based images that use apt-get. This validation remains unchanged but is implemented in Go.

#### Scenario: Ubuntu-based image
- **WHEN** user provides an Ubuntu-based image (e.g., ubuntu:22.04)
- **THEN** the CLI SHALL proceed with the build

#### Scenario: Debian-based image
- **WHEN** user provides a Debian-based image (e.g., debian:bullseye)
- **THEN** the CLI SHALL proceed with the build

#### Scenario: Non-Debian base image
- **WHEN** user provides a non-Debian-based image (e.g., alpine:latest)
- **THEN** the CLI SHALL print a warning or error indicating only Ubuntu/Debian images are supported

### Requirement: Go Binary Build
The CLI SHALL be built as a standalone Go binary using `go build` command.

#### Scenario: Build command
- **WHEN** developer runs `go build -o dist/spinner`
- **THEN** a standalone binary SHALL be created at dist/spinner

#### Scenario: Binary execution
- **WHEN** user runs `./dist/spinner setup --name test`
- **THEN** the binary SHALL execute without requiring Node.js runtime

#### Scenario: Cross-platform builds
- **WHEN** developer uses `GOOS` and `GOARCH` environment variables
- **THEN** the CLI SHALL support cross-compilation for different platforms (Linux, macOS, Windows)

### Requirement: Cobra Command Structure

The setup command SHALL use Cobra and Viper for flag definition and configuration management, implemented in Go. The
command SHALL be fully testable via dependency injection.

#### Scenario: Cobra command initialization

- **WHEN** the CLI starts
- **THEN** the setup command SHALL be registered as a Cobra subcommand with all flags (--name, --backend,
  --bake-script, --provider-args, and GCP routing flags --project, --zone, --state-bucket)

### Requirement: Viper Configuration Support
The CLI SHALL use Viper for environment variable configuration (future-proofing).

#### Scenario: Environment variable binding
- **WHEN** Viper is initialized in cmd/root.go
- **THEN** the CLI SHALL be capable of reading configuration from environment variables

#### Scenario: Precedence handling
- **WHEN** both CLI flags and environment variables are provided
- **THEN** CLI flags SHALL take precedence over environment variables (Cobra + Viper default behavior)

#### Scenario: Configuration file support (optional)
- **WHEN** a configuration file is present
- **THEN** Viper SHALL optionally read configuration from the file

### Requirement: Unit Test Coverage for Setup Command
The setup command SHALL have comprehensive unit tests that validate argument parsing, flag validation, and error handling without requiring Docker operations.

#### Scenario: Test missing name flag validation
- **GIVEN** setup command is invoked without --name flag
- **WHEN** the command is executed in a test
- **THEN** the test SHALL verify an error is returned with message containing "Missing required flag"

#### Scenario: Test mutually exclusive flags validation
- **GIVEN** setup command is invoked with both --base-image and --dockerfile flags
- **WHEN** the command is executed in a test
- **THEN** the test SHALL verify an error is returned with message containing "mutually exclusive"

#### Scenario: Test successful argument parsing
- **GIVEN** setup command is invoked with valid --name flag
- **WHEN** the command arguments are parsed
- **THEN** the test SHALL verify the name value is correctly extracted

#### Scenario: Test base-image flag parsing
- **GIVEN** setup command is invoked with --name and --base-image flags
- **WHEN** the command arguments are parsed
- **THEN** the test SHALL verify both flag values are correctly extracted

#### Scenario: Test dockerfile flag parsing
- **GIVEN** setup command is invoked with --name and --dockerfile flags
- **WHEN** the command arguments are parsed
- **THEN** the test SHALL verify both flag values are correctly extracted

#### Scenario: Test dockerfile path validation
- **GIVEN** setup command is invoked with non-existent dockerfile path
- **WHEN** the command is executed in a test
- **THEN** the test SHALL verify an error is returned indicating file not found

### Requirement: Unit Test Coverage for Docker Build Logic
The Docker build operations SHALL have unit tests with mocked Docker client to verify build logic without actual Docker calls.

#### Scenario: Test successful image build with mocked Docker
- **GIVEN** a mocked Docker client that returns success
- **WHEN** BuildImage is called with valid configuration
- **THEN** the test SHALL verify the build completes without error

#### Scenario: Test image build with custom base image
- **GIVEN** a mocked Docker client
- **WHEN** BuildImage is called with custom base-image configuration
- **THEN** the test SHALL verify the correct base image is used in Docker build

#### Scenario: Test image build with custom Dockerfile
- **GIVEN** a mocked Docker client
- **WHEN** BuildImage is called with dockerfile configuration
- **THEN** the test SHALL verify the custom Dockerfile is used in Docker build

#### Scenario: Test image build failure handling
- **GIVEN** a mocked Docker client that returns build error
- **WHEN** BuildImage is called
- **THEN** the test SHALL verify the error is properly propagated

### Requirement: Integration Test Coverage for Setup Command
The setup command SHALL have integration tests that verify end-to-end behavior with real Docker operations.

#### Scenario: Integration test for successful image build
- **GIVEN** Docker is running on the host system
- **WHEN** setup command is executed with valid arguments
- **THEN** the test SHALL verify a Docker image is created with the specified name

#### Scenario: Integration test for image existence verification
- **GIVEN** setup command has successfully built an image
- **WHEN** docker images is queried
- **THEN** the test SHALL verify the image exists in Docker's image list

#### Scenario: Integration test for git installation in image
- **GIVEN** setup command has successfully built an image
- **WHEN** a container is run from the image to check git
- **THEN** the test SHALL verify git is available in the image

#### Scenario: Integration test for claude-code installation in image
- **GIVEN** setup command has successfully built an image
- **WHEN** a container is run from the image to check claude-code
- **THEN** the test SHALL verify claude-code is available in the image

#### Scenario: Integration test cleanup
- **GIVEN** an integration test has created Docker images
- **WHEN** the test completes (success or failure)
- **THEN** the test SHALL clean up created images to prevent resource leaks

### Requirement: Test Utility Infrastructure
The project SHALL provide test utilities for Docker operations, CLI execution, and resource cleanup to support both unit and integration tests.

#### Scenario: Docker test helpers available
- **GIVEN** tests need to verify Docker resource state
- **WHEN** test utilities are imported
- **THEN** helper functions for image existence, container status, and cleanup SHALL be available

#### Scenario: CLI execution helpers available
- **GIVEN** integration tests need to execute CLI commands
- **WHEN** test utilities are imported
- **THEN** helper functions for building CLI binary and running commands SHALL be available

#### Scenario: Test cleanup utilities available
- **GIVEN** tests create Docker resources
- **WHEN** tests complete or fail
- **THEN** cleanup utilities SHALL ensure resources are removed

### Requirement: Backend Selection for Setup

The setup command SHALL accept a `--backend` flag to select which provider handles the setup operation.

#### Scenario: Default backend

- **WHEN** user runs `spinner setup --name my-env` without `--backend` flag
- **THEN** the CLI SHALL use the `docker` backend (backward compatible)

#### Scenario: GCP backend selected

- **WHEN** user runs `spinner setup --backend gcp --name my-env --project my-project --zone us-central1-a`
- **THEN** the CLI SHALL use the GCP provider to bake a custom Compute Engine image

#### Scenario: Unknown backend

- **WHEN** user provides an unknown backend name (e.g., `--backend kubernetes`)
- **THEN** the CLI SHALL print an error listing available backends and exit with non-zero status

#### Scenario: Backend flag via environment variable

- **WHEN** `SPINNER_BACKEND` environment variable is set
- **THEN** the CLI SHALL use that value as the default backend
- **AND** the `--backend` flag SHALL take precedence over the environment variable

### Requirement: GCP-Specific Setup Flags

The setup command SHALL accept GCP-specific flags when `--backend gcp` is selected.

#### Scenario: Required project flag for GCP

- **WHEN** user runs `spinner setup --backend gcp --name my-env` without `--project`
- **THEN** the CLI SHALL print an error indicating `--project` is required for GCP backend

#### Scenario: Required zone flag for GCP

- **WHEN** user runs `spinner setup --backend gcp --name my-env --project p` without `--zone`
- **THEN** the CLI SHALL print an error indicating `--zone` is required for GCP backend

#### Scenario: Optional machine-type flag

- **WHEN** user runs `spinner setup --backend gcp --name my-env --project p --zone z --machine-type e2-standard-4`
- **THEN** the CLI SHALL pass the machine type to the GCP provider for the bake VM

#### Scenario: Default machine-type

- **WHEN** user does not provide `--machine-type` with GCP backend
- **THEN** the CLI SHALL default to `e2-standard-2`

#### Scenario: Optional disk-size flag

- **WHEN** user runs `spinner setup --backend gcp --name my-env --project p --zone z --disk-size 50`
- **THEN** the CLI SHALL pass the disk size (in GB) to the GCP provider for the bake VM

#### Scenario: Default disk-size

- **WHEN** user does not provide `--disk-size` with GCP backend
- **THEN** the CLI SHALL default to 30 GB

#### Scenario: Required state-bucket flag for GCP

- **WHEN** user runs `spinner setup --backend gcp --name my-env --project p --zone z` without `--state-bucket`
- **THEN** the CLI SHALL print an error indicating `--state-bucket` is required for GCP backend
- **AND** explain that GCS bucket names are globally unique and must be pre-created

#### Scenario: Docker flags rejected for GCP backend

- **WHEN** user provides `--base-image` or `--dockerfile` CLI flags with `--backend gcp`
- **THEN** the CLI SHALL return an error indicating these flags require `--backend docker`

#### Scenario: GCP flags rejected for Docker backend

- **WHEN** user provides `--project`, `--zone`, `--machine-type`, `--disk-size`, or `--state-bucket` CLI flags
  without `--backend gcp`
- **THEN** the CLI SHALL return an error indicating these flags require `--backend gcp`

#### Scenario: Config file values not rejected cross-backend

- **WHEN** `.spinner.json` contains keys for a different backend (e.g., `project` when using Docker)
- **THEN** the CLI SHALL silently ignore those values (no error)
- **AND** only CLI flags that are explicitly set trigger cross-backend validation

### Requirement: Configuration File Support

The setup command SHALL read infrastructure defaults from a `.spinner.json` file, including `provider-args`.

#### Scenario: Config file with provider-args for setup

- **WHEN** `.spinner.json` contains `{"provider-args": ["--no-cache", "--build-arg=NODE_ENV=production"]}`
- **AND** user runs `spinner setup --name my-env`
- **THEN** the CLI SHALL forward both args to the `docker build` command

### Requirement: Grouped Help Output

The setup command help SHALL organize flags into backend-specific groups for clarity.

#### Scenario: Help shows flag groups

- **WHEN** user runs `spinner setup --help`
- **THEN** flags SHALL be organized into labeled sections: General, Docker Backend, GCP Backend

### Requirement: Provider Pass-Through Arguments for Setup

The setup command SHALL accept an optional repeatable `--provider-args` flag that passes raw arguments directly to the
underlying backend provider. Arguments are forwarded verbatim to the backend's environment provisioning command
(`docker build` for Docker, `gcloud compute instances create` for GCP image bake).

#### Scenario: Provider arg for Docker build

- **WHEN** user runs `spinner setup --name my-env --provider-args="--build-arg=MY_ARG=value"`
- **THEN** the CLI SHALL append `--build-arg=MY_ARG=value` to the `docker build` command before the context directory

#### Scenario: Provider arg for Docker no-cache

- **WHEN** user runs `spinner setup --name my-env --provider-args="--no-cache"`
- **THEN** the CLI SHALL append `--no-cache` to the `docker build` command

#### Scenario: Provider arg for GCP setup

- **WHEN** user runs `spinner setup --backend gcp --name my-env --project p --zone z --state-bucket b --provider-args="--labels=env=staging"`
- **THEN** the CLI SHALL append `--labels=env=staging` to the GCP bake VM creation command

#### Scenario: No provider args provided

- **WHEN** user runs `spinner setup --name my-env` without any `--provider-args` flags
- **THEN** the CLI SHALL behave identically to the current implementation (no change in behavior)

#### Scenario: Provider args from config file

- **WHEN** `.spinner.json` contains `{"provider-args": ["--no-cache"]}`
- **AND** user runs `spinner setup --name my-env`
- **THEN** the CLI SHALL forward `--no-cache` to the backend build command

### Requirement: Provider Args Conflict Detection for Setup

The CLI SHALL reject `--provider-args` values that conflict with arguments managed by Spinner during setup.

#### Scenario: Docker build tag conflict

- **WHEN** user provides `--provider-args="-t my-tag"` with the Docker backend setup
- **THEN** the CLI SHALL print an error indicating `-t` is managed by Spinner and exit with non-zero status

#### Scenario: Non-conflicting build args pass through

- **WHEN** user provides `--provider-args="--no-cache"` with the Docker backend setup
- **THEN** the CLI SHALL accept and forward the argument without error

### Requirement: Removed Backend-Specific Setup Flags

The setup command SHALL NOT accept the following backend-specific flags, which have been replaced by `--provider-args`:
`--base-image`, `--dockerfile`, `--machine-type`, `--disk-size`.

#### Scenario: Removed flag produces error

- **WHEN** user provides `--base-image=node:20`
- **THEN** the CLI SHALL print an unknown flag error and exit with non-zero status

### Requirement: Setup documentation MUST present ANTHROPIC_API_KEY and CLAUDE_CODE_OAUTH_TOKEN as equal peers

User-facing documentation for initial setup SHALL list `ANTHROPIC_API_KEY` first
(reflecting runtime check order) and `CLAUDE_CODE_OAUTH_TOKEN` as the alternative.
The docs MUST make clear that both are valid and only one is required. Neither key
SHALL be described as deprecated or preferred.

#### Scenario: User follows setup docs using ANTHROPIC_API_KEY

Given a user reads the "Initial Setup" section of the usage docs
When they follow the primary instructions
Then they run `spinner secret set ANTHROPIC_API_KEY` (not the OAuth token)
And the subsequent `spinner spin` command succeeds with that key

#### Scenario: User follows setup docs using CLAUDE_CODE_OAUTH_TOKEN

Given a user reads the "Initial Setup" section of the usage docs
When they follow the alternative instructions
Then they run `spinner secret set CLAUDE_CODE_OAUTH_TOKEN`
And the subsequent `spinner spin` command succeeds with that key

