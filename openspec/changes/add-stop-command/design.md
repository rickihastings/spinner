# Design: add-stop-command

## Technical Implementation Plan

### Component Map

| File                                    | Action | Notes                                       |
|-----------------------------------------|--------|---------------------------------------------|
| `cmd/stop.go`                           | CREATE | New command, mirrors `destroy.go` structure |
| `cmd/stop_test.go`                      | CREATE | Unit tests with mock factory                |
| `cmd/spin.go`                           | MODIFY | Update "To stop:" hint lines (Docker + GCP) |
| `docs/usage.md`                         | MODIFY | Add `spinner stop` section                  |
| `tests/integration/docker_stop_test.go` | CREATE | Docker integration test                     |
| `tests/integration/gcp_stop_test.go`    | CREATE | One GCP integration test                    |

### Approach

**`cmd/stop.go`** — Follow `destroy.go` exactly:

```
NewStopCommand(f *provider.Factory) *cobra.Command
  cobra.Command{Use: "stop <instance-name>...", Args: MinimumNArgs(1)}
  for each name:
    status := p.Status(ctx, name)
    if status == InstanceStatusNone → print "not found", continue
    if status == InstanceStatusStopped → print "already stopped", continue
    p.Stop(ctx, name) → print success or failure
  return aggregate error if any failed
```

Key behaviours:
- Multiple instance names supported (consistent with destroy)
- Already-stopped instances: skip gracefully with a warning (not an error)
- Not-found instances: skip with warning (not an error), consistent with destroy
- Aggregate error if any Stop() call fails

**`cmd/spin.go`** update:

```go
// Docker
fmt.Printf("To stop:    spinner stop %s\n", instance.Name)
// GCP
fmt.Printf("To stop:    spinner stop %s --backend gcp --project %s --zone %s\n",
    instance.Name, gcpProject, gcpZone)
```

### Key Decisions

1. **No `--force` flag** — `Stop()` already does a graceful stop at the provider level; force/kill is out of scope.
2. **Already-stopped = skip, not error** — avoids confusing failures in scripts that call stop defensively.
3. **Not-found = skip, not error** — consistent with destroys handling.
4. **No state-directory changes** — stop doesn't clean up state (destroy does). State persists so the instance can be
   restarted.
5. **GCP flags** — the "To stop:" hint includes `--backend gcp --project ... --zone ...` so it's copy-pasteable. The
   command itself requires these flags when `--backend gcp` is set (via existing `addGCPQueryFlags`).

### Risks / Trade-offs

- **GCP flag verbosity**: The stop hint for GCP becomes longer. Acceptable — it's already long for destroy.
- **No `spinner start`**: Stopping a Docker container and then wanting to restart it requires `docker start` still.
  Out of scope but worth noting for future work.