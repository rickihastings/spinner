## MODIFIED Requirements

### Requirement: CLI-Based GCP Operations

The GCP provider SHALL use the `gcloud` CLI tool for all Compute Engine and Cloud Storage operations instead of the
GCP Go SDK. All structured data SHALL be obtained via `--format=json` output parsing into plain Go types.

#### Scenario: gcloud CLI prerequisite

- **WHEN** any GCP operation is requested
- **THEN** the system SHALL verify that `gcloud` is available on the PATH
- **AND** if `gcloud` is not found, SHALL return a clear error with install instructions

#### Scenario: Authentication via gcloud

- **WHEN** the GCP provider executes operations
- **THEN** it SHALL delegate authentication to `gcloud` (which uses Application Default Credentials)
- **AND** it SHALL support all ADC methods: `gcloud auth login`, `gcloud auth application-default login`,
  service account key, workload identity

#### Scenario: Missing credentials

- **WHEN** no valid GCP credentials are configured in `gcloud`
- **THEN** the `gcloud` command SHALL return a clear error message indicating how to configure credentials
- **AND** the provider SHALL propagate this error to the user

#### Scenario: Invalid project ID

- **WHEN** the provided project ID does not exist or is inaccessible
- **THEN** the `gcloud` command SHALL return an error indicating the project is invalid or inaccessible
- **AND** the provider SHALL propagate this error to the user

#### Scenario: JSON output parsing

- **WHEN** a `gcloud` command returns structured data
- **THEN** the provider SHALL use `--format=json` and parse the output into plain Go structs
- **AND** the Go structs SHALL have JSON tags matching `gcloud` output field names

#### Scenario: GCE environment detection

- **WHEN** code needs to detect whether it is running on a GCE VM
- **THEN** it SHALL make an HTTP GET request to `http://metadata.google.internal/computeMetadata/v1/` with header
  `Metadata-Flavor: Google`
- **AND** a successful response (HTTP 200) SHALL indicate GCE environment
- **AND** the request SHALL have a 1-second timeout to avoid blocking on non-GCE hosts

#### Scenario: CLI error handling

- **WHEN** a `gcloud` command exits with non-zero status
- **THEN** the provider SHALL capture stderr and wrap it in a Go error with command context
- **AND** the `isNotFoundError()` function SHALL continue to match "not found" and "404" in error messages
