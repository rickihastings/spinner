# Proposal: add-stop-command

## Summary

Add a `spinner stop` CLI command that stops a running instance without destroying it. This command is an ergonomic
wrapper around what `spinner spin` currently instructs users to run manually (`docker stop <name>` or
`gcloud compute instances stop <name> ...`).

## Motivation

After `spinner spin` launches an instance, the output currently reads:

```
To stop:   docker stop <instance-name>
To stop:   gcloud compute instances stop <instance-name> --project <p> --zone <z>
```

Users must copy/paste a raw Docker or GCP CLI command. A native `spinner stop` command:

- Is consistent with `spinner destroy` (which already wraps provider removal)
- Hides backend details from users who don't care about Docker/GCP internals
- Enables future automation (scripts, CI) without backend-specific logic
- Keeps the `spin` output self-contained within the spinner CLI

## What Changes

- **NEW** `cmd/stop.go` — `spinner stop <instance-name>...` command
- **MODIFIED** `cmd/spin.go` — update "To stop:" hint to use `spinner stop <name>` (with backend/GCP flags appended
  for GCP)
- **NEW** `openspec/changes/add-stop-command/specs/cli-stop/spec.md` — new capability spec
- **NEW** tests: Docker unit/integration tests in `cmd/stop_test.go`; one GCP integration test in
  `tests/integration/`
- **MODIFIED** `docs/usage.md` — document the stop command

## No New Provider Methods Needed

`Provider.Stop(ctx, name)` already exists on the interface and is implemented by both Docker and GCP backends. This
change is purely a CLI layer addition.

## Impact

| Area               | Impact                            |
|--------------------|-----------------------------------|
| cli-spin spec      | MODIFIED — update "To stop:" hint |
| cli-stop spec      | ADDED — new capability            |
| Provider interface | None — Stop() already exists      |
| Docs               | Minor addition                    |
| Tests              | New unit + integration tests      |

## Out of Scope

- `spinner start` (restart a stopped instance) — separate concern, not requested
- Batch stop (stopping all instances) — keep it simple
- Force-stop / SIGKILL variant — not requested