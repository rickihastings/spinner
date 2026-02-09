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

1. Read `openspec/agents/implement.md` — it is the single source of truth for the implementation workflow
2. Implement ALL sub-tasks (X.1, X.2, X.3...) within the ONE assigned vertical slice
3. Follow the stopping conditions in `openspec/agents/implement.md` Critical Rules exactly

**Workflow**

Read and follow `openspec/agents/implement.md`. Key points:

1. **Read Context** — Read `changes/<id>/proposal.md`, `design.md`, `tasks.md`, and identify your slice (X.0)
2. **Per-Task Cycle** — For each sub-task: investigate → implement → verify → mark complete → continue (do NOT commit per sub-task)
3. **Finalize Slice** — After ALL sub-tasks are complete, mark slice header done, commit once, push
4. **Check and Halt** — Apply stopping conditions from `openspec/agents/implement.md` Critical Rules

**Guardrails**

- Do NOT create a TODO list for the entire change — only implement the assigned slice
- Do NOT skip tests — they are part of the vertical slice
- Each commit should leave the codebase in a valid, tested state
- If implementation diverges from spec, update the spec immediately
- Favor straightforward, minimal implementations first
- Keep changes tightly scoped to the requested outcome

<!-- OPENSPEC:END -->
