# Design: Add Docker Socket Support

## Technical Implementation Plan

### Component Map

| File                                                          | Action | Purpose                                                                        |
|---------------------------------------------------------------|--------|--------------------------------------------------------------------------------|
| `internal/backend/docker/run.go`                              | modify | Mount Docker socket + add host.docker.internal + set DOCKER_DEFAULT_LABELS env |
| `internal/backend/docker/run_test.go`                         | modify | Tests for socket mounting, opt-out, labeling                                   |
| `internal/backend/docker/templates/docker/extending.template` | modify | Install Docker CLI + compose plugin, add spinner to docker group               |
| `internal/backend/docker/docker_provider.go`                  | modify | Pass docker-socket config through to run command builder                       |
| `internal/backend/docker/destroy.go` or `docker_provider.go`  | modify | Cleanup sibling containers on Remove()                                         |
| `internal/provider/provider.go`                               | modify | Add DockerSocket field to CreateConfig                                         |
| `cmd/spin.go`                                                 | modify | Add --no-docker-socket flag, read from config                                  |
| `cmd/destroy.go`                                              | modify | Wire sibling cleanup into destroy flow                                         |
| `test/integration/spin_test.go`                               | modify | Add integration test for Docker socket support                                 |

### Approach

#### Phase 1: Image — Docker CLI installation

Modify the extending.template Dockerfile to install:
- Docker CE CLI (`docker-ce-cli`) via Docker's official apt repository
- Docker Compose plugin (`docker-compose-plugin`)
- Add `spinner` user to `docker` group

This adds ~50MB to the image but is essential for DooD to work. The installation uses the same signed-apt-repo pattern
already used for Node.js and GitHub CLI.

```dockerfile
# Install Docker CLI and Compose plugin
RUN install -m 0755 -d /etc/apt/keyrings && \
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc && \
    chmod a+r /etc/apt/keyrings/docker.asc && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
    tee /etc/apt/sources.list.d/docker.list > /dev/null && \
    apt-get update && \
    apt-get install -y docker-ce-cli docker-compose-plugin && \
    rm -rf /var/lib/apt/lists/*

# Add spinner user to docker group for socket access
RUN groupadd -f docker && usermod -aG docker spinner
```

Note: We only install `docker-ce-cli` (the CLI), NOT `docker-ce` (the daemon). The daemon runs on the host.

#### Phase 2: Socket mounting in run.go

In `buildDockerRunCommand`, after the existing volume mounts, check:
1. Is Docker socket mounting enabled? (not opted out via config)
2. Does `/var/run/docker.sock` exist on the host?

If both true, append:
```go
dockerArgs = append(dockerArgs,
    "-v", "/var/run/docker.sock:/var/run/docker.sock",
    "--add-host=host.docker.internal:host-gateway",
)
```

The `host.docker.internal` addition ensures cross-platform consistency. On Docker Desktop (macOS/Windows) this is
already set, but on Linux it requires the `--add-host` flag.

#### Phase 3: Sibling container labeling

Set a `DOCKER_DEFAULT_LABELS` environment variable in the container:
```
DOCKER_DEFAULT_LABELS=spinner-parent=<container-name>
```

Docker Compose (v2.21+) and many Docker tools respect this variable, automatically applying the label to any containers
they create. For older Compose versions or raw `docker run` calls, the label won't be applied — but this covers the
majority of use cases.

Additionally, the `COMPOSE_PROJECT_NAME` can be set to include the spinner container name for better isolation between
multiple spinner instances running the same compose stack.

#### Phase 4: Sibling cleanup on destroy

In the Docker provider's `Remove()` method, before removing the spinner container:
1. Query: `docker ps -aq --filter label=spinner-parent=<container-name>`
2. If any results, remove them: `docker rm -f <ids>`
3. Also clean up networks: `docker network ls --filter label=spinner-parent=<container-name> -q` + `docker network rm`
4. Then proceed with normal spinner container removal

#### Phase 5: Opt-out mechanism

Add `DockerSocket` bool to `CreateConfig` (default true for Docker backend).

CLI: `--no-docker-socket` flag on spin command.
Config: `"docker-socket": false` in `.spinner.json`.

The flag takes precedence over config. Config takes precedence over default.

### Key Decisions

1. **DooD over DinD**: Docker-outside-of-Docker via socket mounting is simpler, more performant, and covers all use
   cases. True DinD (nested daemon) adds storage driver complexity and requires --privileged.

2. **host.docker.internal over --network=host**: Adding `--add-host` preserves container network isolation while
   enabling communication with sibling containers via host-mapped ports. `--network=host` would be simpler but removes
   all network isolation.

3. **DOCKER_DEFAULT_LABELS for cleanup**: This leverages Docker Compose's native label propagation rather than wrapping
   docker commands or intercepting calls. It's non-invasive and works transparently.

4. **Default-on with opt-out**: Most users who spin up dev environments will need Docker. Making it default-on removes
   the barrier to entry. Users who don't need it (or have security concerns) can opt out.

5. **Docker CLI in image by default**: Even users who don't use docker-compose may benefit from having `docker` CLI
   available for debugging. The ~50MB cost is acceptable given the image already installs Node.js, GitHub CLI, etc.

### Risks and Trade-offs

- **Security**: Docker socket access = root on host. Mitigated by: opt-out flag, same security model as running Docker
  normally, spinner is a development tool not a production runtime.
- **Image size**: +~50MB for Docker CLI + compose plugin. Acceptable given existing image size.
- **Compose networking**: Services are reachable via `host.docker.internal:<port>` but NOT by service name. This is a
  known limitation of DooD that we document clearly.
- **Port conflicts**: Multiple spinner containers running the same compose stack will have port conflicts on the host.
  This is inherent to DooD and documented as a known limitation.
- **DOCKER_DEFAULT_LABELS**: Only works with Compose v2.21+ and tools that respect the variable. Older tools won't get
  labels, and sibling containers from those tools won't be auto-cleaned.
