# Spinner

Run Claude agents in sandboxed Docker containers, unsupervised. Built for autonomous agent loops where you want
isolation, reproducibility, and hands-off execution.

Unlike opinionated agent frameworks, Spinner doesn't dictate how you structure your prompts, specs, or tasks—bring
whatever workflow suits your project. And because you control the Docker environment through your own Dockerfile, you
can build on any stack: Node, Python, Go, Rust, or anything else you need.

## Why Spinner?

When running autonomous AI agents (like Ralph loops), you want:

- **Isolation** - Agents run in ephemeral Docker containers, so they can't affect your host system
- **Reproducibility** - Same container image, same environment, every time
- **Hands-off execution** - Start a task and walk away; the agent works until it's done or hits the iteration limit

Spinner handles the container orchestration so you can focus on the prompts.

## Installation

### Download Binary (Recommended)

Download the latest release for your platform from [GitHub Releases](https://github.com/rickihastings/spinner/releases):

```bash
# macOS (Apple Silicon)
curl -Lo spinner_darwin_arm64.tar.gz https://github.com/rickihastings/spinner/releases/latest/download/spinner_darwin_arm64.tar.gz
tar -xzf spinner_darwin_arm64.tar.gz
sudo mv spinner /usr/local/bin/

# macOS (Intel)
curl -Lo spinner_darwin_amd64.tar.gz https://github.com/rickihastings/spinner/releases/latest/download/spinner_darwin_amd64.tar.gz
tar -xzf spinner_darwin_amd64.tar.gz
sudo mv spinner /usr/local/bin/

# Linux (amd64)
curl -Lo spinner_linux_amd64.tar.gz https://github.com/rickihastings/spinner/releases/latest/download/spinner_linux_amd64.tar.gz
tar -xzf spinner_linux_amd64.tar.gz
sudo mv spinner /usr/local/bin/
```

### Build from Source

```bash
git clone https://github.com/rickihastings/spinner.git
cd spinner
make build
sudo mv dist/spinner /usr/local/bin/
```

### Update

Spinner can update itself:

```bash
spinner update
```

## Quick Start

### Prerequisites

- Docker (running)
- `GITHUB_TOKEN` environment variable (for cloning repos)
- `CLAUDE_CODE_OAUTH_TOKEN` environment variable (for the agent)

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
  --prompt "$(cat prompt.md)"
```

The agent will clone the repo, start working, and continue until it signals completion or hits the iteration limit.

## Writing Effective Prompts

The prompt isn't a one-shot instruction — it's the agent's persistent memory. The agent reads it at the start of
every iteration, so design it as a living document the agent updates as it works.

**A static one-liner won't complete cleanly.** The agent must emit `~~ FEATURE_COMPLETED ~~` on its own line for
Spinner to detect success. Without a structured prompt that includes a completion step, the agent may keep iterating
indefinitely or exit with an error.

**The recommended pattern is a task list committed to the repo:**

```markdown
# Feature: Add user authentication

## Tasks
- [x] Scaffold the login route and handler
- [ ] Write unit tests for the auth module
- [ ] Update the README with setup instructions
- [ ] Emit ~~ FEATURE_COMPLETED ~~ when all tasks are done

## Notes
- Use the existing `db` package for queries
```

```bash
./dist/spinner spin \
  --image default \
  --repo https://github.com/your-org/your-repo \
  --prompt "$(cat prompt.md)"
```

Each iteration the agent picks the next unchecked task, does the work, marks it done, commits, and pushes. The file
is the source of truth — if the run is interrupted, the next iteration picks up where it left off.

**For one-shot tasks** that don't need looping, use `--max-iterations 1`:

```bash
./dist/spinner spin \
  --image default \
  --repo https://github.com/your-org/your-repo \
  --prompt "run the test suite and save output to test-results.txt" \
  --max-iterations 1
```

For the full guide see [docs/usage.md — Writing Effective Prompts](docs/usage.md#writing-effective-prompts).

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
  [--recreate] \
  [--watch]
```

| Flag               | Description                                          |
|--------------------|------------------------------------------------------|
| `--image`          | Docker image from setup (required)                   |
| `--repo`           | Git repository URL (required)                        |
| `--prompt`         | Task for the agent; if omitted, container stays idle |
| `--branch`         | Git branch to checkout                               |
| `--max-iterations` | Stop after N iterations (default: 100)               |
| `--recreate`       | Force fresh container, removing any existing one     |
| `--watch`          | Enter watch mode after container is ready            |

### watch

Monitor a running container in real-time:

```bash
./dist/spinner watch <container-name>
```

Watch mode provides a terminal UI with:

- Container status (running/stopped/exited)
- CPU and memory usage metrics
- Streaming container logs with structured formatting

Use `q` or `Ctrl+C` to exit watch mode.

**Example:**

```bash
# Start a container and watch it
./dist/spinner spin --image default --repo https://github.com/user/repo --watch

# Or watch an existing container
./dist/spinner watch spinner-default-<hash>
```

### update

Update spinner to the latest version:

```bash
spinner update
```

This checks GitHub Releases for a newer version and updates the binary in place.

### exec

Execute the autonomous iteration loop inside a Docker container. This command is automatically invoked by containers
created with `spinner spin` and manages the agent execution lifecycle.

**Note:** This command is designed to run inside Docker containers and reads all configuration from environment
variables. You typically won't need to call it manually.

**Environment Variables:**

- `PROMPT` - Task prompt for the iteration loop (required)
- `MAX_ITERATIONS` - Maximum number of iterations (required)
- `BRANCH` - Git branch name (optional)
- `LOG_DIR` - Directory for log files (optional)
- `STATE_DIR` - Directory for state file (optional, defaults to `/state`)

**State Management:**

The exec command persists iteration state to `${STATE_DIR}/state.json`, which is mounted from the host. This allows
progress to survive container restarts and tracks:

- Current iteration count
- Branch name
- Status (running/completed/rate_limited/error/auth_error)
- Timestamps and metadata

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

## Guides

For detailed walkthroughs, see the [guides](docs/guides/) directory:

- **[Docker Sandbox Guide](docs/guides/docker.md)** — Setting up sandbox images, running interactive or autonomous containers, and monitoring progress with watch mode
- **[GCP Sandbox Guide](docs/guides/gcp.md)** — Baking VM images, custom bake scripts, running agents on GCP VMs, auto-stop behavior, and state persistence with GCS

For configuration and prompt authoring, see [docs/usage.md](docs/usage.md).

## Development

For contributing to Spinner itself, see the development documentation:

- [docs/development.md](docs/development.md) - Build system, workflow, and local testing
- [docs/standards.md](docs/standards.md) - Coding standards and conventions
- [docs/testing.md](docs/testing.md) - Testing approach and requirements
- [docs/system-design.md](docs/system-design.md) - Architecture overview

## Requirements

- Docker
- Go 1.21+
- GitHub Personal Access Token
- Claude Code OAuth Token
