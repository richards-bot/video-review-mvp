# Agent Behavior Rules

These rules reduce common LLM coding mistakes. They bias toward caution over speed.

## Think Before Coding

- State assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them — don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## Simplicity First

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If 200 lines could be 50, rewrite it.

## Surgical Changes

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it — don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

Every changed line should trace directly to the user's request.

## Goal-Driven Execution

Transform tasks into verifiable goals:
- "Add validation" -> "Write tests for invalid inputs, then make them pass"
- "Fix the bug" -> "Write a test that reproduces it, then make it pass"

For multi-step tasks, state a brief plan:
1. [Step] -> verify: [check]
2. [Step] -> verify: [check]

## Beads Workflow (MANDATORY)

Every code change requires a tracked beads issue.

### Session start
1. `bd list --json` — check for existing in-progress issues
2. `bd ready --json` — find unblocked issues to work on
3. Pick or create: `bd create "description" -p <priority> --json`
4. Activate: `bd update <id> --status in_progress`

### During work
- **Commit AND push after every meaningful change** — a completed function, a passing test, a config update. Small, frequent commits beat one large commit at the end.
- Commit with issue ID: `git commit -m "type(scope): description (bd-xxx)"` then immediately `git push`
- `bd sync` after issue changes
- Use `bd update <id> --notes "context"` to record decisions
- If you've edited 3+ files without committing and pushing, stop and do it now.

### Session end
1. `bd close <id> --reason "Completed" --json` (if done)
2. `bd sync`
3. `git push` (work is NOT done until pushed)

### Forbidden
- NEVER use `bd edit` (interactive editor — agents cannot use it)
- Use `bd update <id> --title/--description/--notes/--design/--acceptance` instead

## These rules are working if:

- Fewer unnecessary changes in diffs
- Fewer rewrites due to overcomplication
- Clarifying questions come before implementation, not after mistakes
- Every session has a tracked beads issue
