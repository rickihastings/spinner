# gcp-sandbox Specification

## Purpose
TBD - created by archiving change add-gcp-sandbox-backend. Update Purpose after archive.
## Requirements
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

### Requirement: GCP Image Baking

The GCP provider SHALL create custom Compute Engine images during setup by launching a temporary VM, installing
required tooling, and creating an image from the resulting disk.

#### Scenario: Successful image bake

- **WHEN** user runs `spinner setup --backend gcp --name my-env --project my-project --zone us-central1-a`
- **THEN** the provider SHALL create a temporary VM from a base Ubuntu image
- **AND** install git, GitHub CLI, Claude Code CLI, and the spinner binary
- **AND** wait for the VM to shut down after installation completes
- **AND** create a custom image named `spinner-{name}` from the VM's boot disk
- **AND** delete the temporary VM

#### Scenario: Custom machine type for baking

- **WHEN** user provides `--machine-type e2-standard-4` during setup
- **THEN** the temporary VM SHALL use the specified machine type for faster installation

#### Scenario: Default machine type for baking

- **WHEN** user does not provide `--machine-type` during setup
- **THEN** the temporary VM SHALL use `e2-standard-2` as the default

#### Scenario: Custom disk size for baking

- **WHEN** user provides `--disk-size 50` during setup
- **THEN** the temporary VM SHALL use the specified disk size in GB

#### Scenario: Default disk size for baking

- **WHEN** user does not provide `--disk-size` during setup
- **THEN** the temporary VM SHALL use 30 GB pd-balanced as the default

#### Scenario: Image already exists

- **WHEN** an image named `spinner-{name}` already exists in the project
- **THEN** the provider SHALL delete the old image and create a new one

#### Scenario: Bake failure cleanup

- **WHEN** the image baking process fails at any step
- **THEN** the provider SHALL clean up all temporary resources (VM, disk) before returning the error

#### Scenario: Spinner binary from GitHub Releases

- **WHEN** the bake VM's startup script installs tooling
- **THEN** it SHALL download the spinner binary from the latest GitHub Release
- **AND** use the GitHub Releases API to resolve the latest version tag
- **AND** download the `spinner_{version}_linux_amd64.tar.gz` archive and extract the binary to `/usr/local/bin/spinner`

#### Scenario: GitHub Release not available

- **WHEN** no GitHub Release exists or the download fails
- **THEN** the bake script SHALL exit with a non-zero status
- **AND** the bake VM SHALL report the error via serial port output before shutting down

#### Scenario: Bake completion detection

- **WHEN** the bake VM's startup script completes installation
- **THEN** the script SHALL shut down the VM
- **AND** the provider SHALL detect the `TERMINATED` state as successful completion

### Requirement: GCP Instance Lifecycle

The GCP provider SHALL manage VM instance lifecycle through the Compute Engine Instances API.

#### Scenario: Create new instance

- **WHEN** `Provider.Create()` is called with valid configuration
- **THEN** the provider SHALL create a VM from the custom image with runtime metadata
- **AND** use the machine type from `--machine-type` (default: `e2-standard-2`)
- **AND** use the disk size from `--disk-size` (default: 30 GB pd-balanced)
- **AND** wait for the VM to reach `RUNNING` status
- **AND** return an `Instance` with the VM name and running status

#### Scenario: Instance metadata configuration

- **WHEN** creating a VM instance
- **THEN** the following metadata keys SHALL be set: `GITHUB_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`, `REPO_URL`,
  `PROMPT`, `MAX_ITERATIONS`, `BRANCH`, `startup-script`

#### Scenario: Instance labels

- **WHEN** creating a VM instance
- **THEN** the following labels SHALL be applied: `spinner-managed=true`, `spinner-image={name}`,
  `spinner-repo={sanitized-repo}`

#### Scenario: Instance networking

- **WHEN** creating a VM instance
- **THEN** the VM SHALL have an ephemeral external IP for outbound internet access
- **AND** the default VPC network SHALL be used unless a custom network is specified

#### Scenario: Start stopped instance

- **WHEN** `Provider.Start()` is called for a stopped VM
- **THEN** the provider SHALL call the Start API and wait for `RUNNING` status

#### Scenario: Stop running instance

- **WHEN** `Provider.Stop()` is called for a running VM
- **THEN** the provider SHALL call the Stop API and wait for `TERMINATED` status

#### Scenario: Restart instance

- **WHEN** `Provider.Restart()` is called
- **THEN** the provider SHALL stop then start the VM

#### Scenario: Remove instance

- **WHEN** `Provider.Remove()` is called
- **THEN** the provider SHALL delete the VM and its boot disk

#### Scenario: Instance not found

- **WHEN** any lifecycle operation is called for a non-existent instance
- **THEN** the provider SHALL return an appropriate error

### Requirement: GCP Instance Naming

The GCP provider SHALL generate deterministic instance names compatible with GCP naming constraints.

#### Scenario: Name format

- **WHEN** generating an instance name
- **THEN** the format SHALL be `spinner-{image}-{repo}[-{branch}]`

#### Scenario: GCP naming constraints

- **WHEN** generating an instance name
- **THEN** the name SHALL be lowercase, start with a letter, contain only `[a-z0-9-]`, and be max 63 characters

#### Scenario: Name truncation

- **WHEN** the generated name exceeds 63 characters
- **THEN** the name SHALL be truncated to 63 characters with a trailing hash suffix for uniqueness

#### Scenario: Deterministic naming

- **WHEN** the same image, repo, and branch are provided
- **THEN** the same instance name SHALL be generated every time

### Requirement: GCP Instance Status Mapping

The GCP provider SHALL map Compute Engine VM statuses to `provider.InstanceStatus`.

#### Scenario: Running VM

- **WHEN** the VM status is `RUNNING`
- **THEN** `Provider.Status()` SHALL return `InstanceStatusRunning`

#### Scenario: Stopped VM

- **WHEN** the VM status is `TERMINATED` or `STOPPED`
- **THEN** `Provider.Status()` SHALL return `InstanceStatusStopped`

#### Scenario: Non-existent VM

- **WHEN** the VM does not exist (404 from API)
- **THEN** `Provider.Status()` SHALL return `InstanceStatusNone`

#### Scenario: Transitional states

- **WHEN** the VM status is `STAGING`, `PROVISIONING`, `STOPPING`, or `SUSPENDING`
- **THEN** `Provider.Status()` SHALL return the closest mapped status with no error

### Requirement: GCP Log Streaming via GCS

The GCP provider SHALL support log retrieval and real-time streaming via GCS, tapping directly into the
executor's existing TeeReader pipeline on the VM side.

#### Scenario: VM-side log sync via executor pipeline

- **WHEN** `spinner exec` runs inside a GCP VM
- **AND** the `SPINNER_LOG_BUCKET` environment variable is set
- **THEN** exec SHALL create a `GCSSink` (`io.Writer`) and pass it to the executor as an additional writer
- **AND** the executor SHALL use `io.MultiWriter(logFile, gcsSink)` in its `TeeReader` pipeline
- **AND** the GCS sink SHALL buffer writes and flush to GCS at `{bucket}/{instance-name}/logs/raw.log` every 2 seconds
- **AND** the GCS sink SHALL be closed when exec completes

#### Scenario: VM-side log sync not on GCP

- **WHEN** `spinner exec` runs inside a Docker container (no `SPINNER_LOG_BUCKET` set)
- **THEN** the executor SHALL use only the log file writer (no behavior change)

#### Scenario: Full log retrieval

- **WHEN** `Provider.Logs()` is called on the control plane
- **THEN** the GCP provider SHALL download the full GCS log object and return as `io.ReadCloser`

#### Scenario: Real-time log streaming

- **WHEN** `Provider.WatchLogs()` is called on the control plane
- **THEN** the GCP provider SHALL poll the GCS log object every 2 seconds
- **AND** track byte offset to only read new content (GCS Range reads)
- **AND** split new bytes into lines and send to the provided channel

#### Scenario: Log stream cancellation

- **WHEN** the context is cancelled during log streaming
- **THEN** the provider SHALL stop polling and return

#### Scenario: Log object not yet created

- **WHEN** `WatchLogs()` is called before the VM has written any logs
- **THEN** the provider SHALL poll until the GCS object appears (with timeout)

#### Scenario: VM not running

- **WHEN** log streaming is requested for a non-running VM
- **THEN** the provider SHALL return an appropriate error

### Requirement: GCP Metrics Streaming

The GCP provider SHALL support resource metrics via Cloud Monitoring API.

#### Scenario: CPU metrics

- **WHEN** `Provider.WatchMetrics()` is called
- **THEN** the provider SHALL query `compute.googleapis.com/instance/cpu/utilization`
- **AND** map the value to `ContainerMetrics.CPUPercent` (scaled to 0-100)

#### Scenario: Memory metrics

- **WHEN** the VM has the Ops Agent installed
- **THEN** the provider SHALL query memory utilization metrics
- **AND** map to `ContainerMetrics.MemoryUsed`, `MemoryLimit`, and `MemoryPercent`

#### Scenario: Metrics polling interval

- **WHEN** metrics streaming is active
- **THEN** the provider SHALL poll Cloud Monitoring at 60-second intervals (minimum API granularity)

#### Scenario: VM state in metrics

- **WHEN** metrics are collected
- **THEN** the provider SHALL include the VM state (`running`, `stopped`, etc.) in `ContainerMetrics.State`

#### Scenario: Metrics stream cancellation

- **WHEN** the context is cancelled during metrics streaming
- **THEN** the provider SHALL stop polling and close the channel

### Requirement: GCP State Persistence

The GCP provider SHALL persist execution state to Google Cloud Storage for durability across VM lifecycle events.
GCS bucket names are globally unique across all of Google Cloud, so the bucket name MUST be user-configured.

#### Scenario: Required state bucket flag

- **WHEN** user runs a GCP operation that requires state persistence
- **THEN** the `--state-bucket` flag SHALL be required
- **AND** the CLI SHALL error if the flag is not provided

#### Scenario: State bucket does not exist

- **WHEN** the specified GCS bucket does not exist
- **THEN** the provider SHALL return a clear error indicating the bucket was not found and must be created

#### Scenario: State write

- **WHEN** execution state needs to be persisted
- **THEN** the provider SHALL write `state.json` to `gs://{state-bucket}/{instance-name}/state.json`

#### Scenario: State read on startup

- **WHEN** a VM instance starts and state exists in GCS
- **THEN** the runtime startup script SHALL download state to `/state/state.json` before running `spinner exec`

#### Scenario: State not found

- **WHEN** no state exists in GCS for an instance
- **THEN** the runtime startup script SHALL proceed without state (fresh start)

### Requirement: GCP Async Operation Handling

The GCP provider SHALL correctly handle Compute Engine's asynchronous operations (Long-Running Operations).

#### Scenario: Operation wait

- **WHEN** a Compute Engine operation is initiated (create, delete, start, stop)
- **THEN** the provider SHALL wait for the operation to complete before returning

#### Scenario: Operation timeout

- **WHEN** an operation does not complete within a reasonable timeout
- **THEN** the provider SHALL return a timeout error with the operation details

#### Scenario: Operation failure

- **WHEN** an operation fails
- **THEN** the provider SHALL return the operation error with descriptive context

#### Scenario: Context cancellation during operation wait

- **WHEN** the context is cancelled while waiting for an operation
- **THEN** the provider SHALL return immediately with a cancellation error

### Requirement: GCP VM Completion Behavior

The GCP provider SHALL keep VMs running after `spinner exec` completes, matching Docker's container behavior.

#### Scenario: VM stays running after exec completes

- **WHEN** `spinner exec` finishes inside a GCP VM (success or failure)
- **THEN** the VM SHALL remain in `RUNNING` state
- **AND** the user can SSH into the VM for debugging or inspection

#### Scenario: VM stays running when no prompt provided

- **WHEN** a VM is created without a `--prompt` flag
- **THEN** the VM SHALL remain running and accessible via SSH

### Requirement: GCP Resource Cleanup

The GCP provider SHALL support cleanup of all managed resources.

#### Scenario: Instance removal cleans up

- **WHEN** `Provider.Remove()` is called
- **THEN** the VM and its boot disk SHALL be deleted
- **AND** the state in GCS SHALL be preserved (for debugging)

#### Scenario: Labels enable identification

- **WHEN** managed resources are created
- **THEN** the `spinner-managed=true` label SHALL be applied
- **AND** resources can be listed/cleaned up by label

### Requirement: GCP Error Handling

The GCP provider SHALL provide clear, actionable error messages for common failure modes.

#### Scenario: Quota exceeded

- **WHEN** a VM creation fails due to quota limits
- **THEN** the error message SHALL indicate which quota was exceeded and in which region

#### Scenario: Permission denied

- **WHEN** an operation fails due to missing IAM permissions
- **THEN** the error message SHALL indicate which permission is required

#### Scenario: Image not found

- **WHEN** a spin operation references a non-existent image
- **THEN** the error message SHALL indicate the image was not found and suggest running setup first

#### Scenario: Zone unavailable

- **WHEN** a VM cannot be created in the specified zone
- **THEN** the error message SHALL indicate the zone issue and suggest alternatives

### Requirement: GCP Client Interface Testability

The GCP provider SHALL use an internal client interface to enable unit testing without GCP API calls.

#### Scenario: Mock client for unit tests

- **WHEN** unit tests run for GCP provider logic
- **THEN** a `MockGCPClient` SHALL be used (testify mock implementing `Client` interface)

#### Scenario: Real client for integration

- **WHEN** integration tests or production code runs
- **THEN** `RealGCPClient` SHALL use `gcloud` CLI commands with `--format=json` output parsing

#### Scenario: Client interface coverage

- **WHEN** the `Client` interface is defined
- **THEN** it SHALL cover all GCP operations used by the provider: instance CRUD, image CRUD, operation wait, serial
  port, storage, and monitoring

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

