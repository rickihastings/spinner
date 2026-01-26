# Design: Migrate CLI from Node.js/TypeScript to Golang

## Context
Current implementation:
- ~760 lines of TypeScript across 7 files
- React (Ink) for terminal UI rendering
- Custom CLI argument parsing
- Synchronous execution with shell commands via Node.js child_process
- Integration tests in bash that invoke the compiled CLI

Project constraints:
- OpenSpec tooling requires package.json with @fission-ai/openspec dependency
- All 17+ integration tests must pass without modification
- Docker commands and container management are the core functionality
- Help text, version info, and error messages must match existing output

## Goals / Non-Goals

### Goals
- Self-contained Go binary (no Node.js runtime required for execution)
- Use Cobra for command structure and flag parsing
- Use Viper for environment variable configuration
- Maintain 100% behavioral compatibility with existing CLI
- Keep package.json for OpenSpec tooling
- All integration tests pass without modification

### Non-Goals
- Changing CLI behavior or output formats
- Rewriting integration tests
- Removing OpenSpec support
- Changing Docker interaction patterns
- Adding new features beyond the migration

## Technical Implementation Plan

### Component Map

**New Go files to create:**
- `main.go` - Entry point, initializes Cobra root command (replaces src/cli.tsx)
- `cmd/root.go` - Root command with --help, --version (replaces src/App.tsx)
- `cmd/setup.go` - Setup command implementation (replaces src/commands/Setup.tsx)
- `cmd/spin.go` - Spin command implementation (replaces src/commands/Spin.tsx)
- `internal/docker/docker.go` - Docker operations including container lifecycle management: create, check existence, restart, remove, and deterministic naming (replaces src/utils/docker.ts)
- `internal/docker/dockerfile.go` - Dockerfile generation (replaces src/utils/dockerfile.ts)
- `internal/prerequisites/prerequisites.go` - Prerequisite checks (replaces src/utils/prerequisites.ts)
- `go.mod` - Go module definition
- `go.sum` - Dependency checksums

**Files to modify:**
- `package.json` - Update build/test scripts, keep OpenSpec dependency
- `CLAUDE.md` - Update build instructions (yarn build → go build)
- `docs/setup.md` - Update development workflow commands
- `openspec/project.md` - Update tech stack from TypeScript/Node.js to Go
- `.gitignore` - Add Go binary artifacts

**Files to keep:**
- `tests/**/*` - All integration test scripts (unchanged)
- `openspec/**/*` - All OpenSpec files (unchanged)
- `docs/**/*` - Documentation (minor updates only)
- `.husky/**/*` - Git hooks (unchanged)

**Files to delete:**
- `src/**/*` - All TypeScript source files
- `tsconfig.json` - TypeScript configuration
- `.eslintrc.*` - ESLint configuration
- `.prettierrc` - Prettier configuration

### Approach

**Phase 1: Setup Go project structure**
1. Initialize Go module: `go mod init github.com/rickihastings/spinner`
2. Add dependencies: cobra, viper, testify (for future unit tests)
3. Create directory structure: `cmd/`, `internal/docker/`, `internal/prerequisites/`

**Phase 2: Implement core logic (internal packages)**
4. Port `src/utils/prerequisites.ts` → `internal/prerequisites/prerequisites.go`
5. Port `src/utils/dockerfile.ts` → `internal/docker/dockerfile.go`
6. Port `src/utils/docker.ts` → `internal/docker/docker.go`
   - Include container reuse logic: `CheckContainerExists()`, `RestartContainer()`, `RemoveContainer()`
   - Implement deterministic naming: `GenerateContainerName()` with `sanitizeComponent()` and `extractRepoName()` helpers
7. Focus on behavior parity, not line-by-line translation

**Phase 3: Implement CLI commands**
8. Create `cmd/root.go` with Cobra root command, --help, --version flags
9. Create `cmd/setup.go` with setup command and flags (--name, --base-image, --dockerfile)
10. Create `cmd/spin.go` with spin command and flags (--image, --repo, --prompt, --branch, --max-iterations, --recreate)
    - Implement container reuse logic: check existing container, reuse/restart/recreate as needed
    - Display appropriate messages for created/reused/restarted containers
11. Implement validation and error handling matching TypeScript behavior

**Phase 4: Wire up entry point**
12. Create `main.go` that invokes Cobra root command
13. Add Viper configuration for environment variables (future-proofing)

**Phase 5: Build and test**
14. Build binary: `go build -o dist/spinner`
15. Update package.json scripts: build → `go build`, test → existing test runner
16. Run full integration test suite: `bash tests/run.sh`
17. Debug and fix any behavioral differences

**Phase 6: Documentation and cleanup**
18. Update CLAUDE.md, docs/setup.md, openspec/project.md
19. Delete TypeScript source files and related configuration
20. Update .gitignore for Go artifacts
21. Final validation: all tests pass, binary works end-to-end

### Patterns to Follow

**Cobra command structure:**
```go
// Example from cmd/setup.go
var setupCmd = &cobra.Command{
    Use:   "setup",
    Short: "Build a Docker sandbox image",
    Long:  `Build a Docker sandbox image with custom base image or Dockerfile`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
    },
}
```

**Flag registration:**
```go
func init() {
    setupCmd.Flags().String("name", "", "Name for the Docker image (required)")
    setupCmd.MarkFlagRequired("name")
    rootCmd.AddCommand(setupCmd)
}
```

**Error handling:**
```go
// Match TypeScript error output format
if err != nil {
    return fmt.Errorf("Error: %s", err.Error())
}
```

**Shell command execution:**
```go
// Use exec.Command for docker commands
cmd := exec.Command("docker", "build", "-t", imageName, ".")
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
if err := cmd.Run(); err != nil {
    return fmt.Errorf("docker build failed: %w", err)
}
```

**Dockerfile generation:**
- Keep exact same Dockerfile template structure as TypeScript version
- See `src/utils/dockerfile.ts:10-50` for reference

### Key Decisions

**Decision**: Keep package.json with minimal scripts
**Rationale**: OpenSpec tooling requires npm ecosystem; package.json stays for `@fission-ai/openspec` dependency

**Decision**: Use Cobra + Viper instead of standard library flag package
**Rationale**: User explicitly requested Cobra and Viper; Cobra provides better subcommand structure and help text generation

**Decision**: Place business logic in `internal/` packages
**Rationale**: Follows Go project layout conventions; prevents external imports; separates CLI concerns from Docker logic

**Decision**: Output to `dist/spinner` matching current structure
**Rationale**: Tests expect binary at `dist/cli.js`, can easily change to `dist/spinner` without test modification

**Decision**: No TUI library (like bubbletea)
**Rationale**: Current implementation uses simple text output; React Ink was overkill; fmt.Print* is sufficient

**Decision**: Delete TypeScript source files immediately after Go implementation is validated
**Rationale**: Once tests pass, no need to maintain old code; git history provides rollback capability if needed

## Risks / Trade-offs

**Risk**: Integration tests fail due to subtle output format differences
→ **Mitigation**: Carefully compare error messages and help text; use identical wording

**Risk**: Docker command execution differs between Node.js exec and Go exec.Command
→ **Mitigation**: Test extensively with all test scenarios; ensure stdout/stderr handling matches

**Risk**: Flag parsing differences between custom parser and Cobra
→ **Mitigation**: Test all flag combinations (missing required, mutually exclusive, etc.)

**Risk**: Loss of React Ink's terminal UI capabilities
→ **Mitigation**: Current UI is simple text output; no complex TUI needed; Go's fmt package sufficient

**Trade-off**: Initial development time vs long-term maintenance benefits
→ Single binary, faster startup, no runtime dependencies outweigh migration effort

**Trade-off**: Go learning curve for contributors familiar with TypeScript
→ Go is simpler than TypeScript; excellent documentation; worth the investment

## Open Questions

None. Implementation strategy is clear. User has approved the general direction (Golang with Cobra/Viper).
