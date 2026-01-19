<!-- OPENSPEC:START -->

# OpenSpec Instructions

These instructions are for AI assistants working in this project.

Always open `@/openspec/AGENTS.md` when the request:

- Mentions planning or proposals (words like proposal, spec, change, plan)
- Introduces new capabilities, breaking changes, architecture shifts, or big performance/security work
- Sounds ambiguous and you need the authoritative spec before coding

Use `@/openspec/AGENTS.md` to learn:

- How to create and apply change proposals
- Spec format and conventions
- Project structure and guidelines

Keep this managed block so 'openspec update' can refresh the instructions.

<!-- OPENSPEC:END -->

# Project Documentation

This project's documentation is organized into focused guides:

- **@docs/communication.md** - AI assistant communication guidelines and best practices
- **@docs/setup.md** - Development workflow, package manager, and command examples
- **@docs/standards.md** - Coding standards, SOLID principles, TypeScript conventions, and git commit format
- **@docs/system-design.md** - Architecture, code organization, and design patterns
- **@docs/testing.md** - Testing approach, coverage requirements, and testability guidelines

## Quick Reference

### Essential Commands

```bash
# Build and test workflow
yarn build
node dist/cli.js setup --name default
node dist/cli.js spin --image default --repo . --prompt "your task"
```

### Key Principles

- Always use **yarn**, never npm
- Build before testing CLI commands
- Follow SOLID principles
- All code must have tests
- Keep functions small and focused (< 30 lines)
- Use TypeScript interfaces for clear contracts
