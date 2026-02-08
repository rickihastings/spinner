# Tasks: add-destroy-command

## 1.0 Destroy command with tests

- [ ] 1.1 Create `cmd/destroy.go` with `NewDestroyCommand(f *provider.Factory)` following the `watch` command pattern:
  positional arg for instance name, `--backend` flag, GCP flags, `resolveAndValidateBackend()`, status check, and
  `Provider.Remove()` call
- [ ] 1.2 Create `cmd/destroy_test.go` with unit tests using mock provider: destroy running instance, destroy stopped
  instance, instance not found error, missing argument error
- [ ] 1.3 Verify build succeeds and all tests pass
