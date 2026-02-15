# Tasks: add-destroy-command

- [x] 1.0 Destroy command with state cleanup, multi-instance support, and spin output update

- [x] 1.1 Create `cmd/destroy.go` with `NewDestroyCommand(f *provider.Factory)` following the `watch` command pattern:
  `cobra.MinimumNArgs(1)` for one or more instance names, `--backend` flag, GCP flags,
  `resolveAndValidateBackend()`, loop over args calling `p.Status()` + `p.Remove()` + state directory cleanup,
  per-instance output, aggregate error on failure
- [x] 1.2 Update `cmd/spin.go` management instructions to replace backend-specific "To remove:" lines with
  `spinner destroy <instance-name>` for both Docker and GCP output blocks
- [x] 1.3 Create `cmd/destroy_test.go` with unit tests using mock provider: destroy single running instance,
  destroy single stopped instance, destroy multiple instances, instance not found error, partial failure with
  multiple instances, missing argument error, state directory cleanup verification
- [x] 1.4 Verify build succeeds and all tests pass
