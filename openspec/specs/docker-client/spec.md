# docker-client Specification

## Purpose

The docker-client spec defines how the application interacts with the Docker Engine. It uses Docker CLI commands (`docker`) to provide access to Docker operations including image building, container lifecycle management, and real-time log streaming. This abstraction enables the application to create isolated environments for code execution while maintaining testability through well-defined interfaces.
## Requirements
### Requirement: CLI-Based Docker Operations

The Docker client SHALL use Docker CLI commands for all Docker Engine operations.

#### Scenario: Image existence check

- **WHEN** checking if a Docker image exists
- **THEN** `docker image inspect` is executed
- **AND** the exit code determines existence (0 = exists, non-zero = not found)

#### Scenario: Container inspection

- **WHEN** checking container status
- **THEN** `docker inspect --format` is used to query container state
- **AND** the status (running/stopped/none) is correctly determined

### Requirement: Container Lifecycle via CLI

The Docker client SHALL manage container lifecycle using CLI commands.

#### Scenario: Container start

- **WHEN** starting a container
- **THEN** `docker start` is executed

#### Scenario: Container stop

- **WHEN** stopping a container
- **THEN** `docker stop -t 10` is executed with a 10-second grace period

#### Scenario: Container removal

- **WHEN** removing a container
- **THEN** `docker rm -f` is used to force-remove the container

#### Scenario: Container creation and run

- **WHEN** running a new container
- **THEN** `docker run` is executed with appropriate flags
- **AND** volume mounts, labels, and environment variables are passed as CLI flags

### Requirement: Image Building via CLI

The Docker client SHALL build images using `docker build`.

#### Scenario: Standard image build

- **WHEN** building a Docker image
- **THEN** `docker build -t <tag> -f <dockerfile> <context>` is executed
- **AND** build arguments are passed via `--build-arg` flags
- **AND** build output streams directly to stdout/stderr

#### Scenario: User Dockerfile support

- **WHEN** a user-provided Dockerfile path is specified
- **THEN** the user's image is built first using `docker build`
- **AND** the resulting image is used as the base for the spinner image

### Requirement: Container Log Streaming

The Docker client SHALL support real-time container log streaming via CLI.

#### Scenario: Log stream initiation

- **WHEN** requesting container logs with streaming
- **THEN** `docker logs --follow --timestamps` is executed
- **AND** a channel of log events is returned with line-by-line reading

#### Scenario: Log stream cancellation

- **WHEN** the context is cancelled during log streaming
- **THEN** the CLI process is terminated via context cancellation
- **AND** the channel is closed

#### Scenario: Non-streaming log retrieval

- **WHEN** requesting container logs without streaming
- **THEN** `docker logs` is executed
- **AND** the full log output is returned as a string

### Requirement: Error Handling

The Docker client SHALL provide consistent error messages regardless of implementation.

#### Scenario: Operation failure

- **WHEN** a Docker operation fails
- **THEN** the error message includes operation context
- **AND** the error format is consistent with expected error handling

#### Scenario: Connection failure

- **WHEN** the Docker daemon is not accessible
- **THEN** a clear error message indicates the connection failure
- **AND** the user is informed how to resolve the issue

### Requirement: Container Discovery Labels

The Docker provider SHALL apply a `spinner-managed=true` label to all containers created via `Provider.Create()`.

#### Scenario: Label applied at creation

- **WHEN** `Provider.Create()` creates a new Docker container
- **THEN** the container SHALL have the label `spinner-managed=true`
- **AND** the label SHALL be passed via `--label spinner-managed=true` in the Docker run arguments

### Requirement: Container Listing

The Docker Client interface SHALL support listing containers filtered by labels.

#### Scenario: List by label filter

- **WHEN** `ListContainers()` is called with a label filter
- **THEN** `docker ps -a --filter label=<key>=<value> --format json` is executed
- **AND** JSON output is parsed into `ContainerListEntry` structs
- **AND** all matching containers (both running and stopped) are returned

### Requirement: Docker Instance Listing

The Docker provider SHALL implement `Provider.List()` to discover all spinner-managed containers.

#### Scenario: Successful listing

- **WHEN** `Provider.List()` is called
- **THEN** the provider SHALL query Docker for containers with label `spinner-managed=true`
- **AND** return results as `[]InstanceInfo`

#### Scenario: State enrichment from host

- **WHEN** a container is discovered during listing
- **THEN** the provider SHALL read `~/.spinner/<name>/state/state.json` from the host
- **AND** populate iteration, agent status, and timestamp fields in `InstanceInfo`
- **AND** if the state file does not exist, those fields SHALL be zero-valued

#### Scenario: Docker not available

- **WHEN** Docker is not running or not installed
- **THEN** `List()` SHALL return an error indicating Docker is unavailable

### Requirement: Metrics Collection via CLI

The Docker client SHALL collect container metrics using CLI commands.

#### Scenario: Stats collection

- **WHEN** collecting container metrics
- **THEN** `docker stats --no-stream --format json` is executed
- **AND** CPU and memory usage are parsed from the JSON output

#### Scenario: Container state inspection

- **WHEN** inspecting container state for metrics
- **THEN** `docker inspect --format '{{json .State}}'` is executed
- **AND** state fields (Running, ExitCode, Dead, OOMKilled) are parsed from JSON
