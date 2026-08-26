#!/bin/sh
# gtkai PostToolUse hook for Claude Code.

# Locate gtkai: check PATH first, then common install locations.
GTKAI=$(command -v gtkai 2>/dev/null)
if [ -z "$GTKAI" ]; then
  for candidate in "$HOME/.local/bin/gtkai" "/usr/local/bin/gtkai" "/opt/homebrew/bin/gtkai"; do
    if [ -x "$candidate" ]; then
      GTKAI="$candidate"
      break
    fi
  done
fi

[ -z "$GTKAI" ] && exit 0

# Only activate in projects that have run `gtkai init`.
_root=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
[ -f "$_root/.gtk-ai" ] || exit 0

exec "$GTKAI" hook-post --agent=claudecode
