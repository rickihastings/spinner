# Coding Standards

## SOLID Principles

This project follows SOLID principles for maintainable, testable, and scalable code:

### Single Responsibility Principle (SRP)

- Each function should have one clear, well-defined purpose
- Each module should have one reason to change
- Example: `validatePrerequisites()` only validates, `executeDockerRun()` only executes

### Open/Closed Principle (OCP)

- Code should be open for extension, closed for modification
- Use interfaces and abstractions to allow behavior changes without modifying existing code
- Example: `SpinConfig` interface allows new fields without changing function signatures

### Liskov Substitution Principle (LSP)

- Subtypes must be substitutable for their base types
- Return types should be consistent and predictable
- Example: All validation functions return a result object with consistent structure

### Interface Segregation Principle (ISP)

- Keep interfaces focused and minimal
- Don't force clients to depend on methods they don't use
- Example: `ValidationResult`, `ContainerResult` - each interface serves a specific purpose

### Dependency Inversion Principle (DIP)

- Depend on abstractions, not concrete implementations
- High-level modules should not depend on low-level modules
- Example: `Spin.tsx` depends on utility function interfaces, not their implementation details

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

- Place command implementations in `cmd/` package
- Place business logic in `internal/` packages
- Keep internal packages focused (e.g., `internal/docker`, `internal/prerequisites`)
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
