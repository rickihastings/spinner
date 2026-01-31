# Coding Standards

## Communication Guidelines

- **Be honest about uncertainty**: Say "I don't know" when uncertain rather than providing incorrect information
- **Probe for requirements**: Ask clarifying questions about requirements, constraints, and expected behavior before implementing
- **Avoid assumptions**: When requirements are ambiguous, explicitly state assumptions and ask for confirmation

## Design Principles

Follow SOLID principles: Single Responsibility, Open/Closed, Liskov Substitution, Interface Segregation, and Dependency Inversion. Keep functions small and focused with one clear purpose. Depend on abstractions, not implementations.

## Code Integration Policy

**Principle: Integrate or don't implement. No dead code, no "just in case" code.**

- ✅ **Integrate immediately**: If code serves a current need, integrate and use it right away
- ✅ **Skip implementation**: If there's no current use case, don't write it at all
- ❌ **No "just in case" code**: Don't create unused functions for hypothetical future needs
- ❌ **No dead code**: Remove unused code rather than leaving it around
- 📝 **Document decisions**: When removing code or deciding not to implement something, document why

**Rationale**: Dead code creates maintenance burden, confusion, and false assumptions about what's actually used. If we need functionality later, we can implement it then with full context of the actual requirements.

**Example**:
```go
// ❌ Bad: Creating unused helper "just in case"
func CheckAllPrerequisites() error {
    // Not called anywhere, keeping for potential future use
}

// ✅ Good: Only implement what's needed now
func CheckPrerequisites() error {
    // Used in setup command
}

func CheckEnvironmentVariables() error {
    // Used in spin command validation
}
```

## Go Standards

### Type Safety and Conventions

- Use explicit types for function parameters and return values
- Define structs for complex data structures
- Export types (capitalize first letter) that are used across packages
- Use Go's error handling patterns (return error as last value)
- Follow Go naming conventions (MixedCaps, not snake_case)

### Struct Conventions

```go
// Configuration structs (input)
type SpinConfig struct {
    Image         string
    Repo          string
    Prompt        string // optional fields use pointer (*string) or check for empty string
    Branch        string
    MaxIterations int
    Recreate      bool
}

// Result structs (output)
type ValidationResult struct {
    Valid    bool
    Error    string
    Warnings []string
}
```

### Error Handling

- Return errors as the last return value
- Use `fmt.Errorf` for formatted error messages
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Check errors immediately after function calls

```go
func DoSomething() error {
    result, err := SomeOperation()
    if err != nil {
        return fmt.Errorf("failed to do something: %w", err)
    }
    // ... use result
    return nil
}
```

### Code Organization

- Commands: `cmd/` package - CLI command implementations
- Business logic: `internal/` packages - Keep focused (e.g., `internal/docker`, `internal/prerequisites`)
- Use Go modules for dependency management

## Code Documentation

- **Prefer self-documenting code**: Write clear, descriptive variable and function names that make the code's intent obvious
- **Avoid excessive comments**: Rarely use single-line comments. If code needs explanation, consider refactoring for clarity first
- **Use godoc comments for exported items**: When documentation is necessary, use godoc-style comments (complete sentences starting with the item name) for exported functions, types, and packages
- **Document structs and interfaces**: Always document non-obvious types, structs, and their fields
- **Don't be too verbose**: Keep documentation concise and focused. Avoid obvious or redundant explanations

Example:
```go
// GenerateContainerName creates a deterministic container name from the image, repo, and branch.
// The name is sanitized to meet Docker naming requirements (lowercase alphanumeric and hyphens).
func GenerateContainerName(image, repo, branch string) string {
    // implementation
}
```

## Documentation Standards

### Specifications

- **Keep specs in sync with code**: When implementing changes, always update related specification documents (in
  `openspec/changes/`) to reflect the actual implementation
- **Update design documents**: If implementation deviates from the design, update the design document with rationale
- **Document breaking changes**: Clearly document any breaking changes in both specs and commit messages

## Git Commit Standards

- Use conventional commit format: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`
- Write clear, descriptive commit messages
- Reference issues/PRs when applicable
- Keep commits atomic and focused
