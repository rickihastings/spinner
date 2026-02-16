# Design: Migrate GCP SDK to CLI

## Context

The GCP backend in `internal/backend/gcp/` uses the official GCP Go SDK (`cloud.google.com/go/compute/apiv1`,
`cloud.google.com/go/storage`, `cloud.google.com/go/compute/metadata`) for all operations. These SDK packages
bring 29+ transitive dependencies including gRPC, protobuf, OpenTelemetry, and envoy control plane libraries.

Scaling to additional cloud providers (AWS, Azure) using their respective SDKs would exponentially increase
the dependency graph. Using CLI tools with JSON parsing keeps the binary lean and establishes a consistent
pattern across all backends.

## Goals / Non-Goals

**Goals:**

- Replace all GCP SDK calls with `gcloud` CLI commands using `--format=json`
- Define plain Go types to replace protobuf types (`computepb.Instance`, `computepb.Image`, `computepb.Metadata`)
- Replace `metadata.OnGCE()` with direct HTTP call to GCE metadata server
- Remove all `cloud.google.com/go/*` dependencies from go.mod
- Add `gcloud` CLI prerequisite check
- Keep all existing tests passing via mock client
- Preserve Provider layer logic unchanged (only Client layer changes)

**Non-Goals:**

- Changing the Provider interface or its consumers
- Modifying command-layer code (`cmd/`)
- Adding new user-facing features
- Changing the Docker backend
- Supporting `gsutil` (deprecated in favor of `gcloud storage`)

## Technical Implementation Plan

### Component Map

| File | Action | Purpose |
|---|---|---|
| `internal/backend/gcp/types.go` | **modify** | Add GCPInstance, GCPImage, GCPMetadata structs with JSON tags |
| `internal/backend/gcp/client.go` | **modify** | Rewrite Client interface returns; replace RealGCPClient SDK with CLI |
| `internal/backend/gcp/cli_runner.go` | **create** | Shared `runGcloud()` helper for executing gcloud commands |
| `internal/backend/gcp/gcp_provider.go` | **modify** | Replace protobuf accessor methods with plain struct field access |
| `internal/backend/gcp/mock_client.go` | **modify** | Update type assertions to use GCPInstance/GCPImage/GCPMetadata |
| `internal/backend/gcp/exec_hooks.go` | **modify** | Replace `metadata.OnGCE()` with HTTP-based detection |
| `internal/backend/gcp/object_writer.go` | **modify** | Replace storage.Client with Client interface |
| `internal/backend/gcp/*_test.go` | **modify** | Replace computepb type construction with plain Go structs |
| `go.mod` | **modify** | Remove cloud.google.com/go/* dependencies after `go mod tidy` |

### New Go Types

Structs with JSON tags matching `gcloud --format=json` output:

```go
// GCPInstance represents a Compute Engine VM instance.
type GCPInstance struct {
    Name              string                `json:"name"`
    Status            string                `json:"status"`
    MachineType       string                `json:"machineType"`
    Zone              string                `json:"zone"`
    Disks             []GCPDisk             `json:"disks"`
    NetworkInterfaces []GCPNetworkInterface `json:"networkInterfaces"`
    Metadata          *GCPMetadata          `json:"metadata"`
    Labels            map[string]string     `json:"labels"`
    ServiceAccounts   []GCPServiceAccount   `json:"serviceAccounts"`
}

type GCPDisk struct {
    Source     string `json:"source"`
    Boot       bool   `json:"boot"`
    AutoDelete bool   `json:"autoDelete"`
}

type GCPNetworkInterface struct {
    Network       string            `json:"network"`
    Subnetwork    string            `json:"subnetwork"`
    AccessConfigs []GCPAccessConfig `json:"accessConfigs"`
}

type GCPAccessConfig struct {
    Name  string `json:"name"`
    Type  string `json:"type"`
    NatIP string `json:"natIP"`
}

type GCPMetadata struct {
    Fingerprint string            `json:"fingerprint"`
    Items       []GCPMetadataItem `json:"items"`
}

type GCPMetadataItem struct {
    Key   string `json:"key"`
    Value string `json:"value"`
}

type GCPServiceAccount struct {
    Email  string   `json:"email"`
    Scopes []string `json:"scopes"`
}

type GCPImage struct {
    Name        string            `json:"name"`
    Status      string            `json:"status"`
    SourceDisk  string            `json:"sourceDisk"`
    Description string            `json:"description"`
    Labels      map[string]string `json:"labels"`
}
```

### CLI Command Mapping

| Client Method | gcloud Command |
|---|---|
| `CreateInstance` | `gcloud compute instances create NAME --project=P --zone=Z --machine-type=T --image=I --image-project=P --boot-disk-size=NGB --boot-disk-type=T --network=N [--subnet=S] [--no-address] --metadata=K=V,... --labels=K=V,... [--service-account=SA --scopes=S,...] --quiet --format=json` |
| `GetInstance` | `gcloud compute instances describe NAME --project=P --zone=Z --format=json` |
| `SetMetadata` | `gcloud compute instances add-metadata NAME --project=P --zone=Z --metadata=K=V,...` |
| `StartInstance` | `gcloud compute instances start NAME --project=P --zone=Z --quiet` |
| `StopInstance` | `gcloud compute instances stop NAME --project=P --zone=Z --quiet` |
| `ResetInstance` | `gcloud compute instances reset NAME --project=P --zone=Z --quiet` |
| `DeleteInstance` | `gcloud compute instances delete NAME --project=P --zone=Z --quiet` |
| `ListInstances` | `gcloud compute instances list --project=P --zones=Z --filter=EXPR --format=json` |
| `GetSerialPortOutput` | `gcloud compute instances get-serial-port-output NAME --project=P --zone=Z --start=N` |
| `CreateImage` | `gcloud compute images create NAME --project=P --source-disk=D --source-disk-zone=Z --labels=K=V,... --description=DESC --quiet --format=json` |
| `GetImage` | `gcloud compute images describe NAME --project=P --format=json` |
| `DeleteImage` | `gcloud compute images delete NAME --project=P --quiet` |
| `WriteObject` | `gcloud storage cp - gs://BUCKET/OBJECT` (pipe stdin) |
| `ReadObject` | `gcloud storage cat gs://BUCKET/OBJECT` |
| `ReadObjectRange` | `gcloud storage cat gs://BUCKET/OBJECT` + slice from offset in Go |
| `ObjectSize` | `gcloud storage ls -l gs://BUCKET/OBJECT --format=json` |
| `ObjectExists` | `gcloud storage ls gs://BUCKET/OBJECT` (exit code 0=exists) |
| `DeleteObjectsWithPrefix` | `gcloud storage rm gs://BUCKET/PREFIX/**` |

### CLI Runner Helper

```go
// runGcloud executes a gcloud command and returns its stdout.
// On non-zero exit, returns an error wrapping stderr.
func runGcloud(ctx context.Context, args ...string) ([]byte, error) {
    cmd := exec.CommandContext(ctx, "gcloud", args...)
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("gcloud %s failed: %s: %w",
            args[0], strings.TrimSpace(stderr.String()), err)
    }

    return stdout.Bytes(), nil
}

// runGcloudJSON executes a gcloud command with --format=json and unmarshals into target.
func runGcloudJSON(ctx context.Context, target interface{}, args ...string) error {
    output, err := runGcloud(ctx, args...)
    if err != nil {
        return err
    }
    return json.Unmarshal(output, target)
}
```

### Approach

**Slice 1: Type System Migration**

Replace all `computepb` types with plain Go structs across the entire package. Key change patterns:

```go
// BEFORE (protobuf accessor style):
instance.GetName()              → instance.Name
instance.GetStatus()            → instance.Status
instance.GetLabels()            → instance.Labels
instance.Disks[0].GetSource()   → instance.Disks[0].Source
item.Value = strPtr(newVal)     → item.Value = newVal

// BEFORE (test construction):
&computepb.Instance{Name: strPtr("vm"), Status: &running}
// AFTER:
&GCPInstance{Name: "vm", Status: "RUNNING"}
```

The `updateMetadata` method in `gcp_provider.go:260-308` is the most complex conversion — it manipulates
metadata items in-place using pointer semantics. With plain structs, pointer assignments become direct field
assignments, and `append` uses value types instead of pointers.

**Slice 2: Compute CLI Implementation**

Replace SDK-based `RealGCPClient` methods with `gcloud` CLI calls. `CreateInstance` is the most complex
because it translates `instanceConfig` fields to CLI flags. For `SetMetadata`, `gcloud compute instances
add-metadata` handles fingerprint conflicts automatically with retries, simplifying the code vs. the current
SDK approach of explicitly reading and passing the fingerprint.

**Slice 3: Image CLI Implementation**

Straightforward mapping — image operations are simpler than instance operations.

**Slice 4: GCS CLI Implementation**

`WriteObject` needs stdin piping (`gcloud storage cp - gs://...`). For `ReadObjectRange`, reading the full
object and slicing in Go is sufficient — state files and log tails are small.

**Slice 5: Metadata Detection + Prerequisite**

Replace `metadata.OnGCE()` with HTTP GET to `http://metadata.google.internal/computeMetadata/v1/` (1-second
timeout, `Metadata-Flavor: Google` header). Add `exec.LookPath("gcloud")` prerequisite check.

**Slice 6: Dependency Cleanup**

`go mod tidy` removes `cloud.google.com/go/compute`, `cloud.google.com/go/storage`,
`cloud.google.com/go/compute/metadata` and their transitive deps. Some indirects may be retained if used by
the Docker SDK.

### object_writer.go Simplification

After migration, `gcsObjectWriter` uses the `Client` interface instead of `storage.Client`:

```go
type gcsObjectWriter struct {
    client Client
}

func newGCSObjectWriter(client Client) *gcsObjectWriter {
    return &gcsObjectWriter{client: client}
}

func (w *gcsObjectWriter) WriteObject(ctx context.Context, bucket, object string, data []byte) error {
    return w.client.WriteObject(ctx, bucket, object, data)
}

func (w *gcsObjectWriter) Close() error {
    return nil // no resources to release with CLI-based client
}
```

On the VM, `exec_hooks.go` creates a `RealGCPClient` (now a thin CLI wrapper, no SDK initialization) and
passes it to the writer.

### Key Decisions

| Decision | Rationale |
|---|---|
| Plain Go structs with JSON tags | Directly deserializable from `gcloud --format=json`; no mapping layer |
| `runGcloud` shared helper | DRY command execution, consistent error handling |
| `exec.LookPath` for prerequisite | Same pattern as Docker prerequisite check |
| HTTP-based GCE detection | Removes metadata SDK; standard GCE detection pattern |
| `add-metadata` not `set-metadata` | Handles fingerprint conflicts internally |
| Full object read for ReadObjectRange | State files and log tails are small |
| `gcloud storage` not `gsutil` | `gsutil` is deprecated |
| Type migration as first slice | All files compile against new types before CLI implementation |

### Error Handling

CLI errors follow a consistent pattern — `gcloud` exits non-zero and writes to stderr. Error messages
contain enough context (e.g., "The resource 'projects/.../instances/foo' was not found"). The existing
`isNotFoundError()` function in `gcp_provider.go` already does string matching for "not found" and "404",
which works identically with CLI errors.
