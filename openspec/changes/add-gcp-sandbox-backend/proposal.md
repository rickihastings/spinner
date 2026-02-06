# Proposal: Add GCP Sandbox Backend

## Summary

Add Google Cloud Platform (GCP) Compute Engine as a second sandbox backend for Spinner, alongside the existing Docker
backend. The GCP provider uses the official Go SDK (`cloud.google.com/go/compute`) — not CLI commands — for all
interactions with GCP APIs. This also introduces a provider selection mechanism (`--backend` flag) so users can choose
between Docker and GCP at runtime.

## Motivation

Docker is excellent for local development but has limitations for production-scale agent loops:

- **Resource constraints**: Host machine CPU/memory limits the number of concurrent agents
- **Isolation**: Docker containers share the host kernel; VMs provide hardware-level isolation
- **Scalability**: Cloud VMs can be provisioned on-demand across regions, enabling parallel agent runs
- **Persistence**: Cloud VMs with persistent disks survive host reboots and can be managed remotely
- **Cost flexibility**: Spot/preemptible VMs reduce costs for long-running agent loops

GCP Compute Engine provides the right abstraction — full VMs with predictable performance, deep API access, and
mature tooling for monitoring and lifecycle management.

## What Changes

### New Capability: `gcp-sandbox`

A complete GCP Compute Engine provider implementing the `provider.Provider` interface:

- **GCPClient interface** — SDK-based abstraction (mirrors Docker's two-layer pattern) with mock for testing
- **Authentication** — Application Default Credentials (ADC) for zero-config auth in GCP environments
- **Setup** — Bakes a custom GCE image by launching a temp VM, installing tooling (git, gh, claude-code, spinner),
  then creating a machine image from the disk
- **Create** — Launches a VM instance from the baked image with runtime metadata (repo, prompt, branch, tokens)
- **Start/Stop/Restart/Remove** — Full VM lifecycle management via Compute Engine Instances API
- **Logs** — GCS-based log streaming; exec syncs `raw.log` to GCS via reused `LogWatcher`; control plane polls GCS
- **Metrics** — Cloud Monitoring API for CPU utilization; Ops Agent for memory metrics
- **State** — Cloud Storage (GCS) bucket for state persistence across VM stop/start cycles
- **Secrets** — Passed via instance metadata (matches Docker's env-var security model); Secret Manager noted as
  future hardening option

### Modified Capability: `cli-setup`

- **ADDED** `--backend` flag (default: `"docker"`) to select which provider handles setup
- **ADDED** GCP-specific setup options: `--project`, `--zone`, `--machine-type`, `--disk-size`, `--state-bucket`

### Modified Capability: `cli-spin`

- **ADDED** `--backend` flag (default: `"docker"`) to select which provider handles spin
- **ADDED** GCP-specific spin options: `--project`, `--zone`, `--machine-type`, `--disk-size`, `--state-bucket`

### New Internal Package: Provider Factory

- `internal/provider/factory.go` — Registry/factory that maps backend names to `Provider` constructors
- Allows commands to remain backend-agnostic while supporting runtime selection

### New Capability: Configuration File (`.spinner.json`)

A JSON configuration file at the repo root that stores infrastructure defaults:

```json
{
  "backend": "gcp",
  "project": "my-gcp-project",
  "zone": "us-central1-a",
  "state-bucket": "my-org-spinner-state",
  "machine-type": "e2-standard-4",
  "disk-size": 50
}
```

- Loaded via Viper's config file support (already in use for `.env`)
- Precedence: **CLI flags > env vars (`SPINNER_*`) > `.spinner.json` > defaults**
- Can be committed to the repo so team members share the same infra config
- Only infrastructure defaults belong here — runtime values (`prompt`, `branch`, `repo`) stay as CLI flags

### Modified Capability: Conditional Flag Validation

Backend-specific flags are organized into groups and validated at runtime:

- **Hard error** if a backend-specific flag is used with the wrong `--backend` (e.g., `--project` without `--backend gcp`)
- **Grouped help output** — `spinner setup --help` shows Docker flags and GCP flags in separate labeled sections
- Uses Cobra's `AddGroup` for visual organization + `RunE` validation for enforcement

## GCP APIs Used

| API / Service | SDK Package | Purpose |
|---|---|---|
| Compute Engine Instances | `cloud.google.com/go/compute/apiv1` (`InstancesClient`) | VM lifecycle: create, start, stop, delete, get, reset |
| Compute Engine Images | `cloud.google.com/go/compute/apiv1` (`ImagesClient`) | Custom image creation and lookup |
| Compute Engine Operations | `cloud.google.com/go/compute/apiv1` (zone/global ops) | Wait for async operations to complete |
| Cloud Storage | `cloud.google.com/go/storage` | State file persistence, startup script hosting |
| Cloud Logging | `cloud.google.com/go/logging` | Structured log reading and streaming |
| Cloud Monitoring | `cloud.google.com/go/monitoring/apiv3/v2` (`MetricClient`) | CPU/memory metrics for watch mode |
| IAM / Service Accounts | `google.golang.org/api/iam/v1` | (read-only) Validate service account permissions |
| Resource Manager | `cloud.google.com/go/resourcemanager/apiv3` | (read-only) Validate project access |

## Impact

### Affected Specs

- `cli-setup` — New `--backend` flag and GCP-specific options
- `cli-spin` — New `--backend` flag and GCP-specific options
- `cli-watch` — Backend-aware provider resolution (uses provider from spin or standalone flag)

### Affected Code

| Area | Change Type |
|---|---|
| `internal/provider/` | New `factory.go` for provider registry |
| `internal/logs/` | **New package** — GCS log sink (`io.Writer` for exec → GCS streaming) |
| `internal/gcp/` | **New package** — GCP provider, client, types, templates |
| `cmd/constructors.go` | Modify command constructors to accept factory instead of single provider |
| `cmd/setup.go` | Wire factory; add GCP-specific flags with conditional validation |
| `cmd/spin.go` | Wire factory; add GCP-specific flags with conditional validation |
| `cmd/watch.go` | Wire factory for standalone watch mode |
| `cmd/root.go` | Add `.spinner.json` config file loading via Viper |
| `templates/scripts/` | New GCP-specific startup scripts (bake + runtime) |
| `go.mod` | New dependencies: `cloud.google.com/go/*` packages |

### Also Affected (refactor only, no behavior change)

- `internal/agent/claude/executor.go` — Accepts optional `AdditionalWriter` for GCS sink via `io.MultiWriter`
- `internal/exec/loop.go` — Creates GCS sink and passes to executor when `SPINNER_LOG_BUCKET` env var is set

### Not Affected

- `internal/agent/` — Agent abstraction is provider-agnostic
- Provider interface definition — No changes to `provider.Provider`

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| GCP SDK adds significant dependency weight | Larger binary, longer builds | Tree-shaking via Go modules; lazy import only when GCP backend is selected |
| Image baking is slow (~5-10 min) | Poor UX for first-time setup | Cache images; show progress; document expected time |
| Cloud costs from forgotten VMs | Unexpected charges | VM stays running after completion (Docker parity); labels for cost tracking; clear teardown instructions; auto-stop is a follow-up |
| GCS bucket name collision | Bucket names are globally unique across all GCP | User-configurable `--state-bucket` flag; no auto-generated default that could collide |
| Metadata-based secrets visible in console | Security concern for shared projects | Document threat model; recommend Secret Manager for multi-tenant use |
| Async operations (LROs) add complexity | Harder error handling | Consistent operation-wait pattern with timeout and progress reporting |
| Network connectivity requirements | VM needs outbound internet | Default to VPC with NAT or external IP; document firewall requirements |

## Non-Goals

- **Kubernetes backend** — Out of scope; could be a separate proposal
- **AWS/Azure backends** — Out of scope; factory pattern makes future backends easy to add
- **Secret Manager integration** — Metadata approach matches Docker's env-var model; Secret Manager is a future hardening option
- **Custom VPC creation** — Uses default VPC or user-specified existing VPC; no VPC provisioning
- **GPU support** — Standard machine types only for initial implementation
- **Multi-region** — Single zone per command invocation; multi-region orchestration is out of scope
- **Spot/preemptible VMs** — Follow-up; agent loops can be long-running, preemption adds complexity
- **Auto-stop on completion** — Follow-up; VM stays running after exec completes (Docker parity), auto-stop flag added later
