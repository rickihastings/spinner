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

# What is Spinner

Spinner is a CLI tool that runs Claude agents in sandboxed Docker containers for autonomous, unsupervised execution.
Users run `setup` to build a Docker image, then `spin` to launch a container that clones a repo and runs an iteration
loop. Each iteration invokes Claude CLI, checks the output for completion/errors/rate-limits, pushes changes to git,
and persists state. The agent works until it outputs `~~ FEATURE_COMPLETED ~~` or hits the iteration limit.

## Key Concepts

- **Iteration loop** (`internal/exec/loop.go`) — the core execution cycle. Each iteration: run Claude → check result →
  push to git → save state → repeat. Rate limits trigger a 61-minute wait. Auth errors stop the loop.
- **Completion signal** — the agent outputs `~~ FEATURE_COMPLETED ~~` in its response to signal it's done. This is
  detected by the parser in `internal/agent/claude/executor.go`.
- **Provider abstraction** (`internal/provider/provider.go`) — backend-agnostic interface. Docker is the only provider
  today, but the architecture supports VMs, K8s, etc. Commands depend on the Provider interface, never on Docker
  directly.
- **State file** (`/state/state.json` in container) — JSON tracking iteration count, status (`running`/`completed`/
  `rate_limited`/`error`/`auth_error`), and timestamps. Mounted from host at `~/.spinner/<container-name>/state/`.
- **Container naming** — deterministic: `{image}-{repo}[-branch]` (sanitized). Existing containers are reused unless
  `--recreate` is passed.
- **max-iterations** — default is 100 when not specified.

# Project Documentation

Detailed guides for specific topics:

- **docs/usage.md** - Development workflow, package manager, and command examples
- **docs/standards.md** - Coding standards, communication guidelines, Go conventions, and git commit format
- **docs/system-design.md** - Architecture, code organization, and design patterns
- **docs/testing.md** - Testing approach, coverage requirements, and testability guidelines

## Quick Reference

### Essential Commands

```bash
go build -o dist/spinner
./dist/spinner setup --name default
./dist/spinner spin --image default --repo https://github.com/user/repo --prompt "your task"
```

### Key Principles

- Build before testing CLI commands
- All code must have tests
- Keep functions small and focused
- Follow SOLID principles and Go conventions

### Sandbox Proxy Fix

**IMPORTANT**: this is ONLY a concern when running inside the Claude Web sandbox

If `go mod download` fails with DNS errors for `storage.googleapis.com`, the sandbox `no_proxy`/`NO_PROXY` vars are
blocking the HTTP proxy. Fix by removing Google domains before running Go commands:

```bash
export no_proxy=$(echo "$no_proxy" | sed 's/,\*\.googleapis\.com//g;s/,\*\.google\.com//g;s/,storage\.googleapis\.com//g')
export NO_PROXY=$(echo "$NO_PROXY" | sed 's/,\*\.googleapis\.com//g;s/,\*\.google\.com//g;s/,storage\.googleapis\.com//g')
```
