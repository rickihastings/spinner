# Implementation Tasks

- [x] Reorganize templates directory structure
  - [x] Create `templates/docker/` directory
  - [x] Create `templates/scripts/` directory
  - [x] Create `templates/skills/` directory
  - [x] Move `startup.sh` to `templates/scripts/startup.sh`
  - [x] Move `task-implementation-lifecycle.skill.md` to `templates/skills/task-implementation-lifecycle.skill.md`
  - [x] Delete old `templates/Dockerfile.template` (will be replaced)
  - [x] Create new `templates/docker/extending.template` for minimal base-extending Dockerfile
  - [x] **Validation**: New directory structure exists, old template is removed

- [x] Update template path references in code
  - [x] Update dockerfile.ts to reference `templates/docker/extending.template`
  - [x] Update any references to `templates/startup.sh` to use `templates/scripts/startup.sh`
  - [x] Update any references to skill template to use `templates/skills/task-implementation-lifecycle.skill.md`
  - [x] **Validation**: All imports resolve correctly, TypeScript compiles

- [x] Update CLI argument parsing
  - [x] Remove `--jvm-url` and `--node-version` flags from setup command
  - [x] Add `--base-image` flag (string, optional)
  - [x] Add `--dockerfile` flag (string, optional)
  - [x] Add validation to ensure `--base-image` and `--dockerfile` are mutually exclusive
  - [x] Update help text to document the new flags and Ubuntu/Debian requirement
  - [x] **Validation**: `setup --help` shows new flags, running with old flags shows error

- [x] Update SetupProps interface
  - [x] Remove `jvmUrl: string` and `nodeVersion: string` from SetupProps
  - [x] Add `baseImage?: string` and `dockerfile?: string` to SetupProps
  - [x] Update Setup.tsx to accept new props
  - [x] **Validation**: TypeScript compilation succeeds

- [x] Add Dockerfile path validation utility
  - [x] Create utility function to check if Dockerfile path exists
  - [x] Add validation in Setup.tsx before proceeding to build
  - [x] Display clear error message if path is invalid
  - [x] **Validation**: Running with `--dockerfile ./missing.txt` shows appropriate error

- [x] Implement custom Dockerfile build logic
  - [x] Add function in docker.ts to build user's Dockerfile first
  - [x] Tag user's image as `spinner-base:<name>`
  - [x] Handle Docker build failures and surface errors to user
  - [x] **Validation**: User Dockerfile is built and tagged correctly

- [x] Create new minimal Dockerfile template
  - [x] Design template that extends user's base with ARG for base image
  - [x] Add conditional RUN statements that check for git before installing
  - [x] Add conditional RUN statements that check for claude before installing
  - [x] Use `command -v git` to detect existing git installation
  - [x] Use `command -v claude` to detect existing claude installation
  - [x] Preserve startup script copying logic
  - [x] Preserve skill template copying logic
  - [x] **Validation**: Generated Dockerfile only installs missing dependencies

- [x] Update generateDockerfile function
  - [x] Replace existing implementation to use new extending template
  - [x] Accept baseImage parameter (either user-provided or spinner-base:<name>)
  - [x] Generate FROM statement using baseImage parameter
  - [x] Remove NODE_VERSION and JVM_URL template variable substitution
  - [x] Remove nvm installation logic
  - [x] Remove JDK download and installation logic
  - [x] **Validation**: Generated Dockerfile matches expected minimal format

- [x] Update docker.ts buildImage function
  - [x] Update function signature to accept baseImage or dockerfile parameters
  - [x] Remove nodeVersion and jvmUrl parameters
  - [x] Add logic to build user's Dockerfile first if dockerfile param provided
  - [x] Pass correct base image reference to generateDockerfile
  - [x] Ensure final image is tagged as spinner:<name>
  - [x] **Validation**: Both base-image and dockerfile flows create correct final image

- [x] Update integration tests
  - [x] Remove tests for --jvm-url and --node-version flags
  - [x] Add tests for --base-image flag with ubuntu:22.04
  - [x] Add test for mutual exclusivity of flags (both provided = error)
  - [x] Keep tests verifying git is present in final image
  - [x] Keep tests verifying claude is present in final image
  - [x] **Validation**: All tests pass

- [x] Update documentation
  - [x] Update README or CLI help with new usage examples
  - [x] Document Ubuntu/Debian requirement clearly
  - [x] Add example: using --base-image ubuntu:22.04
  - [x] Add example: using --base-image node:20-bullseye
  - [x] Add example: using --dockerfile with custom Dockerfile
  - [x] Document breaking change from --jvm-url/--node-version to new flags
  - [x] Add migration guide for existing users
  - [x] **Validation**: Documentation accurately reflects new behavior
