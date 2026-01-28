# Tasks: Add Setup Flag to Spin Command

## Implementation Tasks

### 1. Add setup-related flags to spin command
- [ ] Add `--setup` boolean flag to `cmd/spin.go`
- [ ] Add `--base-image` string flag to `cmd/spin.go` (optional, used with --setup)
- [ ] Add `--dockerfile` string flag to `cmd/spin.go` (optional, used with --setup)
- [ ] Bind new flags to Viper for environment variable support
- [ ] **Validation**: Flags are registered and accessible via `spinner spin --help`

### 2. Add flag validation logic
- [ ] Validate that `--base-image` and `--dockerfile` are mutually exclusive
- [ ] Validate that `--base-image` and `--dockerfile` are only used when `--setup` is present
- [ ] Error with clear message if invalid flag combinations are provided
- [ ] **Validation**: Command rejects invalid flag combinations with helpful error messages

### 3. Add image build orchestration to spin command
- [ ] Check if `--setup` flag is provided
- [ ] If true, call `prerequisites.CheckPrerequisites()` before building
- [ ] If true, construct `docker.BuildConfig` from flags (reusing setup command logic)
- [ ] If true, call `docker.BuildImage()` to build/rebuild the image
- [ ] Handle build errors and exit before attempting to spin
- [ ] **Validation**: Image is built successfully when --setup flag is used

### 4. Handle image name resolution
- [ ] When `--setup` is provided, use `--image` value as the setup name
- [ ] Ensure the built image is tagged as `spinner:<image-value>`
- [ ] Existing spin logic should use this image
- [ ] **Validation**: Built image matches expected tag format

### 5. Update command documentation and examples
- [ ] Update `spinCmd.Long` help text to include --setup flag and related flags
- [ ] Add examples showing `--setup` usage with different configurations
- [ ] Document that --setup always rebuilds the image
- [ ] **Validation**: `spinner spin --help` shows updated documentation

### 6. Add integration tests
- [ ] Test: `spin --setup --image test --base-image ubuntu:22.04 --repo <url>` builds and spins
- [ ] Test: `spin --setup --image test --dockerfile ./custom.Dockerfile --repo <url>` uses custom Dockerfile
- [ ] Test: `spin --setup --base-image ubuntu:22.04 --dockerfile ./custom.Dockerfile` returns error (mutually exclusive)
- [ ] Test: `spin --base-image ubuntu:22.04 --repo <url>` returns error (--base-image requires --setup)
- [ ] Test: `spin --setup --image test --repo <url>` with existing image rebuilds it
- [ ] **Validation**: All test scenarios pass

## Task Dependencies
- Task 1 must complete before Task 2, 3, 4
- Task 5 can be done in parallel with Task 2, 3, 4
- Task 6 depends on Tasks 1-5 being complete

## Estimated Deliverables
1. Modified `cmd/spin.go` with new flags and orchestration logic
2. Updated help documentation
3. Integration tests demonstrating the new functionality