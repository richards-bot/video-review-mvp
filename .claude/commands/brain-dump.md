# Brain Dump

Turn an unstructured project brain dump into an execution-ready brief with minimal back-and-forth.

## Instructions

### Step 0: Ensure tracking/spec tooling exists

If `.beads/` is missing:
```bash
bd init
bd hooks install
```

If OpenSpec is installed, ensure instructions are current:
```bash
openspec update
```

### Step 1: Parse the dump into a structured brief

Populate:
- `docs/SPEC.md` — goals, scope, non-goals, success criteria
- `docs/DECISIONS.md` — key technical decisions + rationale
- `CLAUDE.md` — stack-specific commands/structure
- `README.md` — project quick start and purpose

Also include these required sections in `docs/SPEC.md`:
- Assumptions
- Risks and mitigations
- Open questions that block coding
- Checkpoint plan (where human approval is required)

### Step 2: Create OpenSpec artifacts (preferred)

Prefer OpenSpec-native change artifacts under `openspec/changes/<change-id>/` (proposal/tasks/spec deltas).
If the project is using the simpler legacy mode, create/update `openspec/specs/*.md`.

Every feature/change must have explicit acceptance criteria.

### Step 3: Create Beads execution queue

Create beads issues tied to spec/change artifacts:
```bash
bd create "feat: [feature] (spec: openspec/...)" -p N --json
```

Add dependencies where needed:
```bash
bd dep add <id> --blocked-by <other-id>
```

Store traceability in notes:
```bash
bd update <id> --notes "Spec source: openspec/..."
```

### Step 4: Build checkpointed execution plan

Output phases with:
- objective
- outside-in tests to write first
- completion evidence
- checkpoint gate (requires human approval before next phase)

### Step 5: Summarize and handoff

Summarize generated artifacts, unresolved blockers, and the first recommended bead.
Suggest:
```bash
bd ready --json
```
