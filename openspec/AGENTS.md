# OpenSpec + Beads Workflow

## System Roles

- **OpenSpec** = requirements, change intent, decisions, acceptance criteria
- **Beads** = executable task queue, dependencies, progress/state

Both are required for non-trivial work. Keep them linked.

## For AI Agents

1. Check OpenSpec artifacts first:
   - Prefer `openspec/changes/*` for active change work
   - Otherwise use `openspec/specs/*`
2. Start or pick a beads issue (`bd ready --json`), mark in progress.
3. Add spec/change linkage in bead notes (`Spec source: openspec/...`).
4. Implement in small slices with outside-in tests first.
5. At phase boundaries, produce checkpoint summary and wait for human approval.
6. On completion, update spec/change status and close bead.

## Outside-In Testing Order

1. Acceptance/E2E (or API contract) for user-visible outcome
2. Integration tests for boundaries and contracts
3. Unit tests for complex logic and edge cases

## Creating Specs / Changes

For significant changes, prefer OpenSpec-native change artifacts under `openspec/changes/<change-id>/`.
For simpler/legacy flow, use `openspec/specs/[feature].md`.

Minimum required in each artifact:
- explicit acceptance criteria
- open questions / assumptions
- risks and mitigations

## Integration with Beads

```bash
bd create "feat: [feature] (spec: openspec/...)" -p 1 --json
bd update <id> --status in_progress
bd update <id> --notes "Spec source: openspec/..."
```

If a change has multiple beads, record dependencies:

```bash
bd dep add <id> --blocked-by <other-id>
```
