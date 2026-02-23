# docker-client Specification Delta

## ADDED Requirements

### Requirement: Sibling Container Discovery

The Docker client SHALL support discovering containers created by a spinner container via label filtering.

#### Scenario: Query sibling containers by parent label

- **WHEN** the client queries for sibling containers of a spinner container
- **THEN** it SHALL execute `docker ps -aq --filter label=spinner-parent=<container-name>`
- **AND** return a list of container IDs

#### Scenario: No sibling containers found

- **WHEN** the client queries for sibling containers
- **AND** no containers have the `spinner-parent=<container-name>` label
- **THEN** it SHALL return an empty list

### Requirement: Sibling Network Discovery

The Docker client SHALL support discovering Docker networks created by sibling containers.

#### Scenario: Query sibling networks by parent label

- **WHEN** the client queries for sibling networks of a spinner container
- **THEN** it SHALL execute `docker network ls --filter label=spinner-parent=<container-name> -q`
- **AND** return a list of network IDs

### Requirement: Bulk Container Removal

The Docker client SHALL support removing multiple containers in a single operation.

#### Scenario: Remove multiple containers by ID

- **WHEN** the client is given a list of container IDs to remove
- **THEN** it SHALL execute `docker rm -f <id1> <id2> ...` to forcefully remove all containers

#### Scenario: Empty container list

- **WHEN** the client is given an empty list of container IDs
- **THEN** it SHALL be a no-op (no Docker commands executed)

### Requirement: Network Removal

The Docker client SHALL support removing Docker networks.

#### Scenario: Remove networks by ID

- **WHEN** the client is given a list of network IDs to remove
- **THEN** it SHALL execute `docker network rm <id1> <id2> ...` to remove the networks

#### Scenario: Network removal failure

- **WHEN** a network cannot be removed (e.g., still in use)
- **THEN** the client SHALL return an error with context about which network failed
