## 1. Implementation

- [ ] 1.1 Create `spin.ts` command file with CLI flag definitions (--image, --repo)
- [ ] 1.2 Implement flag validation (both required flags present)
- [ ] 1.3 Verify Docker image exists locally using `docker image inspect`
- [ ] 1.4 Check SSH_AUTH_SOCK environment variable and socket existence
- [ ] 1.5 Check if ~/.npmrc exists and prepare mount configuration
- [ ] 1.6 Generate unique container name from repository name + timestamp/random suffix
- [ ] 1.7 Build Docker run command with:
  - [ ] Detached mode (-d)
  - [ ] SSH agent socket mount (-v $SSH_AUTH_SOCK:/ssh-agent)
  - [ ] SSH_AUTH_SOCK environment variable (-e SSH_AUTH_SOCK=/ssh-agent)
  - [ ] .npmrc mount if file exists (-v ~/.npmrc:/root/.npmrc)
  - [ ] Container name (--name)
  - [ ] Startup command: `bash -c "git clone <repo-url> /workspace && tail -f /dev/null"`
- [ ] 1.8 Execute docker run command and capture output
- [ ] 1.9 Display success message with container management instructions
- [ ] 1.10 Handle errors and display appropriate messages
- [ ] 1.11 Add spin command to CLI application entry point

## 2. Integration Tests

- [ ] 2.1 Create test/spin/ directory for integration test scripts
- [ ] 2.2 Write test script: missing --image flag exits with error code 1
- [ ] 2.3 Write test script: missing --repo flag exits with error code 1
- [ ] 2.4 Write test script: non-existent Docker image exits with error
- [ ] 2.5 Write test script: SSH_AUTH_SOCK not set exits with error
- [ ] 2.6 Write test script: successful container creation with valid flags
- [ ] 2.7 Write test script: container is named correctly based on repo name
- [ ] 2.8 Write test script: container is running after spin command completes
- [ ] 2.9 Write test script: repository is cloned into /workspace inside container
- [ ] 2.10 Write test script: SSH agent socket is mounted at /ssh-agent
- [ ] 2.11 Write test script: .npmrc is mounted when file exists
- [ ] 2.12 Write test script: warning displayed when .npmrc missing
- [ ] 2.13 Write test script: container keeps running after clone (status is "Up")
- [ ] 2.14 Write test script: container can be exec'd into with bash
- [ ] 2.15 Create master test runner script that executes all tests and cleans up containers
