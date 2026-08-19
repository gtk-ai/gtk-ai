# gtk-ai architecture

## Scope

gtk-ai reduces token usage in agentic sessions by intercepting commands before they execute and filtering output before the agent reads it. Filtering is heuristic: truncation, line caps, grouping, comment stripping. It is not semantic compression, intelligent deduplication, or ML-based summarisation.

Use: heuristic pruning, rule-based filtering, deterministic truncation.  
Avoid: intelligent compression, semantic optimisation, smart deduplication.

---

## Three-layer model

```
Agentic runtime (Claude Code, Cursor, Codex, OpenCode, …)
        │  runtime-specific hook event (JSON wire format differs per agent)
        ▼
gtk-ai/gtk-ai — core binary (gtkai)
  hook-pre · hook-post · proxy · registry · rewrite · never_worse · gain
        │  filter contract: id + filters + Rewrite + FilterOutput
        ▼
gtk-ai/gtkai-<command>  (one repository per intercepted shell argv0)
```

### Layer 1 — runtime integration (hook-pre / hook-post)

`gtkai hook-pre` and `gtkai hook-post` live in the core binary. They understand the wire format of each supported runtime. Adding a new runtime requires extending these two entry points, not creating an external adapter binary.

Supported runtimes and their shell tool names:

| Runtime | Shell tool name(s) | Pre-exec rewrite | Post-exec filter |
|---|---|---|---|
| Claude Code | `Bash` | `PreToolUse` | `PostToolUse` |
| Cursor | `Shell` | Rules hook | Rules hook |
| Codex | `shell_command`, `container_exec` | `PreToolUse` | — |
| OpenCode | `local_shell`, `bash`, `shell` | `PreToolUse` | — |

Each runtime sends a different JSON envelope. `hook-pre` normalises the input, performs the rewrite, and serialises the result in the format that runtime expects. `hook-post` does the same for output filtering.

The runtime scripts in `plugin/scripts/` invoke `gtkai hook-pre --agent <name>` and `gtkai hook-post --agent <name>`. The `--agent` flag selects the correct I/O serialisation.

### Layer 2 — core (`gtk-ai/gtk-ai`)

The binary `gtkai`. Responsibilities:

- `gtkai <cmd> [args…]` — proxy: runs the real command, captures stdout/stderr, calls the active filter, applies `never_worse`, records `gain`, propagates exit code.
- `gtkai hook-pre --agent <name>` — pre-execution: reads the runtime event, extracts the shell command, resolves the active filter by argv0, returns the rewritten command in the agent's envelope.
- `gtkai hook-post --agent <name>` — post-execution: reads the runtime event, filters output, returns the result in the agent's envelope. Used for Bash output and for native tools (Read, MCP, Grep, Glob) that do not go through a shell.
- `gtkai filter install|uninstall|list` — filter registry management.
- `gtkai gain` — token savings analytics.
- `gtkai mcp-scan` — lists MCP tools, suggests passthrough patterns.

The core is responsible for:

- Normalising different runtime event envelopes.
- Resolving the active filter for a given argv0.
- Applying ANSI stripping, `never_worse`, and `gain` recording (filters cannot bypass these).
- Serialising the response in the format the runtime expects.

### Layer 3 — filters (`gtk-ai/gtkai-<command>`)

One repository per intercepted shell argv0. Each filter:

- Declares an `id` (`author/gtkai-<command>`) and a `filters` field (the argv0 it intercepts).
- Implements `Rewrite` and `FilterOutput` (and optionally `ExtraEnv`).
- Does not register agent hooks. Does not know which runtime called it.
- Cannot bypass `never_worse`, ANSI stripping, or `gain` recording; the core applies those unconditionally.

---

## Filter identity and registry

### Naming rule

```
author/gtkai-<command>
```

- `author` — GitHub organisation or username owning the filter implementation.
- `gtkai-<command>` — fixed suffix; `<command>` is the shell argv0 (basename) intercepted.

Examples: `gtk-ai/gtkai-ls`, `gtk-ai/gtkai-git`, `acme/gtkai-ls`.

Built-in filters (compiled into the binary) use `gtk-ai` as author. Third-party filters use their own prefix and the same naming rule. The namespace never appears in Bash commands.

### Active filter resolution

Many filters may declare the same `filters` value. Only one is **active** at a time:

- **Active** = most recently installed among all filters whose `filters` equals that command.
- On `filter install`, if another filter already targets the same command, print a warning on stderr naming both. The new one becomes active.
- On `filter uninstall author/gtkai-ls` (full id required):
  - No filters remain for that command → pass through (no rewrite).
  - Other filters remain → active = most recent among survivors.

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

`Name()` returns the argv0 (e.g. `"ls"`). The namespaced `id` is declared separately in the filter manifest. `exitCode` is -1 when unknown (native tools in PostToolUse).

---

## Distribution (filter marketplace)

### Repository layout

```
gtk-ai/gtk-ai        — core binary, runtime integrations
gtk-ai/gtkai-ls      — filter for ls
gtk-ai/gtkai-git     — filter for git
gtk-ai/gtkai-go      — filter for go test/build/vet
…                    — one repo per intercepted argv0
```

Rules:
- One repository per shell command intercepted. No monorepo for filters.
- Built-in filters (compiled in) and packaged filters (separate repos) share the same id scheme. A packaged `gtk-ai/gtkai-ls` can replace the built-in one.
- Third-party authors use their own prefix: `acme/gtkai-ls`.

### install transport (sketch)

```
gtkai filter install github.com/gtk-ai/gtkai-ls@v1
gtkai filter install ./path/to/local/filter
gtkai filter uninstall gtk-ai/gtkai-ls
gtkai filter list
```

The exact transport (Go plugin, subprocess + manifest, static binary) is an implementation detail of §4. The id rules and conflict/uninstall semantics above are not.

---

## What is not in scope

- rtk TOML rule engine, `rtk init`, `discover`/`learn`, telemetry, on-disk `tee`.
- Auto-discovery from PATH or GitHub without an explicit `filter install`.
- One agent plugin per filter (filters are not registered as agent hooks).
- Semantic compression or ML-based summarisation.
- `kubectl`, `helm`, `pulumi`, `terraform`, `dotnet`, `gradle`, `phpunit` — deferred until `gain` shows demand.

---

## Current state (0.10.0)

§1 proxy, §2 corrections, and §3 runners are complete. Multi-runtime support (Claude Code, Cursor, Codex, OpenCode) landed in 0.10.0.

| Intercepted | Module | Done |
|---|---|---|
| `ls` | `modules/ls` | 0.4.0 |
| `git` (status, log, diff, branch, show, write, stash) | `modules/git` | 0.4.0–0.5.0 |
| `grep`, `rg` | `modules/grep`, `modules/rg` | 0.4.0 |
| `find` | `modules/find` | 0.4.0 |
| `cat`, `head`, `tail` | `modules/readcmd` | 0.5.0 |
| `tree` | `modules/tree` | 0.5.0 |
| `Read`, MCP | `internal/hook` (PostToolUse) | 0.5.0 |
| `go` (test, build, vet) | `modules/go` | 0.6.0 |
| `cargo` (test, build, check, clippy) | `modules/cargo` | 0.7.0 |
| `pytest`, `python -m pytest` | `modules/pytest`, `modules/python` | 0.8.0 |
| `npm`, `pnpm`, `npx` (test) | `modules/npmtest` | 0.9.0 |
| `docker` (ps, images, logs, compose ps/logs) | `modules/docker` | 0.9.0 |
| Multi-runtime: Cursor, Codex, OpenCode | `internal/hook` | 0.10.0 |

## §4 — External filter transport

### Transport: subprocess/v1

Each external filter is a standalone binary. The core communicates with it via stdin/stdout using a line-delimited JSON protocol. No shared memory, no dynamic linking.

Go plugin (`.so`) is explicitly out of scope and will not be implemented.

### JSON contract

Request (core → filter binary):

```json
{
  "operation": "rewrite" | "filter_output",
  "args": ["<argv0>", "…"],
  "output": "<stdout+stderr, filter_output only>",
  "exit_code": 0
}
```

Response (filter binary → core):

```json
{
  "args": ["<rewritten args, rewrite only>"],
  "changed": true,
  "output": "<filtered output, filter_output only>"
}
```

`exit_code` is -1 when unknown (native tool post-hook). `changed` signals whether the core should use the rewritten value; unchanged responses short-circuit further processing.

### filter.json manifest (required)

Every filter binary ships a `filter.json` at the root of its repository:

```json
{
  "id": "author/gtkai-<command>",
  "filters": ["<argv0>"],
  "version": "1.2.3",
  "platforms": ["linux/amd64", "darwin/arm64"],
  "contract": "subprocess/v1",
  "min_gtkai_version": "0.11.0"
}
```

All fields are required. `id` must match `^[a-z0-9_-]+/gtkai-[a-z0-9_-]+$`. `version` and `min_gtkai_version` must be valid semver.

### Validation on install

`gtkai filter install` performs these checks in order before committing anything to the registry:

1. `filter.json` is present and parses without error.
2. `id` matches the naming rule regex.
3. `version` and `min_gtkai_version` are valid semver; running `gtkai` satisfies `min_gtkai_version`.
4. `contract` is `subprocess/v1`.
5. Running platform appears in `platforms`.
6. Liveness check: spawn the binary with `{"operation":"rewrite","args":[],"output":"","exit_code":0}` and expect a valid response within 500 ms.

Any failure aborts the install with a descriptive error. No partial state is written.

### Persistence

Installed filters are recorded in `~/.gtk-ai/filters.db` (SQLite). The path is resolved via `internal/storage.Dir()`. Schema: `id`, `filters`, `version`, `contract`, `binary_path`, `installed_at`.

### Binary layout

```
~/.gtk-ai/filters/<id>/
    <binary>
    filter.json
```

Binaries are downloaded from the GitHub Releases page of the filter repository and stored under the namespaced path.

### Official filters and install.sh

`install.sh` downloads the official `gtk-ai/gtkai-*` filters by default alongside the core binary.

The reference template repository for building a new filter is `gtk-ai/gtkai-date`.

### Built-ins as fallback

Built-in filters (compiled into `gtkai`) remain active as fallback when no external filter is installed for a given argv0. Installing an external filter for the same command makes it active and shadows the built-in; uninstalling it restores the built-in.

## Pending

- §4 — filter registry with namespaced ids (`gtk-ai/gtkai-*`), `filter install/uninstall/list`, migrate built-ins.
- Native `Grep`/`Glob` filtering in PostToolUse.
- `gain` per-filter attribution by `filter_id`.
- `gtk-ai` GitHub organisation + per-filter repositories (marketplace prerequisite; not required for §4 core work).
