# Design: Add Setup Flag to Spin Command

## Architecture Overview

This change integrates the setup command's image building capability into the spin command via a new `--setup` flag. The design reuses existing internal packages to minimize code duplication.

## Component Integration

```
cmd/spin.go
    |
    +-- When --setup flag is present:
    |   |
    |   +-- prerequisites.CheckPrerequisites()
    |   |   (reuse from setup command)
    |   |
    |   +-- Validate --base-image vs --dockerfile
    |   |   (replicate setup command logic)
    |   |
    |   +-- docker.BuildImage(config)
    |   |   (reuse from setup command)
    |   |
    |   +-- Continue to container creation
    |
    +-- When --setup flag is absent:
        |
        +-- Existing spin logic (no changes)
```

## Code Reuse Strategy

### 1. Prerequisite Validation
- **Reuse**: `internal/prerequisites.CheckPrerequisites()`
- **Rationale**: Identical prerequisites for both standalone setup and spin with setup
- **Implementation**: Call before `docker.BuildImage()` when `--setup` flag is present

### 2. Image Building
- **Reuse**: `internal/docker.BuildImage(config docker.BuildConfig)`
- **Rationale**: Identical build logic, same Dockerfile generation, same docker build execution
- **Implementation**: Construct `BuildConfig` from spin flags when `--setup` is present

### 3. Flag Validation
- **Replicate**: Mutual exclusivity check for `--base-image` and `--dockerfile`
- **Rationale**: Same validation logic needed in spin command
- **Implementation**: Duplicate the if-statement from `cmd/setup.go` into `cmd/spin.go`
- **Alternative Considered**: Extract to shared validation function, but deemed unnecessary for single conditional

### 4. Dockerfile Path Validation
- **Replicate**: `os.Stat()` check for Dockerfile existence
- **Rationale**: Same validation needed before calling BuildImage
- **Implementation**: Duplicate from `cmd/setup.go` into `cmd/spin.go`

## Data Flow

### Spin with Setup Flow
```
1. User runs: spinner spin --setup --image my-env --base-image ubuntu:22.04 --repo <url>

2. Flag Parsing (Cobra + Viper)
   - setup = true
   - image = "my-env"
   - baseImage = "ubuntu:22.04"
   - repo = "<url>"

3. If setup == true:
   a. CheckPrerequisites()
      - Verify docker, git, claude-code installed

   b. Validate flags
      - Error if both --base-image and --dockerfile provided
      - Validate Dockerfile path exists (if --dockerfile used)

   c. BuildImage(BuildConfig{
        Name: image,           // "my-env"
        BaseImage: baseImage,  // "ubuntu:22.04"
        Dockerfile: dockerfile // or ""
      })
      - Always rebuilds image (no existence check)
      - Tags as "spinner:my-env"

   d. Fall through to existing spin logic

4. Existing Spin Logic
   - Generate container name from image/repo/branch
   - Check container existence
   - Create/reuse/restart container
   - Display management instructions
```

### Spin without Setup Flow (Unchanged)
```
1. User runs: spinner spin --image my-env --repo <url>

2. Flag Parsing
   - setup = false (default)
   - image = "my-env"
   - repo = "<url>"

3. Skip setup logic entirely

4. Existing Spin Logic
   - Proceed directly to container creation
   - No changes to current behavior
```

## Flag Design

### New Flags Added to Spin Command

| Flag | Type | Required | Used With | Default | Environment Variable |
|------|------|----------|-----------|---------|---------------------|
| `--setup` | bool | No | Always optional | false | SPINNER_SETUP |
| `--base-image` | string | No | Requires --setup | ubuntu:22.04 | SPINNER_BASE_IMAGE |
| `--dockerfile` | string | No | Requires --setup | "" (empty) | SPINNER_DOCKERFILE |

### Flag Validation Matrix

| Flags Provided | Valid? | Behavior |
|----------------|--------|----------|
| `--setup` | ✅ | Build with default ubuntu:22.04 |
| `--setup --base-image X` | ✅ | Build with base image X |
| `--setup --dockerfile X` | ✅ | Build with custom Dockerfile X |
| `--setup --base-image X --dockerfile Y` | ❌ | Error: mutually exclusive |
| `--base-image X` (no --setup) | ❌ | Error: requires --setup |
| `--dockerfile X` (no --setup) | ❌ | Error: requires --setup |
| (no --setup) | ✅ | Normal spin, no image build |

## Error Handling Strategy

### Phase Separation
Errors are handled in two distinct phases:

**Setup Phase** (only when --setup is true):
- Prerequisite check failures → exit before build
- Flag validation failures → exit before build
- Dockerfile path validation failures → exit before build
- Image build failures → exit before container creation

**Spin Phase** (always executed):
- Container creation failures → standard error handling
- Container reuse/restart failures → standard error handling

### Error Messages
Error messages should clearly indicate which phase failed:
- Setup phase: "Error building image: ..."
- Spin phase: "Error creating container: ..." (existing messages)

## Backwards Compatibility

### No Breaking Changes
- All existing `spin` command usage remains unchanged
- New flags are optional
- Default behavior (no --setup) is identical to current implementation

### Opt-in Behavior
- Users must explicitly provide `--setup` flag to trigger image building
- Users who don't use --setup see zero behavior changes

## Implementation Considerations

### Code Location
All changes are isolated to `cmd/spin.go`:
- Add flag definitions in `init()` function
- Add conditional setup logic in `RunE` function before existing spin logic
- No changes to `internal/` packages needed

### Testing Strategy
- Integration tests validate flag combinations
- Integration tests validate setup → spin workflow
- Existing spin tests remain unchanged and continue to pass

### Build Performance
- When `--setup` is used, image rebuild always occurs (no caching)
- This is intentional and documented behavior
- Users who want to avoid rebuild simply omit `--setup` flag

## Alternative Designs Considered

### 1. Conditional Rebuild Based on Image Existence
**Rejected**: Adds complexity and ambiguity. Users can't predict when rebuild will occur. Always rebuilding when --setup is used is simpler and more predictable.

### 2. Separate --force-rebuild Flag
**Rejected**: --setup itself implies rebuild intent. Adding another flag adds unnecessary complexity.

### 3. Auto-setup When Image Missing
**Rejected**: Doesn't solve the "force rebuild" use case. Would require additional flag anyway. Also creates unexpected behavior when image name is mistyped.

### 4. Extract Shared Validation Logic
**Rejected**: Only 2-3 simple conditionals would be shared. Extraction overhead not justified for minimal duplication.

## Future Enhancements (Out of Scope)

- Smart caching: Only rebuild if base image or Dockerfile changed
- Build progress indicators with percentage
- Parallel build and container preparation
- Image build caching configuration options