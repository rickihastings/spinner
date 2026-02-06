# cli-setup Specification (Delta)

## ADDED Requirements

### Requirement: Backend Selection for Setup

The setup command SHALL accept a `--backend` flag to select which provider handles the setup operation.

#### Scenario: Default backend

- **WHEN** user runs `spinner setup --name my-env` without `--backend` flag
- **THEN** the CLI SHALL use the `docker` backend (backward compatible)

#### Scenario: GCP backend selected

- **WHEN** user runs `spinner setup --backend gcp --name my-env --project my-project --zone us-central1-a`
- **THEN** the CLI SHALL use the GCP provider to bake a custom Compute Engine image

#### Scenario: Unknown backend

- **WHEN** user provides an unknown backend name (e.g., `--backend kubernetes`)
- **THEN** the CLI SHALL print an error listing available backends and exit with non-zero status

#### Scenario: Backend flag via environment variable

- **WHEN** `SPINNER_BACKEND` environment variable is set
- **THEN** the CLI SHALL use that value as the default backend
- **AND** the `--backend` flag SHALL take precedence over the environment variable

### Requirement: GCP-Specific Setup Flags

The setup command SHALL accept GCP-specific flags when `--backend gcp` is selected.

#### Scenario: Required project flag for GCP

- **WHEN** user runs `spinner setup --backend gcp --name my-env` without `--project`
- **THEN** the CLI SHALL print an error indicating `--project` is required for GCP backend

#### Scenario: Required zone flag for GCP

- **WHEN** user runs `spinner setup --backend gcp --name my-env --project p` without `--zone`
- **THEN** the CLI SHALL print an error indicating `--zone` is required for GCP backend

#### Scenario: Optional machine-type flag

- **WHEN** user runs `spinner setup --backend gcp --name my-env --project p --zone z --machine-type e2-standard-4`
- **THEN** the CLI SHALL pass the machine type to the GCP provider for the bake VM

#### Scenario: Default machine-type

- **WHEN** user does not provide `--machine-type` with GCP backend
- **THEN** the CLI SHALL default to `e2-standard-2`

#### Scenario: Optional disk-size flag

- **WHEN** user runs `spinner setup --backend gcp --name my-env --project p --zone z --disk-size 50`
- **THEN** the CLI SHALL pass the disk size (in GB) to the GCP provider for the bake VM

#### Scenario: Default disk-size

- **WHEN** user does not provide `--disk-size` with GCP backend
- **THEN** the CLI SHALL default to 30 GB

#### Scenario: Required state-bucket flag for GCP

- **WHEN** user runs `spinner setup --backend gcp --name my-env --project p --zone z` without `--state-bucket`
- **THEN** the CLI SHALL print an error indicating `--state-bucket` is required for GCP backend
- **AND** explain that GCS bucket names are globally unique and must be pre-created

#### Scenario: Docker flags rejected for GCP backend

- **WHEN** user provides `--base-image` or `--dockerfile` CLI flags with `--backend gcp`
- **THEN** the CLI SHALL return an error indicating these flags require `--backend docker`

#### Scenario: GCP flags rejected for Docker backend

- **WHEN** user provides `--project`, `--zone`, `--machine-type`, `--disk-size`, or `--state-bucket` CLI flags
  without `--backend gcp`
- **THEN** the CLI SHALL return an error indicating these flags require `--backend gcp`

#### Scenario: Config file values not rejected cross-backend

- **WHEN** `.spinner.json` contains keys for a different backend (e.g., `project` when using Docker)
- **THEN** the CLI SHALL silently ignore those values (no error)
- **AND** only CLI flags that are explicitly set trigger cross-backend validation

### Requirement: Configuration File Support

The setup command SHALL read infrastructure defaults from a `.spinner.json` file at the repo root.

#### Scenario: Config file provides backend default

- **WHEN** `.spinner.json` contains `{"backend": "gcp", "project": "my-proj", "zone": "us-central1-a", "state-bucket": "my-bucket"}`
- **AND** user runs `spinner setup --name my-env`
- **THEN** the CLI SHALL use the GCP backend with values from the config file

#### Scenario: CLI flags override config file

- **WHEN** `.spinner.json` contains `{"zone": "us-central1-a"}`
- **AND** user runs `spinner setup --backend gcp --name my-env --zone us-east1-b`
- **THEN** the CLI SHALL use `us-east1-b` (CLI flag takes precedence)

#### Scenario: Environment variables override config file

- **WHEN** `.spinner.json` contains `{"project": "file-project"}`
- **AND** `SPINNER_PROJECT=env-project` is set
- **THEN** the CLI SHALL use `env-project` (env var takes precedence over config file)

#### Scenario: No config file present

- **WHEN** no `.spinner.json` exists in the current directory
- **THEN** the CLI SHALL continue normally using CLI flags, env vars, and defaults

#### Scenario: Invalid config file

- **WHEN** `.spinner.json` exists but contains invalid JSON
- **THEN** the CLI SHALL print a warning and continue using CLI flags and defaults

### Requirement: Grouped Help Output

The setup command help SHALL organize flags into backend-specific groups for clarity.

#### Scenario: Help shows flag groups

- **WHEN** user runs `spinner setup --help`
- **THEN** flags SHALL be organized into labeled sections: General, Docker Backend, GCP Backend
