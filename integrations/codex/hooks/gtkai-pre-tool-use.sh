#!/bin/sh
# gtkai PreToolUse hook for Codex CLI (matcher Bash and shell aliases).

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
exec "$GTKAI" hook-pre --agent=codex
