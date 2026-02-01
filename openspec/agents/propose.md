# Creating Change Proposals

This guide covers Stage 1: Creating proposals for new features, breaking changes, or architectural updates.

## TL;DR Quick Checklist

- Search existing work: `openspec spec list --long`, `openspec list` (use `rg` only for full-text search)
- Decide scope: new capability vs modify existing capability
- Pick a unique `change-id`: kebab-case, verb-led (`add-`, `update-`, `remove-`, `refactor-`)
- Scaffold: `proposal.md`, `tasks.md`, `design.md` (only if needed), and delta specs per affected capability
- Write deltas: use `## ADDED|MODIFIED|REMOVED|RENAMED Requirements`; include at least one `#### Scenario:` per
  requirement
- Validate: `openspec validate [change-id] --strict --no-interactive` and fix issues
- Request approval: Do not start implementation until proposal is approved

## When to Create a Proposal

Create a proposal when you need to:

- Add features or functionality
- Make breaking changes (API, schema)
- Change architecture or patterns
- Optimize performance (changes behavior)
- Update security patterns

Triggers (examples):

- "Help me create a change proposal"
- "Help me plan a change"
- "Help me create a proposal"
- "I want to create a spec proposal"
- "I want to create a spec"

Loose matching guidance:

- Contains one of: `proposal`, `change`, `spec`
- With one of: `create`, `plan`, `make`, `start`, `help`

Skip proposal for:

- Bug fixes (restore intended behavior)
- Typos, formatting, comments
- Dependency updates (non-breaking)
- Configuration changes
- Tests for existing behavior

## Proposal Workflow

1. Review `openspec/project.md`, `openspec list`, and `openspec list --specs` to understand current context.
2. Choose a unique verb-led `change-id` and scaffold `proposal.md`, `design.md`, `tasks.md`, and spec deltas under
   `openspec/changes/<id>/`.
3. Draft spec deltas using `## ADDED|MODIFIED|REMOVED Requirements` with at least one `#### Scenario:` per requirement.
4. Run `openspec validate <id> --strict --no-interactive` and resolve any issues before sharing the proposal.

## Decision Tree

```
New request?
├─ Bug fix restoring spec behavior? → Fix directly
├─ Typo/format/comment? → Fix directly
├─ New feature/capability? → Create proposal
├─ Breaking change? → Create proposal
├─ Architecture change? → Create proposal
└─ Unclear? → Create proposal (safer)
```

## Proposal Structure

### 1. Create Directory

Create `changes/[change-id]/` (kebab-case, verb-led, unique)

### 2. Write proposal.md

```markdown
# Change: [Brief description of change]

## Why

[1-2 sentences on problem/opportunity]

## What Changes

- [Bullet list of changes]
- [Mark breaking changes with **BREAKING**]

## Impact

- Affected specs: [list capabilities]
- Affected code: [key files/systems]
```

### 3. Create Spec Deltas

File: `specs/[capability]/spec.md`

```markdown
## ADDED Requirements

### Requirement: New Feature

The system SHALL provide...

#### Scenario: Success case

- **WHEN** user performs action
- **THEN** expected result

## MODIFIED Requirements

### Requirement: Existing Feature

[Complete modified requirement]

## REMOVED Requirements

### Requirement: Old Feature

**Reason**: [Why removing]
**Migration**: [How to handle]
```

If multiple capabilities are affected, create multiple delta files under
`changes/[change-id]/specs/<capability>/spec.md`—one per capability.

### 4. Create tasks.md

Tasks MUST be organized as **vertical feature slices**, not horizontal layers. Each top-level task (X.0) represents a
complete, testable, committable feature.

**Structure:**

- `X.0` = Complete feature (e.g., "Implement `--prune` command")
- `X.1-X.n` = Sub-tasks that together deliver the feature
- Each `X.0` MUST include: implementation + tests + docs + verification
- After completing `X.0`, the codebase should be in a valid, tested, documented state

**What belongs in a vertical slice:**

1. Implementation code
2. Tests for that feature
3. Documentation updates (help text, comments, README sections)
4. Verification that everything works

**Example:**

```markdown
## 1.0 Implement `--prune` command

- [ ] 1.1 Add `--prune` flag to CLI argument parser
- [ ] 1.2 Implement container pruning logic in utils
- [ ] 1.3 Update help text to document `--prune` behavior
- [ ] 1.4 Add tests for `--prune` functionality
- [ ] 1.5 Verify tests pass and feature works end-to-end

## 2.0 Implement container listing

- [ ] 2.1 Add `--list` flag to CLI
- [ ] 2.2 Implement listing logic with filtering
- [ ] 2.3 Update help text to document `--list` behavior
- [ ] 2.4 Add tests for listing behavior
- [ ] 2.5 Verify tests pass
```

**Anti-pattern (DO NOT do this):**

```markdown
## 1. CLI Changes ← Horizontal slice, tests deferred

- [ ] 1.1 Add --prune flag
- [ ] 1.2 Add --list flag

## 2. Backend Logic

- [ ] 2.1 Implement pruning
- [ ] 2.2 Implement listing

## 3. Tests ← Tests come last, no TDD

- [ ] 3.1 Test pruning
- [ ] 3.2 Test listing
```

**Why vertical slices:**

- Each feature is independently committable
- Aligns with implementation workflow (verify → commit → next)
- Enables TDD workflow
- Reduces context switching between unrelated code areas
- Clear progress: "Feature 1 done" vs "CLI half-done, backend half-done"

### 5. Create design.md

Every change MUST include a `design.md` with a Technical Implementation Plan. This provides implementing agents with
guidance on how to approach the work.

`design.md` template:

```markdown
## Context

[Background, constraints, stakeholders]

## Goals / Non-Goals

- Goals: [...]
- Non-Goals: [...]

## Technical Implementation Plan

### Component Map

- `path/to/file.ts` - [what changes] (create|modify|delete)
- `path/to/other.ts` - [what changes]

### Approach

[High-level strategy for implementation - what order, what patterns]

### Patterns to Follow

- See `path/to/example.ts:45-60` for [pattern name]
- [Other patterns to reference]

### Key Decisions

- [Decision]: [rationale]
- [Decision]: [rationale]

## Risks / Trade-offs

- [Risk] → Mitigation

## Open Questions

- [...]
```

The Technical Implementation Plan section is critical for guiding agents during implementation. At minimum, include
Component Map and Approach.

## Happy Path Script

```bash
# 1) Explore current state
openspec spec list --long
openspec list
# Optional full-text search:
# rg -n "Requirement:|Scenario:" openspec/specs
# rg -n "^#|Requirement:" openspec/changes

# 2) Choose change id and scaffold
CHANGE=add-two-factor-auth
mkdir -p openspec/changes/$CHANGE/{specs/auth}
printf "## Why\n...\n\n## What Changes\n- ...\n\n## Impact\n- ...\n" > openspec/changes/$CHANGE/proposal.md
printf "## 1.0 Implement OTP verification\n- [ ] 1.1 Add OTP input to login flow\n- [ ] 1.2 Implement OTP validation logic\n- [ ] 1.3 Add tests for OTP scenarios\n- [ ] 1.4 Verify tests pass\n" > openspec/changes/$CHANGE/tasks.md

# 3) Add deltas (example)
cat > openspec/changes/$CHANGE/specs/auth/spec.md << 'EOF'
## ADDED Requirements
### Requirement: Two-Factor Authentication
Users MUST provide a second factor during login.

#### Scenario: OTP required
- **WHEN** valid credentials are provided
- **THEN** an OTP challenge is required
EOF

# 4) Validate
openspec validate $CHANGE --strict --no-interactive
```

## Multi-Capability Example

```
openspec/changes/add-2fa-notify/
├── proposal.md
├── tasks.md
└── specs/
    ├── auth/
    │   └── spec.md   # ADDED: Two-Factor Authentication
    └── notifications/
        └── spec.md   # ADDED: OTP email notification
```

auth/spec.md:

```markdown
## ADDED Requirements

### Requirement: Two-Factor Authentication

...
```

notifications/spec.md:

```markdown
## ADDED Requirements

### Requirement: OTP Email Notification

...
```

## Spec File Format

### Critical: Scenario Formatting

**CORRECT** (use #### headers):

```markdown
#### Scenario: User login success

- **WHEN** valid credentials provided
- **THEN** return JWT token
```

**WRONG** (don't use bullets or bold):

```markdown
- **Scenario: User login**  ❌
**Scenario**: User login ❌

### Scenario: User login ❌
```

Every requirement MUST have at least one scenario.

### Requirement Wording

- Use SHALL/MUST for normative requirements (avoid should/may unless intentionally non-normative)

### Delta Operations

- `## ADDED Requirements` - New capabilities
- `## MODIFIED Requirements` - Changed behavior
- `## REMOVED Requirements` - Deprecated features
- `## RENAMED Requirements` - Name changes

Headers matched with `trim(header)` - whitespace ignored.

### When to use ADDED vs MODIFIED

- **ADDED**: Introduces a new capability or sub-capability that can stand alone as a requirement. Prefer ADDED when the
  change is orthogonal (e.g., adding "Slash Command Configuration") rather than altering the semantics of an existing
  requirement.
- **MODIFIED**: Changes the behavior, scope, or acceptance criteria of an existing requirement. Always paste the full,
  updated requirement content (header + all scenarios). The archiver will replace the entire requirement with what you
  provide here; partial deltas will drop previous details.
- **RENAMED**: Use when only the name changes. If you also change behavior, use RENAMED (name) plus MODIFIED (content)
  referencing the new name.

Common pitfall: Using MODIFIED to add a new concern without including the previous text. This causes loss of detail at
archive time. If you aren't explicitly changing the existing requirement, add a new requirement under ADDED instead.

### Authoring a MODIFIED Requirement

1. Locate the existing requirement in `openspec/specs/<capability>/spec.md`.
2. Copy the entire requirement block (from `### Requirement: ...` through its scenarios).
3. Paste it under `## MODIFIED Requirements` and edit to reflect the new behavior.
4. Ensure the header text matches exactly (whitespace-insensitive) and keep at least one `#### Scenario:`.

### Example for RENAMED

```markdown
## RENAMED Requirements

- FROM: `### Requirement: Login`
- TO: `### Requirement: User Authentication`
```

## Before Creating Specs

- Always check if capability already exists
- Prefer modifying existing specs over creating duplicates
- Use `openspec show [spec]` to review current state
- If request is ambiguous, ask 1–2 clarifying questions before scaffolding
