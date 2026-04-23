# Coding Style Rules

## Size Limits

- **Files:** >200 lines consider splitting, >400 should split, >600 must split
- **Functions:** >30 lines consider extracting, >50 should extract, >80 must extract
- **Nesting:** Maximum 4 levels — flatten with early returns

## Patterns

- Prefer immutable operations (`[...arr, item]` over `arr.push(item)`)
- Async/await over `.then()` chains
- Named constants over magic numbers
- Proper logging (logger.info/error) over console.log
- Explicit error handling — no silent catches, specific error types

## Imports Order

1. External packages
2. Internal modules (absolute paths)
3. Relative imports
4. Types (if separate)

## Comments

- Explain "why", not "what"
- No commented-out code
- No obvious comments (`// increment counter`)
