# cli-spin Specification Delta

## ADDED Requirements

### Requirement: Docker Socket Mounting

The Docker backend SHALL automatically mount the host's Docker socket into spinner containers, enabling Docker-in-Docker
(DooD) workflows such as docker-compose, testcontainers, and custom Docker tooling.

#### Scenario: Socket mounted when available

- **WHEN** the Docker backend creates a container
- **AND** `/var/run/docker.sock` exists on the host
- **AND** Docker socket mounting is not disabled
- **THEN** the CLI SHALL mount `/var/run/docker.sock` read-write into the container at `/var/run/docker.sock`
- **AND** the CLI SHALL add `--add-host=host.docker.internal:host-gateway` to the docker run command

#### Scenario: Socket not mounted when missing

- **WHEN** the Docker backend creates a container
- **AND** `/var/run/docker.sock` does NOT exist on the host
- **THEN** the CLI SHALL NOT attempt to mount the Docker socket
- **AND** the CLI SHALL NOT add `--add-host=host.docker.internal:host-gateway`
- **AND** container creation SHALL proceed normally

#### Scenario: Sibling container labeling

- **WHEN** the Docker socket is mounted into a spinner container
- **THEN** the CLI SHALL set the environment variable `DOCKER_DEFAULT_LABELS=spinner-parent=<container-name>` in the container
- **AND** containers created inside the spinner container (via docker-compose or other tools) SHALL inherit this label

#### Scenario: Socket not mounted for GCP backend

- **WHEN** the GCP backend creates a VM
- **THEN** Docker socket mounting SHALL NOT apply (GCP VMs have their own Docker daemon)

### Requirement: Docker Socket Opt-Out

The CLI SHALL support disabling Docker socket mounting via a CLI flag and configuration file.

#### Scenario: Opt-out via CLI flag

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --no-docker-socket`
- **THEN** the Docker socket SHALL NOT be mounted into the container
- **AND** `host.docker.internal` SHALL NOT be added
- **AND** `DOCKER_DEFAULT_LABELS` SHALL NOT be set

#### Scenario: Opt-out via config file

- **WHEN** `.spinner.json` contains `{"docker-socket": false}`
- **AND** user runs `spinner spin --image <image> --repo <repo>`
- **THEN** the Docker socket SHALL NOT be mounted into the container

#### Scenario: CLI flag overrides config file

- **WHEN** `.spinner.json` contains `{"docker-socket": false}`
- **AND** user runs `spinner spin --image <image> --repo <repo>` without `--no-docker-socket`
- **THEN** the CLI flag absence SHALL NOT override the config file
- **AND** the Docker socket SHALL NOT be mounted

#### Scenario: Default behavior (enabled)

- **WHEN** no `--no-docker-socket` flag is provided
- **AND** `.spinner.json` does not contain `"docker-socket": false`
- **THEN** Docker socket mounting SHALL be enabled by default
