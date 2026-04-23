# CLAUDE.md

## Project Overview

**What:** Secure Video Review MVP — CLI uploader + static browser viewer for sharing encrypted video
**Stack:** Go 1.22, vanilla JS/HTML/CSS, AWS S3, Terraform
**Status:** Active development

## Commands

```bash
# Go
PATH=.tools/go/bin:$PATH go build ./...   # Build
PATH=.tools/go/bin:$PATH go test ./...    # Tests
PATH=.tools/go/bin:$PATH go vet ./...     # Vet

# Viewer (no build step — open directly in browser or serve statically)
cd viewer && npm install && npm test        # Run viewer JS unit tests (Vitest)
# scripts/
./scripts/build.sh                         # Cross-platform binary build
./scripts/deploy-viewer.sh                 # Deploy viewer to S3
```

## Structure

```
cmd/video-review/     Go CLI
internal/
  crypto/             AES-256-GCM helpers
  chunker/            Streaming chunk reader
  ffmpeg/             ffmpeg/ffprobe wrappers
  manifest/           Manifest schema
  s3upload/           S3 uploader
  share/              URL construction
viewer/               Static viewer
infrastructure/       Terraform
scripts/              Build + deploy
```

## Naming

- **Files:** snake_case for Go (per convention), kebab-case for scripts
- **Go types:** PascalCase
- **Go functions/vars:** camelCase
- **Constants:** SCREAMING_SNAKE

## Operating Model

- **OpenSpec** defines what to build (requirements, acceptance criteria, decisions)
- **Beads** tracks how work executes (queue, dependencies, progress)
- **Human checkpoint approvals** gate transitions between major phases

## Beads — MANDATORY (enforced by hooks)

**STOP. Do not write or edit code without an active beads issue.**

### Before ANY code change

1. Run `bd list` to see current issues
2. If no in-progress issue exists:
   - Pick one: `bd ready` (shows unblocked issues)
   - Or create one: `bd create "description" -p <priority> --json`
3. Mark it active: `bd update <id> --status in_progress`

### During work

- **Commit after every meaningful change** — don't batch all changes into one commit at the end
- Commit with issue ID: `git commit -m "type(scope): description (bd-xxx)"`
- Sync periodically: `bd sync`

### When done

1. Close the issue: `bd close <id> --reason "Completed" --json`
2. Sync: `bd sync`
3. Run tests before finishing

### Critical rules

- NEVER use `bd edit` (interactive — agents cannot use it)
- Use `bd update <id> --title/--description/--notes` instead
- Always use `--json` flag when creating/querying for structured output
- Run `bd sync` after making issue changes
- Include issue ID `(bd-xxx)` in commit messages

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`

## Execution Loop (default)

1. Read active OpenSpec artifact (`openspec/changes/*` preferred, else `openspec/specs/*`)
2. Start/pick a bead and mark `in_progress`
3. Implement one thin vertical slice with outside-in tests
4. Run verification (`go test ./...`, `go vet ./...`)
5. Checkpoint summary for human approval at phase boundary
6. Commit with `(bd-xxx)`, sync beads, continue or close

## Git

Commit format: `type(scope): description (bd-xxx)`
Branch naming: `feature/*`, `fix/*` from `main`

## Do Not Modify

- `go.sum` manually (use `go mod tidy`)
- `.env` files
- `infrastructure/` without explicit request

## Always

- Follow outside-in test order: acceptance/E2E -> integration -> unit
- Run tests before marking work complete
- Update `docs/DECISIONS.md` for architectural changes
- Keep bead notes linked to spec paths (`Spec source: openspec/...`)
- Run `bd sync` before ending a session

## Environment

- Go toolchain: `.tools/go/bin/go` — always add to PATH: `export PATH=.tools/go/bin:$PATH`
- ffmpeg/ffprobe: NOT installed in this environment — integration tests requiring them must be skipped
- AWS: use standard credential chain (env vars, ~/.aws/credentials, IAM role)

## Context Files

- `SPEC.md` - Full technical specification
- `docs/DECISIONS.md` - Architecture decisions
