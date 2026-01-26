# Change: Migrate CLI from Node.js/TypeScript to Golang

## Why
The current CLI is built with Node.js, TypeScript, React (Ink), and a custom argument parser. This introduces runtime dependencies and complexity. Migrating to Golang will provide a self-contained binary, better performance, simpler deployment, and leverage Go's excellent CLI tooling (Cobra and Viper). OpenSpec support will be maintained via package.json.

## What Changes
- Replace TypeScript/React/Ink CLI with Go implementation using Cobra for commands and Viper for environment variable configuration
- Maintain all existing functionality: `setup` and `spin` commands with identical flags and behavior
  - Including deterministic container naming based on image, repo, and branch
  - Container reuse logic (reuse running, restart stopped containers)
  - --recreate flag for forcing fresh container creation
- Keep package.json for OpenSpec dependency and tooling
- Preserve all existing integration tests without modification
- Replace yarn build/test scripts with Go equivalents (go build, go test)
- Output a standalone `spinner` binary instead of requiring Node.js runtime

## Impact
- Affected specs: `cli-setup`, `cli-spin`
- Affected code: Complete rewrite from `src/` TypeScript to Go files
- **BREAKING**: Build process changes from `yarn build` to `go build`
- **BREAKING**: Development dependencies shift from npm packages to Go modules
- Tests remain unchanged but execution shifts to testing Go binary
- Users benefit from simpler installation (single binary vs node_modules)
