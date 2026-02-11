# Tasks: Config File Home Directory Traversal

## 1.0 Config file traversal and home directory fallback

- [ ] 1.1 Implement `findConfigFile()` function in `cmd/root.go` that traverses from cwd upward and falls back to `$HOME`
- [ ] 1.2 Update `init()` in `cmd/root.go` to use `findConfigFile()` instead of `viper.AddConfigPath(".")`
- [ ] 1.3 Add unit tests for `findConfigFile()` covering: cwd match, ancestor match, home fallback, no file found
- [ ] 1.4 Add integration test verifying config precedence with traversal (repo config overrides home config)
- [ ] 1.5 Update `docs/usage.md` with new config search behavior and precedence
- [ ] 1.6 Verify build and all tests pass
