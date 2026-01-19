# System Design & Architecture

## Code Organization

### Directory Structure

```
src/
├── commands/          # UI components for CLI commands
│   ├── Setup.tsx     # Setup command component
│   └── Spin.tsx      # Spin command component
├── utils/            # Business logic and utilities
│   ├── docker.ts     # Docker-related utilities
│   └── dockerfile.ts # Dockerfile generation
└── App.tsx           # Main CLI router
```

### Separation of Concerns

- **Commands** (`src/commands/`): React components for UI and state management only
    - Handle user interaction and display
    - Orchestrate calls to utility functions
    - No business logic or direct system calls

- **Utils** (`src/utils/`): Pure business logic and system interactions
    - No UI code or React dependencies
    - Testable in isolation
    - Clear input/output contracts via TypeScript interfaces

## Function Design

- Functions should be small and focused (ideally < 30 lines)
- Use descriptive names that clearly indicate purpose
- Document complex logic with JSDoc comments
- Return structured results (interfaces) rather than throwing errors when possible

## Error Handling

- Utility functions return result objects with success/error fields
- UI components handle error display
- Preserve error context through the call stack
