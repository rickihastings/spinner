# cli-spin Specification (Delta)

## ADDED Requirements

### Requirement: Backend Selection for Spin

The spin command SHALL accept a `--backend` flag to select which provider handles the spin operation.

#### Scenario: Default backend

- **WHEN** user runs `spinner spin --image my-env --repo <url>` without `--backend` flag
- **THEN** the CLI SHALL use the `docker` backend (backward compatible)

#### Scenario: GCP backend selected

- **WHEN** user runs `spinner spin --backend gcp --image my-env --repo <url> --project my-project --zone us-central1-a`
- **THEN** the CLI SHALL use the GCP provider to create a Compute Engine VM instance

#### Scenario: Unknown backend

- **WHEN** user provides an unknown backend name
- **THEN** the CLI SHALL print an error listing available backends and exit with non-zero status

#### Scenario: Backend flag via environment variable

- **WHEN** `SPINNER_BACKEND` environment variable is set
- **THEN** the CLI SHALL use that value as the default backend
- **AND** the `--backend` flag SHALL take precedence over the environment variable

### Requirement: GCP-Specific Spin Flags

The spin command SHALL accept GCP-specific flags when `--backend gcp` is selected.

#### Scenario: Required project flag for GCP

- **WHEN** user runs `spinner spin --backend gcp --image my-env --repo <url>` without `--project`
- **THEN** the CLI SHALL print an error indicating `--project` is required for GCP backend

#### Scenario: Required zone flag for GCP

- **WHEN** user runs `spinner spin --backend gcp --image my-env --repo <url> --project p` without `--zone`
- **THEN** the CLI SHALL print an error indicating `--zone` is required for GCP backend

#### Scenario: Optional machine-type flag

- **WHEN** user provides `--machine-type n2-standard-4` with GCP backend
- **THEN** the CLI SHALL create the VM with the specified machine type

#### Scenario: Default machine-type for spin

- **WHEN** user does not provide `--machine-type` with GCP backend
- **THEN** the CLI SHALL default to `e2-standard-2`

#### Scenario: GCP instance management instructions

- **WHEN** a GCP instance is created successfully
- **THEN** the CLI SHALL display GCP-specific management instructions:
- **AND** include `gcloud compute ssh` for access
- **AND** include `gcloud compute instances stop` for stopping
- **AND** include `gcloud compute instances delete` for removal

#### Scenario: GCP instance reuse

- **WHEN** user runs spin and a GCP VM with the deterministic name already exists and is running
- **THEN** the CLI SHALL reuse the existing VM (same as Docker container reuse behavior)

#### Scenario: GCP instance restart

- **WHEN** user runs spin and a GCP VM exists but is stopped (TERMINATED)
- **THEN** the CLI SHALL start the existing VM

#### Scenario: GCP instance recreation

- **WHEN** user runs spin with `--recreate` and `--backend gcp`
- **THEN** the CLI SHALL delete the existing VM and create a new one

#### Scenario: Watch mode with GCP backend

- **WHEN** user runs spin with `--watch` and `--backend gcp`
- **THEN** the CLI SHALL enter watch mode using the GCP provider's WatchLogs and WatchMetrics

#### Scenario: Setup flag with GCP backend

- **WHEN** user runs spin with `--setup` and `--backend gcp`
- **THEN** the CLI SHALL bake a GCP image before creating the VM instance

#### Scenario: Docker flags ignored for GCP

- **WHEN** user provides Docker-specific flags (e.g., `--dockerfile`) with `--backend gcp`
- **THEN** the CLI SHALL print a warning that these flags are Docker-specific and will be ignored

#### Scenario: GCP flags ignored for Docker

- **WHEN** user provides GCP-specific flags (e.g., `--project`, `--zone`) without `--backend gcp`
- **THEN** the CLI SHALL print a warning that these flags are GCP-specific and will be ignored
