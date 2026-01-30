# Tasks: Migrate Bash Tests to Golang

## 1.0 Test Infrastructure Setup

**Goal:** Establish testing foundation and utilities before writing tests

- [x] 1.1 Install testify dependency (`go get github.com/stretchr/testify`)
- [x] 1.2 Create test utilities package structure (`tests/testutil/`)
- [x] 1.3 Implement Docker test helpers (`tests/testutil/docker.go`)
  - Image existence checking
  - Container status checking
  - Resource cleanup functions
- [x] 1.4 Implement CLI execution helpers (`tests/testutil/cli.go`)
  - Binary build function
  - Command execution with output capture
  - Exit code verification
- [x] 1.5 Implement test fixtures and utilities (`tests/testutil/fixtures.go`)
  - Generate unique test identifiers
  - Test data generation
  - Common assertions
- [x] 1.6 Create integration test package structure (`tests/integration/`)
- [x] 1.7 Verify test infrastructure with a simple smoke test

**Validation:** `go test ./tests/testutil/... -v` passes

**Dependencies:** None

---

## 2.0 Docker Client Interface Refactoring

**Goal:** Enable dependency injection for testability

- [x] 2.1 Define `DockerClient` interface in `internal/docker/client.go`
  - `BuildImage(ctx, config) error`
  - `RunContainer(ctx, config) (containerID string, error)`
  - `ImageExists(ctx, image) (bool, error)`
  - `ContainerExists(ctx, name) (bool, error)`
  - `RemoveContainer(ctx, name) error`
- [x] 2.2 Implement `RealDockerClient` struct that wraps exec.Command calls
- [x] 2.3 Create mock Docker client in `internal/docker/mock_client_test.go` using testify/mock
- [x] 2.4 Update `cmd/setup.go` to accept DockerClient via dependency injection
- [x] 2.5 Update `cmd/spin.go` to accept DockerClient via dependency injection
- [x] 2.6 Create command constructor functions (`NewSetupCommand`, `NewSpinCommand`)
- [x] 2.7 Update `cmd/root.go` to use new constructors with RealDockerClient

**Validation:** Project still builds and runs successfully (`go build -o dist/spinner && ./dist/spinner --help`)

**Dependencies:** None

---

## 3.0 Setup Command Unit Tests

**Goal:** Test setup command validation and logic without Docker

- [x] 3.1 Create `cmd/setup_test.go` with test suite skeleton
- [x] 3.2 Implement test for missing --name flag validation
- [ ] 3.3 Implement test for mutually exclusive flags (--base-image and --dockerfile)
- [ ] 3.4 Implement test for successful argument parsing with --name only
- [ ] 3.5 Implement test for argument parsing with --name and --base-image
- [ ] 3.6 Implement test for argument parsing with --name and --dockerfile
- [ ] 3.7 Implement test for non-existent Dockerfile path validation
- [ ] 3.8 Create table-driven test for all flag combinations

**Validation:** `go test ./cmd/setup_test.go -v` passes with 100% coverage of validation logic

**Dependencies:** Task 2.0 (Docker client interface)

---

## 4.0 Docker Build Logic Unit Tests

**Goal:** Test Docker build operations with mocked client

- [ ] 4.1 Create `internal/docker/build_test.go`
- [ ] 4.2 Implement test for successful image build with default base image
- [ ] 4.3 Implement test for image build with custom base image
- [ ] 4.4 Implement test for image build with custom Dockerfile
- [ ] 4.5 Implement test for build failure error handling
- [ ] 4.6 Implement test for Dockerfile generation logic
- [ ] 4.7 Implement test for startup script inclusion in Dockerfile
- [ ] 4.8 Create table-driven test for various build configurations

**Validation:** `go test ./internal/docker/build_test.go -v` passes

**Dependencies:** Task 2.0 (Docker client interface)

---

## 5.0 Setup Command Integration Tests

**Goal:** Verify setup command end-to-end with real Docker

- [ ] 5.1 Create `tests/integration/setup_test.go`
- [ ] 5.2 Implement integration test for successful image build (maps to bash test 04-successful-build.sh)
- [ ] 5.3 Implement integration test for image existence verification (maps to bash test 05-image-exists.sh)
- [ ] 5.4 Implement integration test for git installation verification (maps to bash test 06-git-verification.sh)
- [ ] 5.5 Implement integration test for claude-code installation verification (maps to bash test 07-claude-verification.sh)
- [ ] 5.6 Implement integration test for custom base-image flag (maps to bash test 03-base-image-flag.sh)
- [ ] 5.7 Add cleanup logic using `t.Cleanup()` for all integration tests
- [ ] 5.8 Verify all setup bash tests have Go equivalents (cross-reference)

**Validation:** `go test ./tests/integration/setup_test.go -v` passes and creates/cleans Docker images

**Dependencies:** Tasks 1.0 (test utilities), 2.0 (client interface)

---

## 6.0 Spin Command Unit Tests

**Goal:** Test spin command validation and logic without Docker

- [ ] 6.1 Create `cmd/spin_test.go` with test suite skeleton
- [ ] 6.2 Implement test for missing --image flag validation (maps to bash test 01-missing-image-flag.sh)
- [ ] 6.3 Implement test for missing --repo flag validation (maps to bash test 02-missing-repo-flag.sh)
- [ ] 6.4 Implement test for --prompt flag parsing
- [ ] 6.5 Implement test for --branch flag parsing
- [ ] 6.6 Implement test for --max-iterations flag parsing and default value (maps to bash test 14-max-iterations-default.sh)
- [ ] 6.7 Implement test for --recreate flag parsing (maps to bash test 18-recreate-flag.sh)
- [ ] 6.8 Implement test for --setup flag parsing and validation
- [ ] 6.9 Implement test for --setup with --base-image validation (maps to bash test 22-setup-flag-required-for-base-image.sh)
- [ ] 6.10 Implement test for --setup with mutually exclusive flags (maps to bash test 21-setup-mutually-exclusive-flags.sh)
- [ ] 6.11 Create table-driven test for all flag combinations

**Validation:** `go test ./cmd/spin_test.go -v` passes with 100% coverage of validation logic

**Dependencies:** Task 2.0 (Docker client interface)

---

## 7.0 Docker Run Logic Unit Tests

**Goal:** Test Docker container operations with mocked client

- [ ] 7.1 Create `internal/docker/run_test.go`
- [ ] 7.2 Implement test for successful container creation
- [ ] 7.3 Implement test for container naming logic (maps to bash test 07-container-naming.sh)
- [ ] 7.4 Implement test for container name sanitization (maps to bash test 17-name-sanitization.sh)
- [ ] 7.5 Implement test for deterministic naming with branch (maps to bash test 16-deterministic-naming-with-branch.sh)
- [ ] 7.6 Implement test for container reuse logic (maps to bash test 14-reuse-running-container.sh)
- [ ] 7.7 Implement test for container recreation logic (maps to bash test 15-restart-stopped-container.sh)
- [ ] 7.8 Create table-driven test for various naming scenarios

**Validation:** `go test ./internal/docker/run_test.go -v` passes

**Dependencies:** Task 2.0 (Docker client interface)

---

## 8.0 Prerequisites Unit Tests

**Goal:** Test prerequisite validation independently

- [ ] 8.1 Create `internal/prerequisites/prerequisites_test.go`
- [ ] 8.2 Implement test for GitHub token validation (maps to bash test 04-github-token-not-set.sh)
- [ ] 8.3 Implement test for Claude token validation (maps to bash test 05-claude-token-not-set.sh)
- [ ] 8.4 Implement test for Docker availability check
- [ ] 8.5 Implement test for all prerequisites passing
- [ ] 8.6 Create table-driven test for various prerequisite failure scenarios

**Validation:** `go test ./internal/prerequisites/prerequisites_test.go -v` passes

**Dependencies:** None

---

## 9.0 Spin Command Integration Tests - Basic Scenarios

**Goal:** Verify basic spin command end-to-end with real Docker

- [ ] 9.1 Create `tests/integration/spin_test.go`
- [ ] 9.2 Implement integration test for successful container creation (maps to bash test 06-successful-container-creation.sh)
- [ ] 9.3 Implement integration test for container naming verification (maps to bash test 07-container-naming.sh)
- [ ] 9.4 Implement integration test for container running status (maps to bash test 08-container-running.sh)
- [ ] 9.5 Implement integration test for repository cloning (maps to bash test 09-repository-cloned.sh)
- [ ] 9.6 Implement integration test for container exec capability (maps to bash test 11-container-exec.sh)
- [ ] 9.7 Implement integration test for non-existent image error (maps to bash test 03-non-existent-image.sh)
- [ ] 9.8 Add cleanup logic using `t.Cleanup()` for all integration tests

**Validation:** `go test ./tests/integration/spin_test.go -v -run TestSpin_Basic` passes

**Dependencies:** Tasks 1.0 (test utilities), 2.0 (client interface), 5.0 (setup creates test image)

---

## 10.0 Spin Command Integration Tests - Advanced Scenarios

**Goal:** Verify advanced spin command scenarios with real Docker

- [ ] 10.1 Implement integration test for prompt without branch (maps to bash test 12-prompt-without-branch.sh)
- [ ] 10.2 Implement integration test for branch without prompt (maps to bash test 13-branch-without-prompt.sh)
- [ ] 10.3 Implement integration test for container reuse (maps to bash test 14-reuse-running-container.sh)
- [ ] 10.4 Implement integration test for restart stopped container (maps to bash test 15-restart-stopped-container.sh)
- [ ] 10.5 Implement integration test for private repo clone (maps to bash test 15-private-repo-clone.sh)
- [ ] 10.6 Implement integration test for deterministic naming with branch (maps to bash test 16-deterministic-naming-with-branch.sh)
- [ ] 10.7 Implement integration test for name sanitization (maps to bash test 17-name-sanitization.sh)
- [ ] 10.8 Implement integration test for recreate flag (maps to bash test 18-recreate-flag.sh)
- [ ] 10.9 Implement integration test for .npmrc warning (maps to bash test 10-npmrc-warning.sh)
- [ ] 10.10 Implement integration test for --setup with --base-image (maps to bash test 19-setup-with-base-image.sh)
- [ ] 10.11 Implement integration test for --setup with --dockerfile (maps to bash test 20-setup-with-dockerfile.sh)
- [ ] 10.12 Implement integration test for --setup rebuilds existing image (maps to bash test 23-setup-rebuilds-existing-image.sh)

**Validation:** `go test ./tests/integration/spin_test.go -v` passes for all scenarios

**Dependencies:** Task 9.0 (basic integration tests)

---

## 11.0 Test Coverage Analysis & Gap Identification

**Goal:** Ensure comprehensive coverage and identify missing tests

- [ ] 11.1 Run coverage analysis: `go test -cover -coverprofile=coverage.out ./...`
- [ ] 11.2 Generate coverage report: `go tool cover -html=coverage.out -o coverage.html`
- [ ] 11.3 Review coverage report and identify gaps
- [ ] 11.4 Cross-reference all bash tests with Go tests (create mapping document)
- [ ] 11.5 Identify any bash test scenarios not covered by Go tests
- [ ] 11.6 Add additional tests for identified gaps
- [ ] 11.7 Document test coverage metrics in `tests/README.md`

**Validation:** Coverage report shows ≥80% coverage for cmd/ and internal/ packages

**Dependencies:** Tasks 3.0-10.0 (all test implementations)

---

## 12.0 CI/CD Integration & Documentation

**Goal:** Enable automated test execution and document testing approach

- [ ] 12.1 Update `.github/workflows/` (if exists) to run Go tests
- [ ] 12.2 Add GitHub Actions step for unit tests (`go test -short ./...`)
- [ ] 12.3 Add GitHub Actions step for integration tests (`go test ./tests/integration/...`)
- [ ] 12.4 Configure test timeout and Docker service in CI
- [ ] 12.5 Update `tests/README.md` with Go testing documentation
  - How to run unit tests
  - How to run integration tests
  - How to run with coverage
  - How to skip integration tests with `-short` flag
- [ ] 12.6 Update root `README.md` with testing section
- [ ] 12.7 Document migration from bash to Go tests (why, how, what changed)

**Validation:** CI pipeline runs and passes all Go tests; documentation is clear and complete

**Dependencies:** Task 11.0 (coverage analysis)

---

## 13.0 Bash Test Validation & Deprecation

**Goal:** Validate Go tests match bash test behavior and deprecate bash tests

- [ ] 13.1 Run all bash tests: `./tests/run.sh`
- [ ] 13.2 Run all Go tests: `go test ./...`
- [ ] 13.3 Compare results and verify equivalent coverage
- [ ] 13.4 Address any discrepancies or failing tests
- [ ] 13.5 Add deprecation notice to `tests/README.md` about bash tests
- [ ] 13.6 Move bash tests to `tests/bash-legacy/` directory (preserve for reference)
- [ ] 13.7 Update test documentation to reference Go tests as primary

**Validation:** All Go tests pass; bash tests deprecated but preserved; documentation updated

**Dependencies:** Tasks 3.0-12.0 (all Go tests implemented and validated)

---

## Execution Strategy

**Recommended approach:** Complete vertical slices in order (1.0 → 13.0)

**Parallel work opportunities:**
- Tasks 3.0 and 4.0 can run in parallel (both depend only on 2.0)
- Tasks 6.0, 7.0, and 8.0 can run in parallel (all depend only on 2.0)
- Tasks 9.0 and 10.0 can be split if desired (both depend on 1.0, 2.0, 5.0)

**Critical path:**
1. Infrastructure (1.0, 2.0) - Required for all tests
2. Setup tests (3.0, 4.0, 5.0) - Creates test environment for spin tests
3. Spin tests (6.0, 7.0, 8.0, 9.0, 10.0) - Majority of test coverage
4. Validation (11.0, 12.0, 13.0) - Ensure completeness

**Testing milestones:**
- After 5.0: Setup command fully tested
- After 10.0: All commands fully tested
- After 13.0: Migration complete and validated

## Test Count Summary

**Bash tests to migrate:**
- Setup: 7 tests (tests/setup/*.sh)
- Spin: 24 tests (tests/spin/*.sh)
- **Total: 31 tests**

**Notable new tests since proposal:**
- spin/19-setup-with-base-image.sh - Tests `--setup` flag with `--base-image`
- spin/20-setup-with-dockerfile.sh - Tests `--setup` flag with `--dockerfile`
- spin/21-setup-mutually-exclusive-flags.sh - Tests `--setup` flag validation
- spin/22-setup-flag-required-for-base-image.sh - Tests `--base-image` requires `--setup`
- spin/23-setup-rebuilds-existing-image.sh - Tests `--setup` rebuilds existing images

**Expected Go tests (minimum):**
- Setup unit tests: ~8-10 tests
- Setup integration tests: ~5-6 tests
- Docker build unit tests: ~6-8 tests
- Spin unit tests: ~12-15 tests (including --setup flag tests)
- Spin integration tests: ~18-22 tests (including --setup flag tests)
- Docker run unit tests: ~8-10 tests
- Prerequisites unit tests: ~4-6 tests
- **Total: ~61-77 tests** (approximately 2x bash tests due to unit + integration split)
