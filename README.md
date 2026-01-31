# Spinner

Run Claude agents in sandboxed Docker containers, unsupervised. Built for autonomous agent loops where you want isolation, reproducibility, and hands-off execution.

Unlike opinionated agent frameworks, Spinner doesn't dictate how you structure your prompts, specs, or tasks—bring whatever workflow suits your project. And because you control the Docker environment through your own Dockerfile, you can build on any stack: Node, Python, Go, Rust, or anything else you need.

## Why Spinner?

When running autonomous AI agents (like Ralph loops), you want:

- **Isolation** - Agents run in ephemeral Docker containers, so they can't affect your host system
- **Reproducibility** - Same container image, same environment, every time
- **Hands-off execution** - Start a task and walk away; the agent works until it's done or hits the iteration limit

Spinner handles the container orchestration so you can focus on the prompts.

## Quick Start

### Prerequisites

- Docker (running)
- Go 1.21+ (for building)
- `GITHUB_TOKEN` environment variable (for cloning repos)
- `CLAUDE_CODE_OAUTH_TOKEN` environment variable (for the agent)

### Build

```bash
git clone https://github.com/rickihastings/spinner.git
cd spinner
go build -o dist/spinner
```

### Create a Sandbox Image

```bash
./dist/spinner setup --name default
```

This builds a Docker image with Claude Code and git pre-installed.

### Run an Agent

```bash
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx
export CLAUDE_CODE_OAUTH_TOKEN=your_token_here

./dist/spinner spin \
  --image default \
  --repo https://github.com/your-org/your-repo \
  --prompt "implement the feature described in TASKS.md"
```

The agent will clone the repo, start working, and continue until it signals completion or hits the iteration limit.

## Writing Effective Prompts

**Spec-driven development works best.** Point the agent at a specification file, design doc, or task list in your repo:

```bash
--prompt "implement the changes described in specs/feature-x.md"
```

**Task lists are also effective.** If you don't have a spec, a clear task list in your prompt works well:

```bash
--prompt "1. Add user authentication 2. Create login page 3. Add session management 4. Write tests"
```

The more context you provide upfront, the better the agent performs autonomously.

## Optimizing Agent Performance

### Minimize Context Window Bloat

Long-running agents accumulate context, leading to degraded performance and hallucinations. Structure your tasks to keep context windows fresh:

**Work in vertical slices.** Instruct your agent to complete tasks one at a time:

```
For each task:
1. Implement the change
2. Add tests
3. Run formatting and linting
4. Verify tests pass
5. Commit
6. Move to next task
```

This pattern keeps each unit of work small and verifiable.

### Use Back Pressure for Correctness

Configure your project with automated checks that run on each iteration:

- **Formatting** (prettier, gofmt, black)
- **Linting** (eslint, golangci-lint, ruff)
- **Type checking** (tsc, mypy)
- **Tests** (unit → integration → e2e)

When these checks fail, the agent must fix issues before proceeding. This "back pressure" forces correctness and naturally segments work into smaller context windows.

Spinner automatically pushes changes after each iteration, so progress is preserved even if the agent needs to restart with a fresh context.

### Example Task Structure

```markdown
## Tasks

Work through each task individually. After completing each one, run the
test suite, fix any failures, then commit before moving to the next task.

1. Add user model with email validation
2. Create registration endpoint
3. Add login endpoint with JWT tokens
4. Implement session middleware
5. Add integration tests for auth flow
```

## Command Reference

### setup

Build a sandbox Docker image:

```bash
# Default Ubuntu base
./dist/spinner setup --name my-env

# Custom base image
./dist/spinner setup --name node-env --base-image node:20-bullseye

# Custom Dockerfile
./dist/spinner setup --name custom-env --dockerfile ./Dockerfile.custom
```

### spin

Launch a container and optionally start an agent:

```bash
./dist/spinner spin \
  --image <image-name> \
  --repo <git-url> \
  [--prompt "task description"] \
  [--branch feature-branch] \
  [--max-iterations 50] \
  [--recreate]
```

| Flag | Description |
|------|-------------|
| `--image` | Docker image from setup (required) |
| `--repo` | Git repository URL (required) |
| `--prompt` | Task for the agent; if omitted, container stays idle |
| `--branch` | Git branch to checkout |
| `--max-iterations` | Stop after N iterations (default: 30) |
| `--recreate` | Force fresh container, removing any existing one |

### Container Access

Containers persist after the agent finishes. Access them directly:

```bash
docker exec -it <container-name> bash
```

Container names are deterministic: `spinner-<image>-<repo>[-branch]`

## Ephemeral by Design

Spinner containers are meant to be disposable. When you're done:

```bash
docker stop <container-name>
docker rm <container-name>
```

Or use `--recreate` on the next run to start fresh.

## Development

For contributing to Spinner itself, see the development documentation:

- [docs/usage.md](docs/usage.md) - Development workflow and commands
- [docs/standards.md](docs/standards.md) - Coding standards and conventions
- [docs/testing.md](docs/testing.md) - Testing approach and requirements
- [docs/system-design.md](docs/system-design.md) - Architecture overview

## Requirements

- Docker
- Go 1.21+
- GitHub Personal Access Token
- Claude Code OAuth Token
