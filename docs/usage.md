# Usage Guide

## Secret Management

Spinner uses an encrypted secret store to manage sensitive tokens like `GITHUB_TOKEN` and Claude auth credentials
(`ANTHROPIC_API_KEY` or `CLAUDE_CODE_OAUTH_TOKEN`). All secrets are stored in an AES-256-GCM encrypted file at
`~/.spinner/secrets.enc` and delivered to containers as an encrypted blob — never as environment variables.

**This is a breaking change:** environment variable fallback for tokens has been removed. You must store all tokens in
the secret store before running `spinner spin`.

### Initial Setup

Store your required tokens:

```bash
# Required
spinner secret set GITHUB_TOKEN

# Claude auth — set one of the following (ANTHROPIC_API_KEY is checked first)
spinner secret set ANTHROPIC_API_KEY
# or
spinner secret set CLAUDE_CODE_OAUTH_TOKEN
```

Each command prompts for the value with hidden input. Alternatively, pass the value directly:

```bash
spinner secret set GITHUB_TOKEN --value ghp_xxxxxxxxxxxx
```

### Managing Secrets

```bash
# List stored secret key names (values are never shown)
spinner secret list

# Delete a secret
spinner secret delete GITHUB_TOKEN
```

### Using Secrets with `spinner spin`

Built-in tokens (`GITHUB_TOKEN` and one of `ANTHROPIC_API_KEY` / `CLAUDE_CODE_OAUTH_TOKEN`) are resolved automatically
from the store. For additional secrets, use the `--secret` flag:

```bash
# Built-in tokens are resolved automatically; add custom secrets with --secret
spinner spin \
  --image default \
  --repo https://github.com/user/repo \
  --prompt "deploy the app" \
  --secret NPM_TOKEN \
  --secret API_KEY
```

All resolved secrets (built-in + custom) are encrypted into a per-session blob using a random AES-256 key and
delivered to the container at `/run/spinner/secrets.enc`. The decryption key is stored separately (in GCS for GCP,
or written directly for Docker) and never exposed as a plaintext environment variable.

### Passphrase

The store passphrase is sourced from:

1. `SPINNER_SECRET_PASSPHRASE` environment variable (for CI/scripts)
2. Interactive prompt (for local development)

```bash
# Non-interactive usage (CI)
export SPINNER_SECRET_PASSPHRASE=my-passphrase
spinner spin --image default --repo https://github.com/user/repo --prompt "task"
```

### In-Container Secret Access

Secrets are never exposed as container environment variables. Inside a container, there are two ways secrets are
accessed:

**Automatic (`--prompt` mode):** `spinner exec` decrypts the blob at startup and injects secrets into the Claude CLI
child process via `cmd.Env`. No user action is needed.

**Manual (no `--prompt` / SSH mode):** Use `spinner secret inject` to decrypt the blob and run a command with secrets:

```bash
# Run a single command with secrets injected
spinner secret inject -- claude -p "implement feature X"

# Start an interactive shell with secrets available
spinner secret inject -- bash

# The passphrase is read from SPINNER_SECRET_PASSPHRASE env if available,
# otherwise prompts interactively
```

### Inception (Spinner Inside Spinner)

The encrypted blob format matches the store format, so an outer Spinner's blob can serve as the inner Spinner's store.
The `SPINNER_SECRET_STORE` environment variable controls where the store file is read from:

```bash
# Inside a container, use the blob as the inner spinner's store
spinner secret inject -- sh -c '
  SPINNER_SECRET_STORE=/run/spinner/secrets.enc \
  spinner spin --backend docker --secret NPM_TOKEN --repo ... --prompt "sub-task"
'
```

Each layer decrypts what it needs and re-encrypts for the next. Same passphrase at every layer, separate salts per blob.

### Configurable Store Path

Set `SPINNER_SECRET_STORE` to override the default store location (`~/.spinner/secrets.enc`):

```bash
export SPINNER_SECRET_STORE=/path/to/custom/secrets.enc
spinner secret set MY_KEY
```

## Configuration

### Environment Variables

Spinner uses Viper to support environment variable configuration. All command-line flags can be overridden using
environment variables with the `SPINNER_` prefix.

**Setup Command:**

- `SPINNER_NAME` - Override `--name` flag

**Spin Command:**

- `SPINNER_IMAGE` - Override `--image` flag
- `SPINNER_REPO` - Override `--repo` flag
- `SPINNER_PROMPT` - Override `--prompt` flag
- `SPINNER_BRANCH` - Override `--branch` flag
- `SPINNER_MAX_ITERATIONS` - Override `--max-iterations` flag
- `SPINNER_RECREATE` - Override `--recreate` flag (set to `true` or `false`)
- `SPINNER_WATCH` - Override `--watch` flag (set to `true` or `false`)

```bash
# Set default image via environment variable
export SPINNER_IMAGE=spinner:default
spinner spin --repo https://github.com/user/repo --prompt "task"

# Command-line flags take precedence over environment variables
export SPINNER_IMAGE=spinner:default
spinner spin --image spinner:custom --repo https://github.com/user/repo
# Uses spinner:custom, not spinner:default
```

### Configuration File

Spinner supports `.spinner.json` for team-shared defaults (commit to repo) and `.env` for local overrides (don't
commit).

**`.spinner.json` example:**

```json
{
  "backend": "gcp",
  "project": "my-project",
  "zone": "us-central1-a",
  "state-bucket": "my-state-bucket",
  "image": "my-default-image",
  "spin-provider-args": ["--machine-type=e2-standard-2", "--boot-disk-size=30GB"]
}
```

**`.env` example:**

```env
SPINNER_PROJECT=my-local-project
SPINNER_ZONE=us-west1-a
```

Spinner searches for `.spinner.json` by traversing up from the current directory to the filesystem root, then falls
back to `$HOME/.spinner.json` if no file is found. The first file found is used (no merging between config files).

This allows you to:
- Place `.spinner.json` in your home directory (`~/.spinner.json`) for personal defaults across all projects
- Override home defaults with a `.spinner.json` in any repository or parent directory
- Use team-shared config by committing `.spinner.json` to a repository

The `.env` file is only loaded from the current directory (no traversal).

### Configuration Precedence

Configuration values are applied in this order (highest to lowest priority):

1. Command-line flags
2. Environment variables (`SPINNER_*`)
3. `.env` file (current directory only)
4. `.spinner.json` file (nearest ancestor or `$HOME/.spinner.json`)
5. Default values

## Writing Effective Prompts

### The Prompt as a Living Document

Unlike a one-shot chat message, your prompt (or prompt file) is the agent's persistent memory. The agent reads it at the
start of each iteration, so it can track what has already been done and what remains. Design your prompt accordingly:
write it as a living spec that the agent updates as it works, not as a set of instructions it reads once and discards.

A static, one-liner prompt generally will not cause the agent to stop cleanly — the agent must emit `~~ FEATURE_COMPLETED ~~`
on its own line for Spinner to detect completion. Without a clear signal of doneness baked into the prompt, the agent
may keep iterating or exit with an error.

### Task Lists Are the Core Pattern

Structure your prompt as an ordered task list. Each iteration the agent:

1. Reads the prompt/spec to find the next unchecked task
2. Does the work
3. Marks the task complete in the file (or a separate checklist)
4. Commits and pushes
5. Moves on to the next task

When all tasks are done the agent emits the completion signal. The commit history gives you a clean record of each
incremental step, and if the run is interrupted mid-way the next iteration picks up exactly where it left off because
the state is in the file itself, not in the agent's memory.

**Example prompt structure (`prompt.md`):**

```markdown
# Feature: Add user authentication

## Tasks
- [x] Scaffold the login route and handler
- [x] Add session middleware
- [ ] Write unit tests for the auth module
- [ ] Update the README with setup instructions
- [ ] Emit ~~ FEATURE_COMPLETED ~~ when all tasks are done

## Notes
- Use the existing `db` package for queries
- JWT secret is in `.env` as `JWT_SECRET`
```

Pass this file as your prompt:

```bash
spinner spin \
  --image my-env \
  --repo https://github.com/user/repo \
  --prompt "$(cat prompt.md)"
```

Or commit `prompt.md` to the repo and instruct the agent to read and update it each iteration, so the file itself
becomes the source of truth that survives restarts.

### Single-Iteration / One-Shot Prompts

`--max-iterations 1` is a fully supported mode for tasks that are self-contained and don't need multiple rounds. This
is useful when you want to run a single sandboxed command and not have the agent loop:

```bash
spinner spin \
  --image my-env \
  --repo https://github.com/user/repo \
  --prompt "run the test suite and paste the output into test-results.txt" \
  --max-iterations 1
```

The agent runs once, does the work, and the container exits regardless of whether the completion signal was emitted.
Use this for tasks where you know one iteration is enough and you don't need the loop logic.

### Prompt Tips

- **Be explicit about completion.** Tell the agent to emit `~~ FEATURE_COMPLETED ~~` when all tasks are done, or
  structure the task list so it naturally ends with a "emit completion signal" step.
- **Keep context in the file.** Anything the agent needs to remember between iterations should be written to a file in
  the repo — agent memory does not persist across iterations.
- **Prefer small, concrete tasks.** "Add login endpoint" is better than "implement authentication". Smaller tasks mean
  cleaner commits and easier recovery if something goes wrong.
- **Include constraints up front.** Style guides, libraries to use, files to avoid — put them in the prompt so the
  agent doesn't have to guess.
