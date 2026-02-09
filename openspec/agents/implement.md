# Implementing Approved Changes

This guide covers Stage 2: Implementing approved change proposals through vertical slices.

## Critical Rules

**UNDERSTANDING SLICES:**

A "slice" is the X.0 header AND ALL its numbered sub-tasks:

```markdown
## 10.0 Documentation updates          ← This is the slice header
- [ ] 10.1 Update README.md            ← Sub-task 1
- [ ] 10.2 Update docs/usage.md        ← Sub-task 2
- [ ] 10.3 Add migration notes         ← Sub-task 3
- [ ] 10.4 Update examples             ← Sub-task 4
- [ ] 10.5 Verify documentation        ← Sub-task 5
```

**ONE VERTICAL SLICE PER INVOCATION:**

- Complete ALL sub-tasks (10.1 AND 10.2 AND 10.3 AND 10.4 AND 10.5) before committing
- After ALL sub-tasks are complete, mark the slice header (10.0) complete
- Then commit ONCE for the entire slice
- ALWAYS halt after completing and committing a slice
- Never automatically continue to the next slice

**STOPPING CONDITIONS (read carefully - the completion signal must ONLY appear when every slice is done):**

- After completing a slice → Read tasks.md and check if ANY uncompleted slices remain
  - **ANY uncompleted slices remain** → HALT immediately. Do NOT output `~~ FEATURE_COMPLETED ~~`. No signal whatsoever.
  - **ALL slices are complete (zero uncompleted slices in the entire tasks.md)** → Output `~~ FEATURE_COMPLETED ~~`, then HALT

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

**Example:** If you're working on slice 10.0 with sub-tasks 10.1, 10.2, 10.3, 10.4, 10.5:

1. Complete 10.1 → Check it off → Continue to 10.2 (DO NOT COMMIT YET)
2. Complete 10.2 → Check it off → Continue to 10.3 (DO NOT COMMIT YET)
3. Complete 10.3 → Check it off → Continue to 10.4 (DO NOT COMMIT YET)
4. Complete 10.4 → Check it off → Continue to 10.5 (DO NOT COMMIT YET)
5. Complete 10.5 → Check it off → NOW proceed to step 4 (Finalize Slice)

**For EACH sub-task in the slice:**

1. **Investigate** - Search codebase for existing patterns before coding
2. **Implement** - Make minimal, focused changes following existing patterns
3. **Verify** - Run builds and tests; fix any failures before proceeding
4. **Mark Complete** - Check off the sub-task in tasks.md (`- [x]`)
5. **Continue** - Move to the next sub-task in the slice (DO NOT commit yet)

**Repeat the above cycle for ALL sub-tasks before proceeding to step 4.**

**Rules:**

- Complete the entire vertical slice (all sub-tasks X.1, X.2, X.3...) in one session
- DO NOT commit after each sub-task—commit only after the entire slice is complete
- Each commit should leave the codebase in a valid, tested state
- Never defer tests—they are part of the vertical slice, not a separate phase
- If implementation diverges from spec, update the spec immediately

### 4. Finalize Slice

**ONLY after ALL sub-tasks (X.1, X.2, X.3, X.4, X.5...) are checked off:**

1. **Update tasks.md** - Mark the slice header (X.0) complete with `- [x]`
2. **Commit** - Create one meaningful commit message for the entire slice
3. **Push** - Push changes to remote

**Example:** For slice 10.0, only commit after tasks.md shows:
```markdown
## 10.0 Documentation updates
- [x] 10.1 Update README.md
- [x] 10.2 Update docs/usage.md
- [x] 10.3 Add migration notes
- [x] 10.4 Update examples
- [x] 10.5 Verify documentation
```

### 5. Check and Halt

After finalizing the slice, check for remaining work:

```bash
# Check tasks.md for any uncompleted slices
# Look for ## X.0 headings with unchecked sub-tasks
```

**Decision (the completion signal must ONLY appear when every slice is done):**

- **ANY uncompleted slices remain** → HALT immediately. Do NOT output `~~ FEATURE_COMPLETED ~~`. No signal whatsoever. Stop execution.
- **ALL slices are complete (zero remaining in the entire tasks.md)** → Output `~~ FEATURE_COMPLETED ~~`, then HALT.

**CRITICAL:** Never continue to the next slice automatically. Always halt after completing one vertical slice.

## Quality Guidelines

- Follow existing code patterns (investigate before coding)
- Keep changes minimal and focused
- Run project-specific quality checks before committing
- Each commit must leave codebase in valid, tested state
- If implementation diverges from spec, update the spec delta immediately
