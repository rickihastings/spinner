## REMOVED Requirements

### Requirement: SSH Agent Forwarding

**Reason**: SSH agent forwarding is unreliable in Docker containers. Replaced with GitHub PAT token authentication for
more robust git operations.

**Migration**: Users must set the `GITHUB_TOKEN` environment variable with a GitHub Personal Access Token instead of
running `ssh-agent`. The token should have `repo` scope for private repositories.

## ADDED Requirements

### Requirement: GitHub Token Environment Variable

The CLI SHALL require the `GITHUB_TOKEN` environment variable to be set on the host system before running the spin
command. The token SHALL be passed to the container as an environment variable for git authentication. If `GITHUB_TOKEN`
is not set, the CLI SHALL exit with error code 1 and display an error message.

#### Scenario: GITHUB_TOKEN is set

- **WHEN** the user has set `GITHUB_TOKEN` in their shell environment
- **AND** runs the spin command
- **THEN** the CLI passes the token value to the container as the `GITHUB_TOKEN` environment variable
- **AND** the container can use the token for git authentication

#### Scenario: GITHUB_TOKEN is not set

- **WHEN** the user has not set `GITHUB_TOKEN` in their shell environment
- **AND** runs the spin command
- **THEN** the CLI exits with error code 1
- **AND** displays "Error: GITHUB_TOKEN environment variable is required. Set it with a GitHub Personal Access Token."

#### Scenario: Token is not exposed in logs

- **WHEN** the spin command runs successfully
- **THEN** the token value is NOT displayed in CLI output or container logs
- **AND** the token is NOT stored in bash history (because it comes from environment variable, not CLI flag)

### Requirement: GitHub CLI Installation

Docker images created with the setup command SHALL include the GitHub CLI (`gh`) tool installed and available in the
PATH. The Dockerfile template SHALL include installation steps for `gh` CLI.

#### Scenario: gh CLI is available in container

- **WHEN** a container is created from a spinner base image
- **THEN** the `gh` command is available in the container's PATH
- **AND** running `gh --version` returns a valid version number

#### Scenario: Dockerfile template includes gh installation

- **WHEN** the setup command generates a Dockerfile from the template
- **THEN** the Dockerfile includes steps to install the GitHub CLI
- **AND** the installation completes successfully during image build

### Requirement: Git Credential Configuration

The container startup script SHALL configure git to use GitHub CLI as the credential helper before cloning the
repository. The script SHALL run `gh auth login --with-token` using the `GITHUB_TOKEN` environment variable, then run
`gh auth setup-git` to configure git credential helper, and finally configure git credential cache with a 1-year
timeout.

#### Scenario: Git credential helper is configured

- **WHEN** the container starts with `GITHUB_TOKEN` set
- **THEN** the startup script runs `echo "$GITHUB_TOKEN" | gh auth login --with-token`
- **AND** the startup script runs `gh auth setup-git`
- **AND** the startup script runs `git config --global credential.helper 'cache --timeout=31536000'`
- **AND** git operations use the configured credential helper for authentication

#### Scenario: Token authentication succeeds

- **WHEN** git credential configuration completes successfully
- **AND** the container attempts to clone a private repository
- **THEN** git authentication succeeds using the GitHub token
- **AND** no additional credentials are prompted

#### Scenario: Token authentication fails

- **WHEN** the provided `GITHUB_TOKEN` is invalid or expired
- **AND** the container attempts to configure `gh auth login`
- **THEN** the authentication fails
- **AND** the startup script outputs the error message from `gh`
- **AND** the container exits with a non-zero status

### Requirement: Repository URL Format

The CLI SHALL accept both SSH and HTTPS repository URLs via the `--repo` flag. When using GitHub token authentication,
the container SHALL convert SSH URLs to HTTPS format before cloning.

#### Scenario: HTTPS URL provided

- **WHEN** user runs `spin --repo https://github.com/user/repo.git`
- **THEN** the CLI passes the URL as-is to the container
- **AND** git clone uses the HTTPS URL with token authentication

#### Scenario: SSH URL provided

- **WHEN** user runs `spin --repo git@github.com:user/repo.git`
- **THEN** the startup script converts the URL to `https://github.com/user/repo.git`
- **AND** git clone uses the HTTPS URL with token authentication

#### Scenario: URL conversion handles edge cases

- **WHEN** the repository URL is in SSH format like `git@github.com:org/repo.git`
- **THEN** the startup script correctly converts it to `https://github.com/org/repo.git`
- **AND** preserves organization/user and repository name correctly
