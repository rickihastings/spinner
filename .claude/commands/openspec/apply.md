---
name:
  OpenSpec: Apply
description: Implement an approved OpenSpec change and keep tasks in sync.
category: OpenSpec
tags: [ openspec, apply ]
---

<!-- OPENSPEC:START -->
**CRITICAL: Implement ONE Vertical Slice Only**

You are implementing a SINGLE vertical feature slice (e.g., "1.0 Implement deterministic container naming"). Do NOT
create a TODO list for all tasks in the change. Your job is to:

1. Implement ALL sub-tasks (X.1, X.2, X.3...) within the ONE assigned vertical slice
2. Commit after logical checkpoints as you complete sub-tasks
3. Signal completion when the entire feature is done and there are no remaining tasks

**Workflow**

1. **Read Context**
    - Read `changes/<id>/proposal.md`, `design.md` (if present), and `tasks.md`
    - Identify which vertical slice (X.0) you are implementing
    - If unclear which slice, ask the user

2. **Per-Task Cycle** (repeat for each sub-task X.1, X.2, X.3... in the slice)
    - **Select**: Pick next incomplete sub-task in the vertical slice
    - **Investigate**: Search codebase for existing patterns before coding
    - **Implement**: Make minimal, focused changes following existing patterns
    - **Verify**: Run builds and tests; fix any failures before proceeding
    - **Update**: Mark sub-task complete in `tasks.md` (change `- [ ]` to `- [x]`)
    - **Commit**: Create meaningful commit describing what you just completed
    - **Repeat**: Continue until all sub-tasks in the slice are done

3. **Signal Completion**
    - When all sub-tasks in the vertical slice are complete, you MUST output the exact signal: `~~ FEATURE_COMPLETED ~~`
    - This signal MUST be output as plain text in your response, not in a code block or comment
    - Output this signal immediately after completing the final commit for the vertical slice
    - After outputting the signal, halt - do not continue to the next vertical slice

**Guardrails**

- Do NOT create a TODO list for the entire change - only implement the assigned slice
- Do NOT skip tests - they are part of the vertical slice
- Each commit should leave the codebase in a valid, tested state
- If implementation diverges from spec, update the spec immediately
- Favor straightforward, minimal implementations first
- Keep changes tightly scoped to the requested outcome

**Reference**

- See `openspec/AGENTS.md` Stage 2 for full workflow details
- Use `openspec show <id> --json --deltas-only` for additional context

<!-- OPENSPEC:END -->
