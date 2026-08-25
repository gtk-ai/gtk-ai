# AGENTS.md

## Architecture

Two independent phases:

1. **Go binary** (`gtkai`): command proxy and token reduction. PreToolUse rewrites `git status` to `gtkai git status`; the binary runs the real command, injects flags, and filters output.
2. **Agent integrations**: Claude Code plugin, Cursor hooks, Codex hooks, and the OpenCode plugin. Each one registers hooks and invokes the binary.

Never collapse both phases into one. The integrations depend on the binary. The binary does not write agent config files; `install.sh` does.

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

External plugins are binaries that speak the `stdin/v1` JSON protocol on stdin/stdout. The infrastructure lives in `internal/`:

| Package | Role |
|---|---|
| `pluginregistry` | SQLite DB (`~/.gtk-ai/plugins.db`) — tracks installed plugins |
| `pluginsubprocess` | Adapts an external binary to `registry.Module` via stdin/v1 |
| `plugininstall` | Downloads, validates, and installs plugin binaries |
| `pluginmanifest` | Parses and validates `gtkai.json` plugin manifests |

Built-in plugins in `plugins/` are compiled into the binary and registered via `init()`. External plugins use the subprocess adapter. Both implement `registry.Module` — the proxy treats them identically.

### Plugin naming

```
author/<cmd>
```

`author` is the GitHub org or username. `<cmd>` is the shell argv0 intercepted. For official plugins: `gtk-ai/date`, `gtk-ai/ls`, etc. Third-party authors use their own prefix.

### stdin/v1 protocol

Request (core → plugin binary):

```json
{
  "operation": "rewrite" | "filter_output",
  "args": ["..."],
  "output": "...",
  "exit_code": 0
}
```

Response (plugin binary → core):

```json
{
  "args": ["..."],
  "changed": true,
  "output": "..."
}
```

`changed: false` short-circuits processing — the original value passes through unchanged. `exit_code` is -1 when unknown (native tool post-hook).

### gtkai.json manifest

Every plugin ships a `gtkai.json` at the repo root:

```json
{
  "id": "author/<cmd>",
  "command": "<argv0>",
  "platforms": ["linux/amd64", "darwin/arm64"],
  "contract": "stdin/v1",
  "gtkai-core-version": {
    "version": "0.11.0",
    "constraint": "min"
  }
}
```

`contract` must be `stdin/v1`. `constraint` is `"min"` (running gtkai >= version) or `"exact"` (must match). On install, the core validates the manifest and runs a contract check (sends `rewrite` and `filter_output` probes and expects valid JSON back) before writing anything to the registry.

## Post-merge

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
