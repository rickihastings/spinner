## ADDED Requirements

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

## MODIFIED Requirements

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
