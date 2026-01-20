## Context

The spinner CLI creates Docker containers for isolated development environments. Currently, after cloning a repository, containers idle indefinitely. This change implements the "Ralph loop" pattern to enable autonomous AI-driven task implementation.

The Ralph loop is named after the ghuntley/how-to-ralph-wiggum methodology where Claude runs in a continuous loop, implementing tasks one at a time until all work is complete.

### Stakeholders
- Developers using spinner for autonomous code generation
- CI/CD systems that need containers to complete and exit

### Constraints
- Must work with existing task-implementation-lifecycle skill
- Container must exit cleanly on completion or max iterations

## Goals / Non-Goals

### Goals
- Enable autonomous feature implementation via Ralph loop when prompt provided
- Maintain backward compatibility (containers can still be idle without prompt)
- Ralph loop runs on default branch when branch not specified
- Support passing prompt string via CLI argument
- Support branch selection via CLI argument (optional)
- Configurable max iterations with sensible default (100)
- Detect completion via `~~ FEATURE_COMPLETED ~~` signal
- Refactor spin logic to follow SOLID principles

### Non-Goals
- Planning phase (tasks must pre-exist in repo)
- Automatic git push (user handles pushing)
- Health monitoring or external status reporting
- Prompt file support (prompt is passed as string)
- Breaking changes to existing idle container behavior

## Technical Implementation Plan

### Component Map

| File                      | Change                                               | Type   |
|---------------------------|------------------------------------------------------|--------|
| `src/commands/Spin.tsx`   | Add `--prompt`, `--branch`, `--max-iterations` flags | modify |
| `templates/startup.sh`    | Add branch checkout and Ralph loop execution         | modify |
| `templates/ralph-loop.sh` | New script containing the loop logic                 | create |

### Approach

1. **CLI changes**:
   - Add optional `--prompt` flag that accepts a prompt string
   - Add optional `--branch` flag that accepts a branch name
   - Add optional `--max-iterations` flag (default: 100)
   - Remove validation requiring both prompt and branch together
   - Pass as `PROMPT`, `BRANCH` (if provided), `MAX_ITERATIONS` environment variables to container

2. **Refactor to SOLID principles** (utils/docker.ts):
   - Extract spin logic from Spin.tsx into focused utility functions:
     - `validatePrerequisites()` - Single responsibility: validation
     - `generateContainerName()` - Single responsibility: name generation
     - `buildDockerRunCommand()` - Single responsibility: command building
     - `executeDockerRun()` - Single responsibility: execution
     - `verifyContainerStatus()` - Single responsibility: status checking
   - Define TypeScript interfaces for inputs/outputs (Dependency Inversion)
   - Spin.tsx becomes thin orchestration layer (UI only)

3. **Startup script** (startup.sh):
   - Check if `PROMPT` is set
   - If set:
     - If `BRANCH` is set: checkout or create the branch
     - If `BRANCH` not set: stay on default branch
     - Execute ralph-loop.sh
   - If not set: idle with `tail -f /dev/null`

4. **Ralph loop** (ralph-loop.sh):
   - Read prompt from `$PROMPT` environment variable
   - Loop: pipe prompt to `claude --dangerously-skip-permissions`
   - Track iteration count
   - Capture output, check for `~~ FEATURE_COMPLETED ~~`
   - On signal: exit 0 with success message
   - On max iterations: exit 0 with max iterations message

### Loop Pseudocode

```bash
#!/bin/bash
set -e

ITERATION=0

while [ $ITERATION -lt $MAX_ITERATIONS ]; do
  ITERATION=$((ITERATION + 1))
  echo "=== Ralph Loop Iteration $ITERATION/$MAX_ITERATIONS ==="

  OUTPUT=$(echo "$PROMPT" | claude --dangerously-skip-permissions 2>&1)
  echo "$OUTPUT"

  if echo "$OUTPUT" | grep -q "~~ FEATURE_COMPLETED ~~"; then
    echo "Feature completed after $ITERATION iterations. Exiting."
    exit 0
  fi
done

echo "Max iterations ($MAX_ITERATIONS) reached. Exiting."
exit 0
```

### Patterns to Follow

- See `templates/startup.sh:1-23` for existing bash script patterns
- Use `set -e` for fail-fast behavior
- Echo status messages for observability

### Key Decisions

| Decision                                     | Rationale                                                                        |
|----------------------------------------------|----------------------------------------------------------------------------------|
| Separate ralph-loop.sh script                | Cleaner separation, easier testing                                               |
| Prompt as string, not file                   | Simpler, more flexible - prompt doesn't need to exist in repo                    |
| Prompt and branch both optional              | Maintains backward compatibility; Ralph loop runs on default when branch omitted |
| Max iterations default 100                   | Safety limit to prevent runaway loops                                            |
| No git push on completion                    | User controls when to push; simpler implementation                               |
| Use `--dangerously-skip-permissions`         | Required for autonomous operation                                                |
| Extract spin logic to utils/docker.ts        | Follows SOLID principles; improves testability and maintainability               |
| Use TypeScript interfaces for data contracts | Enables type safety and clear API boundaries                                     |

## Risks / Trade-offs

| Risk                                              | Mitigation                                         |
|---------------------------------------------------|----------------------------------------------------|
| Loop exits at max iterations with incomplete work | 100 is generous default; work is committed locally |
| Large Claude output memory usage                  | Stream output rather than capture all              |
| Branch doesn't exist                              | Auto-create from default branch                    |

## Open Questions

None - all questions resolved during requirements discussion.
