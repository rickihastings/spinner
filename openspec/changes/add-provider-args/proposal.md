# Proposal: Add Provider Pass-Through Arguments

## Summary

Add a `--provider-args` repeatable flag to the `spin` and `setup` commands that passes raw arguments directly to the
underlying backend (Docker or GCP). This enables users to leverage the full power of the underlying tool without
requiring Spinner to expose every possible option as a first-class flag.

## Motivation

Today, every backend-specific option requires a dedicated CLI flag, Viper binding, Options-map wiring, and spec
update. This has led to a growing surface area of GCP-specific flags (`--machine-type`, `--disk-size`,
`--service-account`, `--bake-script`) and Docker-specific flags (`--base-image`, `--dockerfile`). Each new flag
adds maintenance cost and coupling.

Users often need capabilities that aren't exposed yet - Docker volume mounts (`-v`), hostname overrides, network
settings, GCP labels, accelerator attachments, etc. A pass-through mechanism lets them self-serve without waiting
for Spinner to add explicit support.

## What Changes

1. **New `--provider-args` flag** on `spin` and `setup` commands - repeatable string flag that collects raw arguments.
2. **Backend forwarding** - Docker appends args to `docker run` / `docker build`; GCP appends to
   `gcloud compute instances create` / image bake commands.
3. **Conflict detection** - args that conflict with Spinner-managed flags (e.g. `--name`, `-d`, `--env-file` for
   Docker) are rejected with a clear error.

## What Does NOT Change

- All existing first-class flags remain and work as before. No deprecations in this change.
- The `Options map[string]string` internal pattern is unaffected.
- Config file (`.spinner.json`) support is not added for provider-args in this iteration (raw arg lists don't
  map cleanly to JSON key-value pairs).

## Design Discussion Points

### 1. Flag UX: `--provider-args` vs `--` separator

**Option A: `--provider-args` (proposed)**
```bash
spinner spin --image default --repo <url> --provider-args="-v /data:/data" --provider-args="--network=host"
```
Pros: Explicit, self-documenting, works with Cobra's standard flag parsing.
Cons: Verbose for multiple args.

**Option B: `--` double-dash separator**
```bash
spinner spin --image default --repo <url> -- -v /data:/data --network=host
```
Pros: Familiar Unix convention, concise.
Cons: Cobra consumes `--` for its own positional-args parsing, making this harder to implement cleanly. Also
ambiguous when `spin` takes positional args in the future.

**Recommendation:** Option A. It's explicit about intent and avoids Cobra parsing edge cases.

### 2. Should existing flags be replaced?

The user raised the idea of replacing `--machine-type` and similar flags with generic pass-through. There are
two viable strategies:

**Strategy A: Keep both (proposed for this change)**
First-class flags provide discoverability (`--help`), validation, defaults, and config-file support.
`--provider-args` is an escape hatch for advanced use cases. No migration needed.

**Strategy B: Deprecate backend-specific flags over time**
Move `--machine-type`, `--disk-size`, `--service-account` to `--provider-args` only. Reduces Spinner's flag
surface area but loses discoverability, defaults, and `.spinner.json` support.

**Recommendation:** Strategy A for now. First-class flags are better UX for common options. If the flag list
grows unmanageable, we can selectively deprecate rarely-used ones later.

### 3. Safety: should we blocklist dangerous args?

For Docker, args like `--privileged`, `--pid=host`, `--cap-add` can break sandbox isolation.

**Option A: No blocklist** - trust the user, they're already running `docker run` on their machine.
**Option B: Warn on dangerous args** - print a warning but proceed.
**Option C: Blocklist dangerous args** - reject known-dangerous args by default, allow override with `--force`.

**Recommendation:** Option A. Spinner users already have Docker access and could run `docker run` directly.
Adding a blocklist creates a false sense of security and maintenance burden. The `--provider-args` flag name
itself signals "you're on your own."

### 4. Scope: `spin` only or `spin` + `setup`?

Docker `setup` runs `docker build` - users might want `--build-arg`, `--no-cache`, `--platform`.
GCP `setup` bakes images - users might want `--labels`, `--network`.

**Recommendation:** Both `spin` and `setup`. The implementation is symmetric and the use cases are real.

## Impact

- **Affected specs**: `cli-spin`, `cli-setup`
- **Affected code**: `cmd/spin.go`, `cmd/setup.go`, `cmd/helpers.go`, `internal/backend/docker/run.go`,
  `internal/backend/docker/docker_provider.go`, `internal/backend/gcp/gcp_provider.go`
- **Breaking changes**: None. Purely additive.
- **Risk**: Low. Pass-through args are appended to existing command construction; existing behavior unchanged.
