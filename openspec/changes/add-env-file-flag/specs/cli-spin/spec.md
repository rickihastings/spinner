# cli-spin Specification

## MODIFIED Requirements

### Requirement: Cobra Flag Parsing for Spin

The spin command SHALL use Cobra for flag definition and validation.

#### Scenario: Flag registration

- **WHEN** the spin command initializes
- **THEN** all flags (--image, --repo, --prompt, --branch, --max-iterations, --recreate, --env, --env-file) SHALL be
  registered with Cobra

#### Scenario: Optional flag defaults

- **WHEN** user omits optional flags (--prompt, --branch, --max-iterations, --recreate, --env-file)
- **THEN** Cobra SHALL provide default values (empty string for prompt/branch/env-file, 100 for max-iterations, false
  for recreate)

## ADDED Requirements

### Requirement: Env File Flag

The spin command SHALL accept an optional `--env-file <path>` flag that places the file in the instance workspace and
makes its variables available in the runtime environment. The CLI SHALL not parse or validate the file contents.

#### Scenario: Env file provided

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --env-file .env`
- **THEN** the CLI SHALL verify the file exists and is readable
- **AND** the CLI SHALL pass the file path to the backend via `CreateConfig.EnvFile`

#### Scenario: Env file not found

- **WHEN** user provides `--env-file /nonexistent/path`
- **THEN** the CLI SHALL print an error indicating the file was not found and exit with non-zero status

#### Scenario: Env file not provided

- **WHEN** user runs `spinner spin` without the `--env-file` flag
- **THEN** the CLI SHALL behave identically to the current implementation (no change in behavior)

### Requirement: Docker Env File Delivery

The Docker backend SHALL pass the user's env file as a second `--env-file` to `docker run` and mount it into the
container for workspace placement.

#### Scenario: Env file passed to docker run

- **WHEN** `--env-file` is provided with the Docker backend
- **THEN** the Docker backend SHALL pass the user's file as a second `--env-file` argument to `docker run`
- **AND** the Docker backend SHALL mount the file read-only at `/tmp/.env` in the container

#### Scenario: Startup script copies env file to workspace

- **WHEN** the container starts and `/tmp/.env` exists
- **THEN** `startup.sh` SHALL copy the file to the workspace root after cloning the repository

#### Scenario: No env file provided

- **WHEN** `--env-file` is not provided
- **THEN** the Docker backend SHALL not add a second `--env-file` or mount
- **AND** `startup.sh` SHALL skip the copy step
