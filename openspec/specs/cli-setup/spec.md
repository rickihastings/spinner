# cli-setup Specification

## Purpose
TBD - created by archiving change add-setup-command. Update Purpose after archive.
## Requirements
### Requirement: Prerequisite Verification

The CLI SHALL verify that required tools are installed before proceeding with setup operations. The verification MUST
check for docker, git, and claude CLI tools. If any prerequisite is missing, the CLI SHALL exit immediately with an
error message identifying the missing tool.

#### Scenario: All prerequisites installed

- **WHEN** docker, git, and claude are all available in PATH
- **THEN** the CLI proceeds with the setup command

#### Scenario: Docker not installed

- **WHEN** docker is not available in PATH
- **THEN** the CLI exits with error code 1 and displays "Error: docker is not installed"

#### Scenario: Git not installed

- **WHEN** git is not available in PATH
- **THEN** the CLI exits with error code 1 and displays "Error: git is not installed"

#### Scenario: Claude not installed

- **WHEN** claude is not available in PATH
- **THEN** the CLI exits with error code 1 and displays "Error: claude is not installed"

### Requirement: Docker Image Build

The CLI SHALL build a Docker image by extending the user-provided base image or Dockerfile. The final image SHALL
contain git and claude-code (installed only if missing from the base). The image SHALL be tagged as spinner:<name>.

#### Scenario: Successful image build

- **WHEN** setup command completes successfully
- **THEN** a Docker image named spinner:<name> exists locally

#### Scenario: Git inclusion

- **WHEN** the Docker image is built
- **THEN** the container can execute `git --version` successfully

#### Scenario: Claude-code inclusion

- **WHEN** the Docker image is built
- **THEN** the container can execute `claude --version` successfully

#### Scenario: Base image tools preserved

- **WHEN** the base image contains Node.js, Python, or other tools
- **THEN** these tools remain available in the final spinner:<name> image

### Requirement: Startup Script Inclusion

The Docker image SHALL include a startup script at /usr/local/bin/startup.sh that handles repository cloning and container initialization. The script SHALL accept a REPO_URL environment variable, clone the repository to /workspace, verify the clone with `git status`, output a hello message, and keep the container running with `tail -f /dev/null`.

#### Scenario: Startup script exists in image

- **WHEN** the Docker image is built
- **THEN** the file /usr/local/bin/startup.sh exists and is executable

#### Scenario: Startup script clones repository

- **WHEN** the container starts with REPO_URL environment variable set
- **THEN** the startup script clones the repository to /workspace
- **AND** runs `git status` to verify the clone
- **AND** outputs a hello message

#### Scenario: Startup script keeps container running

- **WHEN** the repository clone completes successfully
- **THEN** the startup script executes `tail -f /dev/null`
- **AND** the container remains in running state

### Requirement: No Secrets in Image

The Docker image SHALL NOT contain any authentication tokens, API keys, or secrets. Secrets MUST be mounted at container
runtime.

#### Scenario: Clean image

- **WHEN** the Docker image is built
- **THEN** no environment variables containing tokens are baked into the image
- **AND** no credential files are present in the image filesystem

### Requirement: Base Image Input Options

The setup command SHALL accept either --base-image or --dockerfile flag (mutually exclusive) along with the required
--name flag. The CLI SHALL NOT prompt for interactive input.

#### Scenario: Base image name provided

- **WHEN** user runs `setup --name my-sandbox --base-image ubuntu:22.04`
- **THEN** the CLI uses ubuntu:22.04 as the base image and proceeds with Docker image build

#### Scenario: Dockerfile path provided

- **WHEN** user runs `setup --name my-sandbox --dockerfile ./my-env.Dockerfile`
- **THEN** the CLI builds the user's Dockerfile first, then extends it

#### Scenario: Both flags provided

- **WHEN** user runs `setup --name test --base-image ubuntu:22.04 --dockerfile ./Dockerfile`
- **THEN** the CLI exits with error code 1 and displays "Error: Cannot specify both --base-image and --dockerfile"

#### Scenario: Neither flag provided

- **WHEN** user runs `setup --name test` without --base-image or --dockerfile
- **THEN** the CLI exits with error code 1 and displays usage information

#### Scenario: Missing name flag

- **WHEN** user runs `setup --base-image ubuntu:22.04` without --name
- **THEN** the CLI exits with error code 1 and displays usage information

### Requirement: Custom Dockerfile Build

When the user provides a Dockerfile path via --dockerfile, the CLI SHALL build that Dockerfile first to create a base
image, then extend it with required dependencies.

#### Scenario: Valid Dockerfile path

- **WHEN** user runs `setup --name test --dockerfile ./custom.Dockerfile`
- **AND** ./custom.Dockerfile exists and is valid
- **THEN** the CLI builds the user's Dockerfile with tag spinner-base:<name>
- **AND** uses spinner-base:<name> as the base for the final image

#### Scenario: Invalid Dockerfile path

- **WHEN** user runs `setup --name test --dockerfile ./missing.Dockerfile`
- **AND** the file does not exist
- **THEN** the CLI exits with error code 1 and displays "Error: Dockerfile not found at ./missing.Dockerfile"

#### Scenario: Dockerfile build failure

- **WHEN** the user's Dockerfile fails to build
- **THEN** the CLI exits with error code 1 and displays the Docker build error

### Requirement: Conditional Dependency Installation

The Docker image build process SHALL check if git and claude-code are already installed in the base image. Dependencies
SHALL only be installed if they are missing.

#### Scenario: Git already present

- **WHEN** the base image already contains git
- **THEN** the build process skips git installation
- **AND** the final image can execute `git --version` successfully

#### Scenario: Git missing

- **WHEN** the base image does not contain git
- **THEN** the build process installs git using apt-get
- **AND** the final image can execute `git --version` successfully

#### Scenario: Claude-code already present

- **WHEN** the base image already contains claude-code
- **THEN** the build process skips claude-code installation
- **AND** the final image can execute `claude --version` successfully

#### Scenario: Claude-code missing

- **WHEN** the base image does not contain claude-code
- **THEN** the build process installs claude-code using the native installer
- **AND** the final image can execute `claude --version` successfully

### Requirement: Ubuntu/Debian Base Image Restriction

The CLI SHALL only support Ubuntu or Debian-based base images that use the apt-get package manager. If a non-compatible
base image is detected, the CLI SHALL exit with a clear error message.

#### Scenario: Ubuntu-based image

- **WHEN** user provides --base-image ubuntu:22.04
- **THEN** the CLI proceeds with the image build

#### Scenario: Debian-based image

- **WHEN** user provides --base-image debian:bullseye
- **THEN** the CLI proceeds with the image build

#### Scenario: Non-Debian base image

- **WHEN** user provides --base-image alpine:latest
- **AND** apt-get is not available in the base image
- **THEN** the Docker build fails with an error indicating unsupported base OS

