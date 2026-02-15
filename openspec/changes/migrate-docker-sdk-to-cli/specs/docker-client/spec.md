# docker-client Specification

## MODIFIED Requirements

### Requirement: CLI-Based Docker Operations

The Docker client SHALL use the Docker CLI (`docker` command) for all Docker Engine operations instead of the Docker Go SDK.

#### Scenario: CLI command execution

- **WHEN** any Docker operation is requested
- **THEN** the operation SHALL be executed via `exec.CommandContext` calling the `docker` CLI
- **AND** JSON output format (`--format json` or `--format '{{json .}}'`) SHALL be used where structured data is needed

#### Scenario: Image existence check uses CLI

- **WHEN** checking if a Docker image exists
- **THEN** `docker image inspect <name>` SHALL be executed
- **AND** exit code 0 indicates the image exists, non-zero indicates it does not

#### Scenario: Container inspection uses CLI

- **WHEN** checking container status
- **THEN** `docker inspect --format '{{.State.Running}}' <name>` SHALL be executed
- **AND** the status (running/stopped/none) SHALL be correctly determined from the output

### Requirement: Container Lifecycle via CLI

The Docker client SHALL manage container lifecycle using CLI commands.

#### Scenario: Container start

- **WHEN** starting a stopped container
- **THEN** `docker start <name>` SHALL be executed

#### Scenario: Container stop

- **WHEN** stopping a running container
- **THEN** `docker stop -t 10 <name>` SHALL be executed
- **AND** a 10-second timeout SHALL be used before force-killing

#### Scenario: Container removal

- **WHEN** removing a container
- **THEN** `docker rm -f <name>` SHALL be executed
- **AND** running containers SHALL be forcefully stopped and removed

#### Scenario: Container creation and run

- **WHEN** running a new container
- **THEN** `docker run` SHALL be executed with the constructed argument list
- **AND** this is unchanged from the current CLI-based RunContainer implementation

### Requirement: Image Building via CLI

The Docker client SHALL build images using the `docker build` CLI command.

#### Scenario: Build invocation

- **WHEN** building a Docker image
- **THEN** `docker build -t <tag> -f <dockerfile> <context-dir>` SHALL be executed
- **AND** build args SHALL be passed via `--build-arg KEY=VALUE` flags

#### Scenario: User Dockerfile support

- **WHEN** a user-provided Dockerfile path is specified
- **THEN** the user's image SHALL be built first using `docker build`
- **AND** the resulting image SHALL be used as the base for the spinner image

#### Scenario: Build output handling

- **WHEN** building an image
- **THEN** build output SHALL be streamed directly to stdout/stderr
- **AND** the exit code SHALL determine success or failure

### Requirement: Container Log Streaming

The Docker client SHALL support real-time container log streaming via CLI.

#### Scenario: Log stream initiation

- **WHEN** requesting container logs with streaming
- **THEN** `docker logs --follow --timestamps <name>` SHALL be executed
- **AND** stdout/stderr SHALL be read line-by-line and delivered as log events

#### Scenario: Log retrieval without streaming

- **WHEN** requesting container logs without follow mode
- **THEN** `docker logs <name>` SHALL be executed
- **AND** the combined stdout/stderr output SHALL be returned

#### Scenario: Log stream cancellation

- **WHEN** the context is cancelled during log streaming
- **THEN** the `docker logs` process SHALL be terminated via context cancellation
- **AND** the log event channel SHALL be closed

### Requirement: Error Handling

The Docker client SHALL provide consistent error messages regardless of implementation.

#### Scenario: Operation failure

- **WHEN** a Docker operation fails
- **THEN** the error message SHALL include operation context
- **AND** the error format SHALL be consistent with expected error handling

#### Scenario: Connection failure

- **WHEN** the Docker daemon is not accessible
- **THEN** a clear error message SHALL indicate the connection failure
- **AND** the user SHALL be informed how to resolve the issue

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
- **THEN** `docker ps -a --filter label=<key>=<value> --format json` SHALL be executed
- **AND** all matching containers (both running and stopped) SHALL be returned

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

## REMOVED Requirements

### Requirement: SDK-Based Docker Operations

**Reason**: Replaced by CLI-Based Docker Operations. The SDK approach limits arg passthrough, bloats binary size, and creates inconsistency with existing CLI calls.
**Migration**: All SDK calls replaced with equivalent `docker` CLI invocations. The `Client` interface remains unchanged; only the `RealDockerClient` implementation changes.

### Requirement: Container Lifecycle via SDK

**Reason**: Replaced by Container Lifecycle via CLI. SDK container lifecycle methods replaced with CLI equivalents.
**Migration**: `ContainerCreate`/`ContainerStart`/`ContainerStop`/`ContainerRemove` SDK calls replaced with `docker start`/`docker stop`/`docker rm` CLI calls.

### Requirement: Image Building via SDK

**Reason**: Replaced by Image Building via CLI. SDK tar-streaming build replaced with `docker build` CLI.
**Migration**: `ImageBuild` SDK call replaced with `docker build` CLI invocation. Build context tar creation is no longer needed.

## ADDED Requirements

### Requirement: Container Metrics via CLI

The Docker client SHALL collect container resource metrics using CLI commands instead of the Docker SDK stats API.

#### Scenario: Metrics collection

- **WHEN** collecting metrics for a running container
- **THEN** `docker stats --no-stream --format json <name>` SHALL be executed
- **AND** CPU percentage, memory usage, and memory limit SHALL be parsed from the JSON output

#### Scenario: Container state for metrics

- **WHEN** determining container state for metrics
- **THEN** `docker inspect` SHALL be used to determine the container's running state
- **AND** the state SHALL be mapped to running/stopped/exited/unknown
