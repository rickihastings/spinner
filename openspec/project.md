# Project Context

## Purpose
A CLI tool for running code and commands in isolated Docker containers. Provides a sandboxed execution environment for safely running untrusted or experimental code.

## Tech Stack
- TypeScript
- Node.js
- Docker (container runtime)
- yarn (package manager)

## Project Conventions

### Code Style
- ESLint for linting
- Prettier for code formatting
- Follow standard TypeScript conventions

### Architecture Patterns
- CLI-first design with clear command structure
- Modular architecture separating Docker operations from CLI interface

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
