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
secret store by name. The secret value SHALL be resolved at spin-time and delivered to the container
via an encrypted blob. Secret values SHALL never appear on the command line or as container environment
variables.

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
- **AND** the CLI SHALL suggest running `spinner secret set NONEXISTENT`
- **AND** the CLI SHALL exit with non-zero status

#### Scenario: No secrets provided

- **WHEN** user runs `spinner spin --image <image> --repo <repo>` without any `--secret` flags
- **THEN** built-in tokens (`GITHUB_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`) SHALL still be resolved from the store and included in the blob

### Requirement: Encrypted Blob Delivery

All secrets (built-in tokens and custom `--secret` values) SHALL be delivered to containers as an
encrypted blob file, NOT as environment variables. The decryption key SHALL be delivered via a
file-based side channel — never as an environment variable or instance metadata value.

#### Scenario: Blob generated at spin-time

- **WHEN** the `spin` command resolves secrets (built-in tokens + custom keys)
- **THEN** the CLI SHALL generate a random 32-byte AES-256-GCM key
- **AND** the CLI SHALL encrypt ALL resolved secrets into a single blob using the random key
- **AND** the blob SHALL use AES-256-GCM authenticated encryption (no Argon2id — random key is full entropy)

#### Scenario: Docker blob delivery

- **WHEN** the Docker backend creates a container
- **THEN** the CLI SHALL write the encrypted blob to `~/.spinner/<container>/secrets.enc` on the host
- **AND** the CLI SHALL write the ephemeral key to `~/.spinner/<container>/secrets.key` on the host
- **AND** both files SHALL be mounted read-only into the container at `/run/spinner/secrets.enc` and `/run/spinner/secrets.key`
- **AND** no passphrase or key SHALL appear in environment variables, env-file, or `docker inspect` output
- **AND** secret values SHALL NOT appear in `ps aux` output, container environment, or Docker env-file

#### Scenario: GCP blob delivery

- **WHEN** the GCP backend creates a VM
- **THEN** the CLI SHALL base64-encode the encrypted blob and pass it as instance metadata key `SPINNER_SECRET_BLOB`
- **AND** the CLI SHALL upload the ephemeral key to GCS at `gs://<state-bucket>/<instance>/secrets.key`
- **AND** the ephemeral key SHALL NOT be stored in instance metadata
- **AND** the startup script SHALL decode blob from metadata, write to `/run/spinner/secrets.enc`
- **AND** the startup script SHALL fetch the key from GCS, write to `/run/spinner/secrets.key`
- **AND** raw secret values SHALL NOT appear in instance metadata or VM environment variables

#### Scenario: GCP key survives VM restart

- **WHEN** a stopped GCP VM is started via `Provider.Start()`
- **THEN** the key SHALL persist in GCS and be re-readable by the startup script on next boot
- **AND** `Start()` SHALL re-upload the key to GCS if a new `SecretKey` is provided in `CreateConfig`

#### Scenario: Blob and key retained on disk for inception

- **WHEN** `spinner exec` or `startup.sh` decrypts the blob
- **THEN** the blob file at `/run/spinner/secrets.enc` SHALL NOT be deleted
- **AND** the key file at `/run/spinner/secrets.key` SHALL NOT be deleted
- **AND** both SHALL remain available for inception scenarios (inner `spinner spin` commands)

### Requirement: Startup Script Secret Integration

The container startup script SHALL use `spinner secret inject` to decrypt secrets for token-dependent
operations instead of reading tokens from environment variables. After initial git authentication,
the `gh` credential cache SHALL persist credentials so token env vars are no longer needed.

#### Scenario: Startup uses inject for git auth

- **WHEN** the container starts
- **THEN** `startup.sh` SHALL run `spinner secret inject -- sh -c '...'` to decrypt secrets
- **AND** `spinner secret inject` SHALL read the key from `/run/spinner/secrets.key`
- **AND** the injected shell SHALL run `gh auth setup-git` and `git clone`
- **AND** git credential cache SHALL be configured with a 1-year timeout

#### Scenario: Credentials persist after inject exits

- **WHEN** the `spinner secret inject` wrapper in startup.sh completes
- **THEN** git credentials SHALL remain cached via the credential helper
- **AND** subsequent `git push`, `git fetch` operations SHALL work without `GITHUB_TOKEN` in the environment

#### Scenario: Startup without secrets blob or key

- **WHEN** the container starts and `/run/spinner/secrets.enc` or `/run/spinner/secrets.key` does NOT exist
- **THEN** `startup.sh` SHALL print an error indicating secrets are required
- **AND** the container SHALL exit with non-zero status

### Requirement: Spinner Exec Secret Injection

When running in `--prompt` mode, `spinner exec` SHALL decrypt the secrets blob at startup, hold secrets
in memory, and inject them into Claude CLI child processes. The blob SHALL be retained on disk and
the passphrase SHALL be forwarded to child processes to enable inception.

#### Scenario: Exec decrypts blob at startup

- **WHEN** `spinner exec` starts and `/run/spinner/secrets.enc` exists
- **AND** `/run/spinner/secrets.key` exists
- **THEN** `spinner exec` SHALL read the key file and decrypt the blob into memory

#### Scenario: Exec injects secrets and inception env into child processes

- **WHEN** `spinner exec` spawns a Claude CLI child process
- **AND** decrypted secrets are held in memory
- **THEN** `spinner exec` SHALL inject the secrets as environment variables on the child process via `cmd.Env`
- **AND** `spinner exec` SHALL inject `SPINNER_SECRET_STORE=/run/spinner/secrets.enc` into the child process env
- **AND** `spinner exec` SHALL inject `SPINNER_SECRET_KEY=/run/spinner/secrets.key` into the child process env
- **AND** the secrets SHALL NOT be added to the container's global environment

#### Scenario: Exec without blob continues normally

- **WHEN** `spinner exec` starts and `/run/spinner/secrets.enc` does NOT exist
- **THEN** `spinner exec` SHALL continue normally without secret injection (backward compatible)

#### Scenario: Exec with corrupted or undecryptable blob

- **WHEN** `spinner exec` starts and the blob exists but cannot be decrypted (missing key file or wrong key)
- **THEN** `spinner exec` SHALL log a warning and continue without secret injection
- **AND** the iteration loop SHALL NOT be blocked

### Requirement: Secret Inject Command

The CLI SHALL provide a `spinner secret inject -- <command>` subcommand for decrypting the secrets blob
and running a command with secrets injected as environment variables. This is the primary mechanism for
accessing secrets in no-`--prompt` mode (user SSH) and is also used by `startup.sh`.

#### Scenario: Inject with key file

- **WHEN** user runs `spinner secret inject -- <command>`
- **AND** `/run/spinner/secrets.enc` exists
- **AND** `/run/spinner/secrets.key` exists (or `SPINNER_SECRET_KEY` points to a key file)
- **THEN** the CLI SHALL read the key from the file
- **AND** the CLI SHALL decrypt the blob and run the specified command with secrets as environment variables

#### Scenario: Inject with passphrase fallback

- **WHEN** user runs `spinner secret inject -- <command>`
- **AND** `/run/spinner/secrets.enc` exists
- **AND** no key file is found at `/run/spinner/secrets.key` or `SPINNER_SECRET_KEY`
- **THEN** the CLI SHALL fall back to prompting for a passphrase with hidden input (backward compat)
- **AND** the CLI SHALL decrypt the blob using Argon2id key derivation and run the specified command
- **AND** secrets SHALL only exist in the child command's process tree

#### Scenario: Inject into subshell

- **WHEN** user runs `spinner secret inject -- bash`
- **THEN** the CLI SHALL start a bash subshell with secrets available as environment variables
- **AND** processes started from that subshell SHALL inherit the secrets

#### Scenario: Inject with wrong key or passphrase

- **WHEN** user runs `spinner secret inject -- <command>` with an incorrect key file or passphrase
- **THEN** the CLI SHALL print an error indicating authentication failed
- **AND** the CLI SHALL exit with non-zero status without running the command

#### Scenario: Inject without blob file

- **WHEN** user runs `spinner secret inject -- <command>` and `/run/spinner/secrets.enc` does NOT exist
- **THEN** the CLI SHALL print an error indicating no secrets blob was found
- **AND** the CLI SHALL exit with non-zero status

#### Scenario: Inject without command argument

- **WHEN** user runs `spinner secret inject` without `-- <command>`
- **THEN** the CLI SHALL print usage instructions and exit with non-zero status

#### Scenario: Inject enables inception

- **WHEN** user runs `spinner secret inject -- sh -c 'SPINNER_SECRET_STORE=/run/spinner/secrets.enc SPINNER_SECRET_KEY=/run/spinner/secrets.key spinner spin --backend docker --secret NPM_TOKEN ...'`
- **THEN** the inner `spinner spin` SHALL resolve `NPM_TOKEN` from the outer blob using the outer key
- **AND** the inner `spinner spin` SHALL generate a new random key and blob for the inner container

### Requirement: Configurable Secret Store Path

The CLI SHALL support `SPINNER_SECRET_STORE` and `SPINNER_SECRET_KEY` environment variables to
override the default store file path (`~/.spinner/secrets.enc`) and key file path
(`/run/spinner/secrets.key`). This enables inception scenarios where an encrypted blob from an
outer Spinner layer serves as the store for an inner layer.

#### Scenario: Custom store path via environment variable

- **WHEN** `SPINNER_SECRET_STORE=/run/spinner/secrets.enc` is set
- **AND** user runs `spinner spin --secret NPM_TOKEN ...`
- **THEN** the CLI SHALL resolve `NPM_TOKEN` from `/run/spinner/secrets.enc` instead of `~/.spinner/secrets.enc`

#### Scenario: Custom key path via environment variable

- **WHEN** `SPINNER_SECRET_KEY=/run/spinner/secrets.key` is set
- **AND** user runs `spinner spin --secret NPM_TOKEN ...`
- **THEN** the CLI SHALL use the key at `/run/spinner/secrets.key` to decrypt the store at `SPINNER_SECRET_STORE`

#### Scenario: Default store path when env var not set

- **WHEN** `SPINNER_SECRET_STORE` is not set
- **THEN** the CLI SHALL use `~/.spinner/secrets.enc` as the store path

#### Scenario: Default key path when env var not set

- **WHEN** `SPINNER_SECRET_KEY` is not set
- **THEN** the CLI SHALL use `/run/spinner/secrets.key` as the key file path for blob decryption

#### Scenario: Inception — spinner inside spinner

- **WHEN** an outer Spinner creates a VM/container with secrets
- **AND** the user or agent runs an inner `spinner spin --secret ...`
- **AND** `SPINNER_SECRET_STORE` points to the outer blob at `/run/spinner/secrets.enc`
- **AND** `SPINNER_SECRET_KEY` points to the outer key at `/run/spinner/secrets.key`
- **THEN** the inner Spinner SHALL resolve secrets from the outer blob using the outer key
- **AND** the inner Spinner SHALL generate a new random key and encrypted blob for the inner container

#### Scenario: Store path for secret CLI commands

- **WHEN** `SPINNER_SECRET_STORE` is set
- **AND** user runs `spinner secret list` or `spinner secret set`
- **THEN** the CLI SHALL operate on the file at the custom path

## MODIFIED Requirements

### Requirement: GitHub Token Environment Variable

The CLI SHALL resolve `GITHUB_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` from the encrypted secret store
only. **Environment variable fallback is removed.** Both tokens MUST be stored in the secret store
via `spinner secret set` before `spinner spin` will work. Token values SHALL NOT appear in container
environment variables, docker command output, or logs.

#### Scenario: Token resolved from secret store

- **WHEN** `GITHUB_TOKEN` exists in the secret store
- **THEN** the CLI SHALL use the value from the secret store
- **AND** the value SHALL be included in the encrypted blob

#### Scenario: Token not found in store

- **WHEN** `GITHUB_TOKEN` does not exist in the secret store
- **THEN** the CLI SHALL print an error message: `secret not found: GITHUB_TOKEN — run "spinner secret set GITHUB_TOKEN"`
- **AND** the CLI SHALL exit with non-zero status
- **AND** the CLI SHALL NOT fall back to the `GITHUB_TOKEN` environment variable

#### Scenario: Token is not exposed in container env

- **WHEN** the CLI creates a container or VM
- **THEN** `GITHUB_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` SHALL NOT appear as container environment variables
- **AND** they SHALL only be accessible via blob decryption (`spinner exec` or `spinner secret inject`)

#### Scenario: Token is not exposed in logs

- **WHEN** the CLI executes docker commands
- **THEN** the token value SHALL NOT appear in docker command output or logs

#### Scenario: Both tokens resolved via same mechanism

- **WHEN** the CLI resolves tokens for a spin operation
- **THEN** both `GITHUB_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` SHALL be resolved from the store only (no env fallback)
- **AND** both SHALL be included in the encrypted blob alongside any custom `--secret` values

### Requirement: Custom Environment Variables

The spin command SHALL accept repeatable `--env KEY=VALUE` flags for injecting custom environment variables into the
runtime environment. `--env` SHALL NOT be used for sensitive values — use `--secret` instead.

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
  PROMPT, BRANCH, MAX_ITERATIONS, LOG_DIR, STATE_DIR, SPINNER_LOG_BUCKET, SPINNER_STATE_BUCKET, SPINNER_INSTANCE_NAME,
  SPINNER_SECRET_PASSPHRASE, SPINNER_SECRET_KEY)
- **THEN** the CLI SHALL print an error indicating the variable is reserved and exit with non-zero status

#### Scenario: Invalid format rejected

- **WHEN** user provides `--env NO_EQUALS_SIGN` (missing `=`)
- **THEN** the CLI SHALL print an error indicating invalid format and exit with non-zero status

#### Scenario: Docker backend env var delivery

- **WHEN** `--env` vars are provided with the Docker backend
- **THEN** the CLI SHALL write them to the env-file alongside non-secret built-in variables and pass via `--env-file` to docker run

#### Scenario: GCP backend env var delivery

- **WHEN** `--env` vars are provided with the GCP backend
- **THEN** the CLI SHALL pass them as instance metadata with a `SPINNER_ENV_` prefix (e.g., `SPINNER_ENV_MY_VAR=hello`)
- **AND** the runtime script SHALL extract `SPINNER_ENV_*` metadata keys and export them without the prefix

#### Scenario: Env vars not provided

- **WHEN** user runs `spinner spin` without any `--env` flags
- **THEN** the CLI SHALL proceed normally with no additional environment variables

#### Scenario: Secret and env flags used together

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --env CONFIG=val --secret NPM_TOKEN`
- **THEN** `CONFIG=val` SHALL be delivered as a container environment variable (via env-file or metadata)
- **AND** `NPM_TOKEN` SHALL be delivered via the encrypted blob (accessible only via `spinner exec` or `spinner secret inject`)
