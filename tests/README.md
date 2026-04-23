# Tests

This directory contains all test files for the project.

## Structure

```
tests/
├── fixtures/        # Test data and mocks
│   ├── users.json   # Sample user data
│   └── mocks/       # Mock implementations
├── unit/            # Unit tests (optional subdirectory)
├── integration/     # Integration tests (optional subdirectory)
└── e2e/             # End-to-end tests (optional subdirectory)
```

## Conventions

### File Naming
- Unit tests: `[name].test.ts` or `[name].spec.ts`
- Integration tests: `[name].integration.test.ts`
- E2E tests: `[name].e2e.test.ts`

### Test Structure
```typescript
describe('UserService', () => {
  describe('createUser', () => {
    it('creates a user with valid input', async () => {
      // Arrange
      const input = { name: 'Test', email: 'test@example.com' };

      // Act
      const result = await userService.createUser(input);

      // Assert
      expect(result.name).toBe('Test');
    });

    it('throws on duplicate email', async () => {
      // Test error cases
    });
  });
});
```

### Test Categories

#### Unit Tests
- Test individual functions/classes
- Mock all dependencies
- Fast execution

#### Integration Tests
- Test component interactions
- Use real dependencies where practical
- May use test database

#### End-to-End Tests
- Test complete user flows
- Run against full application
- Slower but comprehensive

## Running Tests

```bash
# All tests
npm test

# Watch mode
npm test -- --watch

# Specific file
npm test -- user-service.test.ts

# Coverage
npm test -- --coverage
```

## For AI Agents

When writing tests:
1. **Test behavior, not implementation** - Tests should survive refactoring
2. **One assertion focus** - Each test checks one thing
3. **Descriptive names** - Test name explains what's being tested
4. **Arrange-Act-Assert** - Clear test structure
5. **Use fixtures** - Store test data in fixtures/
6. **Cover edge cases** - Empty inputs, null values, errors

### Test Checklist
- [ ] Happy path covered
- [ ] Error cases covered
- [ ] Edge cases covered
- [ ] Test is isolated (no shared state)
- [ ] Test is deterministic (same result every time)
