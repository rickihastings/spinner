# Implementing Approved Changes

This guide covers Stage 2: Implementing approved change proposals through vertical slices.

## Critical Rules

**ONE VERTICAL SLICE PER INVOCATION:**

- Complete all sub-tasks (X.1, X.2, X.3...) in a single slice (X.0)
- ALWAYS halt after completing a slice
- Never automatically continue to the next slice
- Update tasks.md before halting
- Commit changes before halting

**STOPPING CONDITIONS:**

- After completing a slice → Check if uncompleted slices remain
  - **Remaining slices exist** → HALT (do not show signal)
  - **NO remaining slices** → Output `~~ FEATURE_COMPLETED ~~`, then HALT

## Implementation Workflow

Follow this sequence for each vertical slice:

### 1. Prepare

- Read `changes/[change-id]/design.md` for technical approach
- Read `changes/[change-id]/tasks.md` for work breakdown
- Verify correct branch and clean working directory
- Review relevant specs in `openspec/specs/[capability]/spec.md`

### 2. Select Slice

- Identify the first incomplete vertical slice (lowest X.0 with unchecked sub-tasks)
- Verify all sub-tasks (X.1, X.2, X.3...) are clear
- If ambiguous, ask clarifying questions before starting

### 3. Implement Slice

Complete ALL sub-tasks in the vertical slice before committing.

**For EACH sub-task in the slice (X.1, X.2, X.3...):**

1. **Investigate** - Search codebase for existing patterns before coding
2. **Implement** - Make minimal, focused changes following existing patterns
3. **Verify** - Run builds and tests; fix any failures before proceeding
4. **Mark Complete** - Check off the sub-task in tasks.md (`- [x]`)

**Repeat the above cycle for ALL sub-tasks in the slice before proceeding to step 4.**

**Rules:**

- Complete the entire vertical slice (all sub-tasks X.1, X.2, X.3...) in one session
- Do NOT commit after each sub-task—commit only after the entire slice is complete
- Each commit should leave the codebase in a valid, tested state
- Never defer tests—they are part of the vertical slice, not a separate phase
- If implementation diverges from spec, update the spec immediately

### 4. Finalize Slice

**ONLY after completing ALL sub-tasks in the slice:**

1. **Update tasks.md** - Mark the slice header complete with `- [x]`
2. **Commit** - Create meaningful commit message
3. **Push** - Push changes to remote

### 5. Check and Halt

After finalizing the slice, check for remaining work:

```bash
# Check tasks.md for any uncompleted slices
# Look for ## X.0 headings with unchecked sub-tasks
```

**Decision:**

- **Uncompleted slices remain** → HALT. Do not show any signal. Stop execution.
- **All slices complete** → Output `~~ FEATURE_COMPLETED ~~`, then HALT.

**CRITICAL:** Never continue to the next slice automatically. Always halt after completing one vertical slice.

## Quality Guidelines

- Follow existing code patterns (investigate before coding)
- Keep changes minimal and focused
- Run project-specific quality checks before committing
- Each commit must leave codebase in valid, tested state
- If implementation diverges from spec, update the spec delta immediately
