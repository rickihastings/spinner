# Spinner Test Suite

Integration tests for the Spinner CLI tool.

## Structure

```
tests/
├── run.sh              # Master test runner (runs all test suites)
├── setup/              # Setup command tests
│   ├── run-all.sh      # Setup test runner
│   └── *.sh            # Individual setup tests
├── spin/               # Spin command tests
│   ├── run-all.sh      # Spin test runner
│   └── *.sh            # Individual spin tests
└── README.md           # This file
```

## Running Tests

### Run All Tests

Run the complete test suite (setup and spin commands):

```bash
./tests/run.sh
```

### Run Individual Test Suites

Run only setup command tests:

```bash
./tests/setup/run-all.sh
```

Run only spin command tests:

```bash
./tests/spin/run-all.sh
```

### Run Individual Tests

Run a single setup test:

```bash
./tests/setup/01-missing-name-flag.sh
./tests/setup/03-successful-build.sh
# ... etc
```

Run a single spin test:

```bash
./tests/spin/01-missing-image-flag.sh
./tests/spin/05-successful-container-creation.sh
# ... etc
```

## Prerequisites

Before running tests, ensure:

1. Docker is installed and running
2. Git is installed
3. Claude CLI is installed
4. Node.js is installed
5. **Project is built**: Run `npm run build` from the project root

**Important**: Tests use the built CLI from `dist/cli.js`, not a globally installed version.

## Test Coverage

### Setup Command Tests
- Missing required flags validation
- Docker image building (creates `spinner:test-env`)
- Image existence verification
- Tool installation verification in container:
  - Java (JDK 21)
  - Node.js (via nvm)
  - npm
  - Git
  - Claude CLI

### Spin Command Tests
- Missing required flags validation
- Docker image existence validation
- SSH agent validation
- Container creation and naming
- Repository cloning
- Container lifecycle management
- .npmrc mounting and warnings
- Container exec capability

## Test Image

The setup tests create a Docker image named `spinner:test-env` that is **reused** by all spin tests. This image:
- Is created once during setup tests (test 03-successful-build.sh)
- Persists between test runs for efficiency
- Contains all required tools (Java, Node.js, Git, Claude)

To clean up the test image:

```bash
docker rmi spinner:test-env
```

## Cleanup

Spin tests automatically clean up any containers they create. If tests fail unexpectedly, you may need to manually clean up:

```bash
# Remove test containers
docker ps -a | grep -E "(Hello-World-|test-)" | awk '{print $1}' | xargs docker rm -f

# Remove test image (will be recreated on next test run)
docker rmi spinner:test-env
```
