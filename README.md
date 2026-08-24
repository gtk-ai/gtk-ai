# gtk-ai

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go&logoColor=white)
![Version](https://img.shields.io/badge/version-0.11.0--beta.2-blue?style=flat)
![License](https://img.shields.io/badge/license-Apache%202.0-blue?style=flat)
![Claude Code](https://img.shields.io/badge/Claude%20Code-plugin-blueviolet?style=flat)
![Cursor](https://img.shields.io/badge/Cursor-hooks-000000?style=flat)
![Codex](https://img.shields.io/badge/Codex-hooks-412991?style=flat)
![OpenCode](https://img.shields.io/badge/OpenCode-plugin-6D28D9?style=flat)
![Build](https://img.shields.io/badge/build-passing-brightgreen?style=flat)

`gtk-ai` is a two-part integration:

- the `gtkai` Go binary, which rewrites shell commands before they run and filters their output
- per-agent hooks, which register `PreToolUse` / `PostToolUse` (or the agent equivalent) and invoke `gtkai`

Registered shell commands are rewritten to `gtkai <cmd>` before execution. gtkai runs the real binary, injects compact flags when needed, and filters stdout. On Claude Code and OpenCode, `Read` and MCP still go through post-tool hooks.

```text
Agent → Shell("git status")
              ↓ PreToolUse → gtkai hook-pre --agent=<agent>
         command becomes gtkai git status
              ↓ gtkai runs git status --porcelain -b
         compact grouped status
              ↓
         Agent receives filtered output
```

## Benchmark

Numbers from `go test ./internal/hook/... -v`. Token estimate: ~4 chars/token.

| Input | Tokens before | Tokens after | Savings |
|---|---:|---:|---:|
| `find`: 150 paths | 1,050 | 374 | **64%** |
| `ls`: 70 entries | 262 | 65 | **75%** |
| `grep`: 250 matches across 20 files | 3,820 | 3,360 | **12%** |
| `git diff`: 400 lines | 3,185 | 813 | **74%** |
| `git log`: 80 commits | 1,917 | 244 | **87%** |
| `Read`: Go file, 100 commented vars | 1,346 | 348 | **74%** |
| `Read`: plain text, 400 lines | 2,772 | 1,380 | **50%** |
| MCP tool response — 5,200 chars | 1,300 | 758 | **42%** |

Savings grow with output size. Small outputs may not be reduced.

## Installation

gtk-ai requires the `gtkai` binary plus one integration per coding agent. The installer default is `--agent=auto`: it configures every compatible agent it finds on the machine.

### Option A: install script

```bash
curl -sSL https://raw.githubusercontent.com/jmeiracorbal/gtk-ai/main/install.sh | sh
```

Explicit targets:

```bash
curl -sSL https://raw.githubusercontent.com/jmeiracorbal/gtk-ai/main/install.sh | sh -s -- --agent=cursor
curl -sSL https://raw.githubusercontent.com/jmeiracorbal/gtk-ai/main/install.sh | sh -s -- --agent=codex
curl -sSL https://raw.githubusercontent.com/jmeiracorbal/gtk-ai/main/install.sh | sh -s -- --agent=opencode
curl -sSL https://raw.githubusercontent.com/jmeiracorbal/gtk-ai/main/install.sh | sh -s -- --agent=all
```

Claude Code still needs the plugin after the marketplace is registered:

```bash
claude plugin install -s user gtk-ai@gtk-ai
```

The install script also runs `gtkai filter install-marketplace` for the entries listed in `marketplace.json` (currently `gtk-ai/date`).

Then restart the agent.

### Option B: build from source

Requires Go 1.22+.

```bash
git clone https://github.com/jmeiracorbal/gtk-ai
cd gtk-ai
go build -o ~/.local/bin/gtkai ./cmd/gtkai/
```

Then configure agents without reinstalling the binary:

```bash
GTKAI_SKIP_BINARY=1 sh install.sh -- --agent=all
```

For Claude Code only, `GTKAI_CLAUDE_ONLY=1 sh install.sh` still works and implies `--agent=claudecode`.

### Agent surfaces

| | Claude Code | Cursor | Codex | OpenCode |
|---|---|---|---|---|
| **Hooks** | plugin `PreToolUse` / `PostToolUse` | `~/.cursor/hooks/` | `~/.codex/hooks/` | `~/.config/opencode/plugins/gtkai.ts` |
| **Hook config** | plugin `hooks.json` | `~/.cursor/hooks.json` | `~/.codex/hooks.json` | global plugin |
| **Shell rewrite** | matcher `Bash` | matcher `Shell` | matcher `Bash` and shell aliases | `tool.execute.before` |
| **Read / MCP post** | yes | MCP only | shell rewrite only | `read` and MCP tools |

## Modules

Each module handles one command. All built-in modules ship with the binary.

| Module | Command | What it does |
|---|---|---|
| `find` | `find` | Groups paths by directory, caps shown files, extension summary |
| `ls` | `ls` | Injects `-l` + `LC_ALL=C` (and `-a` only if the user asked); compact to counts and samples; skips noise dirs |
| `git` | `git` | status/log/diff/branch/show; write subcommands compact on exit 0; push/pull/fetch inject `-q` |
| `grep` / `rg` | `grep`, `rg` | Shared grouping by file with per-file and total caps; `grep` injects `-nH` |
| `cat` / `head` / `tail` | same | Reuses `read.FilterContent` on single-file output |
| `tree` | `tree` | Entry count + capped listing |
| `gain` | — | SQLite analytics: recorded on each proxy run |

Command filters ship as standalone repos (for example `github.com/gtk-ai/date`). `install.sh` installs every entry in `marketplace.json`. Built-in modules remain as fallback until migrated.

### External filter commands

```bash
gtkai filter install github.com/gtk-ai/date@v0.12.0
gtkai filter install github.com/gtk-ai/date@v0.12.0 --replace
gtkai filter install-marketplace marketplace.json --core-version=0.11.0-beta.4
gtkai filter list
gtkai filter uninstall gtk-ai/date
```

| Command | Description |
|---|---|
| `filter install <module@version>` | Download, validate `gtkai.json`, build, register in `~/.gtk-ai/plugins.db` |
| `filter install … --replace` | Required when another filter is already **active** for the same shell command |
| `filter install-marketplace <file>` | Install every entry in `marketplace.json` (`install.sh` uses this; `--replace` is implicit) |
| `filter list` | List installed filters; marks the active one per command |
| `filter uninstall <id>` | Remove by full id (e.g. `gtk-ai/date`); deletes `~/.gtk-ai/filters/<id>/` |

**Conflict policy:** if filter `acme/date` is active for `date`, installing `gtk-ai/date` aborts unless you pass `--replace`. With `--replace`, the new filter becomes active; the previous one stays installed but inactive (not deleted). To remove it: `filter uninstall acme/date`. To switch back: reinstall the other filter with `--replace`.

Same filter id, new version: upgrade without `--replace`. Uninstalling the active filter promotes the most recently installed survivor for that command, or falls back to the built-in module when one exists.

### subprocess/v1

External filters use contract **`subprocess/v1`**: gtk-ai runs the filter as an external program and exchanges JSON on stdin/stdout (rewrite + filter_output). Any language works if the binary implements the protocol. See [ARCHITECTURE.md](ARCHITECTURE.md) and the reference module [gtk-ai/date](https://github.com/gtk-ai/date).

## Adding a module

1. Create `plugins/mycommand/mycommand.go`
2. Implement the `Module` interface
3. Register at `init()` time
4. Import in `cmd/gtkai/main.go`

```go
package mycommand

import "github.com/jmeiracorbal/gtk-ai/internal/registry"

func init() { registry.Register(&Module{}) }

type Module struct{}

func (m *Module) Name() string { return "mycommand" }

func (m *Module) Rewrite(args []string) ([]string, bool) { return nil, false }

func (m *Module) FilterOutput(args []string, output string, exitCode int) string { return output }

func (m *Module) TokensBefore(output string) int { return registry.EstimateTokens(output) }

func (m *Module) TokensAfter(filtered string) int { return registry.EstimateTokens(filtered) }
```

```go
_ "github.com/jmeiracorbal/gtk-ai/plugins/mycommand"
```

No other changes needed.

## MCP passthrough

By default, gtkai truncates all `mcp__*` tool responses above 3,000 chars. To exempt specific tools, set `GTK_MCP_PASSTHROUGH_PATTERNS` in your shell config:

```sh
export GTK_MCP_PASSTHROUGH_PATTERNS="my_tool_*,other_tool"
```

Pattern syntax: exact name or glob prefix (`prefix_*`).

To identify which tools to exempt, check the tool names returned by your MCP servers. Any tool whose output should reach the agent unmodified, such as structured symbol data or memory results, should be listed here.

## Commands

```text
gtkai hook-pre --agent=<agent>   PreToolUse handler — rewrites shell commands to gtkai
gtkai hook-post --agent=<agent>  PostToolUse handler — filters Read and MCP
gtkai json-merge <file>          Deep-merge JSON from stdin into an agent config file
gtkai git status                 Proxy: run a registered command through gtkai
gtkai mcp-scan                   List MCP server tools, suggest passthrough prefixes
gtkai gain                       Token savings analytics
gtkai filter install <mod@ver> [--replace]  Install an external filter module
gtkai filter install-marketplace <file> --core-version=<ver>  Install marketplace entries
gtkai filter uninstall <id>      Remove an installed filter by full id
gtkai filter list                List installed filters (active marked)
gtkai version                    Print version
```

## Architecture

```text
gtk-ai/
├── cmd/gtkai/              # binary entry point
├── internal/
│   ├── registry/           # Module interface + Register() + EstimateTokens()
│   ├── matchgroup/         # shared grep/rg grouping
│   ├── shell/              # shell command rewrite
│   ├── proxy/              # execute + filter + gain
│   ├── text/               # ANSI strip
│   ├── jsonmerge/          # installer config merge
│   └── hook/               # PreToolUse and PostToolUse handlers (per-agent JSON)
├── plugins/
│   ├── find/               # find output filter
│   ├── ls/                 # ls output filter
│   ├── git/                # git subcommand filters
│   ├── grep/               # grep output filter
│   ├── readcmd/            # cat/head/tail via read.FilterContent
│   ├── tree/               # tree output filter
│   └── gain/               # SQLite token savings analytics
├── plugin/                 # Claude Code plugin (hooks + scripts)
└── scripts/
    ├── cursor/             # Cursor hook scripts + rule
    ├── codex/              # Codex hook scripts
    └── opencode/           # OpenCode plugin
```

The `registry` package is the only shared dependency between modules. Modules never import each other.

## Works well with

**[hybrid-coco](https://github.com/jmeiracorbal/hybrid-coco)**: local code intelligence for Claude Code. While gtk-ai filters tool output, hybrid-coco replaces file reads and grep with index queries for code navigation. Both operate independently through separate hooks and complement each other.

## License

Apache 2.0, see [LICENSE](LICENSE). Attribution required on redistribution.
