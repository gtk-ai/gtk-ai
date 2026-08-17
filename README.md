# gtk-ai

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go&logoColor=white)
![Version](https://img.shields.io/badge/version-0.4.0-blue?style=flat)
![License](https://img.shields.io/badge/license-Apache%202.0-blue?style=flat)
![Claude Code](https://img.shields.io/badge/Claude%20Code-plugin%20compatible-blueviolet?style=flat)
![Build](https://img.shields.io/badge/build-passing-brightgreen?style=flat)

`gtk-ai` is a two-part integration for Claude Code:

- the `gtkai` Go binary, which rewrites Bash commands before they run and filters their output
- the Claude plugin, which registers `PreToolUse` and `PostToolUse` hooks and invokes `gtkai`

Registered Bash commands are rewritten to `gtkai <cmd>` before execution. gtkai runs the real binary, injects compact flags when needed, and filters stdout. `Read` and MCP still go through `PostToolUse`.

```text
Claude → Bash("git status")
              ↓ PreToolUse → gtkai hook-pre
         command becomes gtkai git status
              ↓ gtkai runs git status --porcelain -b
         compact grouped status
              ↓
         Claude receives filtered output
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

gtk-ai requires both parts:

1. the `gtkai` Go binary
2. the Claude plugin that registers the hook and calls the binary

### Option A: install script

```bash
curl -sSL https://raw.githubusercontent.com/jmeiracorbal/gtk-ai/main/install.sh | sh
```

The script installs the binary, registers the marketplace, and prepares the Claude Code integration. When it finishes, run:

```bash
claude plugin install -s user gtk-ai@gtk-ai
```

Then restart Claude Code.

### Option B: build from source

Requires Go 1.22+.

```bash
git clone https://github.com/jmeiracorbal/gtk-ai
cd gtk-ai
go build -o ~/.local/bin/gtkai ./cmd/gtkai/
```

Then configure the Claude Code side without reinstalling the binary:

```bash
GTKAI_CLAUDE_ONLY=1 sh install.sh
```

When it finishes, install the plugin:

```bash
claude plugin install -s user gtk-ai@gtk-ai
```

Restart Claude Code when done.

## Modules

Each module handles one command. All built-in modules ship with the binary.

| Module | Command | What it does |
|---|---|---|
| `find` | `find` | Groups paths by directory, caps shown files, extension summary |
| `ls` | `ls` | Injects `-l` + `LC_ALL=C` (and `-a` only if the user asked); compact to counts and samples; skips noise dirs |
| `git` | `git diff/log/status/branch` | `status` injects `--porcelain -b`; `log` injects pretty-format and `-10`; `diff` strips headers and caps hunks |
| `grep` / `rg` | `grep`, `rg` | Shared grouping by file with per-file and total caps; `grep` injects `-nH` |
| `gain` | — | SQLite analytics: recorded on each proxy run |

## Adding a module

1. Create `modules/mycommand/mycommand.go`
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

func (m *Module) FilterOutput(args []string, output string) string { return output }

func (m *Module) TokensBefore(output string) int { return registry.EstimateTokens(output) }

func (m *Module) TokensAfter(filtered string) int { return registry.EstimateTokens(filtered) }
```

```go
_ "github.com/jmeiracorbal/gtk-ai/modules/mycommand"
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
gtkai hook-pre      PreToolUse handler — rewrites Bash commands to gtkai
gtkai hook-post     PostToolUse handler — reads stdin JSON, writes filtered output
gtkai git status    Proxy: run a registered command through gtkai (`ls`, `find`, `grep`, `rg`, `git`)
gtkai mcp-scan      List MCP server tools, suggest passthrough prefixes
gtkai gain          Token savings analytics
gtkai version       Print version
```

## Architecture

```text
gtk-ai/
├── cmd/gtkai/              # binary entry point
├── internal/
│   ├── registry/           # Module interface + Register() + EstimateTokens()
│   ├── matchgroup/         # shared grep/rg grouping
│   ├── shell/              # Bash command rewrite
│   ├── proxy/              # execute + filter + gain
│   ├── text/               # ANSI strip
│   └── hook/               # PreToolUse and PostToolUse handlers
├── modules/
│   ├── find/               # find output filter
│   ├── ls/                 # ls output filter
│   ├── git/                # git subcommand filters
│   ├── grep/               # grep output filter
│   └── gain/               # SQLite token savings analytics
└── plugin/
    ├── hooks/              # Claude plugin hook registration
    └── scripts/            # scripts that invoke gtkai from Claude's hook system
```

The `registry` package is the only shared dependency between modules. Modules never import each other.

## Works well with

**[hybrid-coco](https://github.com/jmeiracorbal/hybrid-coco)**: local code intelligence for Claude Code. While gtk-ai filters tool output, hybrid-coco replaces file reads and grep with index queries for code navigation. Both operate independently through separate hooks and complement each other.

## License

Apache 2.0, see [LICENSE](LICENSE). Attribution required on redistribution.
