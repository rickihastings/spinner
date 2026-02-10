# cli-spin Specification Delta

## ADDED Requirements

### Requirement: Custom Environment Variable Injection

The CLI SHALL accept an optional repeatable `--env` flag on the `spin` command for injecting custom environment
variables into sandboxed instances at runtime. Variables SHALL be passed securely to both Docker and GCP backends
without exposing values in host process listings.

#### Scenario: Single env var provided

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --env NPM_TOKEN=abc123`
- **THEN** the CLI SHALL pass `NPM_TOKEN=abc123` as an environment variable inside the instance

#### Scenario: Multiple env vars provided

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --env NPM_TOKEN=abc --env MY_API_KEY=xyz`
- **THEN** the CLI SHALL pass both `NPM_TOKEN=abc` and `MY_API_KEY=xyz` as environment variables inside the instance

#### Scenario: Env var with equals in value

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --env "CONFIG=key=value"`
- **THEN** the CLI SHALL split on the first `=` only, setting `CONFIG` to `key=value`

#### Scenario: Env var with empty value

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --env "MY_VAR="`
- **THEN** the CLI SHALL set `MY_VAR` to an empty string inside the instance

#### Scenario: Invalid env var format (no equals sign)

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --env "INVALID"`
- **THEN** the CLI SHALL print an error: `--env: invalid format "INVALID", expected KEY=VALUE`
- **AND** the CLI SHALL exit with non-zero status

#### Scenario: Invalid env var format (empty key)

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --env "=value"`
- **THEN** the CLI SHALL print an error: `--env: key must not be empty`
- **AND** the CLI SHALL exit with non-zero status

#### Scenario: Reserved variable override rejected

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --env "GITHUB_TOKEN=override"`
- **THEN** the CLI SHALL print an error: `--env: cannot override reserved variable "GITHUB_TOKEN"`
- **AND** the CLI SHALL exit with non-zero status

#### Scenario: Env vars work with both backends

- **WHEN** user runs `spinner spin --backend gcp --image <image> --repo <repo> --env MY_KEY=val`
- **THEN** the CLI SHALL pass `MY_KEY=val` to the GCP instance via instance metadata
- **AND** the variable SHALL be available as an environment variable inside the VM

#### Scenario: No env vars provided

- **WHEN** user runs `spinner spin --image <image> --repo <repo>` without any `--env` flags
- **THEN** the CLI SHALL behave identically to the current implementation (no change in behavior)

### Requirement: Docker Env-File Secret Passing

The Docker backend SHALL use `--env-file` with a temporary file to pass environment variables to containers,
keeping secret values out of host process listings.

#### Scenario: Env vars passed via env-file

- **WHEN** the Docker backend creates a container with custom env vars
- **THEN** all environment variables (built-in and custom) SHALL be written to a temporary file
- **AND** the file SHALL be passed to `docker run` via `--env-file`
- **AND** secret values SHALL NOT appear in `ps aux` output on the host

#### Scenario: Env-file has restrictive permissions

- **WHEN** the temporary env-file is created
- **THEN** the file SHALL have `0600` permissions (owner read/write only)

#### Scenario: Env-file cleanup after container creation

- **WHEN** the container creation completes (success or failure)
- **THEN** the temporary env-file SHALL be deleted from the host filesystem

#### Scenario: Built-in env vars included in env-file

- **WHEN** the Docker backend creates an env-file
- **THEN** the file SHALL contain `GITHUB_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`, `REPO_URL`, and any other
  built-in variables alongside custom env vars

### Requirement: GCP Metadata Secret Passing

The GCP backend SHALL pass custom environment variables via instance metadata with a `SPINNER_ENV_` prefix
to avoid collisions with internal metadata keys.

#### Scenario: Custom env vars in GCP metadata

- **WHEN** the GCP backend creates a VM with custom env var `MY_KEY=val`
- **THEN** the instance metadata SHALL contain `SPINNER_ENV_MY_KEY=val`

#### Scenario: GCP runtime script exports custom env vars

- **WHEN** the GCP runtime startup script reads instance metadata
- **THEN** it SHALL read all `SPINNER_ENV_*` metadata keys
- **AND** it SHALL export them as environment variables with the prefix stripped (e.g., `MY_KEY=val`)

#### Scenario: No collision with internal metadata

- **WHEN** custom env vars are added to GCP metadata
- **THEN** they SHALL NOT overwrite internal keys like `REPO_URL`, `PROMPT`, `GITHUB_TOKEN`, etc.
