# cli-spin Specification

## Purpose
TBD - created by archiving change add-spin-command. Update Purpose after archive.
## Requirements
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

### Requirement: Prompt Flag for Ralph Loop

The spin command SHALL accept a required `--prompt` flag containing the prompt string to feed to Claude in each iteration of the Ralph loop.

#### Scenario: Prompt provided with branch

- **WHEN** user runs `spin --image spinner:my-env --repo git@github.com:user/repo.git --prompt "study and implement plan for feature-x" --branch feature-x`
- **THEN** the CLI passes `PROMPT="study and implement plan for feature-x"` as an environment variable to the container
- **AND** the CLI passes `BRANCH=feature-x` as an environment variable to the container
- **AND** the container runs in Ralph loop mode on the specified branch after cloning

#### Scenario: Prompt provided without branch

- **WHEN** user runs `spin --image spinner:my-env --repo git@github.com:user/repo.git --prompt "study and implement plan for feature-x"` without `--branch`
- **THEN** the CLI passes `PROMPT="study and implement plan for feature-x"` as an environment variable to the container
- **AND** the container runs in Ralph loop mode on the default branch after cloning

#### Scenario: Prompt not provided

- **WHEN** user runs `spin --image spinner:my-env --repo git@github.com:user/repo.git` without `--prompt`
- **THEN** the container clones the repository and stays idle
- **AND** no Ralph loop is executed

### Requirement: Branch Flag

The spin command SHALL accept an optional `--branch` flag specifying which branch to checkout and work on after cloning the repository. If not provided and `--prompt` is present, the Ralph loop runs on the default branch.

#### Scenario: Branch provided

- **WHEN** user runs `spin --repo git@... --prompt "..." --branch feature-x`
- **THEN** the CLI passes `BRANCH=feature-x` as an environment variable to the container
- **AND** after cloning, the container checks out the specified branch

#### Scenario: Branch does not exist

- **WHEN** the container attempts to checkout a branch that does not exist
- **THEN** the container creates the branch from the default branch
- **AND** continues with Ralph loop execution

#### Scenario: Branch not provided but prompt provided

- **WHEN** user runs `spin --repo git@... --prompt "..."` without `--branch`
- **THEN** the CLI does not pass a `BRANCH` environment variable to the container
- **AND** after cloning, the container stays on the default branch
- **AND** the Ralph loop executes on the default branch

### Requirement: Max Iterations Flag

The spin command SHALL accept an optional `--max-iterations` flag specifying the maximum number of Ralph loop iterations before the container exits. Default is 100.

#### Scenario: Max iterations provided

- **WHEN** user runs `spin --repo git@... --prompt "..." --branch feature-x --max-iterations 50`
- **THEN** the CLI passes `MAX_ITERATIONS=50` as an environment variable to the container

#### Scenario: Max iterations not provided

- **WHEN** user runs `spin --repo git@... --prompt "..." --branch feature-x` without `--max-iterations`
- **THEN** the CLI passes `MAX_ITERATIONS=100` as an environment variable to the container

#### Scenario: Max iterations reached

- **WHEN** the Ralph loop completes 100 iterations (or the configured max)
- **AND** the `~~ FEATURE_COMPLETED ~~` signal has not been detected
- **THEN** the container outputs "Max iterations (100) reached. Exiting."
- **AND** the container exits with status 0

### Requirement: Ralph Loop Execution

The container SHALL execute a Ralph loop that continuously invokes Claude with the provided prompt until the feature is complete or max iterations is reached.

#### Scenario: Ralph loop iteration

- **WHEN** the container starts with `PROMPT` environment variable set
- **THEN** the loop pipes the prompt string to `claude --dangerously-skip-permissions`
- **AND** captures and displays the output
- **AND** increments the iteration counter
- **AND** repeats until completion signal is detected or max iterations reached

### Requirement: Feature Completion Detection

The Ralph loop SHALL monitor Claude's output for the `~~ FEATURE_COMPLETED ~~` signal to detect when all tasks are complete.

#### Scenario: Completion signal detected

- **WHEN** Claude's output contains the string `~~ FEATURE_COMPLETED ~~`
- **THEN** the loop exits
- **AND** the container outputs "Feature completed after N iterations. Exiting."
- **AND** the container exits with status 0

#### Scenario: No completion signal

- **WHEN** Claude's output does not contain `~~ FEATURE_COMPLETED ~~`
- **AND** iteration count is less than max iterations
- **THEN** the loop starts another iteration
- **AND** feeds the prompt to Claude again

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

The container SHALL use the baked-in startup script at /usr/local/bin/startup.sh as its entrypoint. The startup script handles repository cloning, branch checkout, and Ralph loop execution.

#### Scenario: Container startup sequence with branch

- **WHEN** the container starts with `PROMPT` and `BRANCH` environment variables set
- **THEN** the startup script clones the repository from `REPO_URL`
- **AND** checks out the branch specified in `BRANCH` (creating it if it doesn't exist)
- **AND** executes the Ralph loop with the prompt from `PROMPT`
- **AND** runs until `~~ FEATURE_COMPLETED ~~` is detected or `MAX_ITERATIONS` is reached

#### Scenario: Container startup sequence without branch

- **WHEN** the container starts with `PROMPT` environment variable set but `BRANCH` is not set
- **THEN** the startup script clones the repository from `REPO_URL`
- **AND** stays on the default branch
- **AND** executes the Ralph loop with the prompt from `PROMPT`
- **AND** runs until `~~ FEATURE_COMPLETED ~~` is detected or `MAX_ITERATIONS` is reached

#### Scenario: Container startup without prompt

- **WHEN** the container starts without `PROMPT` environment variable set
- **THEN** the startup script clones the repository from `REPO_URL`
- **AND** the container stays idle without executing Ralph loop

#### Scenario: Clone failure

- **WHEN** git clone fails in the startup script
- **THEN** the container outputs the git error message
- **AND** the container exits immediately with non-zero status

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

