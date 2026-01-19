# Change: Support custom base images for Docker setup

## Why

The current setup command hardcodes JVM and Node.js installation via CLI flags (--jvm-url, --node-version), which
doesn't scale for diverse development environments. Users need the flexibility to provide their own base Docker image or
Dockerfile containing their complete development stack (Python, Ruby, custom tools, etc.). The CLI should only ensure
the minimal requirements (git and claude-code) are present rather than prescribing the entire environment.

## What Changes

- Replace `--jvm-url` and `--node-version` flags with `--base-image` or `--dockerfile` flags
- Accept either:
    - `--base-image <image-name>` (e.g., `ubuntu:22.04`, `node:20-bullseye`)
    - `--dockerfile <path>` (path to user's Dockerfile, we build it first)
- Only install missing prerequisites:
    - Check if git is installed, install if missing
    - Check if claude-code is installed, install if missing
- Generate a minimal Dockerfile that extends the user's base image
- Support Ubuntu/Debian-based images only (apt-get package manager)
- Document OS limitation in error messages and help text

## Impact

- Affected specs: `cli-setup` (modified capability)
- Affected code: `Setup.tsx`, `dockerfile.ts`, CLI argument parsing
- Breaking change: Removes `--jvm-url` and `--node-version` flags
- Dependencies: User-provided base images must be Ubuntu/Debian-based
- Users gain full control over their development environment stack
