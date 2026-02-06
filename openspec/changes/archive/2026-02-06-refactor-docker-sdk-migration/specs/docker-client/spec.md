# Docker Client Specification

## ADDED Requirements

### Requirement: SDK-Based Docker Operations

The Docker client SHALL use the Docker Go SDK (`github.com/docker/docker/client`) for all Docker Engine operations instead of CLI command execution.

#### Scenario: SDK client initialization

- **WHEN** any Docker operation is requested
- **THEN** an SDK client is lazily initialized with environment-based configuration
- **AND** API version negotiation is enabled for compatibility

#### Scenario: Image existence check uses SDK

- **WHEN** checking if a Docker image exists
- **THEN** the SDK's `ImageInspectWithRaw` method is used
- **AND** the result is equivalent to `docker image inspect`

#### Scenario: Container inspection uses SDK

- **WHEN** checking container status
- **THEN** the SDK's `ContainerInspect` method is used
- **AND** the status (running/stopped/none) is correctly determined

### Requirement: Container Lifecycle via SDK

The Docker client SHALL manage container lifecycle using SDK methods.

#### Scenario: Container creation and start

- **WHEN** running a new container
- **THEN** the SDK's `ContainerCreate` is called with appropriate config
- **AND** the SDK's `ContainerStart` is called to start the container
- **AND** volume mounts are configured via HostConfig

#### Scenario: Container removal

- **WHEN** removing a container
- **THEN** the SDK's `ContainerRemove` with Force option is used
- **AND** running containers are forcefully stopped and removed

#### Scenario: Container restart

- **WHEN** restarting a stopped container
- **THEN** the SDK's `ContainerStart` method is used

### Requirement: Image Building via SDK

The Docker client SHALL build images using the SDK's ImageBuild method.

#### Scenario: Build context as tar archive

- **WHEN** building a Docker image
- **THEN** the build context directory is packaged as a tar archive
- **AND** the tar is streamed to the SDK's `ImageBuild` method

#### Scenario: User Dockerfile support

- **WHEN** a user-provided Dockerfile path is specified
- **THEN** the user's image is built first using SDK
- **AND** the resulting image is used as the base for the spinner image

#### Scenario: Build output handling

- **WHEN** building an image
- **THEN** build output is processed for errors
- **AND** build failures are reported with appropriate error messages

### Requirement: Container Log Streaming

The Docker client SHALL support real-time container log streaming.

#### Scenario: Log stream initiation

- **WHEN** requesting container logs with streaming
- **THEN** a channel of log events is returned
- **AND** logs are delivered in real-time as they are produced

#### Scenario: Log stream cancellation

- **WHEN** the context is cancelled during log streaming
- **THEN** the log stream is cleanly terminated
- **AND** the channel is closed

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
