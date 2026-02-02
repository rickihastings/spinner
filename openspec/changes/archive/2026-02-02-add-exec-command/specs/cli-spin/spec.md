# cli-spin Specification Delta

## MODIFIED Requirements

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

## ADDED Requirements

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
