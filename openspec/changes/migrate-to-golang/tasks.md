# Implementation Tasks: Migrate CLI from Node.js/TypeScript to Golang

## 1.0 Initialize Go project structure and prerequisites logic
- [ ] 1.1 Initialize Go module with `go mod init github.com/rickihastings/spinner`
- [ ] 1.2 Add dependencies: cobra, viper
- [ ] 1.3 Create directory structure: cmd/, internal/docker/, internal/prerequisites/
- [ ] 1.4 Port prerequisites checking logic from src/utils/prerequisites.ts to internal/prerequisites/prerequisites.go
- [ ] 1.5 Verify prerequisites logic compiles and matches TypeScript behavior

## 2.0 Implement Docker utilities and Dockerfile generation
- [ ] 2.1 Port Dockerfile template generation from src/utils/dockerfile.ts to internal/docker/dockerfile.go
- [ ] 2.2 Port Docker image build logic from src/utils/docker.ts (buildImage function) to internal/docker/docker.go
- [ ] 2.3 Port Docker container operations from src/utils/docker.ts (validatePrerequisites, generateContainerName, buildDockerRunCommand, executeDockerRun, verifyContainerStatus) to internal/docker/docker.go
- [ ] 2.4 Verify all Docker utilities compile and use Go's os/exec for shell commands
- [ ] 2.5 Test Dockerfile generation produces identical output to TypeScript version

## 3.0 Implement setup command with Cobra
- [ ] 3.1 Create cmd/root.go with Cobra root command, --help flag, --version flag
- [ ] 3.2 Create cmd/setup.go implementing setup command with --name, --base-image, --dockerfile flags
- [ ] 3.3 Wire up setup command to use internal/prerequisites and internal/docker packages
- [ ] 3.4 Implement validation: required --name flag, mutually exclusive --base-image and --dockerfile flags
- [ ] 3.5 Match error message format and help text from src/App.tsx and src/commands/Setup.tsx
- [ ] 3.6 Create main.go entry point that executes root command
- [ ] 3.7 Build binary with `go build -o dist/spinner` and test setup command manually
- [ ] 3.8 Run setup integration tests (tests/setup/run-all.sh) and fix any failures
- [ ] 3.9 Verify all setup tests pass

## 4.0 Implement spin command with Cobra
- [ ] 4.1 Create cmd/spin.go implementing spin command with --image, --repo, --prompt, --branch, --max-iterations flags
- [ ] 4.2 Wire up spin command to use internal/docker package
- [ ] 4.3 Implement validation: required --image and --repo flags
- [ ] 4.4 Match error message format and help text from src/App.tsx and src/commands/Spin.tsx
- [ ] 4.5 Implement ralph-loop logic with prompt and branch handling
- [ ] 4.6 Implement container lifecycle management and output display
- [ ] 4.7 Build binary and test spin command manually
- [ ] 4.8 Run spin integration tests (tests/spin/run-all.sh) and fix any failures
- [ ] 4.9 Verify all spin tests pass

## 5.0 Update build tooling and run full test suite
- [ ] 5.1 Update package.json scripts: build → "go build -o dist/spinner", dev → "go build -o dist/spinner --watch" (or similar)
- [ ] 5.2 Update package.json test script to reference Go binary: "test": "go build -o dist/spinner && bash tests/run.sh"
- [ ] 5.3 Update .gitignore to include Go artifacts (dist/spinner, vendor/, go.sum changes)
- [ ] 5.4 Run full integration test suite: `yarn test`
- [ ] 5.5 Debug and fix any remaining test failures
- [ ] 5.6 Verify 100% test pass rate

## 6.0 Update documentation and project metadata
- [ ] 6.1 Update openspec/project.md: change tech stack from TypeScript/Node.js to Go, update architecture patterns
- [ ] 6.2 Update CLAUDE.md: change essential commands from `yarn build` to `go build -o dist/spinner`
- [ ] 6.3 Update docs/setup.md: change development workflow commands to Go equivalents
- [ ] 6.4 Update docs/standards.md: remove TypeScript-specific sections, add Go coding standards
- [ ] 6.5 Create or update README.md with Go build instructions
- [ ] 6.6 Delete TypeScript source: remove src/, tsconfig.json, .eslintrc.*, .prettierrc files
- [ ] 6.7 Run final validation: `yarn test` passes, binary works end-to-end, documentation accurate

## 7.0 Add Viper configuration support (future-proofing)
- [ ] 7.1 Initialize Viper in cmd/root.go to read from environment variables
- [ ] 7.2 Document environment variable configuration patterns in docs/
- [ ] 7.3 Add example .env or configuration file support (optional, based on need)
- [ ] 7.4 Test environment variable overrides work correctly
- [ ] 7.5 Update documentation with Viper configuration examples
