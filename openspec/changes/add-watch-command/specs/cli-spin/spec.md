## ADDED Requirements

### Requirement: Watch Flag for Spin Command

The CLI SHALL accept an optional `--watch` boolean flag for the spin command to automatically enter watch mode after
container creation or reuse.

#### Scenario: Watch flag provided with new container

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --watch`
- **THEN** the CLI SHALL create the container and immediately enter watch mode

#### Scenario: Watch flag provided with existing container

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --watch` and container already exists
- **THEN** the CLI SHALL reuse or restart the container and immediately enter watch mode

#### Scenario: Watch flag compatible with recreate

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --recreate --watch`
- **THEN** the CLI SHALL recreate the container and immediately enter watch mode

#### Scenario: Watch flag compatible with prompt

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --prompt "task" --watch`
- **THEN** the CLI SHALL create the container with ralph-loop and immediately enter watch mode

#### Scenario: Watch flag not provided

- **WHEN** user runs `spinner spin` without the `--watch` flag
- **THEN** the CLI SHALL display management instructions and exit without entering watch mode

### Requirement: Watch Transition After Spin

The CLI SHALL transition to watch mode using the same implementation as the standalone watch command.

#### Scenario: Shared watch implementation

- **WHEN** the `--watch` flag triggers watch mode
- **THEN** the CLI SHALL call the same watch function used by the standalone `watch` command with the container name

#### Scenario: Watch mode inherits container context

- **WHEN** watch mode is entered via `--watch` flag
- **THEN** the watch display SHALL show the same container metadata as the standalone watch command
