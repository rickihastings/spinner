# Design: Support Custom Base Images

## Context

The current spinner CLI hardcodes JVM and Node.js installation via `--jvm-url` and `--node-version` flags. This approach doesn't scale for diverse development environments. Users need flexibility to provide their own base Docker image containing their complete development stack (Python, Ruby, custom tools, etc.).

**Constraints:**
- Must support Ubuntu/Debian-based images only (apt-get package manager)
- Must preserve existing startup script and skill template functionality
- Breaking change: removes `--jvm-url` and `--node-version` flags

**Stakeholders:**
- CLI users who want custom development environments
- Users migrating from the current JVM/Node-focused setup

## Goals / Non-Goals

**Goals:**
- Allow users to provide custom base images via `--base-image` flag
- Allow users to provide custom Dockerfiles via `--dockerfile` flag
- Only install git and claude-code if they're missing from the base
- Default to `ubuntu:22.04` when no base image is specified
- Maintain existing startup script and skill template functionality

**Non-Goals:**
- Supporting non-Debian package managers (Alpine apk, RHEL yum, etc.)
- Auto-detecting and installing user's preferred tools
- Backwards compatibility shims for old flags

## Technical Implementation Plan

### Component Map

| Path | Change | Description |
|------|--------|-------------|
| `templates/docker/extending.template` | create | New minimal Dockerfile template with conditional deps |
| `templates/scripts/startup.sh` | move | Moved from `templates/startup.sh` |
| `templates/skills/task-implementation-lifecycle.skill.md` | move | Moved from `templates/` |
| `templates/Dockerfile.template` | delete | Old template - replaced by extending.template |
| `src/utils/dockerfile.ts` | modify | New config interface, use extending template |
| `src/utils/docker.ts` | modify | Build user Dockerfile first, new BuildConfig |
| `src/commands/Setup.tsx` | modify | New props, Dockerfile path validation |
| `src/App.tsx` | modify | New flag validation, updated help text |
| `src/cli.tsx` | modify | Updated flags type definition |

### Approach

**Phase 1: Template Reorganization**
1. Create new directory structure under `templates/`
2. Move startup.sh and skill template to new locations
3. Create the new `extending.template` Dockerfile
4. Delete old `Dockerfile.template`

**Phase 2: Core Logic Updates**
1. Update `dockerfile.ts` with new `DockerfileConfig` interface accepting `baseImage: string`
2. Update template path reference to `templates/docker/extending.template`
3. Update `docker.ts` to accept `baseImage` or `dockerfile` parameters
4. Add logic to build user's Dockerfile first and tag as `spinner-base:<name>`

**Phase 3: CLI Updates**
1. Update `SetupProps` interface in `Setup.tsx`
2. Add Dockerfile path existence validation
3. Update `App.tsx` flag validation logic for mutually exclusive flags
4. Update help text with new flags and examples
5. Update `cli.tsx` flags type

**Phase 4: Testing & Documentation**
1. Remove tests for old flags
2. Add tests for new flag combinations
3. Update documentation with migration guide

### Patterns to Follow

**Conditional dependency installation pattern** (for extending.template):
```dockerfile
# Check if command exists before installing
RUN command -v git > /dev/null 2>&1 || \
    (apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*)
```

**Template variable substitution** (see `src/utils/dockerfile.ts:17-19`):
```typescript
return template.replace(/{{BASE_IMAGE}}/g, config.baseImage);
```

**User Dockerfile build flow** (for docker.ts):
```typescript
// 1. Build user's Dockerfile to intermediate tag
execSync(`docker build -t spinner-base:${name} -f ${dockerfilePath} .`);
// 2. Use intermediate tag as base for final image
const finalDockerfile = generateDockerfile({ baseImage: `spinner-base:${name}` });
```

### Key Decisions

| Decision | Rationale |
|----------|-----------|
| Default to `ubuntu:22.04` | Provides sensible default when no base specified, maintains backwards-compatible UX |
| Let apt-get fail at build time | Simpler than pre-checking; clear error message from Docker build output |
| Use `command -v` for detection | POSIX-compliant, works reliably in Dockerfile RUN statements |
| Tag user Dockerfile as `spinner-base:<name>` | Clear naming, avoids collision with final `spinner:<name>` image |
| No CMD override warning | Expected behavior, documented in help text |

### New Dockerfile Template (extending.template)

```dockerfile
ARG BASE_IMAGE
FROM ${BASE_IMAGE}

ENV DEBIAN_FRONTEND=noninteractive

# Install git if not present
RUN command -v git > /dev/null 2>&1 || \
    (apt-get update && apt-get install -y git ca-certificates curl && rm -rf /var/lib/apt/lists/*)

# Install claude-code if not present
RUN command -v claude > /dev/null 2>&1 || \
    (curl -fsSL https://claude.ai/install.sh | bash)
ENV PATH=/root/.local/bin:${PATH}

WORKDIR /workspace

# Copy skill templates
RUN mkdir -p /root/.claude/skills
COPY templates/task-implementation-lifecycle.skill.md /root/.claude/skills/

# Copy and set up startup script
COPY templates/startup.sh /usr/local/bin/startup.sh
RUN chmod +x /usr/local/bin/startup.sh

CMD ["/usr/local/bin/startup.sh"]
```

### Updated Interface Definitions

```typescript
// src/utils/dockerfile.ts
export interface DockerfileConfig {
  baseImage: string;
}

// src/commands/Setup.tsx
export interface SetupProps {
  name: string;
  baseImage: string;   // resolved base image (either user-provided or spinner-base:<name>)
  dockerfile?: string; // optional path to user's Dockerfile
}

// src/utils/docker.ts
export interface BuildConfig {
  name: string;
  baseImage?: string;   // e.g., "ubuntu:22.04", "node:20-bullseye"
  dockerfile?: string;  // path to user's Dockerfile
}
```

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Breaking change removes JVM/Node flags | Document migration path clearly; users can pre-install in their base image |
| apt-get failure on non-Debian images is cryptic | Add comment in error output suggesting Ubuntu/Debian requirement |
| User Dockerfile build failures | Surface Docker's error output directly to user |
| Base image may not have curl | Pre-install curl when installing git (needed for claude installer) |

## Open Questions

None - all design questions have been resolved.
