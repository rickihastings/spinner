## ADDED Requirements

### Requirement: Spin Command Flags

The spin command SHALL accept the following required CLI flags: --image (base Docker image name) and --repo (git SSH
clone URL). The CLI SHALL validate that both flags are provided before proceeding.

#### Scenario: All required flags provided

- **WHEN** user runs `spin --image spinner:my-env --repo git@github.com:octocat/Hello-World.git`
- **THEN** the CLI proceeds with container creation

#### Scenario: Missing image flag

- **WHEN** user runs `spin --repo git@github.com:octocat/Hello-World.git` without --image
- **THEN** the CLI exits with error code 1 and displays "Error: --image flag is required"

#### Scenario: Missing repo flag

- **WHEN** user runs `spin --image spinner:my-env` without --repo
- **THEN** the CLI exits with error code 1 and displays "Error: --repo flag is required"

#### Scenario: Invalid image name

- **WHEN** user provides an --image that does not exist locally
- **THEN** the CLI exits with error code 1 and displays "Error: Docker image '<image-name>' not found"

### Requirement: SSH Agent Forwarding

The container SHALL have access to the host's SSH agent for git authentication. The CLI SHALL mount the SSH agent socket
from the host system (typically $SSH_AUTH_SOCK) into the container at /ssh-agent. The container environment SHALL have
SSH_AUTH_SOCK set to /ssh-agent.

#### Scenario: SSH agent available on host

- **WHEN** SSH_AUTH_SOCK is set on the host system
- **THEN** the container can authenticate git operations using the host's SSH agent
- **AND** git clone succeeds without prompting for credentials

#### Scenario: SSH agent not running

- **WHEN** SSH_AUTH_SOCK is not set or the socket does not exist
- **THEN** the CLI exits with error code 1 and displays "Error: SSH agent not running. Start ssh-agent and add your
  key."

#### Scenario: Git authentication via SSH agent

- **WHEN** container attempts to clone a private repository
- **THEN** the SSH agent forwarding allows authentication without copying keys
- **AND** no SSH keys are stored in the container filesystem

### Requirement: NPM Configuration Mount

The CLI SHALL mount the user's .npmrc file from ~/.npmrc to /root/.npmrc in the container. If ~/.npmrc does not exist,
the CLI SHALL proceed without mounting and display a warning message.

#### Scenario: npmrc file exists

- **WHEN** ~/.npmrc exists on the host system
- **THEN** the file is mounted at /root/.npmrc in the container
- **AND** npm commands in the container use the host's registry configuration

#### Scenario: npmrc file missing

- **WHEN** ~/.npmrc does not exist on the host system
- **THEN** the CLI displays "Warning: ~/.npmrc not found, npm will use default registry"
- **AND** the container creation proceeds without the mount

### Requirement: Repository Cloning

The container SHALL automatically clone the repository specified by --repo into /workspace during startup. The CLI SHALL
pass the repository URL to the container via the REPO_URL environment variable. The container's startup script (baked into
the image at /usr/local/bin/startup.sh) SHALL handle the cloning, verification, and initialization. If the clone fails, the
container SHALL exit with a non-zero status.

#### Scenario: Successful repository clone

- **WHEN** the container starts with a valid --repo URL passed as REPO_URL environment variable
- **THEN** the repository is cloned into /workspace by the startup script
- **AND** the startup script runs `git status` to verify the clone
- **AND** the startup script outputs "hello world" to confirm successful initialization
- **AND** the container remains running after clone completes

#### Scenario: Clone failure due to authentication

- **WHEN** SSH agent cannot authenticate to the repository
- **THEN** git clone fails in the startup script
- **AND** the container exits with a non-zero status
- **AND** the CLI displays the git error message

#### Scenario: Clone failure due to invalid URL

- **WHEN** the --repo URL is malformed or repository does not exist
- **THEN** git clone fails in the startup script
- **AND** the container exits with a non-zero status
- **AND** the CLI displays the git error message

### Requirement: Persistent Container

The container SHALL run in detached mode as a persistent background process. The container SHALL NOT be automatically
removed when it exits (no --rm flag). The CLI SHALL assign a unique name to the container based on the repository name
and a timestamp or random suffix.

#### Scenario: Container runs in background

- **WHEN** the spin command completes successfully
- **THEN** the container is running in detached mode
- **AND** the CLI returns to the user's shell prompt immediately

#### Scenario: Container naming

- **WHEN** user spins up a container with --repo git@github.com:user/my-project.git
- **THEN** the container is named like "my-project-<suffix>" where suffix is a timestamp or random value
- **AND** the container name is displayed to the user

#### Scenario: Container persists after exit

- **WHEN** the cloned repository setup completes
- **THEN** the container continues running
- **AND** the container can be restarted with `docker start <container-name>`

#### Scenario: User can exec into container

- **WHEN** the container is running
- **THEN** user can exec into it with `docker exec -it <container-name> bash`
- **AND** the working directory is /workspace

### Requirement: Container Lifecycle Management

The user SHALL be responsible for stopping and removing containers created by the spin command. The CLI SHALL display
instructions on how to manage the container after creation.

#### Scenario: Display management instructions

- **WHEN** the spin command successfully creates a container
- **THEN** the CLI displays the container name
- **AND** the CLI displays "To access: docker exec -it <container-name> bash"
- **AND** the CLI displays "To stop: docker stop <container-name>"
- **AND** the CLI displays "To remove: docker rm <container-name>"

#### Scenario: Container cleanup

- **WHEN** user runs `docker stop <container-name>` then `docker rm <container-name>`
- **THEN** the container and its filesystem are removed
- **AND** the cloned repository data is lost (unless separately backed up)

### Requirement: Container Startup Command

The container SHALL use the baked-in startup script at /usr/local/bin/startup.sh as its entrypoint. The CLI SHALL configure
the container to execute this script, which handles repository cloning, verification, and keeps the container running with
`tail -f /dev/null` after successful initialization.

#### Scenario: Container keeps running after clone

- **WHEN** git clone completes successfully in the startup script
- **THEN** the startup script executes `tail -f /dev/null`
- **AND** the container status is "Up" when checked with `docker ps`

#### Scenario: Container exits after clone failure

- **WHEN** git clone fails in the startup script
- **THEN** the startup script does NOT execute `tail -f /dev/null`
- **AND** the container exits immediately
- **AND** the container status shows as "Exited" with non-zero exit code
