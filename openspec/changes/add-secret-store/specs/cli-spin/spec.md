# cli-spin Specification Delta

## ADDED Requirements

### Requirement: Secret Store CLI

The CLI SHALL provide a `spinner secret` subcommand for managing secrets in a platform-appropriate secure
store (macOS Keychain or encrypted file). Secret values SHALL never be stored in plaintext on the filesystem.

#### Scenario: Set secret with prompted input

- **WHEN** user runs `spinner secret set MY_TOKEN`
- **THEN** the CLI SHALL prompt for the value with hidden input (no echo)
- **AND** the value SHALL be stored in the secret store under the key `MY_TOKEN`

#### Scenario: Set secret with inline value

- **WHEN** user runs `spinner secret set MY_TOKEN --value abc123`
- **THEN** the CLI SHALL store `abc123` in the secret store under the key `MY_TOKEN`
- **AND** no interactive prompt SHALL be shown

#### Scenario: Set secret overwrites existing value

- **WHEN** user runs `spinner secret set MY_TOKEN` for a key that already exists
- **THEN** the CLI SHALL overwrite the existing value with the new value

#### Scenario: List stored secrets

- **WHEN** user runs `spinner secret list`
- **THEN** the CLI SHALL print the names of all stored secret keys (one per line)
- **AND** secret values SHALL NOT be displayed

#### Scenario: List secrets when store is empty

- **WHEN** user runs `spinner secret list` with no secrets stored
- **THEN** the CLI SHALL print nothing (empty output)

#### Scenario: Delete a secret

- **WHEN** user runs `spinner secret delete MY_TOKEN`
- **THEN** the CLI SHALL remove the key `MY_TOKEN` from the secret store

#### Scenario: Delete nonexistent secret

- **WHEN** user runs `spinner secret delete NONEXISTENT`
- **THEN** the CLI SHALL print an error indicating the key was not found
- **AND** the CLI SHALL exit with non-zero status

#### Scenario: Auto-detect backend on macOS

- **WHEN** the CLI runs on macOS with `/usr/bin/security` available
- **THEN** the secret store SHALL use macOS Keychain as the backend

#### Scenario: Auto-detect backend on Linux

- **WHEN** the CLI runs on a non-macOS platform
- **THEN** the secret store SHALL use the encrypted file backend (`~/.spinner/secrets.enc`)

#### Scenario: Backend override via environment variable

- **WHEN** the `SPINNER_SECRET_BACKEND` environment variable is set to `keychain` or `file`
- **THEN** the CLI SHALL use the specified backend regardless of platform detection

#### Scenario: Encrypted file backend passphrase prompt

- **WHEN** the encrypted file backend is used and `SPINNER_SECRET_PASSPHRASE` is not set
- **THEN** the CLI SHALL prompt the user for a passphrase with hidden input

#### Scenario: Encrypted file backend passphrase via environment

- **WHEN** the encrypted file backend is used and `SPINNER_SECRET_PASSPHRASE` is set
- **THEN** the CLI SHALL use the environment variable value as the passphrase without prompting

### Requirement: Secret Flag

The `spin` command SHALL accept an optional repeatable `--secret KEY` flag that references a key in the
secret store by name. The secret value SHALL be resolved at spin-time and injected into the container
environment. Secret values SHALL never appear on the command line.

#### Scenario: Single secret provided

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --secret NPM_TOKEN`
- **THEN** the CLI SHALL resolve `NPM_TOKEN` from the secret store
- **AND** the resolved value SHALL be available as an environment variable inside the instance

#### Scenario: Multiple secrets provided

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --secret NPM_TOKEN --secret API_KEY`
- **THEN** the CLI SHALL resolve both `NPM_TOKEN` and `API_KEY` from the secret store
- **AND** both values SHALL be available as environment variables inside the instance

#### Scenario: Secret not found in store

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --secret NONEXISTENT`
- **AND** `NONEXISTENT` does not exist in the secret store
- **THEN** the CLI SHALL print an error indicating the secret was not found in the store
- **AND** the CLI SHALL exit with non-zero status

#### Scenario: Reserved variable name rejected

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --secret GITHUB_TOKEN`
- **THEN** the CLI SHALL print an error: `--secret: cannot override reserved variable "GITHUB_TOKEN"`
- **AND** the CLI SHALL exit with non-zero status

#### Scenario: Secrets written to Docker env file

- **WHEN** the Docker backend creates a container with `--secret` values
- **THEN** the resolved secret values SHALL be written to the temporary env file alongside built-in and `--env` variables
- **AND** secret values SHALL NOT appear in `ps aux` output on the host

#### Scenario: Secrets written to GCP metadata

- **WHEN** the GCP backend creates a VM with `--secret` values
- **THEN** the resolved secret values SHALL be added to instance metadata with the `SPINNER_ENV_` prefix
- **AND** the runtime script SHALL export them as environment variables

#### Scenario: No secrets provided

- **WHEN** user runs `spinner spin --image <image> --repo <repo>` without any `--secret` flags
- **THEN** the CLI SHALL behave identically to the current implementation (no change in behavior)

## MODIFIED Requirements

### Requirement: GitHub Token Environment Variable

The CLI SHALL resolve `GITHUB_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` from the secret store first, then fall
back to host environment variables via `os.Getenv()`. The CLI SHALL report an error only if neither source
provides a value. Token values SHALL NOT appear in docker command output or logs.

#### Scenario: Token resolved from secret store

- **WHEN** `GITHUB_TOKEN` exists in the secret store
- **AND** the `GITHUB_TOKEN` environment variable is not set on the host
- **THEN** the CLI SHALL use the value from the secret store

#### Scenario: Token resolved from environment variable fallback

- **WHEN** `GITHUB_TOKEN` does not exist in the secret store
- **AND** the `GITHUB_TOKEN` environment variable is set on the host
- **THEN** the CLI SHALL use the value from the environment variable

#### Scenario: Store value takes precedence over environment variable

- **WHEN** `GITHUB_TOKEN` exists in both the secret store and as a host environment variable
- **THEN** the CLI SHALL use the value from the secret store

#### Scenario: Token not found in store or environment

- **WHEN** `GITHUB_TOKEN` does not exist in the secret store
- **AND** the `GITHUB_TOKEN` environment variable is not set on the host
- **THEN** the CLI SHALL print an error message indicating the token is missing and suggest `spinner secret set GITHUB_TOKEN`
- **AND** the CLI SHALL exit with non-zero status

#### Scenario: Token is not exposed in logs

- **WHEN** the CLI executes docker commands
- **THEN** the token value SHALL NOT appear in docker command output or logs

#### Scenario: Both tokens resolved via same mechanism

- **WHEN** the CLI resolves tokens for a spin operation
- **THEN** both `GITHUB_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` SHALL follow the same resolution order (store first, env fallback)

### Requirement: Custom Environment Variables

The spin command SHALL accept repeatable `--env KEY=VALUE` flags for injecting custom environment variables into the
runtime environment. The same reserved variable list SHALL apply to both `--env` and `--secret` flags.

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
  PROMPT, BRANCH, MAX_ITERATIONS, LOG_DIR, STATE_DIR, SPINNER_LOG_BUCKET, SPINNER_STATE_BUCKET, SPINNER_INSTANCE_NAME)
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

#### Scenario: Secret and env flags used together

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --env CONFIG=val --secret NPM_TOKEN`
- **THEN** both the inline `CONFIG=val` and the store-resolved `NPM_TOKEN` SHALL be available as environment variables inside the instance
