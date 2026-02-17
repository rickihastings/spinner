# Tasks: Add Provider Pass-Through Arguments

## 1.0 Core plumbing, Docker spin support, and deprecation

- [ ] 1.1 Add `ProviderArgs []string` to `provider.SetupConfig` and `provider.CreateConfig`
- [ ] 1.2 Add `flagProviderArgs` constant, `--provider-args` flag to `cmd/spin.go`, bind to Viper for `.spinner.json` support
- [ ] 1.3 Deprecate `--machine-type`, `--disk-size`, `--service-account`, `--base-image`, `--dockerfile` flags on spin via `MarkDeprecated`
- [ ] 1.4 Add deprecation translation helper that converts deprecated flags to provider-args
- [ ] 1.5 Add `ExtraArgs []string` to `docker.spinConfig` and forward from `docker_provider.go`
- [ ] 1.6 Inject extra args into `buildDockerRunCommand` (before image argument)
- [ ] 1.7 Implement Docker managed-flag conflict detection for `docker run` args
- [ ] 1.8 Add unit tests for flag parsing, deprecation warnings, conflict detection, and Docker run arg injection
- [ ] 1.9 Build and verify (`go build`, `go test ./...`)

## 2.0 Setup command support and Docker build integration

- [ ] 2.1 Add `--provider-args` flag to `cmd/setup.go`, bind to Viper
- [ ] 2.2 Deprecate `--base-image`, `--dockerfile`, `--machine-type`, `--disk-size` flags on setup via `MarkDeprecated`
- [ ] 2.3 Add `ExtraArgs []string` to `docker.BuildConfig` and forward in `docker_provider.go`
- [ ] 2.4 Inject extra args into `docker build` command (before context directory)
- [ ] 2.5 Implement Docker managed-flag conflict detection for `docker build` args
- [ ] 2.6 Add unit tests for setup flag parsing and build arg injection
- [ ] 2.7 Build and verify

## 3.0 GCP backend support

- [ ] 3.1 Forward `ProviderArgs` in GCP provider's `Create` method to `instanceConfig`
- [ ] 3.2 Forward `ProviderArgs` in GCP provider's `Setup` method to `bakeConfig`
- [ ] 3.3 Append extra args to `gcloud compute instances create` commands
- [ ] 3.4 Implement GCP managed-flag conflict detection
- [ ] 3.5 Add unit tests for GCP arg forwarding and conflict detection
- [ ] 3.6 Build and verify

## 4.0 Help text and documentation

- [ ] 4.1 Update spin command long help to include `--provider-args` examples for both backends
- [ ] 4.2 Update setup command long help to include `--provider-args` examples
- [ ] 4.3 Update docs/usage.md `.spinner.json` examples with `provider-args`
- [ ] 4.4 Verify `--help` output displays correctly
