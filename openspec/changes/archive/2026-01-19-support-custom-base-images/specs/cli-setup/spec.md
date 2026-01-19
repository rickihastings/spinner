# cli-setup Specification Delta

## REMOVED Requirements

### Requirement: Setup Command Flags

The setup command SHALL accept the following CLI flags: --name (required), --jvm-url (required), and --node-version (
optional, defaults to 20). The CLI SHALL NOT prompt for interactive input.

#### Scenario: All required flags provided

- **WHEN** user runs
  `setup --name my-sandbox --jvm-url https://download.oracle.com/java/25/latest/jdk-25_linux-aarch64_bin.tar.gz`
- **THEN** the CLI proceeds with Docker image build

#### Scenario: Missing required flag

- **WHEN** user runs `setup` without --name or --jvm-url
- **THEN** the CLI exits with error code 1 and displays usage information

#### Scenario: Custom node version

- **WHEN** user runs
  `setup --name test --jvm-url https://download.oracle.com/java/25/latest/jdk-25_linux-aarch64_bin.tar.gz --node-version 20`
- **THEN** the Docker image is built with Node.js version 20

### Requirement: JVM Download

The Docker build process SHALL download the JDK from the URL provided via the --jvm-url flag. The user is responsible
for providing a URL compatible with the target container architecture.

#### Scenario: Successful JDK download

- **WHEN** the --jvm-url points to a valid JDK tarball
- **THEN** the JDK is downloaded and installed in the Docker image

#### Scenario: Invalid JVM URL

- **WHEN** the --jvm-url points to an inaccessible or invalid resource
- **THEN** the Docker build fails with an appropriate error message

## ADDED Requirements

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

## MODIFIED Requirements

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
