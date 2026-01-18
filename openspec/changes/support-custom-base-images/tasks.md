# Implementation Tasks

- [ ] Reorganize templates directory structure
  - [ ] Create `templates/docker/` directory
  - [ ] Create `templates/scripts/` directory
  - [ ] Create `templates/skills/` directory
  - [ ] Move `startup.sh` to `templates/scripts/startup.sh`
  - [ ] Move `task-implementation-lifecycle.skill.md` to `templates/skills/task-implementation-lifecycle.skill.md`
  - [ ] Delete old `templates/Dockerfile.template` (will be replaced)
  - [ ] Create new `templates/docker/extending.template` for minimal base-extending Dockerfile
  - [ ] **Validation**: New directory structure exists, old template is removed

- [ ] Update template path references in code
  - [ ] Update dockerfile.ts to reference `templates/docker/extending.template`
  - [ ] Update any references to `templates/startup.sh` to use `templates/scripts/startup.sh`
  - [ ] Update any references to skill template to use `templates/skills/task-implementation-lifecycle.skill.md`
  - [ ] **Validation**: All imports resolve correctly, TypeScript compiles

- [ ] Update CLI argument parsing
  - [ ] Remove `--jvm-url` and `--node-version` flags from setup command
  - [ ] Add `--base-image` flag (string, optional)
  - [ ] Add `--dockerfile` flag (string, optional)
  - [ ] Add validation to ensure exactly one of `--base-image` or `--dockerfile` is provided
  - [ ] Update help text to document the new flags and Ubuntu/Debian requirement
  - [ ] **Validation**: `setup --help` shows new flags, running with old flags shows error

- [ ] Update SetupProps interface
  - [ ] Remove `jvmUrl: string` and `nodeVersion: string` from SetupProps
  - [ ] Add `baseImage?: string` and `dockerfile?: string` to SetupProps
  - [ ] Update Setup.tsx to accept new props
  - [ ] **Validation**: TypeScript compilation succeeds

- [ ] Add Dockerfile path validation utility
  - [ ] Create utility function to check if Dockerfile path exists
  - [ ] Add validation in Setup.tsx before proceeding to build
  - [ ] Display clear error message if path is invalid
  - [ ] **Validation**: Running with `--dockerfile ./missing.txt` shows appropriate error

- [ ] Implement custom Dockerfile build logic
  - [ ] Add function in docker.ts to build user's Dockerfile first
  - [ ] Tag user's image as `spinner-base:<name>`
  - [ ] Handle Docker build failures and surface errors to user
  - [ ] **Validation**: User Dockerfile is built and tagged correctly

- [ ] Create new minimal Dockerfile template
  - [ ] Design template that extends user's base with ARG for base image
  - [ ] Add conditional RUN statements that check for git before installing
  - [ ] Add conditional RUN statements that check for claude before installing
  - [ ] Use `command -v git` to detect existing git installation
  - [ ] Use `command -v claude` to detect existing claude installation
  - [ ] Preserve startup script copying logic
  - [ ] Preserve skill template copying logic
  - [ ] **Validation**: Generated Dockerfile only installs missing dependencies

- [ ] Update generateDockerfile function
  - [ ] Replace existing implementation to use new extending template
  - [ ] Accept baseImage parameter (either user-provided or spinner-base:<name>)
  - [ ] Generate FROM statement using baseImage parameter
  - [ ] Remove NODE_VERSION and JVM_URL template variable substitution
  - [ ] Remove nvm installation logic
  - [ ] Remove JDK download and installation logic
  - [ ] **Validation**: Generated Dockerfile matches expected minimal format

- [ ] Update docker.ts buildImage function
  - [ ] Update function signature to accept baseImage or dockerfile parameters
  - [ ] Remove nodeVersion and jvmUrl parameters
  - [ ] Add logic to build user's Dockerfile first if dockerfile param provided
  - [ ] Pass correct base image reference to generateDockerfile
  - [ ] Ensure final image is tagged as spinner:<name>
  - [ ] **Validation**: Both base-image and dockerfile flows create correct final image

- [ ] Update integration tests
  - [ ] Remove tests for --jvm-url and --node-version flags
  - [ ] Add tests for --base-image flag with ubuntu:22.04
  - [ ] Add tests for --dockerfile flag with sample Dockerfile
  - [ ] Add test for mutual exclusivity of flags (both provided = error)
  - [ ] Add test for neither flag provided (error)
  - [ ] Add test verifying git is present in final image
  - [ ] Add test verifying claude is present in final image
  - [ ] Add test verifying base image tools are preserved (e.g., node if using node:20)
  - [ ] **Validation**: All tests pass

- [ ] Update documentation
  - [ ] Update README or CLI help with new usage examples
  - [ ] Document Ubuntu/Debian requirement clearly
  - [ ] Add example: using --base-image ubuntu:22.04
  - [ ] Add example: using --base-image node:20-bullseye
  - [ ] Add example: using --dockerfile with custom Dockerfile
  - [ ] Document breaking change from --jvm-url/--node-version to new flags
  - [ ] Add migration guide for existing users
  - [ ] **Validation**: Documentation accurately reflects new behavior
