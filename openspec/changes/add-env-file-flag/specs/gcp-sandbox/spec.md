# gcp-sandbox Specification

## ADDED Requirements

### Requirement: GCP Env File Delivery

The GCP backend SHALL pass the env file content via instance metadata and the runtime script SHALL write it to the
workspace and source it.

#### Scenario: Env file encoded in metadata

- **WHEN** `--env-file` is provided with the GCP backend
- **THEN** the GCP backend SHALL read the file, base64-encode the content, and set it as the `SPINNER_ENV_FILE`
  metadata value

#### Scenario: Runtime script writes and sources env file

- **WHEN** the GCP VM starts and `SPINNER_ENV_FILE` metadata exists
- **THEN** the runtime script SHALL base64-decode the value
- **AND** write the decoded content to the workspace directory as `.env`
- **AND** source the file to set variables in the runtime environment

#### Scenario: No env file in metadata

- **WHEN** the GCP VM starts and `SPINNER_ENV_FILE` metadata does not exist
- **THEN** the runtime script SHALL skip env file processing (no change in behavior)
