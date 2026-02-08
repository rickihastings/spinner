# cli-spin Specification (delta)

## MODIFIED Requirements

### Requirement: Container Lifecycle Management

The CLI SHALL display container management instructions after creation, implemented using Go's fmt package.

#### Scenario: Display management instructions

- **WHEN** the container is created successfully
- **THEN** the CLI SHALL print instructions for accessing the container (docker exec or gcloud ssh) and destroying it
  (`spinner destroy <instance-name>`)

#### Scenario: Container cleanup

- **WHEN** user runs the suggested destroy command
- **THEN** the instance SHALL be forcefully removed along with its host-side state directory
