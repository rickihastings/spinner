# cli-spin Spec Delta

## ADDED Requirements

### Requirement: Provider Pass-Through Arguments for Spin

The spin command SHALL accept an optional repeatable `--provider-args` flag that passes raw arguments directly to the
underlying backend provider. Arguments are forwarded verbatim to the backend's instance creation command (`docker run`
for Docker, `gcloud compute instances create` for GCP).

#### Scenario: Single provider arg for Docker

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --provider-args="-v /data:/data"`
- **THEN** the CLI SHALL append `-v /data:/data` to the `docker run` command before the image argument

#### Scenario: Multiple provider args for Docker

- **WHEN** user runs `spinner spin --image <image> --repo <repo> --provider-args="-v /data:/data" --provider-args="--network=host"`
- **THEN** the CLI SHALL append both `-v /data:/data` and `--network=host` to the `docker run` command

#### Scenario: Provider arg for GCP

- **WHEN** user runs `spinner spin --backend gcp --image <image> --repo <repo> --project p --zone z --state-bucket b --provider-args="--hostname=my-host"`
- **THEN** the CLI SHALL append `--hostname=my-host` to the `gcloud compute instances create` command

#### Scenario: No provider args provided

- **WHEN** user runs `spinner spin --image <image> --repo <repo>` without any `--provider-args` flags
- **THEN** the CLI SHALL behave identically to the current implementation (no change in behavior)

#### Scenario: Provider args do not affect instance naming

- **WHEN** user provides `--provider-args` flags
- **THEN** the deterministic instance name SHALL be computed the same way as without `--provider-args`
- **AND** provider args SHALL NOT influence the instance name

### Requirement: Provider Args Conflict Detection

The CLI SHALL reject `--provider-args` values that conflict with arguments managed by Spinner. Each backend defines
its own set of managed arguments.

#### Scenario: Docker managed flag conflict

- **WHEN** user provides `--provider-args="--name=my-container"` with the Docker backend
- **THEN** the CLI SHALL print an error indicating `--name` is managed by Spinner and exit with non-zero status

#### Scenario: Docker detach flag conflict

- **WHEN** user provides `--provider-args="-d"` with the Docker backend
- **THEN** the CLI SHALL print an error indicating `-d` is managed by Spinner and exit with non-zero status

#### Scenario: Docker env-file flag conflict

- **WHEN** user provides `--provider-args="--env-file=my-env"` with the Docker backend
- **THEN** the CLI SHALL print an error indicating `--env-file` is managed by Spinner and exit with non-zero status

#### Scenario: Docker label flag conflict

- **WHEN** user provides `--provider-args="--label=foo=bar"` with the Docker backend
- **THEN** the CLI SHALL print an error indicating `--label` is managed by Spinner and exit with non-zero status

#### Scenario: Non-conflicting args pass through

- **WHEN** user provides `--provider-args="-v /data:/data"` with the Docker backend
- **THEN** the CLI SHALL accept and forward the argument without error

### Requirement: Provider Args in Help Output

The spin command help SHALL document the `--provider-args` flag with examples for each backend.

#### Scenario: Help shows provider-args flag

- **WHEN** user runs `spinner spin --help`
- **THEN** the output SHALL include the `--provider-args` flag description and at least one example per backend

## MODIFIED Requirements

### Requirement: Cobra Flag Parsing for Spin

The spin command SHALL use Cobra for flag definition and validation.

#### Scenario: Flag registration

- **WHEN** the spin command initializes
- **THEN** all flags (--image, --repo, --prompt, --branch, --max-iterations, --recreate, --env, --env-file, --provider-args) SHALL be
  registered with Cobra

#### Scenario: Optional flag defaults

- **WHEN** user omits optional flags (--prompt, --branch, --max-iterations, --recreate, --env-file, --provider-args)
- **THEN** Cobra SHALL provide default values (empty string for prompt/branch/env-file, 100 for max-iterations, false
  for recreate, empty list for provider-args)
