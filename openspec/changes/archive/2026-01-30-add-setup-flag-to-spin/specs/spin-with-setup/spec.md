# Spec Delta: Spin Command with Setup Flag

**Related Spec**: `cli-spin`

## ADDED Requirements

### Requirement: Setup Flag Support
The spin command SHALL accept an optional `--setup` boolean flag that triggers image build before container creation.

#### Scenario: Setup flag builds image before spinning
- **WHEN** user runs `spinner spin --setup --image my-env --repo <url>`
- **THEN** the CLI SHALL build the Docker image `spinner:my-env` before creating the container
- **AND** the build SHALL use the default base image (ubuntu:22.04) if no base image is specified

#### Scenario: Setup with custom base image
- **WHEN** user runs `spinner spin --setup --image my-env --base-image node:20-bullseye --repo <url>`
- **THEN** the CLI SHALL build the Docker image using node:20-bullseye as the base image
- **AND** then create and start the container from the built image

#### Scenario: Setup with custom Dockerfile
- **WHEN** user runs `spinner spin --setup --image my-env --dockerfile ./custom.Dockerfile --repo <url>`
- **THEN** the CLI SHALL build the Docker image using the custom Dockerfile
- **AND** then create and start the container from the built image

#### Scenario: Setup always rebuilds existing image
- **WHEN** user runs `spinner spin --setup --image my-env --repo <url>`
- **AND** an image tagged `spinner:my-env` already exists
- **THEN** the CLI SHALL rebuild the image (no caching or skip logic)
- **AND** then proceed to spin the container using the rebuilt image

#### Scenario: Setup flag absent uses existing image
- **WHEN** user runs `spinner spin --image my-env --repo <url>` without --setup flag
- **THEN** the CLI SHALL skip image building entirely
- **AND** proceed directly to container creation using the existing `spinner:my-env` image
- **AND** behavior is identical to current spin command (no changes)

### Requirement: Base Image Flag in Spin Command
The spin command SHALL accept an optional `--base-image` flag that specifies the base Docker image when used with `--setup`.

#### Scenario: Base image flag requires setup flag
- **WHEN** user runs `spinner spin --image my-env --base-image ubuntu:22.04 --repo <url>` without --setup flag
- **THEN** the CLI SHALL print an error message indicating --base-image requires --setup flag
- **AND** exit with non-zero status

#### Scenario: Base image flag with setup flag
- **WHEN** user runs `spinner spin --setup --image my-env --base-image debian:bullseye --repo <url>`
- **THEN** the CLI SHALL build the image using debian:bullseye as the base
- **AND** proceed with container creation

#### Scenario: Base image flag default value
- **WHEN** user runs `spinner spin --setup --image my-env --repo <url>` without specifying --base-image
- **THEN** the CLI SHALL use ubuntu:22.04 as the default base image

### Requirement: Dockerfile Flag in Spin Command
The spin command SHALL accept an optional `--dockerfile` flag that specifies a custom Dockerfile path when used with `--setup`.

#### Scenario: Dockerfile flag requires setup flag
- **WHEN** user runs `spinner spin --image my-env --dockerfile ./custom.Dockerfile --repo <url>` without --setup flag
- **THEN** the CLI SHALL print an error message indicating --dockerfile requires --setup flag
- **AND** exit with non-zero status

#### Scenario: Dockerfile flag with setup flag
- **WHEN** user runs `spinner spin --setup --image my-env --dockerfile ./Dockerfile.dev --repo <url>`
- **THEN** the CLI SHALL validate the Dockerfile path exists
- **AND** build the image using the custom Dockerfile
- **AND** proceed with container creation

#### Scenario: Missing Dockerfile path
- **WHEN** user runs `spinner spin --setup --image my-env --dockerfile ./nonexistent.Dockerfile --repo <url>`
- **THEN** the CLI SHALL print an error message indicating the Dockerfile was not found
- **AND** exit with non-zero status before attempting to build

### Requirement: Mutually Exclusive Setup Options
The spin command SHALL enforce mutual exclusivity between `--base-image` and `--dockerfile` flags.

#### Scenario: Both base-image and dockerfile provided
- **WHEN** user runs `spinner spin --setup --image my-env --base-image ubuntu:22.04 --dockerfile ./custom.Dockerfile --repo <url>`
- **THEN** the CLI SHALL print an error message indicating the flags are mutually exclusive
- **AND** exit with non-zero status before attempting to build

#### Scenario: Only base-image provided
- **WHEN** user runs `spinner spin --setup --image my-env --base-image ubuntu:22.04 --repo <url>`
- **THEN** the CLI SHALL proceed with image build using the specified base image

#### Scenario: Only dockerfile provided
- **WHEN** user runs `spinner spin --setup --image my-env --dockerfile ./custom.Dockerfile --repo <url>`
- **THEN** the CLI SHALL proceed with image build using the custom Dockerfile

#### Scenario: Neither base-image nor dockerfile provided
- **WHEN** user runs `spinner spin --setup --image my-env --repo <url>`
- **THEN** the CLI SHALL use the default base image (ubuntu:22.04)

### Requirement: Image Name as Setup Name
When `--setup` flag is used, the value of `--image` SHALL serve as both the setup name and the image tag for spinning.

#### Scenario: Image flag used for setup naming
- **WHEN** user runs `spinner spin --setup --image custom-env --repo <url>`
- **THEN** the built image SHALL be tagged as `spinner:custom-env`
- **AND** the container SHALL be created from `spinner:custom-env`

#### Scenario: Image flag format consistency
- **WHEN** user provides `--image` value without "spinner:" prefix
- **THEN** the CLI SHALL automatically use the value as the tag after "spinner:"
- **AND** maintain consistent behavior with standalone setup command

### Requirement: Prerequisite Validation Before Setup
When `--setup` flag is used, the spin command SHALL validate prerequisites before building the image.

#### Scenario: Prerequisites checked before build
- **WHEN** user runs `spinner spin --setup --image my-env --repo <url>`
- **THEN** the CLI SHALL check that docker, git, and claude-code are installed
- **AND** display prerequisite validation results before starting the build

#### Scenario: Prerequisite failure prevents build
- **WHEN** user runs `spinner spin --setup --image my-env --repo <url>`
- **AND** docker is not installed
- **THEN** the CLI SHALL print an error message indicating docker is required
- **AND** exit with non-zero status without attempting to build or spin

#### Scenario: Build failure prevents spin
- **WHEN** user runs `spinner spin --setup --image my-env --repo <url>`
- **AND** the image build fails (e.g., invalid base image)
- **THEN** the CLI SHALL display the build error
- **AND** exit with non-zero status without attempting to create a container

### Requirement: Combined Setup and Spin Error Handling
The spin command SHALL provide clear error messages that distinguish between setup phase errors and spin phase errors.

#### Scenario: Setup phase error message
- **WHEN** an error occurs during image build (setup phase)
- **THEN** the CLI SHALL prefix error messages with context indicating build failure
- **AND** not attempt to proceed to container creation

#### Scenario: Spin phase error message after successful setup
- **WHEN** image build succeeds but container creation fails
- **THEN** the CLI SHALL indicate that setup completed successfully
- **AND** display container creation error separately
- **AND** user can retry spin without --setup flag

### Requirement: Viper Environment Variable Support for Setup Flags
The spin command SHALL support environment variable overrides for setup-related flags via Viper.

#### Scenario: SPINNER_SETUP environment variable
- **WHEN** SPINNER_SETUP environment variable is set to "true"
- **AND** user runs `spinner spin --image my-env --repo <url>` without --setup flag
- **THEN** the CLI SHALL behave as if --setup flag was provided

#### Scenario: SPINNER_BASE_IMAGE environment variable with SPINNER_SETUP
- **WHEN** SPINNER_SETUP=true and SPINNER_BASE_IMAGE=node:20 are set
- **AND** user runs `spinner spin --image my-env --repo <url>`
- **THEN** the CLI SHALL build the image using node:20 as base image

#### Scenario: SPINNER_DOCKERFILE environment variable with SPINNER_SETUP
- **WHEN** SPINNER_SETUP=true and SPINNER_DOCKERFILE=./dev.Dockerfile are set
- **AND** user runs `spinner spin --image my-env --repo <url>`
- **THEN** the CLI SHALL build the image using the specified Dockerfile

#### Scenario: Command-line flags override environment variables
- **WHEN** SPINNER_BASE_IMAGE=ubuntu:22.04 environment variable is set
- **AND** user runs `spinner spin --setup --image my-env --base-image debian:bullseye --repo <url>`
- **THEN** the CLI SHALL use debian:bullseye (command-line flag takes precedence)

## MODIFIED Requirements

### Requirement: Spin Command Flags
The CLI SHALL accept additional optional flags for the spin command when --setup is used: --base-image and --dockerfile, validated by Cobra's flag parsing system. This extends the existing "Spin Command Flags" requirement from cli-spin.

#### Scenario: Setup flag in help documentation
- **WHEN** user runs `spinner spin --help`
- **THEN** the CLI SHALL display documentation for --setup, --base-image, and --dockerfile flags
- **AND** indicate that --base-image and --dockerfile require --setup flag

#### Scenario: Flag validation with setup
- **WHEN** user provides invalid combinations of setup-related flags
- **THEN** Cobra SHALL display appropriate error messages and usage information