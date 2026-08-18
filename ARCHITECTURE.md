# gtk-ai architecture

## Scope

gtk-ai reduces token usage in agentic sessions by intercepting commands before they execute and filtering the output before the agent reads it. Filtering is heuristic: truncation, line caps, grouping, comment stripping. It is not semantic compression, intelligent deduplication, or ML-based summarisation.

Use these terms: heuristic pruning, rule-based filtering, deterministic truncation.  
Avoid: intelligent compression, semantic optimisation, smart deduplication.

---

## Three-layer model

```
Agentic runtime (Claude Code, Cursor, Codex, …)
        │
        │  thin adapter (registers hooks, speaks runtime JSON)
        ▼
gtk-ai/gtk-ai — core binary (gtkai)
  proxy · registry · rewrite · never_worse · gain
        │
        │  filter contract (id + filters + Rewrite + FilterOutput)
        ▼
gtk-ai/gtkai-<command>  (one repo per shell command intercepted)
```

### Layer 1 — adapters

Each agentic runtime gets a thin adapter that:

1. Registers the hooks the runtime supports (e.g. `PreToolUse`, `PostToolUse` in Claude Code).
2. Extracts the command or output from the runtime-specific event payload.
3. Calls `gtkai` using the **core contract** (see below).
4. Injects the result back into the runtime-specific response.

Adapters do not filter. They translate between the runtime wire format and the core contract. A new runtime means a new adapter, not changes to the core or to the filters.

The Claude Code adapter lives in `plugin/` of this repo for now. It can be extracted to `gtk-ai/adapter-claude` when a second adapter exists.

### Layer 2 — core (`gtk-ai/gtk-ai`)

The binary `gtkai`. Responsibilities:

- `gtkai <cmd> [args…]` — proxy: runs the real command, captures stdout/stderr, calls the active filter, applies `never_worse`, records `gain`, propagates exit code.
- `gtkai hook-pre` — pre-execution capture: receives a command string, resolves the active filter by argv0, returns the rewritten command.
- `gtkai hook-post` — post-execution capture: receives command + raw output + exit code, returns filtered output. Used for native tools that do not go through Bash (Read, MCP, Grep, Glob).
- `gtkai filter install|uninstall|list` — filter registry management.
- `gtkai rewrite <command>` — stateless rewrite query (useful for adapter scripting).
- `gtkai gain` — token savings analytics.
- `gtkai mcp-scan` — lists MCP tools and suggests passthrough patterns.

The core never parses runtime-specific JSON (no `hookSpecificOutput`, no `updatedInput`, no `tool_name` field). That is the adapter's job.

### Layer 3 — filters (`gtk-ai/gtkai-<command>`)

One repository per intercepted shell argv0. Each filter:

- Declares an `id` (`author/gtkai-<command>`) and a `filters` field (the argv0 it intercepts).
- Implements `Rewrite` and `FilterOutput` (and optionally `ExtraEnv`).
- Does not register agent hooks. Does not know which runtime called it.
- Cannot bypass `never_worse`, ANSI stripping, or `gain` recording — the core applies those.

---

## Core contract (hook-pre / hook-post)

Adapters call `gtkai hook-pre` and `gtkai hook-post` via stdin/stdout. The payload is the **gtkai contract**, not the runtime format.

### hook-pre (pre-execution rewrite)

```
stdin  → {"command": "git status"}
stdout → {"command": "gtkai git status"}   # rewritten
       or nothing                           # no active filter, pass through
```

If the stdout is empty or `{"command": ""}`, the caller must use the original command unchanged.

### hook-post (post-execution filter)

```
stdin  → {"command": "cat file.go", "output": "…raw…", "exit_code": 0}
stdout → {"output": "…filtered…"}          # never worse than raw
       or nothing                           # no change
```

`exit_code` is -1 when unknown (native tools whose exit code is not available to the adapter).

### Invariants

- Stdout is always valid JSON or empty. Never mixed content.
- A hook that exits non-zero is treated as pass-through by the adapter.
- Adapters must not write to stdout before calling `gtkai`; if they do, the runtime may see corrupted output.

---

## Filter identity and registry

### Naming rule

```
author/gtkai-<command>
```

- `author` — GitHub organisation or username owning the filter implementation.
- `gtkai-<command>` — fixed suffix; `<command>` is the shell argv0 (basename) the filter intercepts.

Examples: `gtk-ai/gtkai-ls`, `gtk-ai/gtkai-git`, `acme/gtkai-ls`.

Built-in filters (compiled into the binary) use `gtk-ai/gtkai-*` as author. Third-party filters use their own author prefix and the same naming rule.

The namespace never appears in Bash commands. The agent calls `ls`; PreToolUse rewrites to `gtkai ls`; the filter id is internal.

### Active filter resolution

Many filters may declare the same `filters` value. Only one is **active** at a time:

- **Active filter** = most recently installed among all filters whose `filters` equals that command.
- On `filter install`, if another filter already targets the same command, print a warning on stderr naming both. The new one becomes active.
- On `filter uninstall author/gtkai-ls` (full id only):
  - If no filters remain for that command → pass through (no rewrite).
  - If other filters remain → active = most recent among survivors.

### Filter contract (Go interface, current)

```go
type Module interface {
    Name() string
    Rewrite(args []string) ([]string, bool)
    FilterOutput(args []string, output string, exitCode int) string
    TokensBefore(output string) int
    TokensAfter(filtered string) int
}

type EnvInjector interface {
    ExtraEnv(args []string) []string
}
```

`Name()` returns the argv0 (e.g. `"ls"`). The namespaced `id` is declared separately in the filter manifest.

---

## Adapter contract (for future adapters)

An adapter must:

1. Detect a Bash/shell command invocation in the runtime event.
2. Send `{"command": "<raw command>"}` to `gtkai hook-pre` via stdin.
3. If stdout is non-empty, replace the command field in the event with the returned command.
4. After execution, send `{"command": "…", "output": "…", "exit_code": N}` to `gtkai hook-post`.
5. If stdout is non-empty, replace the output in the runtime response.

For native tools (Read, Grep, Glob, MCP) that do not go through Bash:

- Send `{"command": "<tool_name>:<file_or_key>", "output": "…", "exit_code": -1}` to `gtkai hook-post`.
- The core matches against registered native-tool filters (today: Read, MCP).

Adapters that only support post-execution hooks (no pre-execution rewrite) still benefit from `hook-post`. Rewrite is an optimisation, not a requirement.

### Runtime support matrix

| Runtime | Pre-exec rewrite | Post-exec filter | Notes |
|---|---|---|---|
| Claude Code | `PreToolUse` (Bash) | `PostToolUse` (Read, MCP) | First adapter; in `plugin/` |
| PATH wrapper | Always | Not applicable | No plugin needed; `gtkai` is on PATH and the shell resolves it |
| Others | Depends on runtime | Depends on runtime | One adapter per runtime |

---

## Distribution (filter marketplace)

### Repository layout

```
gtk-ai/gtk-ai        — core binary + Claude Code adapter (current)
gtk-ai/gtkai-ls      — filter for ls
gtk-ai/gtkai-git     — filter for git
gtk-ai/gtkai-go      — filter for go test/build/vet
…                    — one repo per argv0
```

Rules:
- One repository per shell command intercepted. No monorepo for filters.
- Built-in filters (compiled in the binary) and packaged filters (separate repos) share the same id scheme. A built-in `gtk-ai/gtkai-ls` and a packaged `gtk-ai/gtkai-ls` have the same id; the packaged version can replace the built-in.
- Third-party authors use their own org/username as the `author` prefix.

### install transport (sketch)

```
gtkai filter install github.com/gtk-ai/gtkai-ls@v1
gtkai filter install ./path/to/local/filter
gtkai filter uninstall gtk-ai/gtkai-ls
gtkai filter list
```

The exact install transport (Go plugin, subprocess + manifest, static binary) is an implementation detail of §4. The id and conflict/uninstall rules above are not.

---

## What is not in scope

- rtk TOML rule engine, `rtk init`, `discover`/`learn`, telemetry, on-disk `tee`.
- Auto-discovery from PATH or GitHub without an explicit `filter install`.
- One Claude Code plugin per filter (filters are not registered as agent hooks).
- Semantic compression or ML-based summarisation.
- `kubectl`, `helm`, `pulumi`, `terraform`, `dotnet`, `gradle`, `phpunit` — ecosystem scope deferred until `gain` shows demand.

---

## Current state (0.9.0)

§1 proxy, §2 corrections, and §3 runners are complete.

| Intercepted | Module | Done |
|---|---|---|
| `ls` | `modules/ls` | 0.4.0 |
| `git` (status, log, diff, branch, show, write subcommands, stash) | `modules/git` | 0.4.0–0.5.0 |
| `grep`, `rg` | `modules/grep`, `modules/rg` | 0.4.0 |
| `find` | `modules/find` | 0.4.0 |
| `cat`, `head`, `tail` | `modules/readcmd` | 0.5.0 |
| `tree` | `modules/tree` | 0.5.0 |
| `Read`, MCP | `internal/hook` (PostToolUse) | 0.5.0 |
| `go` (test, build, vet) | `modules/go` | 0.6.0 |
| `cargo` (test, build, check, clippy) | `modules/cargo` | 0.7.0 |
| `pytest`, `python -m pytest` | `modules/pytest`, `modules/python` | 0.8.0 |
| `npm`, `pnpm`, `npx` (test only) | `modules/npmtest` | 0.9.0 |
| `docker` (ps, images, logs, compose ps/logs) | `modules/docker` | 0.9.0 |

## Pending

- §4 — filter registry with namespaced ids, `filter install/uninstall/list`, migrate built-ins to `gtk-ai/gtkai-*`.
- Native `Grep`/`Glob` filtering in PostToolUse.
- `gain` per-filter attribution by `id`.
- Decouple `hook-pre`/`hook-post` from Claude-specific JSON (adapter owns translation; core owns the gtkai contract above).
- Second runtime adapter (when a concrete runtime with equivalent hook support is targeted).
- `gtk-ai` GitHub organisation + per-filter repositories (prerequisite for marketplace; not required for §4 core work).
