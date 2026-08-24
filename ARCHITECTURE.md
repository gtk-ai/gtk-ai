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
gtk-ai/<cmd>  (one repository per intercepted shell argv0)
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

### Layer 3 — filters (`gtk-ai/<cmd>`)

One repository per intercepted shell argv0. Each filter:

- Declares an `id` (`author/<cmd>`) and a `command` field (the argv0 it intercepts).
- Implements `Rewrite` and `FilterOutput` (and optionally `ExtraEnv`).
- Does not register agent hooks. Does not know which runtime called it.
- Cannot bypass `never_worse`, ANSI stripping, or `gain` recording; the core applies those unconditionally.

---

## Filter identity and registry

### Naming rule

```
author/<cmd>
```

- `author` — GitHub organisation or username owning the filter implementation.
- `<cmd>` — the shell argv0 (basename) intercepted; for official filters, matches the repository name under `gtk-ai/`.

Examples: `gtk-ai/date`, `gtk-ai/ls`, `acme/ls`.

Built-in filters (compiled into the binary) use `gtk-ai` as author. Third-party filters use their own prefix and the same naming rule. The namespace never appears in Bash commands.

### Active filter resolution

Many filters may declare the same `command` value. Only one is **active** at a time:

- **Active** = most recently installed among all filters whose `command` equals that argv0.
- On `filter install`, if another filter already targets the same command, abort unless `--replace` is passed. With `--replace`, the new filter becomes active; the previous one stays installed but inactive.
- `filter install-marketplace` (used by `install.sh`) passes `--replace` implicitly.
- On `filter uninstall gtk-ai/ls` (full id required):
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
gtk-ai/gtk-ai   — core binary, runtime integrations
gtk-ai/date     — filter for date (reference template)
gtk-ai/ls       — filter for ls
gtk-ai/git      — filter for git
…               — one repo per intercepted argv0
```

Rules:
- One repository per shell command intercepted. No monorepo for filters.
- Built-in filters (compiled in) and packaged filters (separate repos) share the same id scheme. A packaged `gtk-ai/ls` can replace the built-in one.
- Third-party authors use their own prefix: `acme/ls`.

### install transport (sketch)

```
gtkai filter install github.com/gtk-ai/ls@v1
gtkai filter uninstall gtk-ai/ls
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
| `ls` | `plugins/ls` | 0.4.0 |
| `git` (status, log, diff, branch, show, write, stash) | `plugins/git` | 0.4.0–0.5.0 |
| `grep`, `rg` | `plugins/grep`, `plugins/rg` | 0.4.0 |
| `find` | `plugins/find` | 0.4.0 |
| `cat`, `head`, `tail` | `plugins/readcmd` | 0.5.0 |
| `tree` | `plugins/tree` | 0.5.0 |
| `Read`, MCP | `internal/hook` (PostToolUse) | 0.5.0 |
| `go` (test, build, vet) | `plugins/go` | 0.6.0 |
| `cargo` (test, build, check, clippy) | `plugins/cargo` | 0.7.0 |
| `pytest`, `python -m pytest` | `plugins/pytest`, `plugins/python` | 0.8.0 |
| `npm`, `pnpm`, `npx` (test) | `plugins/npmtest` | 0.9.0 |
| `docker` (ps, images, logs, compose ps/logs) | `plugins/docker` | 0.9.0 |
| Multi-runtime: Cursor, Codex, OpenCode | `internal/hook` | 0.10.0 |

## §4 — External filter transport

### Transport: subprocess/v1

**In one sentence:** `subprocess/v1` means gtk-ai runs your filter as an external program and talks to it via JSON on stdin/stdout, according to the v1 specification.

Each external filter is a standalone binary. The core communicates with it via stdin/stdout using a JSON protocol. No shared memory, no dynamic linking.

#### Why this model

- **Language-agnostic** — the filter can be written in any language (Go, Rust, Python, …) as long as it produces a binary that speaks the JSON protocol on stdin/stdout.
- **Isolation** — if the filter crashes or misbehaves, the core keeps running.
- **Simple installation** — download or compile one binary per filter; no plugins or shared libraries.

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

### gtkai.json manifest (required)

Every filter module ships a `gtkai.json` manifest at the repository root:

```json
{
  "id": "author/<cmd>",
  "command": "<argv0>",
  "platforms": ["linux/amd64", "darwin/arm64"],
  "contract": "subprocess/v1",
  "gtkai-core-version": {
    "version": "0.11.0",
    "constraint": "min"
  }
}
```

Required fields: `id`, `command`, `platforms`, `contract`, `gtkai-core-version`.

- `id` must match `^[a-z0-9_-]+/[a-z0-9_-]+$`.
- `contract` must be `subprocess/v1`.
- `gtkai-core-version.version` must be valid semver.
- `gtkai-core-version.constraint` must be `"min"` or `"exact"`:
  - `"min"` — running `gtkai` must be `>= version`.
  - `"exact"` — running `gtkai` must match `version` exactly.

**Module version is not in the manifest.** It is resolved from the install ref — the git tag of the filter repository. Example: `gtkai filter install github.com/gtk-ai/date@v0.12.0` installs tag `v0.12.0`; the core records that version in the registry at install time.

On install, the core validates `gtkai-core-version` against the running binary:

```go
switch manifest.GtkaiCoreVersion.Constraint {
case "min":
    if semver.Compare(runningGtkai, manifest.GtkaiCoreVersion.Version) < 0 {
        return fmt.Errorf("gtkai %s < required min %s", runningGtkai, manifest.GtkaiCoreVersion.Version)
    }
case "exact":
    if runningGtkai != manifest.GtkaiCoreVersion.Version {
        return fmt.Errorf("gtkai %s != required exact %s", runningGtkai, manifest.GtkaiCoreVersion.Version)
    }
default:
    return fmt.Errorf("unknown constraint %q", manifest.GtkaiCoreVersion.Constraint)
}
```

### Validation on install

`gtkai filter install` performs these checks in order before committing anything to the registry:

1. Git tag/ref resolves to a valid semver (module version).
2. `gtkai.json` is present and parses without error.
3. `id` matches the naming rule regex.
4. `contract` equals `subprocess/v1`.
5. Running platform appears in `platforms`.
6. `gtkai-core-version.version` is valid semver.
7. `gtkai-core-version.constraint` is `"min"` or `"exact"`.
8. Running `gtkai` satisfies the core version constraint (see switch above).
9. Liveness check: spawn the binary with `{"operation":"rewrite","args":[],"output":"","exit_code":0}` and expect a valid response within 500 ms.

Any failure aborts the install with a descriptive error. No partial state is written.

### Persistence

Installed filters are recorded in `~/.gtk-ai/plugins.db` (SQLite). The path is resolved via `internal/storage.Dir()`. Schema: `id`, `filters`, `version`, `contract`, `binary_path`, `installed_at`. The installed module version is stored from the Git tag, not from the manifest.

### Binary layout

```
~/.gtk-ai/filters/gtk-ai/date/
    date
    gtkai.json
```

Binaries are downloaded from the GitHub Releases page of the filter repository and stored under the namespaced path.

### Marketplace and install.sh

`marketplace.json` at the repository root is the single catalog of installable gtk-ai extensions. `install.sh` runs `gtkai filter install-marketplace marketplace.json` by default (`--replace` implicit).

```json
{
  "name": "gtk-ai",
  "entries": [
    {
      "module": "github.com/gtk-ai/date",
      "version": "v0.12.0"
    }
  ]
}
```

Each entry is installed via `filter install` semantics (download, validate `gtkai.json`, register in `~/.gtk-ai/plugins.db`). New filters are added here — no separate official filter list.

The reference template repository for building a new filter is [gtk-ai/date](https://github.com/gtk-ai/date) ([HOWTO.md](https://github.com/gtk-ai/date/blob/main/HOWTO.md)).

### Built-ins as fallback

Built-in filters (compiled into `gtkai`) remain active as fallback when no external filter is installed for a given argv0. Installing an external filter for the same command makes it active and shadows the built-in; uninstalling it restores the built-in.

### Built-in migration

Built-ins migrate to external repos gradually — one `gtk-ai/<cmd>` repository at a time:

1. Publish the filter repo and add an entry to `marketplace.json`.
2. `install.sh` installs it by default; the external filter shadows the built-in.
3. Remove the built-in blank import from `cmd/gtkai/main.go` only when the command should require an external install (as with `date`).

Until step 3, the built-in remains compiled in as fallback.

Filter repos follow `gtk-ai/<cmd>` (Go module `github.com/gtk-ai/<cmd>`, filter id `gtk-ai/<cmd>`).

## Pending

- Migrate remaining built-ins to external `gtk-ai/*` repos (gradual; `date` done).
- Native `Grep`/`Glob` filtering in PostToolUse.
- `gain` per-filter attribution by `filter_id`.
