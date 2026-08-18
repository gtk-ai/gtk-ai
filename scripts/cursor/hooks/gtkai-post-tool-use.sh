#!/bin/sh
# gtkai PostToolUse hook for Cursor (matcher MCP:*).

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
exec "$GTKAI" hook-post --agent=cursor
