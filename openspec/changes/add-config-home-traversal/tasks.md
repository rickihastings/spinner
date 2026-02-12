# Tasks: Config File Home Directory Traversal

## 1.0 Config file traversal and home directory fallback

- [x] 1.1 Implement `findConfigFile()` function in `cmd/root.go` that traverses from cwd upward and falls back to `$HOME`
- [x] 1.2 Update `init()` in `cmd/root.go` to use `findConfigFile()` instead of `viper.AddConfigPath(".")`
- [x] 1.3 Add unit tests for `findConfigFile()` covering: cwd match, ancestor match, home fallback, no file found
- [x] 1.4 Add integration test verifying config precedence with traversal (repo config overrides home config)
- [x] 1.5 Update `docs/usage.md` with new config search behavior and precedence
- [x] 1.6 Verify build and all tests pass
