# cli-spin Specification Delta

## ADDED Requirements

### Requirement: Secret Store CLI

The CLI SHALL provide a `spinner secret` subcommand for managing secrets in an encrypted file store
(`~/.spinner/secrets.enc`). Secret values SHALL be encrypted at rest using AES-256-GCM with Argon2id
key derivation. Secret values SHALL never be stored in plaintext on the filesystem.

#### Scenario: Set secret with prompted input

- **WHEN** user runs `spinner secret set MY_TOKEN`
- **THEN** the CLI SHALL prompt for the value with hidden input (no echo)
- **AND** the value SHALL be stored encrypted in the secret store under the key `MY_TOKEN`

#### Scenario: Set secret with inline value

- **WHEN** user runs `spinner secret set MY_TOKEN --value abc123`
- **THEN** the CLI SHALL store `abc123` encrypted in the secret store under the key `MY_TOKEN`
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

#### Scenario: Passphrase prompt for encrypted store

- **WHEN** a secret store operation is performed and `SPINNER_SECRET_PASSPHRASE` is not set
- **THEN** the CLI SHALL prompt the user for a passphrase with hidden input

#### Scenario: Passphrase via environment variable

- **WHEN** a secret store operation is performed and `SPINNER_SECRET_PASSPHRASE` is set
- **THEN** the CLI SHALL use the environment variable value as the passphrase without prompting

#### Scenario: Wrong passphrase

- **WHEN** the user provides an incorrect passphrase for an existing encrypted store
- **THEN** the CLI SHALL print an error indicating authentication failed
- **AND** the CLI SHALL exit with non-zero status

#### Scenario: First use creates store file

- **WHEN** user runs `spinner secret set` and `~/.spinner/secrets.enc` does not exist
- **THEN** the CLI SHALL create the file with `0600` permissions
- **AND** the user SHALL be prompted to set a passphrase

### Requirement: Secret Flag

The `spin` command SHALL accept an optional repeatable `--secret KEY` flag that references a key in the
secret store by name. The secret value SHALL be resolved at spin-time and injected into the container
environment. Secret values SHALL never appear on the command line.

#### Scenario: Single secret provided

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --secret NPM_TOKEN`
- **THEN** the CLI SHALL resolve `NPM_TOKEN` from the secret store
- **AND** the resolved value SHALL be delivered to the container via an encrypted blob (not as an environment variable)

#### Scenario: Multiple secrets provided

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --secret NPM_TOKEN --secret API_KEY`
- **THEN** the CLI SHALL resolve both `NPM_TOKEN` and `API_KEY` from the secret store
- **AND** both values SHALL be delivered to the container via an encrypted blob

#### Scenario: Secret not found in store

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --secret NONEXISTENT`
- **AND** `NONEXISTENT` does not exist in the secret store
- **THEN** the CLI SHALL print an error indicating the secret was not found in the store
- **AND** the CLI SHALL exit with non-zero status

#### Scenario: Reserved variable name rejected

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --secret GITHUB_TOKEN`
- **THEN** the CLI SHALL print an error: `--secret: cannot override reserved variable "GITHUB_TOKEN"`
- **AND** the CLI SHALL exit with non-zero status

#### Scenario: No secrets provided

- **WHEN** user runs `spinner spin --image <image> --repo <repo>` without any `--secret` flags
- **THEN** the CLI SHALL behave identically to the current implementation (no change in behavior)

### Requirement: Encrypted Blob Delivery for Custom Secrets

Custom `--secret` values SHALL be delivered to containers as an encrypted blob file, NOT as environment
variables. Built-in tokens (`GITHUB_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`) SHALL continue to be delivered
as container environment variables because `startup.sh` requires them.

#### Scenario: Blob generated at spin-time

- **WHEN** the `spin` command resolves custom `--secret` values
- **THEN** the CLI SHALL encrypt the custom secret values into a blob using the user's store passphrase with a fresh Argon2id salt
- **AND** the blob SHALL use AES-256-GCM authenticated encryption

#### Scenario: Docker blob delivery

- **WHEN** the Docker backend creates a container with `--secret` values
- **THEN** the CLI SHALL write the encrypted blob to `~/.spinner/<container>/secrets.enc` on the host
- **AND** the file SHALL be mounted read-only into the container at `/run/spinner/secrets.enc`
- **AND** secret values SHALL NOT appear in `ps aux` output, container environment, or Docker env-file

#### Scenario: GCP blob delivery

- **WHEN** the GCP backend creates a VM with `--secret` values
- **THEN** the CLI SHALL base64-encode the encrypted blob and pass it as instance metadata key `SPINNER_SECRET_BLOB`
- **AND** the startup script SHALL decode the metadata value and write it to `/run/spinner/secrets.enc`
- **AND** the blob SHALL be destroyed when the VM is deleted (no orphaned files in GCS)
- **AND** raw secret values SHALL NOT appear in instance metadata or VM environment variables

#### Scenario: Blob with --prompt mode includes passphrase

- **WHEN** the `spin` command includes both `--secret` and `--prompt` flags
- **THEN** the CLI SHALL pass `SPINNER_SECRET_PASSPHRASE` as a container environment variable (Docker) or instance metadata key (GCP)
- **AND** `spinner exec` SHALL use this passphrase to decrypt the blob at startup

#### Scenario: Blob without --prompt mode excludes passphrase

- **WHEN** the `spin` command includes `--secret` but NOT `--prompt`
- **THEN** the CLI SHALL NOT pass `SPINNER_SECRET_PASSPHRASE` to the container
- **AND** the encrypted blob SHALL remain on disk, requiring interactive passphrase entry via `spinner secret inject`

#### Scenario: No blob when no custom secrets

- **WHEN** the `spin` command has no `--secret` flags
- **THEN** no encrypted blob SHALL be generated or mounted

### Requirement: Spinner Exec Secret Injection

When running in `--prompt` mode, `spinner exec` SHALL decrypt the secrets blob at startup, hold secrets
in memory, and inject them into Claude CLI child processes. After decryption, the blob and passphrase
SHALL be removed from the container's filesystem and environment.

#### Scenario: Exec decrypts blob at startup

- **WHEN** `spinner exec` starts and `/run/spinner/secrets.enc` exists
- **AND** `SPINNER_SECRET_PASSPHRASE` is set in the environment
- **THEN** `spinner exec` SHALL decrypt the blob into memory
- **AND** `spinner exec` SHALL delete `/run/spinner/secrets.enc` from disk
- **AND** `spinner exec` SHALL unset `SPINNER_SECRET_PASSPHRASE` from its own process environment

#### Scenario: Exec injects secrets into child processes

- **WHEN** `spinner exec` spawns a Claude CLI child process
- **AND** decrypted secrets are held in memory
- **THEN** `spinner exec` SHALL inject the secrets as environment variables on the child process via `cmd.Env`
- **AND** the secrets SHALL NOT be added to the container's global environment

#### Scenario: Exec without blob continues normally

- **WHEN** `spinner exec` starts and `/run/spinner/secrets.enc` does NOT exist
- **THEN** `spinner exec` SHALL continue normally without secret injection (backward compatible)

#### Scenario: Exec with corrupted or undecryptable blob

- **WHEN** `spinner exec` starts and the blob exists but cannot be decrypted
- **THEN** `spinner exec` SHALL log a warning and continue without secret injection
- **AND** the iteration loop SHALL NOT be blocked

### Requirement: Secret Inject Command

The CLI SHALL provide a `spinner secret inject -- <command>` subcommand for decrypting the secrets blob
and running a command with secrets injected as environment variables. This is the primary mechanism for
accessing custom secrets in no-`--prompt` mode (user SSH).

#### Scenario: Inject with valid passphrase

- **WHEN** user runs `spinner secret inject -- claude -p "implement feature"`
- **AND** `/run/spinner/secrets.enc` exists
- **THEN** the CLI SHALL prompt for the passphrase with hidden input
- **AND** the CLI SHALL decrypt the blob and run the specified command with secrets as environment variables
- **AND** secrets SHALL only exist in the child command's process tree

#### Scenario: Inject into subshell

- **WHEN** user runs `spinner secret inject -- bash`
- **THEN** the CLI SHALL start a bash subshell with secrets available as environment variables
- **AND** processes started from that subshell SHALL inherit the secrets

#### Scenario: Inject with wrong passphrase

- **WHEN** user runs `spinner secret inject -- <command>` with an incorrect passphrase
- **THEN** the CLI SHALL print an error indicating authentication failed
- **AND** the CLI SHALL exit with non-zero status without running the command

#### Scenario: Inject without blob file

- **WHEN** user runs `spinner secret inject -- <command>` and `/run/spinner/secrets.enc` does NOT exist
- **THEN** the CLI SHALL print an error indicating no secrets blob was found
- **AND** the CLI SHALL exit with non-zero status

#### Scenario: Inject without command argument

- **WHEN** user runs `spinner secret inject` without `-- <command>`
- **THEN** the CLI SHALL print usage instructions and exit with non-zero status

#### Scenario: Unattended agent cannot access secrets

- **WHEN** an agent is started inside the container via `docker exec` or a separate SSH session
- **AND** the agent does not use `spinner secret inject`
- **THEN** the agent SHALL NOT have access to custom `--secret` values
- **AND** the agent SHALL have access to built-in tokens (`GITHUB_TOKEN`) via container environment variables

### Requirement: Configurable Secret Store Path

The CLI SHALL support a `SPINNER_SECRET_STORE` environment variable to override the default store file
path (`~/.spinner/secrets.enc`). This enables inception scenarios where an encrypted blob from an outer
Spinner layer serves as the store for an inner layer.

#### Scenario: Custom store path via environment variable

- **WHEN** `SPINNER_SECRET_STORE=/run/spinner/secrets.enc` is set
- **AND** user runs `spinner spin --secret NPM_TOKEN ...`
- **THEN** the CLI SHALL resolve `NPM_TOKEN` from `/run/spinner/secrets.enc` instead of `~/.spinner/secrets.enc`

#### Scenario: Default store path when env var not set

- **WHEN** `SPINNER_SECRET_STORE` is not set
- **THEN** the CLI SHALL use `~/.spinner/secrets.enc` as the store path

#### Scenario: Inception — spinner inside spinner

- **WHEN** an outer Spinner creates a VM/container with `--secret` flags
- **AND** the user SSHs into the instance and runs an inner `spinner spin --secret ...`
- **AND** `SPINNER_SECRET_STORE` points to the outer blob at `/run/spinner/secrets.enc`
- **THEN** the inner Spinner SHALL resolve secrets from the outer blob (same encryption format)
- **AND** the inner Spinner SHALL generate a new encrypted blob for the inner container

#### Scenario: Store path for secret CLI commands

- **WHEN** `SPINNER_SECRET_STORE` is set
- **AND** user runs `spinner secret list` or `spinner secret set`
- **THEN** the CLI SHALL operate on the file at the custom path

## MODIFIED Requirements

### Requirement: GitHub Token Environment Variable

The CLI SHALL resolve `GITHUB_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` from the encrypted secret store first,
then fall back to host environment variables via `os.Getenv()`. The CLI SHALL report an error only if
neither source provides a value. Token values SHALL NOT appear in docker command output or logs.

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
