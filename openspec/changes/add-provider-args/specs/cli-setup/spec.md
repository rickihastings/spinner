# cli-setup Spec Delta

## ADDED Requirements

### Requirement: Provider Pass-Through Arguments for Setup

The setup command SHALL accept an optional repeatable `--provider-args` flag that passes raw arguments directly to the
underlying backend provider. Arguments are forwarded verbatim to the backend's environment provisioning command
(`docker build` for Docker, `gcloud compute instances create` for GCP image bake).

#### Scenario: Provider arg for Docker build

- **WHEN** user runs `spinner setup --name my-env --provider-args="--build-arg=MY_ARG=value"`
- **THEN** the CLI SHALL append `--build-arg=MY_ARG=value` to the `docker build` command before the context directory

#### Scenario: Provider arg for Docker no-cache

- **WHEN** user runs `spinner setup --name my-env --provider-args="--no-cache"`
- **THEN** the CLI SHALL append `--no-cache` to the `docker build` command

#### Scenario: Provider arg for GCP setup

- **WHEN** user runs `spinner setup --backend gcp --name my-env --project p --zone z --state-bucket b --provider-args="--labels=env=staging"`
- **THEN** the CLI SHALL append `--labels=env=staging` to the GCP bake VM creation command

#### Scenario: No provider args provided

- **WHEN** user runs `spinner setup --name my-env` without any `--provider-args` flags
- **THEN** the CLI SHALL behave identically to the current implementation (no change in behavior)

### Requirement: Provider Args Conflict Detection for Setup

The CLI SHALL reject `--provider-args` values that conflict with arguments managed by Spinner during setup.

#### Scenario: Docker build tag conflict

- **WHEN** user provides `--provider-args="-t my-tag"` with the Docker backend setup
- **THEN** the CLI SHALL print an error indicating `-t` is managed by Spinner and exit with non-zero status

#### Scenario: Non-conflicting build args pass through

- **WHEN** user provides `--provider-args="--no-cache"` with the Docker backend setup
- **THEN** the CLI SHALL accept and forward the argument without error

## MODIFIED Requirements

### Requirement: Cobra Command Structure

The setup command SHALL use Cobra and Viper for flag definition and configuration management, implemented in Go. The
command SHALL be fully testable via dependency injection.

#### Scenario: Cobra command initialization

- **WHEN** the CLI starts
- **THEN** the setup command SHALL be registered as a Cobra subcommand with all flags (--name, --backend, --base-image,
  --dockerfile, --provider-args, and GCP-specific flags)
