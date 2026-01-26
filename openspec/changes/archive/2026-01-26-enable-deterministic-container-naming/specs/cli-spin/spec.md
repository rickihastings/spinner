## MODIFIED Requirements

### Requirement: Persistent Container

The container SHALL run in detached mode as a persistent background process. The container SHALL NOT be automatically removed when it exits (no --rm flag). The CLI SHALL assign a deterministic name to the container based on the image name, repository name, and optionally the branch name.

#### Scenario: Container runs in background

- **WHEN** the spin command completes successfully
- **THEN** the container is running in detached mode
- **AND** the CLI returns to the user's shell prompt immediately

#### Scenario: Deterministic container naming without branch

- **WHEN** user spins up a container with `--image spinner:default --repo git@github.com:user/my-project.git`
- **THEN** the container is named `spinner-default-my-project`
- **AND** the container name is displayed to the user

#### Scenario: Deterministic container naming with branch

- **WHEN** user spins up a container with `--image spinner:default --repo git@github.com:user/my-project.git --branch feature/auth-v2`
- **THEN** the container is named `spinner-default-my-project-feature-auth-v2`
- **AND** the container name is displayed to the user

#### Scenario: Container name sanitization

- **WHEN** the image is `spinner:my-env`, repo is `git@github.com:user/my.project.git`, and branch is `feature/auth-v2`
- **THEN** the container name is `spinner-my-env-my-project-feature-auth-v2`
- **AND** special characters (`:`, `/`, `.`) are replaced with hyphens

#### Scenario: Container persists after exit

- **WHEN** the cloned repository setup completes
- **THEN** the container continues running
- **AND** the container can be restarted with `docker start <container-name>`

#### Scenario: User can exec into container

- **WHEN** the container is running
- **THEN** user can exec into it with `docker exec -it <container-name> bash`
- **AND** the working directory is /workspace

## ADDED Requirements

### Requirement: Container Reuse

The CLI SHALL check if a container with the deterministic name already exists before creating a new one. If a container with the same name exists and is running, the CLI SHALL reuse it. If a container with the same name exists but is stopped, the CLI SHALL restart it. The CLI output SHALL clearly indicate whether a container was created, reused, or restarted.

#### Scenario: Reuse running container

- **WHEN** user runs `spin --image spinner:default --repo git@github.com:user/my-project.git`
- **AND** a container named `spinner-default-my-project` is already running
- **THEN** the CLI does not create a new container
- **AND** the CLI displays "Reusing running container: spinner-default-my-project"
- **AND** the CLI displays the standard management instructions

#### Scenario: Restart stopped container

- **WHEN** user runs `spin --image spinner:default --repo git@github.com:user/my-project.git`
- **AND** a container named `spinner-default-my-project` exists but is stopped
- **THEN** the CLI restarts the existing container with `docker start`
- **AND** the CLI displays "Restarted container: spinner-default-my-project"
- **AND** the CLI displays the standard management instructions

#### Scenario: Create new container when none exists

- **WHEN** user runs `spin --image spinner:default --repo git@github.com:user/my-project.git`
- **AND** no container named `spinner-default-my-project` exists
- **THEN** the CLI creates a new container with that name
- **AND** the CLI displays "Container created successfully: spinner-default-my-project"
- **AND** the CLI displays the standard management instructions

### Requirement: Container Recreation Flag

The CLI SHALL accept an optional `--recreate` boolean flag. When provided, the CLI SHALL remove any existing container with the deterministic name and create a fresh container. This allows users to force a clean slate when reuse is not desired.

#### Scenario: Recreate removes running container

- **WHEN** user runs `spin --image spinner:default --repo git@github.com:user/my-project.git --recreate`
- **AND** a container named `spinner-default-my-project` is currently running
- **THEN** the CLI stops and removes the existing container
- **AND** the CLI creates a new container with the same name
- **AND** the CLI displays "Container recreated: spinner-default-my-project"

#### Scenario: Recreate removes stopped container

- **WHEN** user runs `spin --image spinner:default --repo git@github.com:user/my-project.git --recreate`
- **AND** a container named `spinner-default-my-project` exists but is stopped
- **THEN** the CLI removes the existing container
- **AND** the CLI creates a new container with the same name
- **AND** the CLI displays "Container recreated: spinner-default-my-project"

#### Scenario: Recreate creates when no container exists

- **WHEN** user runs `spin --image spinner:default --repo git@github.com:user/my-project.git --recreate`
- **AND** no container named `spinner-default-my-project` exists
- **THEN** the CLI creates a new container with that name
- **AND** the CLI displays "Container created successfully: spinner-default-my-project"
- **AND** the behavior is identical to running without `--recreate`
