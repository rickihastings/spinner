# cli-setup Specification Delta

## ADDED Requirements

### Requirement: Docker CLI Installation in Image

The Docker image build process SHALL install the Docker CLI and Docker Compose plugin so that containers can interact
with the host Docker daemon via socket mounting.

#### Scenario: Docker CLI available in container

- **WHEN** a container is created from an image built by `spinner setup`
- **THEN** the `docker` CLI SHALL be available in PATH
- **AND** the `docker compose` subcommand SHALL be available

#### Scenario: Docker CLI installed via official repository

- **WHEN** the Dockerfile template installs the Docker CLI
- **THEN** it SHALL use Docker's official apt repository with GPG key verification
- **AND** it SHALL install `docker-ce-cli` and `docker-compose-plugin` packages only (NOT the daemon)

#### Scenario: Spinner user in docker group

- **WHEN** the image is built
- **THEN** the `spinner` user SHALL be a member of the `docker` group
- **AND** the `spinner` user SHALL be able to run `docker` commands without sudo when the socket is mounted
