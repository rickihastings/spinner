# cli-spin Delta: add-model-flag

## ADDED Requirements

### Requirement: Model Flag

The spin command SHALL accept an optional `--model <model-name>` flag that selects which Claude model the agent uses
inside the container/VM. The model name is passed as the `ANTHROPIC_MODEL` environment variable.

#### Scenario: Model flag provided

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --model claude-sonnet-4-5-20250929`
- **THEN** the runtime environment SHALL have `ANTHROPIC_MODEL=claude-sonnet-4-5-20250929` set
- **AND** the Claude CLI inside the container SHALL use that model

#### Scenario: Model flag not provided

- **WHEN** user runs `spinner spin --image <image> --repo <repo>` without `--model`
- **THEN** the `ANTHROPIC_MODEL` environment variable SHALL NOT be set
- **AND** the Claude CLI SHALL use its default model

#### Scenario: Model from config file

- **WHEN** `.spinner.json` contains `{"model": "claude-sonnet-4-5-20250929"}`
- **AND** user runs `spinner spin --image <image> --repo <repo>` without `--model`
- **THEN** the runtime environment SHALL have `ANTHROPIC_MODEL=claude-sonnet-4-5-20250929` set

#### Scenario: CLI flag overrides config file

- **WHEN** `.spinner.json` contains `{"model": "claude-sonnet-4-5-20250929"}`
- **AND** user runs `spinner spin --image <image> --repo <repo> --model claude-opus-4-6`
- **THEN** the runtime environment SHALL have `ANTHROPIC_MODEL=claude-opus-4-6` set

#### Scenario: No model validation

- **WHEN** user provides any string as the model name (e.g., `--model my-custom-model`)
- **THEN** the CLI SHALL pass it through without validation

### Requirement: Model Override on Restart

When a stopped instance is restarted, the model SHALL be updatable. The new `--model` value takes effect on restart,
overriding the original value.

#### Scenario: Docker model override on restart

- **WHEN** a Docker instance was created with `--model claude-sonnet-4-5-20250929`
- **AND** user stops the instance and runs `spinner spin` with `--model claude-opus-4-6`
- **THEN** the CLI SHALL write the new model to the state directory override file
- **AND** the startup script SHALL read the override and export `ANTHROPIC_MODEL=claude-opus-4-6`

#### Scenario: GCP model override on restart

- **WHEN** a GCP instance was created with `--model claude-sonnet-4-5-20250929`
- **AND** user stops the instance and runs `spinner spin` with `--model claude-opus-4-6`
- **THEN** the CLI SHALL update the `ANTHROPIC_MODEL` instance metadata to `claude-opus-4-6`
- **AND** the runtime script SHALL read the updated metadata on boot

## MODIFIED Requirements

### Requirement: Custom Environment Variables

The spin command SHALL accept repeatable `--env KEY=VALUE` flags for injecting custom environment variables into the
runtime environment.

#### Scenario: Single env var provided

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --env MY_VAR=hello`
- **THEN** the runtime environment SHALL have `MY_VAR=hello` set

#### Scenario: Multiple env vars provided

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --env VAR_A=1 --env VAR_B=2`
- **THEN** the runtime environment SHALL have both `VAR_A=1` and `VAR_B=2` set

#### Scenario: Env var value containing equals sign

- **WHEN** user provides `--env DATABASE_URL=postgres://host/db?ssl=true`
- **THEN** the value SHALL be split only on the first `=`, preserving `postgres://host/db?ssl=true` as the value

#### Scenario: Reserved variable name rejected

- **WHEN** user provides `--env GITHUB_TOKEN=fake` (or any other reserved name: CLAUDE_CODE_OAUTH_TOKEN, REPO_URL,
  PROMPT, BRANCH, MAX_ITERATIONS, LOG_DIR, STATE_DIR, SPINNER_LOG_BUCKET, SPINNER_STATE_BUCKET,
  SPINNER_INSTANCE_NAME, ANTHROPIC_MODEL)
- **THEN** the CLI SHALL print an error indicating the variable is reserved and exit with non-zero status

#### Scenario: Invalid format rejected

- **WHEN** user provides `--env NO_EQUALS_SIGN` (missing `=`)
- **THEN** the CLI SHALL print an error indicating invalid format and exit with non-zero status

#### Scenario: Docker backend env var delivery

- **WHEN** `--env` vars are provided with the Docker backend
- **THEN** the CLI SHALL write them to the env-file alongside built-in variables and pass via `--env-file` to docker run

#### Scenario: GCP backend env var delivery

- **WHEN** `--env` vars are provided with the GCP backend
- **THEN** the CLI SHALL pass them as instance metadata with a `SPINNER_ENV_` prefix (e.g., `SPINNER_ENV_MY_VAR=hello`)
- **AND** the runtime script SHALL extract `SPINNER_ENV_*` metadata keys and export them without the prefix

#### Scenario: Env vars not provided

- **WHEN** user runs `spinner spin` without any `--env` flags
- **THEN** the CLI SHALL proceed normally with no additional environment variables
