# Proposal: Add Docker Socket Support

## Summary

Enable Docker-in-Docker (DooD) support for the Docker backend by automatically mounting the host's Docker socket into
spinner containers and installing the Docker CLI in the container image. This allows dev environments that depend on
docker-compose, testcontainers, or other Docker tooling to work seamlessly inside spinner containers without any extra
configuration.

## Motivation

Many development environments require Docker:

- **docker-compose**: Spinning up local Redis, Postgres, Elasticsearch, etc.
- **testcontainers**: Integration tests that create ephemeral containers
- **Custom tooling**: Build scripts that use Docker for compilation or packaging

Today, spinner's Docker backend runs containers without access to the host Docker daemon. The GCP backend doesn't have
this limitation (full VM with Docker installed), but the Docker backend — the most common development target — cannot
run any Docker commands.

Users *can* work around this via `--provider-args="-v /var/run/docker.sock:/var/run/docker.sock"`, but that requires
knowing the incantation, the container image doesn't include the `docker` CLI, and there's no networking support for
reaching sibling containers.

## Changes

### 1. Auto-mount Docker socket (cli-spin) — **MODIFIED**

When the Docker backend creates a container and `/var/run/docker.sock` exists on the host, automatically mount it into
the container at `/var/run/docker.sock`. Add `--add-host=host.docker.internal:host-gateway` so the container can reach
host-mapped ports from sibling containers (docker-compose services, testcontainers, etc.).

Opt-out via:
- `--no-docker-socket` CLI flag
- `"docker-socket": false` in `.spinner.json`

### 2. Install Docker CLI in image (cli-setup) — **MODIFIED**

Add Docker CLI and Docker Compose plugin installation to the extending Dockerfile template. The `spinner` user is added
to the `docker` group so it can use the socket without sudo.

### 3. Label sibling containers (docker-client) — **ADDED**

When Docker socket is mounted, set `DOCKER_DEFAULT_LABELS` environment variable so that containers created inside the
spinner container (via docker-compose, testcontainers, etc.) are labeled with the spinner container name. This enables
cleanup.

### 4. Cleanup sibling containers on destroy (cli-destroy) — **MODIFIED**

When destroying a Docker instance, query for any sibling containers labeled with the spinner container name and remove
them before removing the spinner container itself.

## Impact

- **Affected specs**: `cli-spin`, `cli-setup`, `docker-client`, `cli-destroy`
- **Breaking changes**: None. Docker socket is mounted by default but can be opted out. Existing containers without the
  socket continue to work.
- **Security**: The Docker socket grants root-equivalent access to the host. This is an accepted trade-off for
  development workflows (same as running Docker on the host). Security-conscious users can opt out.

## Alternatives Considered

- **--network=host**: Simplest networking but loses container isolation entirely.
- **Auto-connect to compose networks**: Complex detection logic, fragile, and unnecessary if services expose ports.
- **True DinD (--privileged + nested daemon)**: Heavier, more complex, storage driver issues. DooD is simpler and
  sufficient.