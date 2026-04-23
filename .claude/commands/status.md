# Status

Show implementation status with spec ↔ task traceability.

## Instructions

Run these and summarize:

```bash
bd list --json
bd ready --json
git status --short
git branch --show-current
git log --oneline -5
```

Also report:
1. Active bead(s) and whether they include spec references in notes.
2. Next checkpoint gate and required evidence.
3. Any blockers or open questions from current spec/change artifacts.

Display: active issues, ready tasks, branch, uncommitted changes, recent commits, and checkpoint readiness.
