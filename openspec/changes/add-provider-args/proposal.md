# Proposal: Add Provider Pass-Through Arguments

## Summary

Add a `--provider-args` repeatable flag to the `spin` and `setup` commands that passes raw arguments directly to the
underlying backend (Docker or GCP). Deprecate existing backend-specific tuning flags (`--machine-type`, `--disk-size`,
`--service-account`, `--bake-script`, `--base-image`, `--dockerfile`) in favor of `--provider-args`. Add
`.spinner.json` support for `provider-args` as a JSON string array.

## Motivation

Today, every backend-specific option requires a dedicated CLI flag, Viper binding, Options-map wiring, and spec
update. This has led to a growing surface area of GCP-specific flags (`--machine-type`, `--disk-size`,
`--service-account`, `--bake-script`) and Docker-specific flags (`--base-image`, `--dockerfile`). Each new flag
adds maintenance cost and coupling.

Users often need capabilities that aren't exposed yet - Docker volume mounts (`-v`), hostname overrides, network
settings, GCP labels, accelerator attachments, etc. A pass-through mechanism lets them self-serve without waiting
for Spinner to add explicit support. By deprecating existing backend-specific flags, we keep the CLI surface area
small and consistent.

## What Changes

1. **New `--provider-args` flag** on `spin` and `setup` commands - repeatable string flag that collects raw arguments.
2. **Backend forwarding** - Docker appends args to `docker run` / `docker build`; GCP appends to
   `gcloud compute instances create` / image bake commands.
3. **Conflict detection** - args that conflict with Spinner-managed flags (e.g. `--name`, `-d`, `--env-file` for
   Docker) are rejected with a clear error.
4. **`.spinner.json` support** - `"provider-args": ["--machine-type=e2-standard-2", "-v /data:/data"]` in config.
5. **Deprecation of backend-specific tuning flags** with warnings pointing users to `--provider-args`.

## What Does NOT Change

- **Infrastructure routing flags stay**: `--project`, `--zone`, `--state-bucket` remain first-class because they
  identify *where* to operate, not *how* to configure the backend. They're required flags with validation.
- The `--backend` flag stays (it's a Spinner routing concern, not a provider argument).
- The `Options map[string]string` internal pattern is unaffected for the remaining flags.

## Decisions

### 1. Flag UX: `--provider-args` (decided)

```bash
spinner spin --image default --repo <url> --provider-args="-v /data:/data" --provider-args="--network=host"
```

Explicit, self-documenting, works with Cobra's standard flag parsing. The `--` separator alternative fights with
Cobra's positional-args parsing.

### 2. Deprecate backend-specific tuning flags (decided)

The smaller the API surface, the better. The following flags will be deprecated with warnings, then removed:

**Flags to deprecate:**

| Flag | Backend | Current Default | Migration |
|------|---------|-----------------|-----------|
| `--machine-type` | GCP | `e2-standard-2` | `--provider-args="--machine-type=e2-standard-2"` |
| `--disk-size` | GCP | `30` GB | `--provider-args="--disk-size-gb=30"` |
| `--service-account` | GCP | (none) | `--provider-args="--service-account=..."` |
| `--bake-script` | GCP | (none) | Separate handling (see below) |
| `--base-image` | Docker | `ubuntu:22.04` | `--provider-args="--build-arg=BASE_IMAGE=..."` |
| `--dockerfile` | Docker | (none) | `--provider-args="-f /path/to/Dockerfile"` |

**Flags that STAY first-class:**

| Flag | Reason |
|------|--------|
| `--project` | Required GCP routing, has validation |
| `--zone` | Required GCP routing, has validation |
| `--state-bucket` | Required GCP routing, has validation |
| `--backend` | Spinner routing, not a provider arg |
| `--image`, `--repo`, etc. | Core Spinner flags |

**`--bake-script` note:** This flag is special - it provides a file that Spinner reads and injects into the bake
process, not a raw gcloud argument. It will remain as a Spinner flag since it's Spinner behavior, not a pass-through.

**Deprecation approach:**
- Phase 1 (this change): Add `--provider-args` with full support. Mark deprecated flags with `cmd.Flags().MarkDeprecated()`.
  Deprecated flags still work but print a warning with the migration command. Internally, deprecated flags are
  translated to provider-args before forwarding.
- Phase 2 (future change): Remove deprecated flags entirely.

### 3. No safety blocklist (decided)

Trust the user. They already have Docker/gcloud access. Managed-flag conflicts (e.g. `--name`, `-d`) are still
rejected since they'd break Spinner's internal wiring.

### 4. `.spinner.json` support (decided)

```json
{
  "backend": "gcp",
  "project": "my-project",
  "zone": "us-central1-a",
  "state-bucket": "my-bucket",
  "provider-args": [
    "--machine-type=e2-standard-2",
    "--disk-size-gb=30",
    "--service-account=my-sa@project.iam.gserviceaccount.com"
  ]
}
```

Viper natively supports binding a `StringSlice` flag to a JSON array config key. CLI `--provider-args` values are
**appended** to config file values (not replacing them), allowing the config file to set defaults while the CLI
adds one-off overrides.

**Precedence:** Config file provides base args, CLI `--provider-args` appends additional args. If the same flag
appears in both, the backend tool (docker/gcloud) uses its own last-wins semantics.

## Impact

- **Affected specs**: `cli-spin`, `cli-setup`, `gcp-sandbox`, `docker-client`
- **Affected code**: `cmd/spin.go`, `cmd/setup.go`, `cmd/helpers.go`, `internal/backend/docker/run.go`,
  `internal/backend/docker/docker_provider.go`, `internal/backend/gcp/gcp_provider.go`,
  `internal/provider/provider.go`
- **Breaking changes**: Deprecation warnings (non-breaking in Phase 1). Phase 2 removal would be breaking.
- **Risk**: Low. Pass-through args are appended to existing command construction; existing behavior unchanged
  during Phase 1.
