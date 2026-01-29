# Proposal: Migrate Bash Tests to Golang

## Overview

Migrate the existing bash integration tests to Go's native testing framework, providing both unit tests and integration tests for the CLI commands.

## Problem Statement

Currently, the project has ~29 bash scripts for testing setup and spin commands. While functional, these tests:
- Are not integrated with Go's tooling ecosystem
- Cannot leverage Go's testing features (table-driven tests, benchmarks, test coverage)
- Require bash-specific knowledge and debugging
- Are harder to maintain alongside the Go codebase
- Don't test internal business logic in isolation

## Proposed Solution

Implement a hybrid testing strategy with both unit tests and integration tests:

### Unit Tests
- Test command validation logic, argument parsing, and configuration handling
- Mock Docker operations via interfaces to enable fast, isolated tests
- Co-locate with source code (`cmd/*_test.go`, `internal/**/*_test.go`)
- Use table-driven tests for comprehensive scenario coverage

### Integration Tests
- Test end-to-end CLI behavior with real Docker operations
- Verify actual container creation, image building, and cleanup
- Organize in `tests/integration/` directory
- Maintain 1:1 coverage with existing bash tests at minimum

### Code Refactoring
- Extract Docker operations behind interfaces (e.g., `DockerClient` interface)
- Enable dependency injection in commands for testability
- Maintain existing behavior while improving code structure

## Success Criteria

1. All 29+ bash test scenarios have equivalent Go tests
2. Tests can be run via `go test ./...`
3. Integration tests verify real Docker behavior
4. Unit tests provide fast feedback without Docker dependencies
5. Test coverage reports available via `go test -cover`
6. CI/CD can run tests reliably
7. Bash tests can be deprecated after migration validation

## Impact

- **Developer Experience**: Better integration with Go tooling and IDEs
- **CI/CD**: Native Go test execution, coverage reporting
- **Maintainability**: Tests alongside code, easier refactoring
- **Speed**: Fast unit tests for quick feedback loops
- **Quality**: More comprehensive test coverage with table-driven tests

## Non-Goals

- Performance testing or benchmarking (future work)
- E2E tests beyond current bash test coverage (future work)
- Testing Docker itself (we trust Docker's behavior)

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Breaking existing functionality during refactoring | Keep bash tests until Go tests validated; incremental migration |
| Integration tests flaky due to Docker state | Implement robust cleanup and isolation in test utilities |
| Tests too slow if all are integration tests | Use unit tests with mocks for majority of scenarios |
| Missed test scenarios during migration | 1:1 mapping review, add tests for identified gaps |

## Open Questions

None - all clarifications received from user.

## References

- [Testing Cobra CLI Commands](https://gianarb.it/blog/golang-mockmania-cli-command-with-cobra)
- Current bash tests: `tests/setup/` and `tests/spin/`
- Go testing best practices: https://go.dev/doc/tutorial/add-a-test
