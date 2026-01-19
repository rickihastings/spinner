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

## TypeScript Standards

### Type Safety

- Use explicit types for function parameters and return values
- Avoid `any` - use `unknown` for truly unknown types
- Define interfaces for complex data structures
- Export types that are used across modules

### Interface Conventions

```typescript
// Configuration interfaces (input)
export interface SpinConfig {
    image: string;
    repo: string;
    prompt?: string;  // optional fields marked with ?
    branch?: string;
}

// Result interfaces (output)
export interface ValidationResult {
    valid: boolean;
    error?: string;
    warnings: string[];
    // ... other fields
}
```

## Code Documentation

- **Prefer self-documenting code**: Write clear, descriptive variable and function names that make the code's intent
  obvious
- **Avoid single-line comments**: Rarely use single-line comments. If code needs explanation, consider refactoring for
  clarity first
- **Use JSDoc for complex logic**: When documentation is necessary, use JSDoc block comments to explain the "why" behind
  complex logic, not the "what"
- **Document interfaces and types**: Always document non-obvious types, interfaces, and their fields
- **Don't be too verbose**: Keep documentation concise and focused. Avoid obvious or redundant explanations

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
