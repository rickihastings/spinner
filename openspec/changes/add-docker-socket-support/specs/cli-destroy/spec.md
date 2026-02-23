# cli-destroy Specification Delta

## ADDED Requirements

### Requirement: Sibling Container Cleanup

The destroy command SHALL clean up sibling containers created inside the spinner container (via docker-compose,
testcontainers, etc.) before removing the spinner container itself.

#### Scenario: Sibling containers exist

- **WHEN** user runs `spinner destroy <instance-name>` with the Docker backend
- **AND** containers with label `spinner-parent=<instance-name>` exist
- **THEN** the CLI SHALL forcefully remove all labeled sibling containers before removing the spinner container
- **AND** the CLI SHALL remove any Docker networks with label `spinner-parent=<instance-name>`

#### Scenario: No sibling containers

- **WHEN** user runs `spinner destroy <instance-name>` with the Docker backend
- **AND** no containers with label `spinner-parent=<instance-name>` exist
- **THEN** the CLI SHALL proceed with normal spinner container removal

#### Scenario: Sibling cleanup failure does not block destroy

- **WHEN** sibling container cleanup fails (e.g., permission error, container in use)
- **THEN** the CLI SHALL log a warning about the cleanup failure
- **AND** the CLI SHALL proceed with removing the spinner container
- **AND** the destroy operation SHALL NOT fail due to sibling cleanup issues

#### Scenario: GCP backend skips sibling cleanup

- **WHEN** user runs `spinner destroy <instance-name>` with the GCP backend
- **THEN** sibling container cleanup SHALL NOT apply (GCP VMs manage their own Docker)
