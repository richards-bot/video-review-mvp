# Getting Started

## Prerequisites

```bash
# Claude Code (native installer)
curl -fsSL https://claude.ai/install.sh | bash

# Beads (task tracking)
npm install -g @beads/bd
# or: brew install beads

# OpenSpec
npm install -g @fission-ai/openspec@latest
```

## Setup

```bash
# Clone the repo
git clone <repo-url>
cd video-review-mvp

# Go toolchain is bundled — add to PATH
export PATH=.tools/go/bin:$PATH
go build ./...

# Viewer — no build step needed
cd viewer && npm install && npm test && cd ..

# Configure AWS credentials (standard credential chain)
cp .env.example .env   # edit with your S3 bucket and region

# Infrastructure (one-time, needs Terraform)
cd infrastructure/terraform && terraform init && terraform apply
```

## Daily Workflow

```
1. CHECK      bd list --json / bd ready --json
2. START      bd create "feat: description" -p 1 --json
              bd update <id> --status in_progress
3. WORK       claude  (AI-assisted development)
4. CHECKPOINT git add [files] && git commit -m "checkpoint: msg (bd-xxx)"
5. TEST       go test ./... (Go) or cd viewer && npm test (JS)
6. FINISH     bd close <id> --reason "Completed" --json && git push
```

## Slash Commands (in Claude Code)

| Command | Purpose |
|---------|---------|
| `/start-bead` | Create/pick a beads issue and start work |
| `/complete-bead` | Run tests, commit, close issue, sync |
| `/checkpoint` | Stage, commit, sync current progress |
| `/status` | Show issues, git state, ready tasks |
| `/plan` | Plan before implementing (waits for approval) |
| `/tdd` | Test-driven development cycle |
| `/code-review` | Security and quality review |
| `/brain-dump` | Turn unstructured ideas into docs |

## Beads Commands Reference

| Command | Purpose |
|---------|---------|
| `bd create "msg" -p N --json` | Create issue (priority 1-4) |
| `bd update <id> --status in_progress` | Mark issue active |
| `bd close <id> --reason "msg" --json` | Close issue |
| `bd list --json` | List all issues |
| `bd ready --json` | Show unblocked issues |
| `bd update <id> --notes "msg"` | Add notes to issue |
| `bd show <id> --json` | Show issue details |

**Never use `bd edit`** — it opens an interactive editor. Use `bd update` with flags.

## Token Efficiency Tips

- Reference files by path: `src/services/auth.ts:45-60`
- Small, focused requests — one thing at a time
- Break large tasks into steps
- Use `@file` syntax in Claude Code to reference files

## Documentation

- `CLAUDE.md` — Agent instructions (auto-loaded)
- `docs/SPEC.md` — Project specification
- `docs/DECISIONS.md` — Architecture decisions
- `openspec/specs/` — Feature specifications
