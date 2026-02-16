# Tasks: Migrate GCP SDK to CLI

## 1.0 Define plain Go types and update Client interface

- [x] 1.1 Add `GCPInstance`, `GCPImage`, `GCPMetadata`, `GCPMetadataItem`, `GCPDisk`, `GCPNetworkInterface`,
  `GCPAccessConfig`, `GCPServiceAccount` structs to `types.go` with JSON tags matching `gcloud --format=json` output
- [x] 1.2 Update `Client` interface in `client.go`: change `GetInstance` to return `*GCPInstance`, `GetImage` to return
  `*GCPImage`, `SetMetadata` to accept `*GCPMetadata`, `ListInstances` to return `[]*GCPInstance`
- [x] 1.3 Update `MockGCPClient` in `mock_client.go` to use new Go types
- [x] 1.4 Update `gcp_provider.go` to consume new Go types (replace `.GetName()` with `.Name`, `.GetStatus()` with
  `.Status`, `instance.Disks[0].GetSource()` with `instance.Disks[0].Source`, `.GetLabels()` with `.Labels`, metadata
  item field access, pointer assignments to direct assignments)
- [x] 1.5 Update all test files (`gcp_provider_test.go`, `client_test.go`, `image_test.go`, `metrics_test.go`) to
  construct `GCPInstance`/`GCPImage`/`GCPMetadata` instead of `computepb` types
- [x] 1.6 Remove all `computepb` and `cloud.google.com/go` imports from the `gcp` package
- [x] 1.7 Verify build and all tests pass with new types (mock client still implements interface)

## 2.0 Implement CLI-based Compute Engine operations

- [x] 2.1 Create `cli_runner.go` with `runGcloud(ctx, args...) ([]byte, error)` and
  `runGcloudJSON(ctx, target, args...) error` helpers
- [x] 2.2 Rewrite `NewRealGCPClient` — remove SDK client initialization; store only project/zone; verify `gcloud` on
  PATH
- [x] 2.3 Rewrite `CreateInstance` — `gcloud compute instances create` with flags for machine-type, image, disk-size,
  disk-type, network, subnet, external-ip, metadata, labels, service-account, scopes
- [x] 2.4 Rewrite `GetInstance` — `gcloud compute instances describe --format=json` + JSON unmarshal into `GCPInstance`
- [x] 2.5 Rewrite `SetMetadata` — `gcloud compute instances add-metadata --metadata=KEY=VALUE,...`
- [x] 2.6 Rewrite `StartInstance`, `StopInstance`, `ResetInstance`, `DeleteInstance` — simple
  `gcloud compute instances {start|stop|reset|delete} --quiet`
- [x] 2.7 Rewrite `ListInstances` — `gcloud compute instances list --filter=... --format=json`
- [x] 2.8 Rewrite `GetSerialPortOutput` — `gcloud compute instances get-serial-port-output --start=N`
- [x] 2.9 Rewrite `Close()` — no-op (no SDK clients to release)
- [x] 2.10 Add tests for `runGcloud` helper
- [x] 2.11 Verify build and all tests pass

## 3.0 Implement CLI-based Image operations

- [x] 3.1 Rewrite `CreateImage` —
  `gcloud compute images create --source-disk=... --source-disk-zone=... --labels=... --description=... --quiet --format=json`
- [x] 3.2 Rewrite `GetImage` — `gcloud compute images describe --format=json`
- [x] 3.3 Rewrite `DeleteImage` — `gcloud compute images delete --quiet`
- [x] 3.4 Verify build and all tests pass

## 4.0 Implement CLI-based GCS operations

- [ ] 4.1 Rewrite `WriteObject` — `gcloud storage cp - gs://bucket/object` (pipe data to stdin)
- [ ] 4.2 Rewrite `ReadObject` — `gcloud storage cat gs://bucket/object`
- [ ] 4.3 Rewrite `ReadObjectRange` — read full object via `gcloud storage cat` and slice from offset in Go
- [ ] 4.4 Rewrite `ObjectSize` — `gcloud storage ls -l gs://bucket/object --format=json` + parse size
- [ ] 4.5 Rewrite `ObjectExists` — `gcloud storage ls gs://bucket/object` (check exit code)
- [ ] 4.6 Rewrite `DeleteObjectsWithPrefix` — `gcloud storage rm gs://bucket/prefix/**`
- [ ] 4.7 Rewrite `object_writer.go` — use Client interface `WriteObject` instead of `storage.Client`
- [ ] 4.8 Verify build and all tests pass

## 5.0 Replace metadata.OnGCE() and add gcloud prerequisite check

- [ ] 5.1 Replace `metadata.OnGCE()` in `exec_hooks.go` with HTTP GET to
  `http://metadata.google.internal/computeMetadata/v1/` with `Metadata-Flavor: Google` header (1-second timeout)
- [ ] 5.2 Add `checkGcloudInstalled()` using `exec.LookPath("gcloud")`
- [ ] 5.3 Call `checkGcloudInstalled()` from `NewRealGCPClient`
- [ ] 5.4 Add tests for `isOnGCE()` and `checkGcloudInstalled()`
- [ ] 5.5 Verify build and all tests pass

## 6.0 Remove SDK dependencies and final validation

- [ ] 6.1 Run `go mod tidy` to remove unused SDK dependencies
- [ ] 6.2 Verify no `cloud.google.com/go` imports remain in `internal/backend/gcp/`
- [ ] 6.3 Verify no `google.golang.org/api/iterator` import remains
- [ ] 6.4 Run full test suite: `go test ./...`
- [ ] 6.5 Run `go build -o dist/spinner` and verify binary size reduction
- [ ] 6.6 Update spec (archive after deployment)
