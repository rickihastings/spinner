# Development Setup & Workflow

## Package Manager

- **Always use yarn**: This project uses yarn as the package manager
- Use `yarn install` for installing dependencies, not `npm install`
- Use `yarn build` for building, not `npm run build`
- Use `yarn <script>` for running scripts, not `npm run <script>`

## Development & Debugging

### Running Commands

- **Always build before testing**: Run `yarn build` before testing CLI commands to ensure latest changes are compiled
- **Use yarn, not npm**: All commands must use `yarn`, never `npm`
- **Setup before spin**: When testing the `spin` command, always run `spinner setup` first to ensure the environment is
  properly configured

### Common Debugging Steps

1. Build the project: `yarn build`
2. Run setup if needed: `node dist/cli.js setup [options]`
3. Test the command: `node dist/cli.js spin [options]`
4. Check Docker is running: `docker ps` should execute without errors
5. Verify git repository state: Ensure you're in a git repository with proper configuration

### Testing Workflow

When testing changes to the spin command:

```bash
# 1. Build the project
yarn build

# 2. Run setup (required before first spin)
node dist/cli.js setup

# 3. Test your spin command
node dist/cli.js spin --prompt "your test prompt"
```

## Working Command Examples

These are tested, working examples for future reference:

### Setup Command Examples

```bash
# Basic setup with default base image
node dist/cli.js setup --name spinner:default

# Setup with custom base image
node dist/cli.js setup --name ubuntu --base-image ubuntu:22.04

# Setup with custom Dockerfile
node dist/cli.js setup --name custom --dockerfile ./path/to/Dockerfile
```

### Spin Command Examples

```bash
# Basic spin with prompt only (uses current branch)
node dist/cli.js spin \
  --image spinner:default \
  --repo https://github.com/anthropics/anthropic-quickstarts \
  --prompt "add a readme explaining how to run the customer support agent"

# Spin with specific branch
node dist/cli.js spin \
  --image spinner:default \
  --repo https://github.com/anthropics/anthropic-quickstarts \
  --branch main \
  --prompt "fix any linting errors"

# Spin with max iterations limit
node dist/cli.js spin \
  --image spinner:default \
  --repo https://github.com/anthropics/anthropic-quickstarts \
  --prompt "refactor the authentication module" \
  --max-iterations 10

# Spin without prompt (interactive mode on current branch)
node dist/cli.js spin \
  --image spinner:default \
  --repo https://github.com/user/repo \
  --branch feature/new-feature
```

### Important Notes

- The `--image` parameter must match a `--name` from a previous setup
- Repository must be a valid git URL (https://, http://, or git@)
- Either `--prompt` or `--branch` (or both) must be provided
- Default `max-iterations` is 30 if not specified
