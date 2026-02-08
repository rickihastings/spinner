# Proposal: add-destroy-command

## Summary

Add a `spinner destroy <instance-name>` command that forcefully removes an instance regardless of its current state
(running, stopped, or exited). This is a dedicated destructive operation abstracted behind the existing
`Provider.Remove()` method, working identically across Docker and GCP backends.

## Motivation

Currently, the only way to remove an instance is:

1. Via `spin --recreate`, which couples removal with re-creation
2. Manually running backend-specific commands (`docker rm -f`, `gcloud compute instances delete`)

Users need a first-class CLI command to tear down instances they no longer need without re-creating them or dropping
down to backend-specific tooling. The `spin` command already prints "To remove: docker rm ..." in its output,
indicating this is a recognized user workflow that deserves direct CLI support.

## What Changes

- **New capability**: `cli-destroy` — a new Cobra command at `cmd/destroy.go`
- **No provider changes**: `Provider.Remove()` already exists and is implemented for both Docker and GCP backends
- **No breaking changes**: This is purely additive

## Impact

- **Affected specs**: None modified. New `cli-destroy` spec added.
- **Affected code**: `cmd/destroy.go` (new), `cmd/destroy_test.go` (new)
- **Risk**: Low — uses existing `Provider.Remove()` with no changes to the provider interface

## Open Questions for Discussion

1. **State directory cleanup**: When destroying an instance, should the host-side state directory
   (`~/.spinner/<instance-name>/state/`) also be removed? Or preserved for potential reuse if the user re-spins
   the same config? Recommendation: **preserve by default**, add `--clean` flag to opt into state removal.

2. **Confirmation prompt**: The current codebase pattern (e.g., `spin --recreate`) performs destructive operations
   without confirmation. Recommendation: **no prompt** — consistent with existing patterns. Users invoke `destroy`
   intentionally.

3. **Multiple instances**: Should `destroy` accept multiple names (`spinner destroy foo bar`)? Recommendation:
   **single instance only** for v1 — keeps it simple, can be extended later.
