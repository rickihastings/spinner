# Design: Config File Home Directory Traversal

## Technical Implementation Plan

### Component Map

| File | Action | Purpose |
|------|--------|---------|
| `cmd/root.go` | modify | Add upward directory traversal and `$HOME` fallback to config loading |
| `cmd/root_test.go` | modify | Add tests for traversal behavior |
| `docs/usage.md` | modify | Update config precedence documentation |

### Approach

#### 1. Config Search Algorithm

Replace the single `viper.AddConfigPath(".")` with an upward traversal loop:

```
start = cwd
current = start
while current != parent(current):          # stop at filesystem root
    if .spinner.json exists in current:
        load it and stop
    current = parent(current)

if no file found AND $HOME/.spinner.json exists:
    load $HOME/.spinner.json
```

This is a simple iterative walk. No recursion, no symlink following, no merging.

#### 2. Implementation in root.go

Viper's built-in `AddConfigPath` adds search paths but doesn't do upward traversal. Two options:

- **Option A**: Call `viper.AddConfigPath()` for each ancestor directory in order (nearest first). Viper searches paths
  in order and uses the first match. Simple, uses Viper's native API.
- **Option B**: Manually walk directories, find the file, then set `viper.SetConfigFile()` to the exact path.

**Decision: Option B** — It's explicit, avoids adding potentially hundreds of search paths on deep directory trees, and
makes the behavior easy to test by extracting the search into a pure function.

#### 3. Extracted Function

```go
// findConfigFile searches for .spinner.json starting from startDir,
// traversing up to the filesystem root, then checking homeDir as fallback.
// Returns the path to the first file found, or "" if none exists.
func findConfigFile(startDir, homeDir string) string
```

This is a pure function (no Viper dependency) that can be unit tested in isolation with temp directories.

#### 4. Updated init()

```go
func init() {
    viper.SetEnvPrefix("SPINNER")
    viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
    viper.AutomaticEnv()

    // Find .spinner.json by traversing up from cwd, with $HOME fallback
    cwd, _ := os.Getwd()
    home, _ := os.UserHomeDir()
    if configPath := findConfigFile(cwd, home); configPath != "" {
        viper.SetConfigFile(configPath)
        _ = viper.ReadInConfig()
    }

    // Secondary: .env file (not committed, local overrides)
    viper.SetConfigName(".env")
    viper.SetConfigType("env")
    viper.AddConfigPath(".")
    _ = viper.MergeInConfig()
}
```

### Key Decisions

1. **No merging** — The first `.spinner.json` found wins entirely. Merging home + repo configs adds complexity and
   surprising behavior (partial overrides of nested values). Users can always use env vars or CLI flags for per-repo
   overrides on top of home defaults.

2. **Home dir is fallback, not part of traversal** — The traversal goes from cwd to filesystem root. `$HOME` is only
   checked if no `.spinner.json` was found in the traversal. This means if `$HOME` happens to be an ancestor of cwd
   (which it usually is), it gets checked during traversal anyway. The explicit fallback covers edge cases where cwd is
   outside the home tree (e.g., `/tmp/work/`).

3. **No XDG support yet** — `~/.config/spinner/config.json` is a reasonable future addition but out of scope. The home
   dir file keeps the existing `.spinner.json` naming for consistency.

### Risks / Trade-offs

- **Filesystem access** — The traversal calls `os.Stat()` at each ancestor directory. On typical paths (5-15 levels
  deep) this is negligible. No risk of performance issues.
- **Surprising config pickup** — A `.spinner.json` in an unexpected ancestor directory could apply. This is the same
  pattern used by `.gitignore`, `.npmrc`, `.editorconfig`, etc., so users expect this behavior.
