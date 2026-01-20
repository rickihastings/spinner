# Change: Add Ralph Loop to Docker Container

## Why

Currently, containers created by `spin cmd` clone a repository and then idle indefinitely with `tail -f /dev/null`. Users must manually exec into containers and invoke Claude. This prevents autonomous, hands-off feature implementation.

The Ralph loop pattern (from ghuntley/how-to-ralph-wiggum) enables autonomous AI-driven development by running Claude in a continuous loop until all tasks are complete. This transforms the container from a passive workspace into an active implementation agent.

## What Changes

- **NEW**: Optional `--prompt` CLI flag containing the prompt string to feed Claude
  - When provided, enables Ralph loop for autonomous implementation
  - Without prompt, container clones and stays idle
- **NEW**: Optional `--branch` CLI flag specifying which branch to work on
  - When provided with prompt, Ralph loop runs on specified branch
  - When not provided but prompt is given, Ralph loop runs on default branch
- **NEW**: Optional `--max-iterations` CLI flag (default: 100) to limit loop iterations
- **MODIFIED**: Container startup behavior - conditionally runs Ralph loop based on prompt presence
- **NEW**: Loop monitors Claude output for `~~ FEATURE_COMPLETED ~~` signal
- **MODIFIED**: Container lifecycle can be purpose-driven (exits when complete) or idle (stays running)
- **REFACTORED**: Spin logic moved to `utils/docker.ts` following SOLID principles

## Impact

- Affected specs: `cli-spin`
- Affected code:
  - `src/commands/Spin.tsx` - refactored to use utility functions
  - `src/utils/docker.ts` - new utility functions for spin logic following SOLID principles
  - `src/App.tsx` - updated validation and help text
  - `templates/scripts/startup.sh` - conditional Ralph loop execution based on prompt
  - `templates/scripts/ralph-loop.sh` - new script containing the loop logic
  - `CLAUDE.md` - added SOLID principles and coding standards
  - `tests/spin/*.sh` - updated to reflect new behavior
