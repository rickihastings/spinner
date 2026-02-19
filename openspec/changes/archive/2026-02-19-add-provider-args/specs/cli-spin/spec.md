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

#### Scenario: Provider args from config file

- **WHEN** `.spinner.json` contains `{"provider-args": ["--machine-type=e2-standard-2", "--disk-size-gb=30"]}`
- **AND** user runs `spinner spin --image <image> --repo <repo>`
- **THEN** the CLI SHALL forward `--machine-type=e2-standard-2` and `--disk-size-gb=30` to the backend

#### Scenario: CLI provider args appended to config file args

- **WHEN** `.spinner.json` contains `{"provider-args": ["--machine-type=e2-standard-2"]}`
- **AND** user runs `spinner spin --image <image> --repo <repo> --provider-args="--network=host"`
- **THEN** the CLI SHALL forward both `--machine-type=e2-standard-2` and `--network=host` to the backend

#### Scenario: CLI provider args override config file via last-wins

- **WHEN** `.spinner.json` contains `{"provider-args": ["--machine-type=e2-standard-2"]}`
- **AND** user runs `spinner spin --provider-args="--machine-type=n2-standard-4"`
- **THEN** the backend SHALL receive both args, with `--machine-type=n2-standard-4` appearing last
- **AND** the backend tool's last-wins semantics SHALL cause `n2-standard-4` to take effect

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

### Requirement: Removed Backend-Specific Spin Flags

The spin command SHALL NOT accept the following backend-specific flags, which have been replaced by `--provider-args`:
`--machine-type`, `--disk-size`, `--service-account`, `--base-image`, `--dockerfile`.

#### Scenario: Removed flag produces error

- **WHEN** user provides `--machine-type=n2-standard-4`
- **THEN** the CLI SHALL print an unknown flag error and exit with non-zero status

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

### Requirement: Configuration File Support

The spin command SHALL read infrastructure defaults from a `.spinner.json` file discovered by searching from the current working directory upward through ancestor directories, with a fallback to `$HOME/.spinner.json`.

#### Scenario: Config file in current directory

- **WHEN** `.spinner.json` exists in the current working directory
- **THEN** the CLI SHALL load that file as the configuration source

#### Scenario: Config file in ancestor directory

- **WHEN** no `.spinner.json` exists in the current working directory
- **AND** a `.spinner.json` exists in an ancestor directory (e.g., `$HOME/.spinner.json` when cwd is `$HOME/projects/repo`)
- **THEN** the CLI SHALL traverse upward from cwd and load the nearest `.spinner.json` found

#### Scenario: Config file in home directory as fallback

- **WHEN** no `.spinner.json` exists in the current directory or any ancestor directory
- **AND** `$HOME/.spinner.json` exists
- **THEN** the CLI SHALL load `$HOME/.spinner.json` as a fallback

#### Scenario: First config file wins (no merging)

- **WHEN** `.spinner.json` exists in both the current directory and `$HOME`
- **THEN** the CLI SHALL load only the nearest file (current directory)
- **AND** the home directory file SHALL be ignored entirely (no merging)

#### Scenario: Config file provides full GCP config

- **WHEN** `.spinner.json` contains `{"backend": "gcp", "project": "p", "zone": "z", "state-bucket": "b"}`
- **AND** user runs `spinner spin --image my-env --repo <url> --prompt "Fix bug"`
- **THEN** the CLI SHALL use the GCP backend with project, zone, and state-bucket from the config file

#### Scenario: CLI flags override config file

- **WHEN** `.spinner.json` contains `{"provider-args": ["--machine-type=e2-standard-2"]}`
- **AND** user runs `spinner spin --backend gcp --image my-env --repo <url> --provider-args="--machine-type=n2-standard-4"`
- **THEN** both args SHALL be forwarded, with `n2-standard-4` appearing last (CLI appended after config)

#### Scenario: No config file present

- **WHEN** no `.spinner.json` exists in the current directory, any ancestor directory, or `$HOME`
- **THEN** the CLI SHALL continue normally using CLI flags, env vars, and defaults

#### Scenario: Config file with provider-args

- **WHEN** `.spinner.json` contains `{"provider-args": ["-v /data:/data", "--network=host"]}`
- **AND** user runs `spinner spin --image <image> --repo <repo>`
- **THEN** the CLI SHALL forward `-v /data:/data` and `--network=host` to the backend
