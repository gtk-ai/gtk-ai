# CLAUDE.md

## Architecture

Two independent phases:

1. **Go binary** (`gtkai`): command proxy and token reduction. PreToolUse rewrites `git status` to `gtkai git status`; the binary runs the real command, injects flags, and filters output.
2. **Agent integrations**: Claude Code plugin, Cursor hooks, Codex hooks, and the OpenCode plugin. Each one registers hooks and invokes the binary.

Never collapse both phases into one. The integrations depend on the binary. The binary does not write agent config files; `install.sh` does.

Target flow and remaining work: `ROADMAP.md`. `0.10.0` adds Cursor, Codex, and OpenCode hook integrations. `0.9.0` completed §3 runners (npm, docker).

## Versions

When changing the version, update every file that exposes it:

- `cmd/gtkai/main.go`
- `.claude-plugin/plugin.json`
- `.claude-plugin/marketplace.json`
- `plugin/.claude-plugin/plugin.json`
- `modules/mcpscan/mcpscan.go`
- `README.md`

To check no old references remain:

```bash
grep -rn "X.Y.Z" . --include="*.go" --include="*.json" --include="*.md"
```

The git tag must match the version in code. The release CI workflow enforces this.

## Language

Filtering is heuristic: truncation, extension grouping, comment stripping, line limits. It is not semantic compression or intelligent deduplication.

Use: heuristic pruning, rule-based filtering, deterministic truncation.  
Avoid: intelligent compression, semantic optimization, smart deduplication.

## Before committing

- Tests pass (`go test ./...`).
- If the change affects the hook or filtering, test with a real payload before committing.
- If the install flow changes, validate the full user path, not just compilation.
- If the version is bumped, verify all version-bearing files are updated.
