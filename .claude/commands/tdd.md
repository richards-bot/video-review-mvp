# TDD (Outside-In)

Enforce outside-in TDD: prove user-visible behavior first, then implement inward.

## Instructions

Follow this cycle:

1. **DEFINE OUTCOME** — State user-visible behavior and acceptance criteria.
2. **RED (ACCEPTANCE)** — Add a failing acceptance/E2E (or API contract) test for the behavior.
3. **GREEN (THIN SLICE)** — Implement the thinnest end-to-end slice to pass it.
4. **RED (INTEGRATION/UNIT)** — Add failing integration/unit tests for edge/error logic discovered.
5. **GREEN** — Implement minimal internals to pass.
6. **REFACTOR** — Improve design while keeping all tests green.
7. **VERIFY** — Run full suite + coverage.

## Required Commands

```bash
npm test
npm test -- --coverage
```

If available in the project, also run:

```bash
npm run lint
npm run typecheck
```

## Rules

- Tests MUST fail before implementing corresponding code.
- Start from behavior first, not helper internals.
- Keep acceptance tests small and focused on critical paths.
- Do not skip the final full-suite run.
