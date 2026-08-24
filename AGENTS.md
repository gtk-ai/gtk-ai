# AGENTS.md

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
- `integrations/claude/.claude-plugin/plugin.json`
- `plugins/mcpscan/mcpscan.go`
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

## Plugin infrastructure

External plugins are binaries that implement the `registry.Module` interface via a JSON stdin/stdout protocol. The infrastructure lives in `internal/`:

| Package | Role |
|---|---|
| `pluginregistry` | SQLite DB (`~/.gtk-ai/plugins.db`) — tracks installed plugins |
| `pluginsubprocess` | Adapts an external binary to `registry.Module` via JSON protocol |
| `plugininstall` | Downloads, validates, and installs plugin binaries |
| `pluginmanifest` | Parses and validates `gtkai.json` plugin manifests |

Built-in plugins in `plugins/` are compiled into the binary and registered via `init()`. External plugins from the marketplace are subprocess-based (JSON protocol). Both implement `registry.Module` — the proxy treats them identically.

## Post-merge <!-- post-merge-workflow -->

After merging a PR:

1. Switch to `main` locally and pull.
2. Run `go test ./...` — or the test most relevant to the change.
   - If a gap appears: open a new branch, fix it, push and open a PR.
3. If all green: bump the version in every file listed under **Versions**.
4. Update documentation if the change affects user-facing behavior.
5. Push a new git tag matching the version (`git tag vX.Y.Z && git push origin vX.Y.Z`).

## Before committing

- Tests pass (`go test ./...`).
- If the change affects the hook or filtering, test with a real payload before committing.
- If the install flow changes, validate the full user path, not just compilation.
- If the version is bumped, verify all version-bearing files are updated.
