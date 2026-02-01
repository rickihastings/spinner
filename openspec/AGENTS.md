# OpenSpec Instructions

Instructions for AI coding assistants using OpenSpec for spec-driven development.

## What Stage Are You In?

Choose the appropriate guide based on your current task:

- **Creating or planning a change** → Read `agents/propose.md`
- **Implementing an approved change** → Read `agents/implement.md`
- **Archiving a deployed change** → Read `agents/archive.md`

Each guide provides focused instructions for that stage of the workflow.

## Directory Structure

```
openspec/
├── project.md              # Project conventions
├── specs/                  # Current truth - what IS built
│   └── [capability]/       # Single focused capability
│       ├── spec.md         # Requirements and scenarios
│       └── design.md       # Technical patterns
├── changes/                # Proposals - what SHOULD change
│   ├── [change-name]/
│   │   ├── proposal.md     # Why, what, impact
│   │   ├── tasks.md        # Implementation checklist
│   │   ├── design.md       # Technical implementation plan and decisions
│   │   └── specs/          # Delta changes
│   │       └── [capability]/
│   │           └── spec.md # ADDED/MODIFIED/REMOVED
│   └── archive/            # Completed changes
```

## CLI Commands

```bash
# Essential commands
openspec list                  # List active changes
openspec list --specs          # List specifications
openspec show [item]           # Display change or spec
openspec validate [item]       # Validate changes or specs
openspec archive <change-id> [--yes|-y]   # Archive after deployment (add --yes for non-interactive runs)

# Project management
openspec init [path]           # Initialize OpenSpec
openspec update [path]         # Update instruction files

# Interactive mode
openspec show                  # Prompts for selection
openspec validate              # Bulk validation mode

# Debugging
openspec show [change] --json --deltas-only
openspec validate [change] --strict --no-interactive
```

### Command Flags

- `--json` - Machine-readable output
- `--type change|spec` - Disambiguate items
- `--strict` - Comprehensive validation
- `--no-interactive` - Disable prompts
- `--skip-specs` - Archive without spec updates
- `--yes`/`-y` - Skip confirmation prompts (non-interactive archive)

## Search Guidance

- Enumerate specs: `openspec spec list --long` (or `--json` for scripts)
- Enumerate changes: `openspec list` (or `openspec change list --json` - deprecated but available)
- Show details:
  - Spec: `openspec show <spec-id> --type spec` (use `--json` for filters)
  - Change: `openspec show <change-id> --json --deltas-only`
- Full-text search (use ripgrep): `rg -n "Requirement:|Scenario:" openspec/specs`

## Troubleshooting

### Common Errors

**"Change must have at least one delta"**

- Check `changes/[name]/specs/` exists with .md files
- Verify files have operation prefixes (## ADDED Requirements)

**"Requirement must have at least one scenario"**

- Check scenarios use `#### Scenario:` format (4 hashtags)
- Don't use bullet points or bold for scenario headers

**Silent scenario parsing failures**

- Exact format required: `#### Scenario: Name`
- Debug with: `openspec show [change] --json --deltas-only`

### Validation Tips

```bash
# Always use strict mode for comprehensive checks
openspec validate [change] --strict --no-interactive

# Debug delta parsing
openspec show [change] --json | jq '.deltas'

# Check specific requirement
openspec show [spec] --json -r 1
```

## Best Practices

### Simplicity First

- Default to <100 lines of new code
- Single-file implementations until proven insufficient
- Avoid frameworks without clear justification
- Choose boring, proven patterns

### Complexity Triggers

Only add complexity with:

- Performance data showing current solution too slow
- Concrete scale requirements (>1000 users, >100MB data)
- Multiple proven use cases requiring abstraction

### Clear References

- Use `file.ts:42` format for code locations
- Reference specs as `specs/auth/spec.md`
- Link related changes and PRs

### Capability Naming

- Use verb-noun: `user-auth`, `payment-capture`
- Single purpose per capability
- 10-minute understandability rule
- Split if description needs "AND"

### Change ID Naming

- Use kebab-case, short and descriptive: `add-two-factor-auth`
- Prefer verb-led prefixes: `add-`, `update-`, `remove-`, `refactor-`
- Ensure uniqueness; if taken, append `-2`, `-3`, etc.

## Tool Selection Guide

| Task                  | Tool | Why                      |
|-----------------------|------|--------------------------|
| Find files by pattern | Glob | Fast pattern matching    |
| Search code content   | Grep | Optimized regex search   |
| Read specific files   | Read | Direct file access       |
| Explore unknown scope | Task | Multi-step investigation |

## Error Recovery

### Change Conflicts

1. Run `openspec list` to see active changes
2. Check for overlapping specs
3. Coordinate with change owners
4. Consider combining proposals

### Validation Failures

1. Run with `--strict` flag
2. Check JSON output for details
3. Verify spec file format
4. Ensure scenarios properly formatted

### Missing Context

1. Read project.md first
2. Check related specs
3. Review recent archives
4. Ask for clarification

## Quick Reference

### Stage Indicators

- `changes/` - Proposed, not yet built
- `specs/` - Built and deployed
- `archive/` - Completed changes

### File Purposes

- `proposal.md` - Why and what
- `design.md` - Technical implementation plan (how to approach)
- `tasks.md` - Implementation checklist (what to do)
- `spec.md` - Requirements and behavior

### CLI Essentials

```bash
openspec list              # What's in progress?
openspec show [item]       # View details
openspec validate --strict --no-interactive  # Is it correct?
openspec archive <change-id> [--yes|-y]  # Mark complete (add --yes for automation)
```

Remember: Specs are truth. Changes are proposals. Keep them in sync.
