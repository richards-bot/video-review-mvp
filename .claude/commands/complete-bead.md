# Complete Bead

Finish the current task, run quality gates, and close the beads issue.

## Instructions

1. Find the in-progress issue:
   ```bash
   bd list --json
   ```
   If none: suggest `/start-bead`.

2. Run quality gates for this repo:
   ```bash
   PATH=.tools/go/bin:$PATH go test ./...
   PATH=.tools/go/bin:$PATH go vet ./...
   PATH=.tools/go/bin:$PATH .tools/bin/staticcheck ./...
   npm --prefix viewer test -- --run
   ```
   If `ffmpeg`/`ffprobe` are unavailable, skip only tests that explicitly require them.

3. Confirm outside-in evidence is present:
   - Acceptance/E2E or contract tests for user-visible behavior
   - Integration/unit coverage for critical logic

4. Review uncommitted changes:
   ```bash
   git status
   git diff --stat
   ```

5. If checks pass — stage specific files (NOT `git add -A`), commit, close, verify tracker state, and push:
   ```bash
   git add [specific files]
   git commit -m "type(scope): description (bd-xxx)"
   bd close [id] --reason "Completed" --json
   bd list --json --all
   git push
   ```

6. If checks fail — show failures, fix or propose fixes, do NOT close the issue.

## Human-in-the-loop

Before closing, provide a concise checkpoint summary (what changed, test evidence, risks) and wait for user approval if this is a phase boundary.

## Forbidden

NEVER use `git add -A` (may stage secrets). Stage specific files by name.
