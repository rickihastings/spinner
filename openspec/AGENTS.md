# OpenSpec Instructions

Instructions for AI coding assistants using OpenSpec for spec-driven development.

## ⚠️ CRITICAL RULES - FOLLOW EXACTLY ⚠️

### Stage 2 Implementation Protocol - NON-NEGOTIABLE

**STOP AFTER ONE SLICE:**

- Implement ONLY ONE vertical slice (X.0) per session
- Complete ALL sub-tasks (X.1, X.2, X.3...) for that slice
- DO NOT proceed to (X+1).0 under any circumstances
- After completing X.0: apply stopping conditions from `openspec/agents/implement.md` Critical Rules

**EACH SLICE MUST INCLUDE:**

- Implementation code
- Tests for that implementation
- Documentation updates
- Verification (build + tests pass)

**MANDATORY AFTER EACH VERTICAL SLICE:**

- Mark task `[x]` in tasks.md immediately
- Commit with meaningful message
- Leave codebase in valid, tested state

## Quick Reference

**Essential Commands:**

```bash
openspec list                    # Active changes
openspec list --specs            # Existing specs
openspec show [item]             # View details (--json for filters)
openspec validate [item] --strict --no-interactive  # Validate
openspec archive <change-id> --yes  # Archive after deployment
```

**Before Any Task:**

- Read `specs/[capability]/spec.md` for relevant capabilities
- Check `changes/` for conflicts via `openspec list`
- Review `openspec/project.md` for conventions

**Search:**

- Specs: `openspec spec list --long` or `openspec show <spec-id> --type spec`
- Changes: `openspec list` or `openspec show <change-id> --json --deltas-only`
- Full-text: `rg -n "Requirement:|Scenario:" openspec/specs`

**Proposal Checklist:**

1. Choose unique verb-led `change-id` (kebab-case: `add-`, `update-`, `remove-`, `refactor-`)
2. Scaffold: `proposal.md`, `tasks.md`, `design.md`, delta specs
3. Write deltas: `## ADDED|MODIFIED|REMOVED|RENAMED Requirements` with `#### Scenario:` per requirement
4. Validate: `openspec validate [change-id] --strict --no-interactive`
5. Request approval before implementation

## Three-Stage Workflow

### Stage 1: Creating Changes

**Create proposal for:** new features, breaking changes, architecture changes, performance optimizations, security
updates

**Skip proposal for:** bug fixes (restoring spec behavior), typos/formatting, non-breaking dependency updates, config
changes, tests for existing behavior

**Workflow:**

1. Review context: `openspec/project.md`, `openspec list`, `openspec list --specs`
2. Choose unique verb-led `change-id` and scaffold files in `openspec/changes/<id>/`
3. Draft spec deltas with `## ADDED|MODIFIED|REMOVED Requirements` + `#### Scenario:` per requirement
4. Validate: `openspec validate <id> --strict --no-interactive` before requesting approval

### Stage 2: Implementing Changes

**Implementation Scope:** ONE vertical slice (X.0) per session. Complete all sub-tasks (X.1, X.2, X.3...) then STOP.

**Per-Task Cycle:** Select → Investigate patterns → Implement → Verify (build/tests) → Update tasks.md → Commit → Repeat

**Rules:**

- Complete entire slice before stopping; each commit = valid, tested state
- Never defer tests; they're part of the slice (unless task list specifies otherwise)
- Update spec immediately if implementation diverges
- On completion: apply stopping conditions from `openspec/agents/implement.md` Critical Rules

### Stage 3: Archiving Changes

After deployment: `openspec archive <change-id> --yes` (use `--skip-specs` for tooling-only changes). Validate with
`openspec validate --strict --no-interactive`.

## Directory Structure

```
openspec/
├── project.md              # Project conventions
├── specs/                  # Current truth (what IS built)
│   └── [capability]/
│       ├── spec.md         # Requirements and scenarios
│       └── design.md       # Technical patterns
└── changes/                # Proposals (what SHOULD change)
    ├── [change-name]/
    │   ├── proposal.md     # Why, what, impact
    │   ├── tasks.md        # Implementation checklist (vertical slices)
    │   ├── design.md       # Technical implementation plan
    │   └── specs/[capability]/spec.md  # Delta changes
    └── archive/            # Completed changes
```

## Creating Change Proposals

### Required Files

**1. proposal.md** - Why, what changes (list with **BREAKING** marks), impact (affected specs/code)

**2. spec deltas** - `specs/[capability]/spec.md` with operations:

```markdown
## ADDED Requirements

### Requirement: Feature Name

System SHALL... [requirement text]

#### Scenario: Success case

- **WHEN** condition
- **THEN** outcome

## MODIFIED Requirements

[Full requirement with all scenarios - replaces existing]

## REMOVED Requirements

**Reason**: [why] | **Migration**: [how]
```

**3. tasks.md** - Vertical feature slices (NOT horizontal layers):

```markdown
## X.0 Feature Name (complete, testable, committable)

- [ ] X.1 Implementation
- [ ] X.2 Tests
- [ ] X.3 Documentation
- [ ] X.4 Verification
```

Each X.0 = independently committable feature. Complete ALL sub-tasks before next slice.

**4. design.md** - REQUIRED. Must include Technical Implementation Plan:

- **Component Map**: Files to change (create|modify|delete)
- **Approach**: Implementation order and patterns to follow
- **Key Decisions**: Rationale for choices
- Goals/Non-Goals, Risks/Trade-offs (optional)

## Spec File Format

**Scenario Format (REQUIRED):**

```markdown
#### Scenario: Name

- **WHEN** condition
- **THEN** outcome
```

Use `####` heading (NOT bullets, bold, or `###`). Every requirement needs ≥1 scenario.

**Requirement Wording:** Use SHALL/MUST (not should/may)

**Delta Operations:**

- **ADDED**: New orthogonal capability (preferred for new requirements)
- **MODIFIED**: Changed behavior—copy FULL existing requirement from `specs/[capability]/spec.md`, paste under
  `## MODIFIED`, edit
- **REMOVED**: Include **Reason** and **Migration** path
- **RENAMED**: `FROM: ### Requirement: Old` → `TO: ### Requirement: New`

**Critical:** MODIFIED must include complete requirement text (header + scenarios). Partial deltas lose details at
archive.

## Troubleshooting

**Common Errors:**

- "No delta": Check `changes/[name]/specs/*.md` exists with `## ADDED|MODIFIED|REMOVED` headers
- "No scenario": Ensure `#### Scenario:` format (4 hashtags, not bullets/bold)
- Silent parsing: Debug with `openspec show [change] --json --deltas-only`

**Debug:** `openspec validate [change] --strict --no-interactive`

## Best Practices

- **Simplicity**: Default <100 lines, single-file until proven insufficient, boring patterns
- **Complexity**: Only with data (perf, scale >1000 users, multiple use cases)
- **References**: Use `file.ts:42` format for code, `specs/auth/spec.md` for specs
- **Naming**: Capabilities = verb-noun (`user-auth`), Changes = kebab-case verb-led (`add-2fa`)

**Remember:** Specs = truth (what IS). Changes = proposals (what SHOULD change). Keep in sync.
