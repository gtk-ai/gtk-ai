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

## Clean install validation

Run this before every release. The goal is to catch what unit tests can't: integration scripts with wrong flags, missing `--agent`, stale plugin cache, broken hook wiring.

### 1. Verify integration scripts

```bash
grep -n "hook-post\|hook-pre" integrations/claude/scripts/*.sh
```

Every call to `hook-post` must pass `--agent=claudecode`. Every call to `hook-pre` must pass `--agent=claudecode`. No bare `hook-post` or `hook-pre` without the flag.

### 2. Build and install the binary locally

```bash
go build -o ~/.local/bin/gtkai ./cmd/gtkai/
gtkai version
```

### 3. Test with real payloads

PostToolUse — Bash output filtering:

```bash
echo '{"tool_name":"Bash","tool_input":{"command":"git status"},"tool_response":{"stdout":"On branch main\nnothing to commit\n","interrupted":false},"tool_output":null}' \
  | gtkai hook-post --agent=claudecode
```

Expected: silent (no output) if nothing to filter, or filtered JSON on stdout.

PostToolUse — Read tool:

```bash
echo '{"tool_name":"Read","tool_input":{"file_path":"README.md"},"tool_response":[{"type":"text","text":"# gtk-ai\n..."}],"tool_output":null}' \
  | gtkai hook-post --agent=claudecode
```

PreToolUse — command rewrite:

```bash
echo '{"tool_name":"Bash","tool_input":{"command":"git status"}}' \
  | gtkai hook-pre --agent=claudecode
```

Expected: JSON with `{"hookSpecificOutput":{"hookEventName":"PreToolUse","updatedInput":{"command":"gtkai git status"}}}`.

### 4. Uninstall and reinstall the Claude Code plugin

```bash
claude plugin uninstall gtk-ai
claude plugin install -s user gtk-ai@gtk-ai
```

After reinstall, inspect the installed script to confirm it passes `--agent`:

```bash
cat ~/.claude/plugins/cache/gtk-ai/*/scripts/gtkai-post-tool-use.sh | grep "hook-post"
```

Must show `hook-post --agent=claudecode`, not bare `hook-post`.

### 5. Check other agents

For Cursor — verify the hooks scripts exist and pass the right flags:

```bash
grep "hook-post\|hook-pre" ~/.cursor/hooks/gtkai-*.sh 2>/dev/null || echo "not installed"
```

For Codex — same check:

```bash
grep "hook-pre" ~/.codex/hooks/gtkai-pre-tool-use.sh 2>/dev/null || echo "not installed"
```

---

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
