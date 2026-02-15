# gcp-sandbox Delta: add-model-flag

## ADDED Requirements

### Requirement: Model Metadata on VM Creation

The GCP provider SHALL include `ANTHROPIC_MODEL` in the instance metadata when creating a VM, allowing the
runtime script to set the model for the Claude CLI.

#### Scenario: Model specified in CreateConfig

- **WHEN** the GCP provider creates a VM with `CreateConfig.Model` set to `claude-sonnet-4-5-20250929`
- **THEN** the instance metadata SHALL include `ANTHROPIC_MODEL=claude-sonnet-4-5-20250929`

#### Scenario: Model not specified in CreateConfig

- **WHEN** the GCP provider creates a VM with `CreateConfig.Model` empty
- **THEN** the instance metadata SHALL include `ANTHROPIC_MODEL` with an empty value
- **AND** the Claude CLI SHALL use its default model

### Requirement: Model Metadata Updated on Restart

The GCP provider SHALL update the `ANTHROPIC_MODEL` metadata item when restarting a stopped VM, ensuring the
new model value takes effect on boot.

#### Scenario: Model changed on restart

- **WHEN** a stopped VM is restarted via `Start()` with `CreateConfig.Model` set to `claude-opus-4-6`
- **THEN** the `ANTHROPIC_MODEL` metadata item SHALL be updated to `claude-opus-4-6` before the VM starts

### Requirement: Runtime Script Model Export

The GCP runtime startup script SHALL read `ANTHROPIC_MODEL` from instance metadata and export it as an
environment variable.

#### Scenario: ANTHROPIC_MODEL in metadata

- **WHEN** the runtime script reads instance metadata
- **AND** `ANTHROPIC_MODEL` is set to `claude-sonnet-4-5-20250929`
- **THEN** the script SHALL export `ANTHROPIC_MODEL=claude-sonnet-4-5-20250929`

#### Scenario: ANTHROPIC_MODEL empty in metadata

- **WHEN** the runtime script reads instance metadata
- **AND** `ANTHROPIC_MODEL` is empty or not present
- **THEN** the script SHALL export `ANTHROPIC_MODEL` as empty string
- **AND** the Claude CLI SHALL use its default model
