# Change: Add Ralph Loop to Docker Container

## Why

Currently, containers created by `spin cmd` clone a repository and then idle indefinitely with `tail -f /dev/null`. Users must manually exec into containers and invoke Claude. This prevents autonomous, hands-off feature implementation.

The Ralph loop pattern (from ghuntley/how-to-ralph-wiggum) enables autonomous AI-driven development by running Claude in a continuous loop until all tasks are complete. This transforms the container from a passive workspace into an active implementation agent.

## What Changes

- **NEW**: Required `--prompt` CLI flag containing the prompt string to feed Claude
- **NEW**: Required `--branch` CLI flag specifying which branch to work on
- **NEW**: Optional `--max-iterations` CLI flag (default: 100) to limit loop iterations
- **MODIFIED**: Container startup behavior - runs Ralph loop until completion or max iterations
- **NEW**: Loop monitors Claude output for `~~ FEATURE_COMPLETED ~~` signal
- **MODIFIED**: Container lifecycle is purpose-driven - exits when feature complete or max iterations reached

## Impact

- Affected specs: `cli-spin`
- Affected code:
  - `src/commands/Spin.tsx` - add `--prompt`, `--branch`, `--max-iterations` flags
  - `templates/startup.sh` - replace idle loop with Ralph loop logic
  - `templates/ralph-loop.sh` - new script containing the loop logic
