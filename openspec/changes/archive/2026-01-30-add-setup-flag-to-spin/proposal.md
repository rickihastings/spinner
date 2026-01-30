# Proposal: Add Setup Flag to Spin Command

## Overview

Add an optional `--setup` flag to the `spin` command that combines image setup and container spinning into a single
operation. This eliminates the need to run two separate commands when users want to ensure they have a fresh image
before spinning up a container.

## Why

Users currently must execute two separate commands to set up an image and spin a container. This is inconvenient when users want to ensure they're working with a fresh image build. The `--setup` flag streamlines this workflow into a single command, reducing friction and improving developer experience.

This change is particularly valuable when:
- Testing changes to base image configuration
- Ensuring dependencies are up-to-date before starting work
- Demonstrating the tool to new users (fewer commands to remember)
- Scripting automated workflows that need both setup and spin

## Motivation

Currently, users must run two commands to set up an environment and spin a container:

```bash
spinner setup --name my-env --base-image ubuntu:22.04
spinner spin --image spinner:my-env --repo git@github.com:user/repo.git
```

This proposal allows users to combine both operations:

```bash
spinner spin --setup --image my-env --base-image ubuntu:22.04 --repo git@github.com:user/repo.git
```

## User Impact

- **Convenience**: Single command for setup + spin workflow
- **Rebuild behavior**: When `--setup` is used, the image is always rebuilt (even if it exists)
- **Flexibility**: Users can still use separate `setup` and `spin` commands for more control

## Requirements Summary

1. Add `--setup` boolean flag to spin command
2. When `--setup` is provided, run image build before spinning container
3. Accept `--base-image` and `--dockerfile` flags when `--setup` is used
4. The `--image` flag serves as both the setup name and the image to spin
5. Always rebuild the image when `--setup` is used (no caching/skip logic)
6. Validate flag combinations (e.g., `--base-image` and `--dockerfile` are mutually exclusive)

## Scope

- Modify `cmd/spin.go` to add setup-related flags and orchestration logic
- Reuse existing `internal/docker.BuildImage()` function from setup command
- Reuse existing prerequisite checks from setup command
- No changes to core Docker build or run logic

## Out of Scope

- Image caching or conditional rebuild logic
- Changes to standalone `setup` command
- Changes to container reuse/recreate logic

## Alternatives Considered

1. **Keep separate commands**: Current approach works but requires two steps
2. **Auto-setup on missing image**: Could auto-setup when image doesn't exist, but this doesn't help users who want to
   force a rebuild
3. **Add `--force-rebuild` flag**: Rejected in favor of simpler approach where `--setup` always rebuilds

## Success Criteria

- Users can run `spin --setup` to build image and create container in one command
- Image is always rebuilt when `--setup` flag is present
- All existing spin and setup functionality remains unchanged
- Documentation and examples updated to show new workflow