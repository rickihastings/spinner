# Design: add-list-command

## Technical Implementation Plan

### Component Map

| File | Action | Purpose |
| --- | --- | --- |
| `internal/provider/provider.go` | modify | Add `InstanceInfo` struct and `List()` to Provider interface |
| `internal/backend/docker/client.go` | modify | Add `ListContainers` method to Docker Client interface |
| `internal/backend/docker/docker_provider.go` | modify | Implement `List()` using label filter + name prefix fallback |
| `internal/backend/docker/run.go` | modify | Add `spinner-managed=true` label to container creation |
| `internal/backend/gcp/client.go` | modify | Add `ListInstances` method to GCP Client interface |
| `internal/backend/gcp/gcp_provider.go` | modify | Implement `List()` using label filter |
| `cmd/list.go` | create | New Cobra command with table and JSON output |
| `cmd/list_test.go` | create | Unit tests with mock providers |
| `internal/backend/docker/docker_provider_test.go` | modify | Tests for Docker List() |
| `internal/backend/gcp/gcp_provider_test.go` | modify | Tests for GCP List() |

### Approach

#### 1. Provider Interface Extension

Add a new `InstanceInfo` struct that carries everything the list command needs in a single call per backend.
This avoids N+1 queries (list, then metadata per instance):

```go
type InstanceInfo struct {
    Name          string
    Status        InstanceStatus
    Backend       string         // "docker" or "gcp"
    Image         string         // environment/image name
    Repo          string         // repository (from labels/metadata)
    Branch        string         // git branch (from state or metadata)
    Agent         string         // AI model (if available)
    Iteration     int            // current iteration from state file
    MaxIterations int            // max iterations configured
    AgentStatus   string         // from state: running/completed/rate_limited/error
    StartedAt     *time.Time     // when execution started
    LastUpdated   *time.Time     // last state file update
}
```

Add `List(ctx context.Context) ([]InstanceInfo, error)` to the `Provider` interface.

#### 2. Docker Discovery

**Label addition** (in `run.go`): When building the `docker run` command, add `--label spinner-managed=true`.
This is a one-line change in the argument builder. Existing containers won't have this label, so we also need
a fallback.

**ListContainers** (in `client.go`): New method on the Docker Client interface using the Docker SDK:

```go
ListContainers(ctx context.Context, filters map[string][]string) ([]container.Summary, error)
```

Implementation uses `client.ContainerList()` with a filter. The provider calls it twice:
1. Primary: filter by label `spinner-managed=true`
2. Fallback: filter by name prefix `spinner-` (catches pre-label containers)
3. Deduplicate by container ID

**State reading**: For each discovered container, read `~/.spinner/<name>/state/state.json` from the host.
Parse the state file to extract iteration, agent status, timestamps. If the file doesn't exist, those fields
are zero-valued (instance exists but hasn't run yet or state was cleaned up).

#### 3. GCP Discovery

**ListInstances** (in `client.go`): New method on the GCP Client interface:

```go
ListInstances(ctx context.Context, project, zone string, filter string) ([]*computepb.Instance, error)
```

Uses `InstancesClient.List()` with filter `labels.spinner-managed=true`. GCP already applies this label
during `Create()`, so no creation-path changes needed.

**State reading**: For each discovered VM, read state from GCS at `gs://{bucket}/{name}/state.json`.
The state bucket is provided via `--state-bucket` flag or `.spinner.json` config. If no bucket is configured,
state fields are zero-valued (we still show the instance, just without execution state).

**Metadata extraction**: VM metadata items (`ANTHROPIC_MODEL`, `MAX_ITERATIONS`, `BRANCH`) and labels
(`spinner-image`, `spinner-repo`) provide instance info without needing the state file.

#### 4. Multi-Backend Orchestration (cmd/list.go)

The command iterates over all registered backends from the Factory:

```
1. Get available backends from factory.Available()  → ["docker", "gcp"]
2. For each backend:
   a. Try factory.Create(backend) to get a provider
   b. If provider creation fails (e.g., Docker not installed, GCP not configured):
      → Print warning, skip backend, continue
   c. Call provider.List(ctx)
   d. Collect results
3. Merge all InstanceInfo across backends
4. Sort by backend, then status (running first), then name
5. Render as table or JSON
```

GCP backend creation requires project/zone. The command resolves these from:
1. `--project`/`--zone` flags (if provided)
2. `.spinner.json` config
3. `SPINNER_GCP_PROJECT`/`SPINNER_GCP_ZONE` env vars

If none are available, GCP is silently skipped (most users only use Docker).

#### 5. Output Format

**Table (default)**:

```
BACKEND  NAME                        STATUS   STATE       ITER   AGE        LAST UPDATE
docker   spinner-default-my-repo     running  running     5/100  2h ago     5m ago
docker   spinner-default-other-repo  stopped  completed   42/50  1d ago     23h ago
gcp      spinner-prod-big-refactor   running  rate_limit  12/100 3h ago     1h ago
gcp      spinner-prod-old-task       running  running     88/100 2d ago     2d ago  ⚠ stale
```

- `STATUS` = instance lifecycle (running/stopped)
- `STATE` = agent execution state (running/completed/rate_limited/error)
- `ITER` = current/max iterations
- `AGE` = time since started_at
- `LAST UPDATE` = time since last_updated (with stale warning if >2h for running instances)

**JSON** (`--json`): Array of `InstanceInfo` objects with full timestamps.

### Key Decisions

- **InstanceInfo vs existing types**: We create a dedicated `InstanceInfo` struct rather than reusing
  `Instance` + `InstanceMetadata` because: (a) List needs state file data that neither existing type
  carries, (b) a single struct avoids N+1 per-instance metadata queries, (c) the list use case has
  different data needs than watch/status.

- **Label fallback for Docker**: We filter by label first, then by name prefix, to handle the transition
  period where existing containers don't have labels. After a few releases, the name-prefix fallback could
  be removed.

- **No --cleanup flag in v1**: The `destroy` command already handles removal. The list command focuses
  on visibility. A future `spinner list --stale --destroy` or `spinner cleanup` could compose list + destroy
  but that's a separate proposal.

- **GCP zone scoping**: List only queries the configured zone, not all zones. Cross-zone listing would
  require AggregatedList which is slower and more complex. Users who use multiple zones can filter with
  `--zone`.

### Risks / Trade-offs

- **Provider interface change**: Adding `List()` breaks the interface — both backends and all mocks must
  be updated. This is a one-time cost and the method is natural for the interface.
- **Docker SDK dependency**: The Docker provider currently uses CLI exec for some operations. `ListContainers`
  uses the SDK directly (like `ContainerExists` already does). This is consistent with the `docker-client`
  spec direction.
- **GCP state without bucket**: If `--state-bucket` isn't configured, GCP instances show lifecycle info
  but no execution state. This is acceptable — the instance is still visible, which is the primary goal.
