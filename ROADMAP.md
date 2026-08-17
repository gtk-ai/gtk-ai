# Roadmap: gtk-ai vs rtk 0.42.4

Compared against [rtk-ai/rtk](https://github.com/rtk-ai/rtk) `0.42.4` (`ba7a9ce`). gtk-ai is at `0.4.0`.

**Product decision (2026-08-17):** gtk follows the same path as rtk. It rewrites the command **before** execution (`PreToolUse` → `git status` becomes `gtkai git status`). gtkai runs the real binary, injects flags, and filters the output. Post-filtering remains only for Claude Code native tools that do not go through Bash (`Read`, MCP, `Grep`, `Glob`).

---

## 1. Primary — PreToolUse rewrite (proxy)

Today gtk filters *after* the fact. `Module.Rewrite` exists and is never called. That is the design gap, not a missing command catalog.

### Target flow

```text
Without gtk:  Claude --git status--> shell --> git --> raw stdout --> Claude

With gtk:     Claude --git status--> PreToolUse --> gtkai hook-pre
                                                       |
                                                       v
                                          command: gtkai git status
                                                       |
                                                       v
                                          gtkai runs git (injected flags)
                                          filters stdout/stderr
                                          records gain
                                                       |
                                                       v
                                          Claude receives compact output
```

Claude never sees the rewrite. The agent calls `git status`; the hook replaces it with `gtkai git status`.

### What this changes (and what it does not)

| Piece | Change |
|---|---|
| Plugin | Registers `PreToolUse` (matcher `Bash`) in addition to `PostToolUse`. The plugin still configures Claude Code. |
| Binary | Becomes a CLI proxy: `gtkai git status`, `gtkai ls`, `gtkai hook-pre`. It runs the real command. |
| `Rewrite()` | Stops being dead code: injects flags (`git status --porcelain -b`, `go test -json`, `git log --pretty=…`). |
| `FilterOutput` | Runs **inside** the proxy, on output gtkai just captured. |
| `PostToolUse` | Stays for `Read`, `mcp__*`, and later `Grep`/`Glob`. It is not the Bash path. |
| Repo phases | Still two pieces: the plugin registers hooks; the binary filters and now also executes. They stay separate. |

Do not copy from rtk: multi-agent `rtk init`, the TOML engine, `discover`/`learn`, telemetry, on-disk `tee`. gtk stays heuristic (truncation, grouping, stripping).

### Scope of this phase

One PR (or a few) that makes the rtk path work end-to-end for **one** command, plus the infrastructure for the rest.

1. **`gtkai hook-pre`**: reads PreToolUse JSON, rewrites `tool_input.command` when the binary is registered, writes `updatedInput`. If there is no module, pass through.
2. **Plugin**: `PreToolUse` + `gtkai-pre-tool-use.sh`. Matcher `Bash`. Short timeout.
3. **CLI proxy**: `gtkai <module> [args…]` runs the command, captures stdout+stderr, filters, prints, propagates the exit code, records `gain`.
4. **Command detection**: the first word is not enough. Cover `/usr/bin/git`, `sudo git`, `git -C dir status`, `VAR=1 git status`. Pipelines: rewrite only the last stage if it is safe (`grep`/`rg`); otherwise pass through.
5. **`never_worse` guard**: if filtering estimates more tokens than the raw output, print the raw output.
6. **Strip ANSI** before parsing.
7. **First end-to-end command: `git status`**. Rewrite to `gtkai git status`. The module injects `--porcelain -b` (unless the user already asked for another format) and groups by state.

Done when:

- `git status` in Claude Code is rewritten to `gtkai git status` (PreToolUse test payload).
- `gtkai git status` in a terminal produces compact output and git's exit code.
- `gtkai gain` records that invocation.
- An unregistered command (`echo hi`) is left untouched.

Until this is green, do not add new modules. The current Bash post-filter is removed once the proxy covers existing modules; until then it may coexist so there is no coverage hole.

---

## 2. Corrections — existing modules that do not deliver

With the proxy, these modules can inject flags. That is what they cannot do today, which is why they fail.

| Module | Current problem | Fix with rewrite |
|---|---|---|
| `ls` | Done in `0.4.0`: `-l` + `LC_ALL=C`, `-a` only if requested, counts + samples, noise dirs skipped. | — |
| `git status` | Expects porcelain. Claude runs the long format; the parser never matches. | Covered in the primary phase. |
| `git diff` | Done in `0.4.0`: strip `index`/`+++`/`---`, cap per hunk. | — |
| `git log` | Done in `0.4.0`: inject `%h %s (%ar) <%an>` and `-10`; honor `--pretty`/`--oneline`/`-n`. | — |
| `git branch` | Partial. | Filter remotes; cap. |
| `grep` | Done in `0.4.0`: same grouping as `rg`; injects `-nH`. | — |
| `rg` | Grouping shared with `grep`. Pipelines rewrite the last stage. | — |
| `find` | Done in `0.4.0`: group by directory, cap 50, small outputs unchanged. | — |
| `Read` | Only `//` and `#` lines. | `/* */` blocks, more extensions. No “signatures only” mode (rtk aggressive: unsafe). |
| `gain` | SQLite is ready; the hook never records. | Wired in the proxy runner (phase 1). |

Add in the same block, because they are the same Bash path rtk rewrites to `read`:

- `git show`, `git add`/`commit`/`push`/`pull`/`fetch`/`stash` (confirmation, no progress).
- `cat` / `head` / `tail` → `gtkai read` (reuses `read.FilterContent`).
- `tree`.

Native tools (stay on `PostToolUse`; rtk cannot do this):

- `Grep`, `Glob`.
- MCP: keep truncation + passthrough; compact JSON when the body is text.

Done when: fixtures for long status, unified diff, verbose log, `ls -la`. Measurable savings on all of them, including `ls`. `go test ./...` green.

---

## 3. Remainder — runners and ecosystem

Only after the proxy and the corrections. Here rewrite does inject flags (`go test -json`).

### Runners (high noise, ~90% in rtk)

| Module | Commands | Rewrite | Filter |
|---|---|---|---|
| `gotest` | `go test`, `go build`, `go vet` | `go test -json` unless `-bench` or `-json` is already present | `ok` packages → count; full `FAIL` |
| `cargo` | `test`, `build`, `clippy`, `check` | by subcommand | errors/failures; collapse `Compiling` |
| `pytest` | `pytest`, `python -m pytest` | — | failures + short traceback |
| `npmtest` | `npm test`, `pnpm test`, `npx vitest`/`jest` | — | failures; strip ANSI |
| `docker` | `ps`, `images`, `logs`, `compose ps/logs` | — | essential columns; capped logs |

Done when: `go test` fixture with 40 ok packages + 1 FAIL; the agent sees the FAIL and a count of the ok packages.

### Ecosystem, only if `gain` asks for it

Do not open modules “just in case”:

- linters: `ruff check`, `tsc`, `eslint`/`biome`, `golangci-lint`
- `gh pr`/`issue`/`run`
- `kubectl get`/`logs`
- `curl` JSON (passthrough if the body is not text)

Out of scope until the core and the runners cover a typical session: rtk TOML filters (`helm`, `pulumi`, `terraform`, `dotnet`, `gradle`, `phpunit`, …), multi-agent, `discover`/`learn`.

---

## 4. Filter plugins — namespaced filters (business logic)

This is **not** the Claude Code plugin system and not any agent hook/marketplace. It is the domain model inside gtkai: who implements filtering for which shell command.

Today the registry collapses **command = filter** (`ls` → one `Module`). The target model separates **identity** from **what gets intercepted**.

### Identity

Every filter has a full name:

```text
author/gtkai-<command>
```

Examples: `gtkai/gtkai-ls`, `gtkai/gtkai-git`, `jmeiracorbal/gtkai-ls`.

- `author` — who owns the implementation.
- `gtkai-<command>` — fixed suffix; `<command>` is the shell argv0 this filter is for (`ls`, `git`, `find`, …).

Built-in filters ship as `gtkai/gtkai-<command>` (compiled in). Third-party filters use the same naming rule.

The agent still runs `ls`, `git status`, … PreToolUse still rewrites to `gtkai ls`, `gtkai git status`. The namespace never appears in the Bash command Claude sees.

### Contract

Each filter declares, at minimum:

| Field | Meaning |
|---|---|
| `id` | Full name `author/gtkai-<command>` (must match the naming rule) |
| `filters` | Shell command intercepted (argv0 basename), e.g. `ls` |

Behavior matches today’s `Module`: `Rewrite`, `FilterOutput`, optional `ExtraEnv`. The core keeps ANSI strip, `never_worse`, and `gain`; filters do not bypass them.

A filter owns the full surface of that command or passes through: if it cannot handle an invocation, `Rewrite` returns no change and gtkai runs the original argv unchanged.

### Which filter is active

Many filters may target the same shell command. Only one is **active** at a time:

- **Active filter** = the **most recently installed** among all filters whose `filters` field equals that command.

On **install**, when another filter already targets the same command:

- Print a **warning** (stderr) naming the previous active filter and the new one.
- The newly installed filter becomes active.

On **uninstall** (`gtkai filter uninstall author/gtkai-<command>` — full name only):

- Remove that filter by `id`.
- If **no filters remain** for that shell command → do not rewrite it in PreToolUse; pass through.
- If **other filters remain** → active = most recent among the survivors (same rule as install).

Listing installed filters and which one is active per command is part of this phase.

### CLI (sketch)

```text
gtkai filter install <path-or-package>   # register filter; warn on command conflict
gtkai filter uninstall author/gtkai-ls   # by full id only
gtkai filter list                        # all filters; mark active per command
```

Exact transport (Go package, manifest + subprocess, …) is an implementation detail. The rules above are not.

### What this is not

- Not a Claude Code plugin per filter.
- Not rtk’s TOML rule engine.
- Not auto-discovery from PATH or GitHub without an explicit `filter install`.

Third-party filters are installed into gtkai’s filter registry; they do not register agent hooks.

Done when: native `gtkai/gtkai-*` filters use the same registry; install/uninstall/list work; conflict and uninstall semantics above have tests; `hook-pre` resolves the active filter by shell command before rewrite.

---

## Current vs target

| | gtk-ai 0.4.0 | Remaining |
|---|---|---|
| Bash | `PreToolUse` rewrites registered commands to `gtkai …`; the binary runs and filters | Drop Bash `PostToolUse` once coverage is trusted |
| Filter identity | Flat `registry.Get("ls")` | Namespaced `author/gtkai-<command>`; active = most recent install |
| `Rewrite()` | Injects flags for `git status`, `git log`, `ls`, `grep` | Runners (`go test -json`); third-party filters via `filter install` |
| `Read` / MCP | `PostToolUse` | `/* */` on Read; native `Grep`/`Glob` |
| `gain` | Every proxy execution | Per-filter attribution by `id` |
| Commands | find, ls, git (status/log/diff/branch), grep, rg, Read, MCP | §2 remainder, then runners, then filter plugin CLI |

Filtering stays heuristic. No semantic compression.

---

## Risks

- **Hook contract.** A `PreToolUse` that writes bad JSON disables the hook (Claude Code silences it). Tests with a real payload; stdout is JSON only.
- **Pipelines and `&&`.** Rewriting `find … \| head` as `find` breaks the meaning. Last safe stage only, or pass through.
- **Write commands.** `git commit`, `git push`, `docker run` must not be left half-done. The proxy forwards stdin/TTY when the command is not read-only; if it cannot, do not rewrite.
- **Double filtering.** While Pre and Post both sit on Bash, output can be filtered twice and grow. Drop Bash Post as soon as the proxy covers the module.
- **`never_worse`.** A 3-file status must not become a longer paragraph.
- **Filter stack.** Multiple filters for one command rely on install order; uninstall must not leave a stale active pointer.
- **False positives in runners.** Keep any line that is not classified.

---

## PR order

1. PreToolUse proxy + end-to-end `git status` (section 1) — **done in 0.4.0**.
2. Corrections to current modules + `cat`/`head`/`tail`/`tree` + remaining git (section 2).
3. Runners (section 3).
4. Filter plugin registry + install/uninstall/list + migrate natives to `gtkai/gtkai-*` (section 4).
5. Ecosystem commands as third-party filters according to `gain` (section 3 ecosystem).

The git tag must match every version-bearing file (`cmd/gtkai/main.go`, plugin json, `mcpscan`, README).
