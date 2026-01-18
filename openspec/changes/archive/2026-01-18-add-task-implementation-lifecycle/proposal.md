# Change: Add Task Implementation Lifecycle

## Why

Claude needs a deterministic, reusable skill for implementing tasks that follow a structured lifecycle. This ensures
consistent implementation quality, prevents the agent from running ahead, and maintains alignment between implementation
and specification.

## What Changes

- Add new capability `task-implementation-lifecycle` specifying a 10-step deterministic process
- Skill will be deployed as a template copied to Docker container
- Provides structured workflow from orientation through commit, with mandatory halt after each task
- Spec-agnostic design works with OpenSpec, speckit, or any spec-driven development approach

## Impact

- Affected specs: `task-implementation-lifecycle` (new)
- Affected code: Templates directory (skill implementation), Docker image build
- Changes behavior: Introduces structured implementation workflow for Claude agents
