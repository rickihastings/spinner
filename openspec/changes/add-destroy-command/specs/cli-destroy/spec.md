# cli-destroy Specification

## Purpose

Provide a dedicated CLI command for forcefully destroying one or more instances and their host-side state, abstracted
behind the Provider interface to work across all backends (Docker, GCP).

## ADDED Requirements

### Requirement: Destroy Command

The CLI SHALL provide a `destroy` command that forcefully removes one or more instances by name, regardless of their
current state.

#### Scenario: Destroy a running instance

- **WHEN** user runs `spinner destroy <instance-name>` and the instance is running
- **THEN** the CLI SHALL call Provider.Remove() to forcefully destroy the instance
- **AND** the CLI SHALL print a success message indicating the instance was destroyed

#### Scenario: Destroy a stopped instance

- **WHEN** user runs `spinner destroy <instance-name>` and the instance is stopped
- **THEN** the CLI SHALL call Provider.Remove() to destroy the instance
- **AND** the CLI SHALL print a success message indicating the instance was destroyed

#### Scenario: Destroy multiple instances

- **WHEN** user runs `spinner destroy <name1> <name2> <name3>`
- **THEN** the CLI SHALL attempt to destroy each instance in order
- **AND** the CLI SHALL print per-instance success or failure messages
- **AND** the CLI SHALL continue processing remaining instances if one fails

#### Scenario: Some instances fail to destroy

- **WHEN** user runs `spinner destroy <name1> <name2>` and one instance fails
- **THEN** the CLI SHALL print the failure for that instance
- **AND** the CLI SHALL continue destroying remaining instances
- **AND** the CLI SHALL exit with a non-zero status code

#### Scenario: Instance not found

- **WHEN** user runs `spinner destroy <instance-name>` and no instance with that name exists
- **THEN** the CLI SHALL print an error message indicating the instance was not found
- **AND** the CLI SHALL exit with a non-zero status code

#### Scenario: Missing instance name argument

- **WHEN** user runs `spinner destroy` without any instance names
- **THEN** Cobra SHALL print an error message and usage information

### Requirement: State Directory Cleanup

The destroy command SHALL automatically remove the host-side state directory for each successfully destroyed instance.

#### Scenario: State directory exists

- **WHEN** an instance is successfully destroyed
- **AND** `~/.spinner/<instance-name>/` exists on the host
- **THEN** the CLI SHALL remove the entire `~/.spinner/<instance-name>/` directory including state and logs

#### Scenario: State directory does not exist

- **WHEN** an instance is successfully destroyed
- **AND** `~/.spinner/<instance-name>/` does not exist on the host
- **THEN** the CLI SHALL not treat the missing directory as an error

### Requirement: Destroy Command Backend Support

The destroy command SHALL support backend selection via the `--backend` flag, consistent with other commands.

#### Scenario: Docker backend (default)

- **WHEN** user runs `spinner destroy <instance-name>` without `--backend` flag
- **THEN** the CLI SHALL use the Docker backend to destroy the instance

#### Scenario: GCP backend

- **WHEN** user runs `spinner destroy <instance-name> --backend gcp --project <p> --zone <z> --state-bucket <b>`
- **THEN** the CLI SHALL use the GCP backend to destroy the instance

#### Scenario: GCP backend missing required flags

- **WHEN** user runs `spinner destroy <instance-name> --backend gcp` without required GCP flags
- **THEN** the CLI SHALL print an error indicating which required flags are missing

### Requirement: Destroy Command Factory Injection

The destroy command SHALL use the provider Factory pattern for dependency injection, enabling testability.

#### Scenario: Constructor accepts factory

- **WHEN** `NewDestroyCommand(f *provider.Factory)` is called
- **THEN** it SHALL return a configured Cobra command that uses the provided factory to create providers

#### Scenario: Unit test with mock provider

- **WHEN** a test creates a destroy command with a mock factory
- **THEN** the test SHALL be able to verify destroy behavior without real backend operations
