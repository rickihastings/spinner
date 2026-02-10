# docker-client Specification

## Purpose

Extend the Docker provider with container discovery labels and listing support for the `spinner list` command.

## ADDED Requirements

### Requirement: Container Discovery Labels

The Docker provider SHALL apply a `spinner-managed=true` label to all containers created via `Provider.Create()`.

#### Scenario: Label applied at creation

- **WHEN** `Provider.Create()` creates a new Docker container
- **THEN** the container SHALL have the label `spinner-managed=true`
- **AND** the label SHALL be passed via `--label spinner-managed=true` in the Docker run arguments

#### Scenario: Existing unlabeled containers

- **WHEN** listing instances and containers exist without the `spinner-managed=true` label
- **THEN** the provider SHALL also discover containers matching the `spinner-` name prefix as a fallback
- **AND** deduplicate results by container ID

### Requirement: Container Listing

The Docker Client interface SHALL support listing containers filtered by labels.

#### Scenario: List by label filter

- **WHEN** `ListContainers()` is called with a label filter
- **THEN** the client SHALL use the Docker SDK `ContainerList` API with the provided filters
- **AND** return all matching containers (both running and stopped)

### Requirement: Docker Instance Listing

The Docker provider SHALL implement `Provider.List()` to discover all spinner-managed containers.

#### Scenario: Successful listing

- **WHEN** `Provider.List()` is called
- **THEN** the provider SHALL query Docker for containers with label `spinner-managed=true`
- **AND** also query for containers with name prefix `spinner-` (fallback for pre-label containers)
- **AND** return deduplicated results as `[]InstanceInfo`

#### Scenario: State enrichment from host

- **WHEN** a container is discovered during listing
- **THEN** the provider SHALL read `~/.spinner/<name>/state/state.json` from the host
- **AND** populate iteration, agent status, and timestamp fields in `InstanceInfo`
- **AND** if the state file does not exist, those fields SHALL be zero-valued

#### Scenario: Docker not available

- **WHEN** Docker is not running or not installed
- **THEN** `List()` SHALL return an error indicating Docker is unavailable
