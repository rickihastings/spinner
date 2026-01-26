# Implementation Tasks: Migrate CLI from Node.js/TypeScript to Golang

## 1.0 Initialize Go project structure and prerequisites logic
- [x] 1.1 Initialize Go module with `go mod init github.com/rickihastings/spinner`
- [x] 1.2 Add dependencies: cobra, viper
- [x] 1.3 Create directory structure: cmd/, internal/docker/, internal/prerequisites/
- [x] 1.4 Port prerequisites checking logic from src/utils/prerequisites.ts to internal/prerequisites/prerequisites.go
- [x] 1.5 Verify prerequisites logic compiles and matches TypeScript behavior

## 2.0 Implement Docker utilities and Dockerfile generation
- [x] 2.1 Port Dockerfile template generation from src/utils/dockerfile.ts to internal/docker/dockerfile.go
- [x] 2.2 Port Docker image build logic from src/utils/docker.ts (buildImage function) to internal/docker/docker.go
- [x] 2.3 Port Docker container operations from src/utils/docker.ts (validatePrerequisites, generateContainerName with deterministic naming and sanitization, buildDockerRunCommand, executeDockerRun, verifyContainerStatus, checkContainerExists, restartContainer, removeContainer) to internal/docker/docker.go
- [x] 2.4 Verify all Docker utilities compile and use Go's os/exec for shell commands
- [x] 2.5 Test Dockerfile generation produces identical output to TypeScript version

## 3.0 Implement setup command with Cobra
- [x] 3.1 Create cmd/root.go with Cobra root command, --help flag, --version flag
- [x] 3.2 Create cmd/setup.go implementing setup command with --name, --base-image, --dockerfile flags
- [x] 3.3 Wire up setup command to use internal/prerequisites and internal/docker packages
- [x] 3.4 Implement validation: required --name flag, mutually exclusive --base-image and --dockerfile flags
- [x] 3.5 Match error message format and help text from src/App.tsx and src/commands/Setup.tsx
- [x] 3.6 Create main.go entry point that executes root command
- [x] 3.7 Build binary with `go build -o dist/spinner` and test setup command manually
- [x] 3.8 Run setup integration tests (tests/setup/run-all.sh) and fix any failures
- [x] 3.9 Verify all setup tests pass

## 4.0 Implement spin command with Cobra
- [x] 4.1 Create cmd/spin.go implementing spin command with --image, --repo, --prompt, --branch, --max-iterations, --recreate flags
- [x] 4.2 Wire up spin command to use internal/docker package
- [x] 4.3 Implement validation: required --image and --repo flags
- [x] 4.4 Match error message format and help text from src/App.tsx and src/commands/Spin.tsx
- [x] 4.5 Implement container reuse logic: check for existing containers, reuse running, restart stopped, or recreate based on --recreate flag
- [x] 4.6 Implement ralph-loop logic with prompt and branch handling
- [x] 4.7 Implement container lifecycle management and output display with appropriate messages for created/reused/restarted containers
- [x] 4.8 Build binary and test spin command manually
- [x] 4.9 Run spin integration tests (tests/spin/run-all.sh) and fix any failures
- [x] 4.10 Verify all spin tests pass including container reuse tests (14-reuse-running-container.sh, 15-restart-stopped-container.sh, 18-recreate-flag.sh)

## 5.0 Update build tooling and run full test suite
- [x] 5.1 Update package.json scripts: build → "go build -o dist/spinner", dev → "go build -o dist/spinner --watch" (or similar)
- [x] 5.2 Update package.json test script to reference Go binary: "test": "go build -o dist/spinner && bash tests/run.sh"
- [x] 5.3 Update .gitignore to include Go artifacts (dist/spinner, vendor/, go.sum changes)
- [x] 5.4 Run full integration test suite: `npm test`
- [x] 5.5 Debug and fix any remaining test failures
- [x] 5.6 Verify 100% test pass rate

## 6.0 Update documentation and project metadata
- [x] 6.1 Update openspec/project.md: change tech stack from TypeScript/Node.js to Go, update architecture patterns
- [x] 6.2 Update CLAUDE.md: change essential commands from `npm run build` to `go build -o dist/spinner`
- [x] 6.3 Update docs/usage.md: change development workflow commands to Go equivalents
- [x] 6.4 Update docs/standards.md: remove TypeScript-specific sections, add Go coding standards
- [x] 6.5 Create or update README.md with Go build instructions
- [x] 6.6 Delete TypeScript source: remove src/, tsconfig.json, .eslintrc.*, .prettierrc files
- [x] 6.7 Run final validation: `npm test` passes, binary works end-to-end, documentation accurate

## 7.0 Add Viper configuration support (future-proofing)
- [x] 7.1 Initialize Viper in cmd/root.go to read from environment variables
- [x] 7.2 Document environment variable configuration patterns in docs/
- [x] 7.3 Add example .env or configuration file support (optional, based on need)
- [x] 7.4 Test environment variable overrides work correctly
- [x] 7.5 Update documentation with Viper configuration examples
