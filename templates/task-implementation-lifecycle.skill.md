# Task Implementation Lifecycle Skill

A deterministic process for implementing tasks through a structured 10-step lifecycle. Ensures consistent quality,
prevents runaway execution, and maintains spec alignment.

## Prerequisites

Verify a task list exists before beginning. If not found, alert the user and halt.

## The 10-Step Lifecycle

### 1. Orient

Read available specifications and understand the implementation goal. Spec-agnostic (supports OpenSpec, speckit, or any
format).

### 2. Read Plan

Find and read the task list (tasks.md, TODO.md, etc). Review all tasks and identify dependencies.

### 3. Select Task

Select the next incomplete task and mark it as in-progress. If the spec indicates multiple tasks can be done in
parallel, you may select and process them together.

### 4. Investigate

Spawn subagents to explore the codebase. Search for existing implementations. DO NOT assume functionality doesn't exist.

### 5. Implement

Spawn subagents for file operations. Make minimal, focused changes following existing patterns.

### 6. Validate

Spawn validation subagent to run builds and tests. Fix any failures before proceeding.

### 7. Update Plan

Mark completed task(s) as done (e.g., `- [x]`) in the task list.

### 8. Update Spec on Divergence

Compare implementation against spec. If divergence exists, update the relevant spec document (any format).

### 9. Commit

Stage changes and create a git commit with a meaningful message.

### 10. Halt

Check for remaining tasks. If none remain, output `~~ FEATURE_COMPLETED ~~`. Halt execution and require explicit
re-invocation to continue.

## Rules

1. **Task Execution**: Process one task per invocation, unless the spec explicitly indicates tasks can be done in
   parallel
2. **No Assumptions**: Always investigate before implementing
3. **Mandatory Halt**: Always halt after step 10
4. **Subagent Delegation**: Use subagents for investigation, implementation, and validation
5. **Feature Completion**: Output `~~ FEATURE_COMPLETED ~~` when all tasks complete

## Error Handling

If any step fails, halt immediately. Do not proceed, mark complete, or commit. Report the error and await intervention.

## Supported Formats

- GitHub checkboxes: `- [ ]` / `- [x]`
- Numbered lists with status
- Custom formats with clear markers
