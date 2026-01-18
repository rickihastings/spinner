# Change: Add setup command for building sandbox Docker images

## Why

The project needs a foundational CLI tool that can bootstrap Docker-based development sandbox environments. This command
will verify prerequisites and build a customized Docker image containing JDK, Node.js, git, and claude-code
for isolated development work.

## What Changes

- Initialize CLI project with TypeScript, ESLint, Prettier using Ink framework
- Add `setup` command with prerequisite checking (docker, git, claude)
- Build Docker images based on Ubuntu 22.04 with:
    - JDK downloaded from user-provided URL via --jvm-url flag
    - nvm with configurable Node.js LTS version
    - git and claude-code installed
- No tokens or secrets baked into images (mount at runtime)
- Integration test suite using bash scripts against real Docker

## Impact

- Affected specs: `cli-setup` (new capability)
- Affected code: New project initialization, all CLI infrastructure
- Dependencies: Docker, git, claude-code CLI must be installed on host
- **Note**: JVM is downloaded during Docker build from the URL provided via --jvm-url flag. Users must provide an architecture-appropriate JDK URL for their target platform.
