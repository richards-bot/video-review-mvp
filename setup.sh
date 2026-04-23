#!/usr/bin/env bash
set -euo pipefail

# Project Template Setup
# Usage: ./setup.sh <project-name>

PROJECT_NAME="${1:-}"

if [ -z "$PROJECT_NAME" ]; then
  echo "Usage: ./setup.sh <project-name>"
  echo ""
  echo "Example:"
  echo "  ./setup.sh my-app    # Creates ~/my-app"
  exit 1
fi

DEST="$HOME/$PROJECT_NAME"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

if [ -d "$DEST" ]; then
  echo "Error: $DEST already exists"
  exit 1
fi

echo "Creating project: $PROJECT_NAME"

# Copy template
cp -r "$SCRIPT_DIR" "$DEST"

# Clean up template artifacts
cd "$DEST"
rm -rf .git .beads setup.sh

# Initialize git
git init
git add -A
git commit -m "chore: initialize from project template"

# Initialize beads
if command -v bd &>/dev/null; then
  bd init
  bd hooks install
  echo ""
  echo "Beads initialized with git hooks."
else
  echo ""
  echo "Warning: 'bd' not found. Install beads and run:"
  echo "  cd $DEST && bd init && bd hooks install"
fi

if command -v openspec &>/dev/null; then
  echo ""
  echo "OpenSpec detected. Optional next step in your new project:"
  echo "  cd $DEST && openspec update"
fi

echo ""
echo "Project '$PROJECT_NAME' is ready at $DEST"
echo ""
echo "Next steps:"
echo "  cd $DEST"
echo "  claude"
echo "  > /brain-dump"
