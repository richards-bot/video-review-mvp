# .claude Directory

Claude Code configuration for this project.

## Structure

```
.claude/
├── commands/           # Slash commands (loaded on invocation)
│   ├── start-bead.md
│   ├── complete-bead.md
│   ├── checkpoint.md
│   ├── status.md
│   ├── plan.md
│   ├── tdd.md
│   ├── code-review.md
│   └── brain-dump.md
├── rules/              # Auto-loaded every interaction
│   ├── agent-behavior.md   # Think-before-coding, surgical changes, beads workflow
│   ├── coding-style.md     # Size limits, patterns, naming
│   ├── security.md         # Security checklist
│   └── testing.md          # TDD, coverage requirements
├── hooks.json          # Automated hooks (beads enforcement, console.log checks)
└── README.md
```

## What's Auto-Loaded

- `CLAUDE.md` (project root) — loaded every interaction
- `.claude/rules/*` — all rules loaded every interaction
- `.claude/commands/*` — loaded only when invoked via `/command-name`
- `.claude/hooks.json` — hooks run on matching tool calls

Keep rules terse to minimize token usage.
