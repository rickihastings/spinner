# Project Context

## Purpose
A CLI tool for running code and commands in isolated Docker containers. Provides a sandboxed execution environment for safely running untrusted or experimental code.

## Tech Stack
- Go (Golang)
- Cobra (CLI command structure)
- Viper (configuration management)
- Docker (container runtime)
- yarn (package manager for OpenSpec tooling only)

## Project Conventions

### Code Style
- gofmt for code formatting
- Follow standard Go conventions and idioms
- Keep functions focused and testable

### Architecture Patterns
- CLI-first design with Cobra command structure
- Modular architecture with internal packages separating Docker operations from CLI interface
- Business logic in internal/ packages (internal/docker, internal/prerequisites)
- Command implementations in cmd/ packages

### Testing Strategy
- Minimal testing requirements currently
- Add tests as needed for critical functionality

### Git Workflow
- Trunk-based development
- Commit directly to main with small, frequent changes
- Keep commits focused and atomic

## Domain Context
- Docker containers provide process isolation and resource limits
- Sandbox environments must handle cleanup of containers after execution
- Security considerations for running untrusted code

## Important Constraints
- Requires Docker to be installed and running on the host system
- Container cleanup must be reliable to prevent resource leaks

## External Dependencies
- Docker Engine API for container management
