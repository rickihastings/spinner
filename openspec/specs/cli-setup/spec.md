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
The CLI SHALL use Cobra for command routing and flag parsing.

#### Scenario: Root command
- **WHEN** user runs `spinner --help`
- **THEN** Cobra SHALL display help text matching the format from src/App.tsx

#### Scenario: Subcommand registration
- **WHEN** the CLI initializes
- **THEN** setup and spin commands SHALL be registered with Cobra's AddCommand

#### Scenario: Flag validation
- **WHEN** user provides invalid flags
- **THEN** Cobra SHALL display appropriate error messages and usage information

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

