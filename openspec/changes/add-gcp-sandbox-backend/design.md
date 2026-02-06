# Design: Add GCP Sandbox Backend

## Technical Implementation Plan

### Component Map

| File | Action | Purpose |
|---|---|---|
| `internal/provider/factory.go` | **create** | Provider factory — maps backend names to constructors |
| `internal/provider/factory_test.go` | **create** | Factory tests |
| `internal/gcp/gcp_provider.go` | **create** | GCP provider implementing `provider.Provider` |
| `internal/gcp/gcp_provider_test.go` | **create** | GCP provider tests using mock client |
| `internal/gcp/client.go` | **create** | GCPClient interface + RealGCPClient (SDK) |
| `internal/gcp/client_test.go` | **create** | Client unit tests |
| `internal/gcp/mock_client.go` | **create** | MockGCPClient for testing |
| `internal/gcp/types.go` | **create** | GCP-specific types (config, status, result) |
| `internal/gcp/image.go` | **create** | Image baking logic (temp VM → image) |
| `internal/gcp/image_test.go` | **create** | Image baking tests |
| `internal/gcp/instance.go` | **create** | VM instance lifecycle operations |
| `internal/gcp/instance_test.go` | **create** | Instance lifecycle tests |
| `internal/logs/watcher.go` | **extract** | LogWatcher moved from `internal/docker/logs.go` — fsnotify-based file tailing (Docker control plane) |
| `internal/logs/watcher_test.go` | **extract** | LogWatcher tests (moved from docker package) |
| `internal/logs/gcs_sink.go` | **create** | GCS log sink — `io.Writer` that buffers and flushes to GCS |
| `internal/logs/gcs_sink_test.go` | **create** | GCS sink tests |
| `internal/docker/logs.go` | **modify** | Import LogWatcher from `internal/logs/` instead of defining locally |
| `internal/agent/claude/executor.go` | **modify** | Support optional `AdditionalWriter` in config for GCS sink |
| `internal/gcp/logs.go` | **create** | GCS-based log reader for control plane `WatchLogs` / `Logs` |
| `internal/gcp/logs_test.go` | **create** | GCS log reader tests |
| `internal/gcp/metrics.go` | **create** | Cloud Monitoring integration |
| `internal/gcp/metrics_test.go` | **create** | Metrics tests |
| `internal/gcp/state.go` | **create** | GCS-based state persistence |
| `internal/gcp/state_test.go` | **create** | State persistence tests |
| `internal/gcp/startup.go` | **create** | Startup script generation for GCP VMs |
| `templates/scripts/gcp_bake.sh` | **create** | Image baking startup script (installs tooling) |
| `templates/scripts/gcp_runtime.sh` | **create** | Runtime startup script (clones repo, runs exec) |
| `internal/exec/loop.go` | **modify** | Create GCS sink and pass to executor when `SPINNER_LOG_BUCKET` env var is set |
| `cmd/constructors.go` | **modify** | Accept provider factory; add `--backend` flag; conditional flag validation |
| `cmd/setup.go` | **modify** | Wire factory; register grouped GCP-specific flags |
| `cmd/spin.go` | **modify** | Wire factory; register grouped GCP-specific flags |
| `cmd/constructors_watch.go` | **modify** | Wire factory for standalone watch |
| `cmd/watch.go` | **modify** | Wire factory |
| `cmd/root.go` | **modify** | Add `.spinner.json` config file loading |
| `go.mod` | **modify** | Add GCP SDK dependencies |

### Approach

Implementation follows the existing two-layer architecture (Provider → Client) established by the Docker backend.

#### Phase 1: Provider Factory & Backend Selection

Introduce a factory that decouples commands from specific providers:

```go
// internal/provider/factory.go
package provider

// BackendConstructor creates a Provider for a specific backend.
type BackendConstructor func() (Provider, error)

// Factory holds registered backend constructors.
type Factory struct {
    backends map[string]BackendConstructor
}

func NewFactory() *Factory {
    return &Factory{backends: make(map[string]BackendConstructor)}
}

func (f *Factory) Register(name string, ctor BackendConstructor) {
    f.backends[name] = ctor
}

func (f *Factory) Create(name string) (Provider, error) {
    ctor, ok := f.backends[name]
    if !ok {
        return nil, fmt.Errorf("unknown backend: %q (available: %s)",
            name, strings.Join(f.Available(), ", "))
    }
    return ctor()
}

func (f *Factory) Available() []string { /* sorted keys */ }
```

Commands change from `NewSpinCommand(p provider.Provider)` to `NewSpinCommand(f *provider.Factory)`, resolving the
provider at runtime based on the `--backend` flag.

#### Phase 1b: Configuration File (`.spinner.json`)

Add repo-level JSON config for infrastructure defaults. This builds on Viper's existing config file support:

```go
// cmd/root.go init()
func init() {
    viper.SetEnvPrefix("SPINNER")
    viper.AutomaticEnv()

    // Primary config: .spinner.json in repo root (committed, team-shared)
    viper.SetConfigName(".spinner")
    viper.SetConfigType("json")
    viper.AddConfigPath(".")
    _ = viper.ReadInConfig()

    // Secondary: .env file (not committed, local overrides)
    // Handled separately since Viper only reads one config file
    loadDotEnv()
}
```

**Config file contents** — infrastructure defaults only:

```json
{
  "backend": "gcp",
  "project": "my-gcp-project",
  "zone": "us-central1-a",
  "state-bucket": "my-org-spinner-state",
  "machine-type": "e2-standard-4",
  "disk-size": 50,
  "base-image": "node:20-bullseye"
}
```

**Precedence chain** (highest wins):
1. CLI flags (`--project my-project`)
2. Environment variables (`SPINNER_PROJECT=my-project`)
3. `.spinner.json` in current directory
4. Built-in defaults

**What belongs in `.spinner.json`** vs CLI:
| Config file (infrastructure, rarely changes) | CLI only (runtime, per-invocation) |
|---|---|
| `backend`, `project`, `zone` | `name`, `image`, `repo` |
| `state-bucket`, `machine-type`, `disk-size` | `prompt`, `branch`, `max-iterations` |
| `base-image` | `recreate`, `watch`, `setup` |

This means a typical GCP invocation goes from:
```bash
spinner spin --backend gcp --project my-proj --zone us-central1-a \
  --state-bucket my-bucket --machine-type e2-standard-4 \
  --image my-env --repo git@github.com:org/repo.git --prompt "Fix bug"
```
To just:
```bash
spinner spin --image my-env --repo git@github.com:org/repo.git --prompt "Fix bug"
```
With everything else coming from `.spinner.json`.

#### Phase 1c: Conditional Flag Validation

Backend-specific flags are grouped visually and validated at runtime:

```go
// Register flags in labeled groups
cmd.Flags().StringVar(&project, "project", "", "GCP project ID (GCP backend)")
cmd.Flags().StringVar(&zone, "zone", "", "GCP zone (GCP backend)")
cmd.Flags().StringVar(&machineType, "machine-type", "", "VM machine type (GCP backend)")
cmd.Flags().IntVar(&diskSize, "disk-size", 0, "Boot disk size in GB (GCP backend)")
cmd.Flags().StringVar(&stateBucket, "state-bucket", "", "GCS bucket for state (GCP backend)")

cmd.Flags().StringVar(&baseImage, "base-image", "", "Base Docker image (Docker backend)")
cmd.Flags().StringVar(&dockerfile, "dockerfile", "", "Path to Dockerfile (Docker backend)")
```

**Validation in RunE** — hard errors for mismatched flags:

```go
// Define which flags belong to which backend
gcpFlags := []string{"project", "zone", "machine-type", "disk-size", "state-bucket"}
dockerFlags := []string{"base-image", "dockerfile"}

backend := viper.GetString("backend")

// Reject flags from wrong backend
if backend != "gcp" {
    for _, f := range gcpFlags {
        if cmd.Flags().Changed(f) {
            return fmt.Errorf("--%s requires --backend gcp", f)
        }
    }
}
if backend != "docker" {
    for _, f := range dockerFlags {
        if cmd.Flags().Changed(f) {
            return fmt.Errorf("--%s requires --backend docker (or omit --backend)", f)
        }
    }
}
```

This approach:
- Registers all flags so they appear in `--help` (grouped by backend)
- Only validates flags that the user **explicitly set** (`cmd.Flags().Changed()`)
- Values from `.spinner.json` pass through Viper and aren't "changed" flags, so they don't trigger
  cross-backend errors — only explicit CLI flags do
- Help output uses Cobra's flag grouping annotations for clear sections

#### Phase 2: GCP Client Interface

Mirror Docker's two-layer pattern:

```go
// internal/gcp/client.go
package gcp

type Client interface {
    // Image operations
    CreateInstance(ctx context.Context, config InstanceConfig) (*Operation, error)
    GetInstance(ctx context.Context, project, zone, name string) (*Instance, error)
    StartInstance(ctx context.Context, project, zone, name string) (*Operation, error)
    StopInstance(ctx context.Context, project, zone, name string) (*Operation, error)
    ResetInstance(ctx context.Context, project, zone, name string) (*Operation, error)
    DeleteInstance(ctx context.Context, project, zone, name string) (*Operation, error)

    // Image operations
    CreateImage(ctx context.Context, project string, config ImageConfig) (*Operation, error)
    GetImage(ctx context.Context, project, name string) (*Image, error)
    DeleteImage(ctx context.Context, project, name string) (*Operation, error)

    // Operation waiting
    WaitZoneOperation(ctx context.Context, project, zone, op string) error
    WaitGlobalOperation(ctx context.Context, project, op string) error

    // Serial port (for boot/bake diagnostics only)
    GetSerialPortOutput(ctx context.Context, project, zone, name string, start int64) (*SerialPortOutput, error)

    // Storage (for state persistence + log streaming — GCS)
    WriteObject(ctx context.Context, bucket, object string, data []byte) error
    ReadObject(ctx context.Context, bucket, object string) ([]byte, error)
    ReadObjectRange(ctx context.Context, bucket, object string, offset int64) ([]byte, error)
    ObjectSize(ctx context.Context, bucket, object string) (int64, error)
    ObjectExists(ctx context.Context, bucket, object string) (bool, error)

    // Monitoring
    QueryTimeSeries(ctx context.Context, project string, query MetricsQuery) ([]MetricPoint, error)
}
```

`RealGCPClient` wraps the official SDK clients:
- `compute.InstancesClient` for VM operations
- `compute.ImagesClient` for image operations
- `compute.ZoneOperationsClient` / `compute.GlobalOperationsClient` for LROs
- `storage.Client` for GCS operations
- `monitoring.MetricClient` for metrics
- `logging.Client` for Cloud Logging

`MockGCPClient` uses testify mocks for unit testing.

#### Phase 3: GCP Provider Implementation

```go
// internal/gcp/gcp_provider.go
type Provider struct {
    client    Client
    project   string
    zone      string
    bucket    string // GCS bucket for state/scripts
}
```

##### Setup Flow (Image Baking)

**Spinner binary acquisition:** The bake script downloads the spinner binary from GitHub Releases.
Currently there is no release process — the Docker backend works around this by running `go build`
locally and `COPY`ing the binary into the image. For GCP, this won't work (can't COPY into a VM),
so a lightweight release pipeline is a prerequisite:

- **GoReleaser** (`.goreleaser.yaml`) — builds multi-platform binaries on tag push (already on main)
- **GitHub Actions** (`.github/workflows/release.yaml`) — triggers GoReleaser on `v*` tags (already on main)
- Produces: `spinner_{version}_{os}_{arch}.tar.gz` archives (e.g., `spinner_0.1.0_linux_amd64.tar.gz`)
- This also benefits Docker: `setup` could download the release binary instead of requiring Go on the host

The bake script then does:
```bash
SPINNER_VERSION=$(curl -sf https://api.github.com/repos/rickihastings/spinner/releases/latest | jq -r '.tag_name')
VERSION_NUM="${SPINNER_VERSION#v}"  # Strip leading 'v' for archive name
curl -fsSL "https://github.com/rickihastings/spinner/releases/download/${SPINNER_VERSION}/spinner_${VERSION_NUM}_linux_amd64.tar.gz" \
    -o /tmp/spinner.tar.gz
tar -xzf /tmp/spinner.tar.gz -C /usr/local/bin spinner
chmod +x /usr/local/bin/spinner
rm /tmp/spinner.tar.gz
```

**Full bake flow:**

1. Generate a bake startup script from `templates/scripts/gcp_bake.sh`:
   - Install git, curl, sudo, ca-certificates
   - Install GitHub CLI (gh)
   - Install Claude Code CLI
   - Download spinner binary from GitHub Releases (latest)
   - Write completion marker to serial port
   - Shut down the VM
2. Create a temporary VM with the bake script as `metadata.startup-script`
3. Wait for VM to reach `TERMINATED` state (script shuts down after install)
4. Create a custom image from the VM's boot disk
5. Delete the temporary VM

##### Create Flow (Launch Instance)

1. Generate runtime startup script from `templates/scripts/gcp_runtime.sh`:
   - Configure GitHub authentication
   - Clone repository
   - Handle branch checkout
   - Execute `spinner exec` (or keep alive if no prompt)
2. Create VM instance with:
   - Custom image (from setup)
   - Machine type from config (default: `e2-standard-2`)
   - Boot disk (default: 30 GB pd-balanced)
   - Metadata: `GITHUB_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`, `REPO_URL`, `PROMPT`, `MAX_ITERATIONS`, `BRANCH`,
     `SPINNER_LOG_BUCKET`, `SPINNER_INSTANCE_NAME`
   - Labels: `spinner-image={name}`, `spinner-repo={repo}`, `spinner-managed=true`
   - Service account with minimal scopes
   - External IP for outbound internet (GitHub, Claude API)
   - Startup script via metadata
3. Wait for VM to reach `RUNNING` state
4. Return instance with name and status

##### Instance Name Generation

Deterministic naming matching Docker's pattern:
```
spinner-{image}-{repo}[-{branch}]
```
GCP instance names: lowercase, max 63 chars, `[a-z]([-a-z0-9]*[a-z0-9])?`.

##### Logs Implementation — GCS Streaming

In Docker, `WatchLogs` reads `raw.log` directly from the host filesystem via a volume mount + fsnotify. For GCP,
the file lives on a remote VM, so we use GCS as the transport layer.

**LogWatcher Extraction (Docker control plane)**

The existing `internal/docker/logs.go` (`LogWatcher`, ~250 LOC) is extracted to `internal/logs/watcher.go`. It
provides fsnotify-based local file watching:

- `TailExistingLines(ctx, n)` — ring buffer over existing content
- `WatchLines(ctx, ch)` — stream new lines to channel via fsnotify

Docker's control plane `WatchLogs` continues to use this (reading from the host-side volume mount).
`internal/docker/logs.go` becomes a thin wrapper that imports from `internal/logs/`.

**VM Side: Direct Pipeline via Executor**

The executor already streams Claude's stdout through `io.TeeReader(stdout, logFile)` — the bytes are right
there in-process. Rather than setting up a separate fsnotify watcher on the same file, we tap directly into the
existing pipeline with `io.MultiWriter`:

```go
// internal/agent/claude/executor.go — inside Execute()
var writers []io.Writer
if logFile != nil {
    writers = append(writers, logFile)
}
if e.config.AdditionalWriter != nil {
    writers = append(writers, e.config.AdditionalWriter)
}
if len(writers) > 0 {
    reader = io.TeeReader(stdout, io.MultiWriter(writers...))
}
```

The exec loop creates the GCS sink and passes it to the executor:

```go
// internal/exec/loop.go — inside Run(), before iteration loop
if bucket := os.Getenv("SPINNER_LOG_BUCKET"); bucket != "" {
    instanceName := os.Getenv("SPINNER_INSTANCE_NAME")
    sink, err := logs.NewGCSSink(ctx, bucket, instanceName+"/logs/raw.log")
    if err == nil {
        executorConfig.AdditionalWriter = sink
        defer sink.Close()
    }
}
```

The `GCSSink` (`internal/logs/gcs_sink.go`) implements `io.Writer`:
- Receives bytes directly from the executor's TeeReader — no file watching, no extra goroutines
- Buffers writes and flushes to GCS periodically (every 2 seconds or on buffer threshold)
- Overwrites the GCS object with accumulated content each sync cycle
- Object size serves as the implicit byte offset for readers
- Uses `cloud.google.com/go/storage` directly — small, focused dependency
- Runs a background flush goroutine; `Close()` does a final flush

**Control Plane Side: GCS Log Reader**

GCP provider's `WatchLogs()` (`internal/gcp/logs.go`):
1. Polls GCS object at `{bucket}/{instance}/logs/raw.log` every 2 seconds
2. Checks object size vs last-read byte offset
3. If new content exists, performs a Range read (`Range: bytes={offset}-`)
4. Splits new bytes into lines, sends to channel
5. Updates offset

GCP provider's `Logs()`:
1. Downloads the full GCS log object
2. Returns as `io.ReadCloser`

```
VM (inside executor)               GCS (transport)           Control Plane
┌────────────────────┐         ┌──────────────────┐     ┌──────────────────────┐
│ Claude CLI stdout  │         │ {instance}/logs/  │     │ GCP Provider         │
│   │                │         │   raw.log         │     │   WatchLogs()        │
│   └─ TeeReader     │         │                   │     │     │                │
│        ├─ raw.log  │         │                   │     │   Poll GCS every 2s  │
│        └─ GCSSink ─────────→ │  flush every 2s   │ ──→ │   Range read offset  │
│     (io.MultiWriter)         │                   │     │   Split lines → ch   │
└────────────────────┘         └──────────────────┘     └──────────────────────┘
```

Serial port output is still used for **boot/bake diagnostics** (e.g., `SPINNER_BAKE_COMPLETE` marker during
setup), but agent log streaming uses GCS exclusively.

##### Metrics Implementation

Cloud Monitoring API:
- **CPU**: `compute.googleapis.com/instance/cpu/utilization` (always available, hypervisor-level)
- **Memory**: `compute.googleapis.com/instance/memory/balloon/ram_used` (requires ops agent)
- Poll interval: 60 seconds (Cloud Monitoring's minimum granularity for most metrics)
- Map to `provider.ContainerMetrics` (same struct, despite the name; the fields are generic)

##### State Persistence

GCS bucket (user-configured via `--state-bucket` flag — required for GCP backend):
- GCS bucket names are **globally unique** across all of Google Cloud, so we cannot auto-generate a safe default
- Users must provide their own bucket name (e.g., `--state-bucket my-org-spinner-state`)
- State path: `{instance-name}/state.json`
- Atomic writes via GCS object overwrite (GCS provides strong consistency)
- Read on VM start to resume iteration count
- The runtime startup script downloads state from GCS before running `spinner exec`
- `spinner exec` writes state to local `/state/state.json` as usual
- A periodic sync (or on-completion hook) uploads state back to GCS

##### VM Completion Behavior

The VM stays running after `spinner exec` completes — matching Docker's behavior where containers
remain alive via `tail -f /dev/null`. This enables:
- SSH access for debugging after completion
- Manual inspection of workspace state
- Consistent behavior across backends

The trade-off is cost: idle VMs bill per-minute unlike idle Docker containers. An `--auto-stop`
flag is planned as a follow-up to give users explicit control.

### Key Decisions

| Decision | Rationale |
|---|---|
| **SDK-only, no gcloud CLI** | Avoids runtime dependency; type safety; testable with mocks |
| **Image baking for setup** | Matches Docker's "build image once, run many" model; fast VM boot |
| **Metadata for secrets** | Same security model as Docker env vars; simple; single-tenant VMs |
| **GCS for log streaming** | Reuses LogWatcher code from Docker; exec owns the sync goroutine; no SSH or extra agents needed |
| **Serial port for bake diagnostics** | Always available; used only for boot/bake completion markers, not agent logs |
| **GitHub Releases for binary** | No GCS needed for binary distribution; works without Go on setup machine; benefits Docker too |
| **GCS for state** | Durable; accessible from control plane and VM; strong consistency |
| **Factory pattern for backend selection** | Clean DI; no changes to Provider interface; supports future backends |
| **Default VPC with external IP** | Simplest networking; outbound access for GitHub and Claude API |
| **e2-standard-2 default machine type** | 2 vCPU / 8 GB; good balance for agent workloads; cost-effective; configurable via `--machine-type` |
| **30 GB pd-balanced default disk** | SSD-backed; good performance; cheaper than pd-ssd; configurable via `--disk-size` |
| **Required `--state-bucket` flag** | GCS bucket names are globally unique; no safe auto-generated default; user must provide |
| **VM stays running after completion** | Docker parity; enables debugging; auto-stop is a follow-up feature |
| **Labels for resource management** | Cost attribution; automated cleanup scripts; resource identification |

### Architecture Diagram

```
Command Layer (cmd/)
       │
       ├── --backend docker  ──→  Docker Provider (internal/docker/)
       │                                 │
       │                                 └── DockerClient (Docker SDK)
       │
       └── --backend gcp     ──→  GCP Provider (internal/gcp/)
                                         │
                                         ├── GCPClient (GCP SDK)
                                         │      ├── InstancesClient
                                         │      ├── ImagesClient
                                         │      ├── OperationsClient
                                         │      ├── Storage Client
                                         │      ├── Logging Client
                                         │      └── Monitoring Client
                                         │
                                         └── GCS (state + scripts)

Provider Factory (internal/provider/factory.go)
       │
       ├── Register("docker", dockerCtor)
       └── Register("gcp", gcpCtor)
```

### GCP VM Configuration Detail

```go
// Compute instance creation config
type InstanceConfig struct {
    Name         string
    Project      string
    Zone         string
    MachineType  string            // e.g., "e2-standard-2"
    ImageProject string            // Project containing the custom image
    ImageName    string            // Custom image from setup
    DiskSizeGB   int64             // Boot disk size (default: 30)
    DiskType     string            // "pd-balanced", "pd-ssd", "pd-standard"
    Network      string            // VPC network (default: "default")
    Subnet       string            // Subnetwork (optional)
    ExternalIP   bool              // Assign ephemeral external IP (default: true)
    Metadata     map[string]string // Startup script + env vars
    Labels       map[string]string // Resource labels
    ServiceAccount string          // SA email (optional, uses default)
    Scopes       []string          // OAuth scopes for the SA
    Preemptible  bool              // Use spot/preemptible pricing
}
```

### Error Handling Strategy

GCP operations are async (Long-Running Operations). The pattern:

```go
func (c *RealGCPClient) CreateInstance(ctx context.Context, config InstanceConfig) error {
    op, err := c.instances.Insert(ctx, &computepb.InsertInstanceRequest{...})
    if err != nil {
        return fmt.Errorf("failed to create instance: %w", err)
    }

    // Wait for the operation to complete
    if err := op.Wait(ctx); err != nil {
        return fmt.Errorf("instance creation failed: %w", err)
    }

    return nil
}
```

All operations use context for cancellation and timeouts. Specific error codes are mapped:
- `codes.NotFound` → instance/image doesn't exist
- `codes.AlreadyExists` → duplicate instance name
- `codes.PermissionDenied` → missing IAM permissions
- `codes.ResourceExhausted` → quota exceeded

### Startup Script Templates

**gcp_bake.sh** (runs during setup, installs tooling, then shuts down):
```bash
#!/bin/bash
set -e

# Install system dependencies
apt-get update && apt-get install -y git curl sudo ca-certificates jq

# Install GitHub CLI
# ... (same as Docker template)

# Create spinner user
useradd -m -s /bin/bash spinner
echo "spinner ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Install Claude Code
su - spinner -c 'curl -fsSL https://claude.ai/install.sh | bash'

# Download spinner binary from latest GitHub Release
SPINNER_VERSION=$(curl -sf https://api.github.com/repos/rickihastings/spinner/releases/latest \
    | jq -r '.tag_name')
VERSION_NUM="${SPINNER_VERSION#v}"  # Strip leading 'v' for archive name
curl -fsSL "https://github.com/rickihastings/spinner/releases/download/${SPINNER_VERSION}/spinner_${VERSION_NUM}_linux_amd64.tar.gz" \
    -o /tmp/spinner.tar.gz
tar -xzf /tmp/spinner.tar.gz -C /usr/local/bin spinner
chmod +x /usr/local/bin/spinner
rm /tmp/spinner.tar.gz

# Copy startup script (passed via metadata)
curl -sf -H "Metadata-Flavor: Google" \
    "http://metadata.google.internal/computeMetadata/v1/instance/attributes/startup-script-runtime" \
    > /usr/local/bin/startup.sh
chmod +x /usr/local/bin/startup.sh

# Set up workspace directory
mkdir -p /home/spinner/workspace && chown spinner:spinner /home/spinner/workspace

# Signal completion and shut down
echo "SPINNER_BAKE_COMPLETE" > /dev/ttyS0
shutdown -h now
```

**gcp_runtime.sh** (runs on each spin, clones repo, runs exec):
```bash
#!/bin/bash
set -e

# Read metadata
META_URL="http://metadata.google.internal/computeMetadata/v1/instance/attributes"
GITHUB_TOKEN=$(curl -sf -H "Metadata-Flavor: Google" "$META_URL/GITHUB_TOKEN")
REPO_URL=$(curl -sf -H "Metadata-Flavor: Google" "$META_URL/REPO_URL")
PROMPT=$(curl -sf -H "Metadata-Flavor: Google" "$META_URL/PROMPT" || echo "")
BRANCH=$(curl -sf -H "Metadata-Flavor: Google" "$META_URL/BRANCH" || echo "")
MAX_ITERATIONS=$(curl -sf -H "Metadata-Flavor: Google" "$META_URL/MAX_ITERATIONS" || echo "100")
CLAUDE_CODE_OAUTH_TOKEN=$(curl -sf -H "Metadata-Flavor: Google" "$META_URL/CLAUDE_CODE_OAUTH_TOKEN")
SPINNER_LOG_BUCKET=$(curl -sf -H "Metadata-Flavor: Google" "$META_URL/SPINNER_LOG_BUCKET" || echo "")
SPINNER_INSTANCE_NAME=$(curl -sf -H "Metadata-Flavor: Google" "$META_URL/SPINNER_INSTANCE_NAME" || echo "")

export GITHUB_TOKEN REPO_URL PROMPT BRANCH MAX_ITERATIONS CLAUDE_CODE_OAUTH_TOKEN
export SPINNER_LOG_BUCKET SPINNER_INSTANCE_NAME

# Switch to spinner user and run startup
su - spinner -c "cd /home/spinner/workspace && /usr/local/bin/startup.sh"
```

The runtime script delegates to the existing `startup.sh` after setting environment from metadata,
maximizing code reuse between Docker and GCP backends.

### Risks & Trade-offs

| Trade-off | Analysis |
|---|---|
| **Image bake time vs boot speed** | Baking takes ~5-10 min but VMs boot in ~30s. Worth the upfront cost for repeated spins. |
| **GCS log streaming latency** | ~2s polling delay vs Docker's real-time fsnotify. Acceptable for watch TUI; GCS transport reuses LogWatcher code, avoiding 250 LOC duplication. |
| **GCS for state vs persistent disk** | GCS is simpler (no disk attach/detach) and works across VM recreations. Persistent disks are faster but more complex to manage. |
| **Metadata secrets vs Secret Manager** | Metadata is simpler (matches Docker model) but visible in GCP Console. Acceptable for single-tenant; document the trade-off. |
| **External IP vs Cloud NAT** | External IP is simpler but exposes VM on internet (mitigated by firewall denying all ingress). Cloud NAT is more secure but adds setup complexity. |
