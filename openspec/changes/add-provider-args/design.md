# Design: Add Provider Pass-Through Arguments

## Technical Implementation Plan

### Component Map

| File | Action | Purpose |
|------|--------|---------|
| `internal/provider/provider.go` | modify | Add `ProviderArgs []string` to `CreateConfig` and `SetupConfig` |
| `cmd/helpers.go` | modify | Add `flagProviderArgs` constant, deprecation translation helpers |
| `cmd/spin.go` | modify | Add `--provider-args` flag, deprecate tuning flags, pass to `CreateConfig` |
| `cmd/setup.go` | modify | Add `--provider-args` flag, deprecate tuning flags, pass to `SetupConfig` |
| `internal/backend/docker/run.go` | modify | Accept and inject extra args into `docker run` command |
| `internal/backend/docker/docker_provider.go` | modify | Forward `ProviderArgs` from `CreateConfig`/`SetupConfig` |
| `internal/backend/docker/build.go` | modify | Add `ExtraArgs` to `BuildConfig` |
| `internal/backend/docker/client.go` | modify | Forward extra args in `BuildImage` |
| `internal/backend/gcp/gcp_provider.go` | modify | Forward `ProviderArgs` to instance creation and image bake |
| `internal/backend/gcp/types.go` | modify | Add `ExtraArgs []string` to `instanceConfig` and `bakeConfig` |
| `internal/backend/gcp/client.go` | modify | Append extra args to `gcloud` commands |
| `cmd/spin_test.go` | modify | Add tests for `--provider-args`, deprecation warnings, conflict detection |
| `cmd/setup_test.go` | modify | Add tests for `--provider-args` flag on setup |
| `internal/backend/docker/run_test.go` | modify | Test extra arg injection into docker run args |

### Approach

**1. Provider config plumbing + Docker spin support (vertical slice 1)**

Add `ProviderArgs []string` to both `SetupConfig` and `CreateConfig` in `provider.go`. This is the universal
carrier - backends receive raw string args and decide how to use them.

In the CLI layer (`cmd/spin.go`, `cmd/setup.go`), add a repeatable `--provider-args` flag:
```go
cmd.Flags().StringSliceVar(&providerArgs, flagProviderArgs, []string{},
    "Extra arguments passed directly to the backend (repeatable)")
```

Bind it to Viper so `.spinner.json` support comes for free:
```go
_ = viper.BindPFlag(flagProviderArgs, cmd.Flags().Lookup(flagProviderArgs))
```

With `.spinner.json`:
```json
{
  "provider-args": ["--machine-type=e2-standard-2", "--disk-size-gb=30"]
}
```

CLI args are appended to config file args. Viper's `GetStringSlice` returns the config file values, and we
append any CLI-provided values on top.

**2. Conflict detection (part of slice 1)**

Before forwarding args, validate they don't conflict with Spinner-managed args. Each backend defines its own
blocklist of managed flags:

```go
// Docker managed flags (used in docker run)
var dockerManagedRunFlags = []string{
    "-d", "--detach",
    "--name",
    "--label",
    "--env-file",
}
```

The validation function scans the raw args list and rejects any that start with a managed flag prefix. This
catches `--name=foo`, `--name foo`, and `-d` forms.

**3. Deprecation of existing flags (part of slice 1)**

Use Cobra's `MarkDeprecated` to emit warnings:
```go
cmd.Flags().MarkDeprecated(flagMachineType,
    "use --provider-args=\"--machine-type=e2-standard-2\" instead")
```

When deprecated flags are set, translate them to provider-args internally before forwarding:
```go
func translateDeprecatedFlags(cmd *cobra.Command) []string {
    var args []string
    if mt := viper.GetString(flagMachineType); mt != "" {
        args = append(args, "--machine-type="+mt)
    }
    // ... etc
    return args
}
```

These translated args are prepended to the provider-args list so explicit `--provider-args` values win via
last-wins semantics.

**4. Docker integration (vertical slice 2)**

For `docker run`: extra args are inserted into the args slice *before* the image name (which must be last):

```go
// In buildDockerRunCommand:
// ... existing mounts and flags ...

// Append provider pass-through args before the image
dockerArgs = append(dockerArgs, config.ExtraArgs...)

// Image must be last
dockerArgs = append(dockerArgs, config.Image)
```

For `docker build`: extra args are inserted before the context directory argument.

**5. GCP integration (vertical slice 3)**

For GCP instance creation: extra args are appended to the `gcloud compute instances create` command.

For GCP image bake: extra args are appended to the `gcloud compute instances create` command used for the
bake VM.

### Key Decisions

1. **`[]string` not `map[string]string`**: Provider args are raw CLI fragments, not key-value pairs. A `-v`
   Docker flag takes a complex value (`host:container:mode`), and some flags are boolean (`--privileged`).
   A string slice preserves the original form exactly.

2. **Conflict detection is a blocklist, not parsing**: We don't try to fully parse Docker/gcloud flag syntax.
   We check for known managed flag prefixes (e.g., anything starting with `--name` or `--label`). Simple and
   sufficient.

3. **`.spinner.json` uses JSON array**: `"provider-args": ["--machine-type=e2-standard-2"]`. Viper binds
   StringSlice flags to JSON arrays natively. CLI values are appended to config file values.

4. **Args go before the positional argument**: Both `docker run` and `gcloud compute instances create` require
   the positional arg (image name / instance name) to be last. Extra args are inserted before it.

5. **Deprecated flag translation is prepended**: When `--machine-type=X` is used (deprecated), it becomes
   `--machine-type=X` in the provider-args list, prepended before explicit `--provider-args`. This means
   explicit provider-args win via last-wins semantics of docker/gcloud.

6. **`--bake-script` stays**: It's Spinner behavior (reads a file and injects contents), not a raw gcloud arg.

### Risks / Trade-offs

- **User can break things**: Passing `--rm` to Docker would make the container self-destruct, confusing
  Spinner's lifecycle management. This is acceptable - `--provider-args` is explicitly an escape hatch.
- **No validation of arg correctness**: We don't check if `-v /foo:/bar` is a valid Docker flag. Docker/gcloud
  will report the error, which is fine.
- **Deprecation of defaults**: `--machine-type` currently defaults to `e2-standard-2` and `--disk-size` to 30.
  After deprecation, users who relied on these defaults need to add them to `.spinner.json` or pass them
  explicitly. The deprecation warning message will include the default value to ease migration.
