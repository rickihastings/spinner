# Proposal: add-list-command

## Summary

Add a `spinner list` command that discovers and displays all spinner-managed instances across all configured backends
(Docker, GCP). Extend the Provider interface with a `List()` method and add `spinner-managed=true` labels to Docker
containers to enable consistent label-based discovery across both backends.

## Motivation

Currently, spinner has no way to enumerate existing instances. Users must remember instance names and use
backend-specific tooling (`docker ps`, `gcloud compute instances list`) to find them. This is especially problematic
for GCP instances that incur ongoing cost — forgotten VMs can run up bills with no visibility from the spinner CLI.

The `watch` and `destroy` commands both require an instance name as input, but there's no discovery mechanism to find
that name. Users need a first-class command to answer: "what instances do I have, and what are they doing?"

## What Changes

- **New capability**: `cli-list` — a new Cobra command that queries all backends and renders a table of instances
  with rich execution state
- **New provider method**: `List(ctx) ([]InstanceInfo, error)` on the `Provider` interface — backend-agnostic
  instance discovery
- **Modified capability**: `docker-client` — add `spinner-managed=true` label to containers at creation time;
  add `ListContainers` to the Docker client interface for label-filtered listing
- **Modified capability**: `gcp-sandbox` — add `ListInstances` to the GCP client interface for label-filtered listing
- **No breaking changes**: Existing commands are unaffected. The Docker label addition is invisible to users.

## Decisions

1. **Auto-scan all backends**: `spinner list` queries every registered backend. Docker is always attempted. GCP is
   attempted only if project/zone configuration exists (from `.spinner.json`, env vars, or flags). Individual backend
   errors (e.g., Docker not running) are shown as warnings, not fatal errors.

2. **Label-based discovery**: Both backends use `spinner-managed=true` labels for filtering. Docker containers
   currently lack these labels, so we add them during `Create()`. Existing unlabeled containers are also discovered
   via the `spinner-` name prefix as a fallback.

3. **Rich state display**: The list output includes execution state from state files (iteration count, agent status,
   started_at, last_updated) alongside lifecycle status. This enables users to spot stale, completed, or errored
   instances at a glance.

4. **Backend filter flag**: Optional `--backend docker|gcp` flag to restrict listing to a single backend.

5. **Output format**: Default is a human-readable table. `--json` flag for machine-readable output.

## Impact

- **Affected specs**: `docker-client` (modified — labels + list), `gcp-sandbox` (modified — list method),
  `cli-list` (new)
- **Affected code**: `internal/provider/provider.go` (new types + interface method), `cmd/list.go` (new),
  `internal/backend/docker/` (labels + list), `internal/backend/gcp/` (list), `cmd/root.go` (register command)
- **Risk**: Medium — extends the Provider interface, requiring updates to both backend implementations and mocks.
  The Docker label change touches the container creation path but is purely additive.
