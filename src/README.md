# Source Code

This directory contains the application source code.

## Structure

Organize code by feature or layer depending on project size:

### Small Projects (Feature-based)
```
src/
├── auth/
│   ├── auth.service.ts
│   ├── auth.controller.ts
│   └── auth.test.ts
├── users/
│   ├── user.service.ts
│   ├── user.controller.ts
│   └── user.test.ts
└── shared/
    ├── utils/
    └── types/
```

### Larger Projects (Layer-based)
```
src/
├── controllers/     # Request handlers
├── services/        # Business logic
├── repositories/    # Data access
├── models/          # Data structures
├── utils/           # Helpers
├── types/           # Type definitions
├── config/          # Configuration
└── middleware/      # Request middleware
```

## Conventions

### File Naming
- Use kebab-case: `user-service.ts`
- Test files: `user-service.test.ts`
- Type files: `user.types.ts`

### Code Organization
- One primary export per file
- Keep files under 300 lines
- Extract shared logic to utils/

### Imports
```typescript
// External imports first
import express from 'express';

// Internal imports second
import { UserService } from './services/user-service';

// Types last
import type { User } from './types/user';
```

## Getting Started

1. Review existing code patterns
2. Check CLAUDE.md for project conventions
3. Follow test-driven development
4. Keep changes atomic and focused

## For AI Agents

When adding code to this directory:
- Match existing patterns
- Add corresponding tests
- Update relevant documentation
- Keep functions small and focused
