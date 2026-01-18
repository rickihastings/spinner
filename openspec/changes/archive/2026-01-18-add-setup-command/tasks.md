# Implementation Tasks

## 1. Project Setup
- [x] 1.1 Initialize package.json with yarn
- [x] 1.2 Configure TypeScript (tsconfig.json)
- [x] 1.3 Configure ESLint with TypeScript support
- [x] 1.4 Configure Prettier
- [x] 1.5 Install Ink and React dependencies
- [x] 1.6 Set up build scripts

## 2. CLI Infrastructure
- [x] 2.1 Create CLI entry point with Ink
- [x] 2.2 Implement command routing structure
- [x] 2.3 Add --help and --version flags

## 3. Prerequisite Checks
- [x] 3.1 Implement docker installation check (docker --version)
- [x] 3.2 Implement git installation check (git --version)
- [x] 3.3 Implement claude installation check (claude --version)
- [x] 3.4 Implement fail-fast error handling

## 4. Setup Command
- [x] 4.1 Define CLI flags: --name (required), --jvm-url (required), --node-version (optional)
- [x] 4.2 Validate required flags are provided
- [x] 4.3 Generate Dockerfile dynamically with JVM URL template variable
- [x] 4.4 Execute docker build with appropriate context
- [x] 4.5 Tag image as spinner:<name>

## 5. Dockerfile Template
- [x] 5.1 Base image: Ubuntu 22.04
- [x] 5.2 Download and install JDK from user-provided URL ({{JVM_URL}} template variable)
- [x] 5.3 Install nvm and specified Node.js version
- [x] 5.4 Install git via apt
- [x] 5.5 Install claude-code via curl install script
- [x] 5.6 Create startup.sh script template
- [x] 5.7 Copy startup.sh to /usr/local/bin/startup.sh in Docker image
- [x] 5.8 Make startup.sh executable (chmod +x)
- [x] 5.9 Set startup.sh as container CMD

## 6. Integration Tests
- [x] 6.1 Create test script directory structure
- [x] 6.2 Test prerequisite check failures and CLI validation
- [x] 6.3 Test --jvm-url is required
- [x] 6.4 Test successful Docker image build with --jvm-url parameter
- [x] 6.5 Verify container can start from built image
- [x] 6.6 Verify java, node, git, claude --version work inside container
- [x] 6.7 Verify startup.sh exists and is executable in built image
