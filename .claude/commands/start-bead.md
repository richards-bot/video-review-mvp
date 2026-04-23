# Start Bead

Start work with minimal manual intervention while preserving traceability.

## Instructions

1. Check for in-progress issues:
   ```bash
   bd list --json
   ```

2. **If exactly one issue is `in_progress`**: continue it automatically.

3. **If none are `in_progress`**:
   - Check ready queue:
     ```bash
     bd ready --json
     ```
   - If ready issues exist: auto-pick the highest-priority unblocked issue and mark in progress.
     ```bash
     bd update <id> --status in_progress
     ```
   - If no ready issues: create one from user intent and mark in progress.
     ```bash
     bd create "[task summary]" -p [priority] --json
     bd update <id> --status in_progress
     ```

4. Link the issue to spec context:
   - Prefer OpenSpec change artifacts (`openspec/changes/*`) when present.
   - Otherwise use `openspec/specs/*`.
   - Add notes to issue with the source file path(s):
     ```bash
     bd update <id> --notes "Spec source: openspec/..."
     ```

5. Confirm active issue, objective, and **first checkpoint boundary** (what must be shown before continuing).

## Human-in-the-loop rule

Proceed autonomously within the current slice, but pause at checkpoints for approval before the next phase.

## Forbidden

NEVER use `bd edit`. Use `bd update <id> --flags` for modifications.
