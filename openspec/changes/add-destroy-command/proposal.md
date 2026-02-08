# Proposal: add-destroy-command

## Summary

Add a `spinner destroy <instance-name>...` command that forcefully removes one or more instances regardless of their
current state (running, stopped, or exited), including cleanup of the host-side state directory
(`~/.spinner/<instance-name>/`). Update the `spin` command output to reference `spinner destroy` instead of
backend-specific removal commands.

## Motivation

Currently, the only way to remove an instance is:

1. Via `spin --recreate`, which couples removal with re-creation
2. Manually running backend-specific commands (`docker rm -f`, `gcloud compute instances delete`)

Users need a first-class CLI command to tear down instances they no longer need without re-creating them or dropping
down to backend-specific tooling. The `spin` command currently prints "To remove: docker rm ..." in its output,
indicating this is a recognized user workflow that deserves direct CLI support.

## What Changes

- **New capability**: `cli-destroy` — a new Cobra command at `cmd/destroy.go`
- **Modified capability**: `cli-spin` — update management instructions to reference `spinner destroy`
- **No provider changes**: `Provider.Remove()` already exists and is implemented for both Docker and GCP backends
- **No breaking changes**: This is purely additive (the spin output change is cosmetic)

## Decisions

1. **State directory cleanup**: Automatically remove `~/.spinner/<instance-name>/` (state + logs) on destroy.
   Clean slate by default.
2. **Multiple instances**: Accept one or more names (`spinner destroy foo bar baz`). Iterate through all,
   report per-instance success/failure, continue on individual failures.
3. **No confirmation prompt**: Consistent with existing patterns (`spin --recreate` removes without asking).
4. **Spin output update**: Replace backend-specific "To remove:" instructions with `spinner destroy <name>`.

## Impact

- **Affected specs**: `cli-spin` (modified — management instructions), `cli-destroy` (new)
- **Affected code**: `cmd/destroy.go` (new), `cmd/destroy_test.go` (new), `cmd/spin.go` (modified output)
- **Risk**: Low — uses existing `Provider.Remove()` with no changes to the provider interface
