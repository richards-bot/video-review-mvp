# Checkpoint

Create a safety save point to preserve current progress.

## Instructions

1. Show current changes:
   ```bash
   git status
   git diff --stat
   ```

2. Ask for a brief description of current state.

3. Stage specific changed files (NOT `git add -A`), commit, and sync:
   ```bash
   git add [specific files]
   git commit -m "checkpoint: [description] (bd-xxx)"
   bd sync
   ```

4. Confirm with commit hash and files included.

## When to checkpoint

- After getting something working
- Before risky changes
- At logical stopping points
- Every 30-60 minutes during long work
