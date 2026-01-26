# Development Setup & Workflow

## Package Manager

- **Go modules for dependencies**: This project uses Go modules for managing Go dependencies
- **yarn for OpenSpec only**: yarn is used only for OpenSpec tooling (`@fission-ai/openspec` package)
- Use `go mod download` for Go dependencies
- Use `yarn install` only when working with OpenSpec features

## Development & Debugging

### Running Commands

- **Always build before testing**: Run `go build -o dist/spinner` before testing CLI commands to ensure latest changes are compiled
- **Go build command**: Use `go build -o dist/spinner` to compile the binary
- **Setup before spin**: When testing the `spin` command, always run `spinner setup` first to ensure the environment is properly configured

### Common Debugging Steps

1. Build the project: `go build -o dist/spinner`
2. Run setup if needed: `./dist/spinner setup [options]`
3. Test the command: `./dist/spinner spin [options]`
4. Check Docker is running: `docker ps` should execute without errors
5. Verify git repository state: Ensure you're in a git repository with proper configuration

### Testing Workflow

When testing changes to the spin command:

```bash
# 1. Build the project
go build -o dist/spinner

# 2. Run setup (required before first spin)
./dist/spinner setup --name default

# 3. Test your spin command
./dist/spinner spin --image default --repo . --prompt "your test prompt"
```

## Working Command Examples

These are tested, working examples for future reference:

### Setup Command Examples

```bash
# Basic setup with default base image
./dist/spinner setup --name spinner:default

# Setup with custom base image
./dist/spinner setup --name ubuntu --base-image ubuntu:22.04

# Setup with custom Dockerfile
./dist/spinner setup --name custom --dockerfile ./path/to/Dockerfile
```

### Spin Command Examples

```bash
# Basic spin with prompt only (uses current branch)
./dist/spinner spin \
  --image spinner:default \
  --repo https://github.com/anthropics/anthropic-quickstarts \
  --prompt "add a readme explaining how to run the customer support agent"

# Spin with specific branch
./dist/spinner spin \
  --image spinner:default \
  --repo https://github.com/anthropics/anthropic-quickstarts \
  --branch main \
  --prompt "fix any linting errors"

# Spin with max iterations limit
./dist/spinner spin \
  --image spinner:default \
  --repo https://github.com/anthropics/anthropic-quickstarts \
  --prompt "refactor the authentication module" \
  --max-iterations 10

# Spin without prompt (interactive mode on current branch)
./dist/spinner spin \
  --image spinner:default \
  --repo https://github.com/user/repo \
  --branch feature/new-feature
```

### Important Notes

- The `--image` parameter must match a `--name` from a previous setup
- Repository must be a valid git URL (https://, http://, or git@)
- Either `--prompt` or `--branch` (or both) must be provided
- Default `max-iterations` is 30 if not specified
