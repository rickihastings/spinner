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

#### Scenario: Docker flags ignored for GCP

- **WHEN** user provides `--base-image` or `--dockerfile` with `--backend gcp`
- **THEN** the CLI SHALL print a warning that these flags are Docker-specific and will be ignored

#### Scenario: GCP flags ignored for Docker

- **WHEN** user provides `--project` or `--zone` without `--backend gcp`
- **THEN** the CLI SHALL print a warning that these flags are GCP-specific and will be ignored
