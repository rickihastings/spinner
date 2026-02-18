# Tasks: Add Provider Pass-Through Arguments

## 1.0 Core plumbing and Docker spin support ✅

- [x] 1.1 Add `ProviderArgs []string` to `provider.SetupConfig` and `provider.CreateConfig`
- [x] 1.2 Add `flagProviderArgs` constant, `--provider-args` flag to `cmd/spin.go`, bind to Viper for `.spinner.json` support
- [x] 1.3 Remove backend-specific flags from spin: `--machine-type`, `--disk-size`, `--service-account`, `--base-image`, `--dockerfile`
- [x] 1.4 Clean up `Options map[string]string` plumbing, `gcpOptionsFromViper()`, `dockerOptionsFromViper()` helpers
- [x] 1.5 Add `ExtraArgs []string` to `docker.spinConfig` and forward from `docker_provider.go`
- [x] 1.6 Inject extra args into `buildDockerRunCommand` (before image argument)
- [x] 1.7 Implement Docker managed-flag conflict detection for `docker run` args
- [x] 1.8 Add unit tests for flag parsing, conflict detection, and Docker run arg injection
- [x] 1.9 Build and verify (`go build`, `go test ./...`)

## 2.0 Setup command support and Docker build integration ✅

- [x] 2.1 Add `--provider-args` flag to `cmd/setup.go`, bind to Viper
- [x] 2.2 Remove backend-specific flags from setup: `--base-image`, `--dockerfile`, `--machine-type`, `--disk-size`
- [x] 2.3 Add `ExtraArgs []string` to `docker.BuildConfig` and forward in `docker_provider.go`
- [x] 2.4 Inject extra args into `docker build` command (before context directory)
- [x] 2.5 Implement Docker managed-flag conflict detection for `docker build` args
- [x] 2.6 Add unit tests for setup flag parsing and build arg injection
- [x] 2.7 Build and verify

## 3.0 GCP backend support ✅

- [x] 3.1 Forward `ProviderArgs` in GCP provider's `Create` method to instance creation
- [x] 3.2 Forward `ProviderArgs` in GCP provider's `Setup` method to image bake
- [x] 3.3 Clean up GCP provider code that read removed flags from `Options`
- [x] 3.4 Append extra args to `gcloud compute instances create` commands
- [x] 3.5 Implement GCP managed-flag conflict detection
- [x] 3.6 Add unit tests for GCP arg forwarding and conflict detection
- [x] 3.7 Build and verify

## 4.0 Help text and documentation ✅

- [x] 4.1 Update spin command long help to include `--provider-args` examples for both backends
- [x] 4.2 Update setup command long help to include `--provider-args` examples
- [x] 4.3 Update docs/usage.md `.spinner.json` examples with `provider-args`, remove references to removed flags
- [x] 4.4 Verify `--help` output displays correctly
