# Test Fixtures

This directory contains test data, mocks, and utilities for testing.

## Structure

```
fixtures/
├── data/            # Static test data
│   ├── users.json
│   └── products.json
├── mocks/           # Mock implementations
│   ├── email-service.mock.ts
│   └── database.mock.ts
├── factories/       # Test data factories
│   └── user.factory.ts
└── helpers/         # Test utilities
    └── setup.ts
```

## Usage

### Static Test Data

```typescript
// tests/fixtures/data/users.json
[
  {
    "id": "user-1",
    "name": "Test User",
    "email": "test@example.com"
  }
]

// In tests
import users from '../fixtures/data/users.json';
```

### Factories

```typescript
// tests/fixtures/factories/user.factory.ts
export function createUser(overrides = {}) {
  return {
    id: `user-${Date.now()}`,
    name: 'Test User',
    email: `test-${Date.now()}@example.com`,
    createdAt: new Date(),
    ...overrides
  };
}

// In tests
import { createUser } from '../fixtures/factories/user.factory';

const user = createUser({ name: 'Custom Name' });
```

### Mocks

```typescript
// tests/fixtures/mocks/email-service.mock.ts
export const mockEmailService = {
  send: jest.fn().mockResolvedValue({ success: true }),
  verify: jest.fn().mockResolvedValue(true),
};

// In tests
import { mockEmailService } from '../fixtures/mocks/email-service.mock';

beforeEach(() => {
  jest.clearAllMocks();
});
```

## Best Practices

### Do
- Keep fixture data realistic but minimal
- Use factories for dynamic data
- Clear mocks between tests
- Document non-obvious fixture data

### Don't
- Use production data in fixtures
- Create overly complex test data
- Share state between tests
- Include sensitive information

## For AI Agents

When creating fixtures:
1. **Keep data minimal** - Only what's needed for tests
2. **Make data realistic** - Reflects actual use cases
3. **Use factories** - For data that varies between tests
4. **Document edge cases** - Why certain data exists
5. **No secrets** - Never include real credentials
