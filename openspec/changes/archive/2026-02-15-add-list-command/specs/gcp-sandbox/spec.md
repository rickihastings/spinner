# gcp-sandbox Specification

## Purpose

Extend the GCP provider with VM instance listing support for the `spinner list` command, leveraging existing
`spinner-managed=true` labels and GCS state files.

## ADDED Requirements

### Requirement: GCP Instance Listing

The GCP provider SHALL implement `Provider.List()` to discover all spinner-managed VM instances.

#### Scenario: Successful listing

- **WHEN** `Provider.List()` is called
- **THEN** the provider SHALL query Compute Engine for VMs with label `spinner-managed=true` in the configured zone
- **AND** return results as `[]InstanceInfo` with metadata from VM labels and metadata items

#### Scenario: State enrichment from GCS

- **WHEN** a VM is discovered during listing and a state bucket is configured
- **THEN** the provider SHALL read `gs://{bucket}/{name}/state.json` from GCS
- **AND** populate iteration, agent status, and timestamp fields in `InstanceInfo`
- **AND** if the state object does not exist, those fields SHALL be zero-valued

#### Scenario: No state bucket configured

- **WHEN** listing instances without `--state-bucket` configured
- **THEN** the provider SHALL return instance info from VM metadata only (no execution state)

#### Scenario: Metadata extraction

- **WHEN** populating `InstanceInfo` for a GCP VM
- **THEN** the provider SHALL extract image name from the `spinner-image` label
- **AND** extract repo from the `spinner-repo` label
- **AND** extract branch from the `BRANCH` metadata item
- **AND** extract agent from the `ANTHROPIC_MODEL` metadata item
- **AND** extract max iterations from the `MAX_ITERATIONS` metadata item

### Requirement: GCP Client List Method

The GCP Client interface SHALL support listing VM instances filtered by labels.

#### Scenario: List instances with filter

- **WHEN** `ListInstances()` is called with a project, zone, and filter string
- **THEN** the client SHALL use the Compute Engine Instances API `List` method
- **AND** apply the provided filter (e.g., `labels.spinner-managed=true`)
- **AND** return all matching instances
