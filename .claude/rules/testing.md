# Testing Rules

Code without tests is incomplete.

## Testing Strategy (default): Outside-In

Start from user-visible behavior, then move inward only as needed.

1. **E2E / acceptance test** for the user journey (or API contract test if no UI)
2. **Integration test** for module/service boundaries
3. **Unit tests** for critical logic and edge cases

Prefer testing contracts and outcomes over internal implementation details.

## Coverage Requirements

- General code: 80% minimum
- Business logic: 90% minimum
- Financial/payment and auth/security: 100%

## TDD Workflow (mandatory for new features)

1. **RED** — Write failing test first (outside-in level first)
2. **GREEN** — Write minimal code to pass
3. **REFACTOR** — Improve while keeping green
4. Repeat

Tests MUST fail before implementing. This verifies the test works.

## What to Test

- **Acceptance/E2E (thin set):** Critical user flows and happy-path outcomes
- **Integration:** Success (200/201), validation errors (400), auth errors (401/403), not found (404), persistence and boundary behavior
- **Unit:** Edge cases, error cases, boundary values, complex pure logic

## Rules

- Test behavior, not implementation
- Keep acceptance tests focused on outcomes, not brittle selectors/internal details
- Isolate tests — no shared mutable state
- No flaky tests (timing-dependent)
- Don't mock everything — prefer real dependencies at boundaries when practical
- Don't test private methods directly
- Run full suite before committing: `npm test`
- Run lint + typecheck before completion: `npm run lint` and `npm run typecheck` (if available)
