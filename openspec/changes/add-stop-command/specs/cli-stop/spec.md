## ADDED Requirements

### Requirement: Stop Command

The CLI SHALL provide a `stop` command that gracefully stops one or more running instances by name without destroying
them or their state.

#### Scenario: Stop a running instance

- **WHEN** user runs `spinner stop <instance-name>` and the instance is running
- **THEN** the CLI SHALL call Provider.Stop() to stop the instance
- **AND** the CLI SHALL print a success message indicating the instance was stopped

#### Scenario: Stop multiple instances

- **WHEN** user runs `spinner stop <name1> <name2>`
- **THEN** the CLI SHALL attempt to stop each instance in order
- **AND** the CLI SHALL print per-instance success or failure messages
- **AND** the CLI SHALL continue processing remaining instances if one fails

#### Scenario: Some instances fail to stop

- **WHEN** user runs `spinner stop <name1> <name2>` and one instance fails
- **THEN** the CLI SHALL print the failure for that instance
- **AND** the CLI SHALL continue stopping remaining instances
- **AND** the CLI SHALL exit with a non-zero status code

#### Scenario: Instance already stopped

- **WHEN** user runs `spinner stop <instance-name>` and the instance is already stopped
- **THEN** the CLI SHALL print a message indicating the instance is already stopped
- **AND** the CLI SHALL NOT treat this as an error
- **AND** the CLI SHALL exit with a zero status code

#### Scenario: Instance not found

- **WHEN** user runs `spinner stop <instance-name>` and no instance with that name exists
- **THEN** the CLI SHALL print a message indicating the instance was not found
- **AND** the CLI SHALL NOT treat this as an error
- **AND** the CLI SHALL exit with a zero status code

#### Scenario: Missing instance name argument

- **WHEN** user runs `spinner stop` without any instance names
- **THEN** Cobra SHALL print an error message and usage information

### Requirement: Stop Command Backend Support

The stop command SHALL support backend selection via the `--backend` flag, consistent with other commands.

#### Scenario: Docker backend (default)

- **WHEN** user runs `spinner stop <instance-name>` without `--backend` flag
- **THEN** the CLI SHALL use the Docker backend to stop the instance

#### Scenario: GCP backend

- **WHEN** user runs `spinner stop <instance-name> --backend gcp --project <p> --zone <z>`
- **THEN** the CLI SHALL use the GCP backend to stop the instance

#### Scenario: GCP backend missing required flags

- **WHEN** user runs `spinner stop <instance-name> --backend gcp` without required GCP flags
- **THEN** the CLI SHALL print an error indicating which required flags are missing

### Requirement: Stop Command Factory Injection

The stop command SHALL use the provider Factory pattern for dependency injection, enabling testability.

#### Scenario: Constructor accepts factory

- **WHEN** `NewStopCommand(f *provider.Factory)` is called
- **THEN** it SHALL return a configured Cobra command that uses the provided factory to create providers

#### Scenario: Unit test with mock provider

- **WHEN** a test creates a stop command with a mock factory
- **THEN** the test SHALL be able to verify stop behaviour without real backend operations

### Requirement: Spin Command Stop Hint

The `spin` command output SHALL reference `spinner stop` instead of raw backend CLI commands for the stop hint.

#### Scenario: Docker spin output

- **WHEN** `spinner spin` successfully creates or reuses a Docker instance
- **THEN** the output SHALL include `spinner stop <instance-name>` as the stop hint

#### Scenario: GCP spin output

- **WHEN** `spinner spin` successfully creates or reuses a GCP instance
- **THEN** the output SHALL include `spinner stop <instance-name> --backend gcp --project <p> --zone <z>` as the stop
  hint