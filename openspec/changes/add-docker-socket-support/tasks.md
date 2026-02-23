# Tasks: Add Docker Socket Support

## 1.0 Install Docker CLI in container image

- [ ] 1.1 Modify extending.template to install docker-ce-cli and docker-compose-plugin via Docker's official apt repo
- [ ] 1.2 Add spinner user to docker group in extending.template
- [ ] 1.3 Add unit test verifying Dockerfile template contains Docker CLI installation
- [ ] 1.4 Verify image builds successfully with `go build && ./dist/spinner setup --name test`

## 2.0 Auto-mount Docker socket on container creation

- [ ] 2.1 Add DockerSocket field to provider.CreateConfig
- [ ] 2.2 Modify buildDockerRunCommand to mount /var/run/docker.sock when enabled and socket exists on host
- [ ] 2.3 Add --add-host=host.docker.internal:host-gateway when socket is mounted
- [ ] 2.4 Set DOCKER_DEFAULT_LABELS=spinner-parent=<container-name> env var when socket is mounted
- [ ] 2.5 Add unit tests for socket mounting (enabled, disabled, socket missing)
- [ ] 2.6 Add unit tests for host.docker.internal flag
- [ ] 2.7 Add unit tests for DOCKER_DEFAULT_LABELS env var

## 3.0 Add opt-out mechanism (CLI flag + config)

- [ ] 3.1 Add --no-docker-socket flag to spin command in cmd/spin.go
- [ ] 3.2 Add docker-socket config support to .spinner.json config reading
- [ ] 3.3 Wire flag and config into CreateConfig.DockerSocket field
- [ ] 3.4 Add unit tests for flag and config precedence
- [ ] 3.5 Update spin command help text to document the flag

## 4.0 Cleanup sibling containers on destroy

- [ ] 4.1 In Docker provider Remove(), query for containers with spinner-parent=<name> label before removing
- [ ] 4.2 Remove labeled sibling containers and networks
- [ ] 4.3 Add unit tests for sibling cleanup (with and without sibling containers)
- [ ] 4.4 Add unit test verifying destroy continues if sibling cleanup fails

## 5.0 Integration test for Docker socket support

- [ ] 5.1 Add integration test that creates a spinner container with Docker socket, runs `docker ps` inside it
- [ ] 5.2 Add integration test that runs docker-compose inside spinner container and verifies sibling container creation
- [ ] 5.3 Add integration test that destroys spinner container and verifies sibling containers are cleaned up
- [ ] 5.4 Add integration test for --no-docker-socket opt-out (verify socket not mounted)

## 6.0 Documentation

- [ ] 6.1 Update docs/usage.md with Docker socket support section (networking, known limitations, opt-out)
- [ ] 6.2 Update CLAUDE.md if needed
