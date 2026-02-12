# Proposal: Add Config File Home Directory Traversal

## Summary

Change `.spinner.json` config loading to traverse up the directory tree, enabling a shared config file in the user's
home directory (`~/.spinner.json`) instead of requiring one per repository.

## Motivation

Currently, `.spinner.json` is only loaded from the current working directory (`.`). This means users who work across
many repositories must duplicate the same configuration (e.g., `backend`, `project`, `zone`, `state-bucket`) in every
repo. A home directory config provides a single place for personal defaults while still allowing per-repo overrides.

## What Changes

1. **Config loading order** — `.spinner.json` is now searched by traversing from the current directory upward to the
   filesystem root, then falling back to `$HOME/.spinner.json`. The first file found wins as the "project config."
2. **Precedence update** — The full precedence becomes:
   CLI flags > env vars (`SPINNER_*`) > `.env` file > `.spinner.json` (nearest ancestor) > `~/.spinner.json` > defaults
3. **Home dir as global defaults** — `~/.spinner.json` acts as user-wide defaults. A `.spinner.json` anywhere in the
   ancestor path fully overrides it (no merging between config files).

## Impact

- **Specs affected:** `cli-spin/spec.md`, `cli-setup/spec.md` — both reference "Configuration File Support" requirement
- **Code affected:** `cmd/root.go` (config loading in `init()`)
- **No breaking changes** — existing `.spinner.json` in repo root continues to work identically. The only behavioral
  change is that config files in ancestor directories and `$HOME` are now also discovered.

## Non-Goals

- Merging multiple config files (e.g., home + repo). Only one config file is loaded.
- XDG config directory support (`~/.config/spinner/`). Can be added later if needed.
- `.env` file traversal — `.env` remains current-directory only.
