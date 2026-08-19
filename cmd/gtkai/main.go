// gtk-ai — PreToolUse proxy and PostToolUse filter for coding agents.
// Binary: gtkai
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/jmeiracorbal/gtk-ai/internal/filterregistry"
	"github.com/jmeiracorbal/gtk-ai/internal/hook"
	"github.com/jmeiracorbal/gtk-ai/internal/jsonmerge"
	"github.com/jmeiracorbal/gtk-ai/internal/proxy"
	"github.com/jmeiracorbal/gtk-ai/internal/registry"
	"github.com/jmeiracorbal/gtk-ai/modules/gain"
	"github.com/jmeiracorbal/gtk-ai/modules/mcpscan"

	_ "github.com/jmeiracorbal/gtk-ai/modules/cargo"
	_ "github.com/jmeiracorbal/gtk-ai/modules/docker"
	_ "github.com/jmeiracorbal/gtk-ai/modules/find"
	_ "github.com/jmeiracorbal/gtk-ai/modules/git"
	_ "github.com/jmeiracorbal/gtk-ai/modules/go"
	_ "github.com/jmeiracorbal/gtk-ai/modules/grep"
	_ "github.com/jmeiracorbal/gtk-ai/modules/ls"
	_ "github.com/jmeiracorbal/gtk-ai/modules/npmtest"
	_ "github.com/jmeiracorbal/gtk-ai/modules/pytest"
	_ "github.com/jmeiracorbal/gtk-ai/modules/python"
	_ "github.com/jmeiracorbal/gtk-ai/modules/readcmd"
	_ "github.com/jmeiracorbal/gtk-ai/modules/rg"
	_ "github.com/jmeiracorbal/gtk-ai/modules/tree"
)

const version = "0.11.0"

func usage() {
	fmt.Fprintf(os.Stderr, `gtkai %s

Usage:
  gtkai hook-pre --agent=<agent>   PreToolUse hook — rewrites shell commands to gtkai
  gtkai hook-post --agent=<agent>  PostToolUse hook — reads stdin, writes filtered output
  gtkai json-merge <file>          Deep-merge JSON from stdin into <file>
  gtkai <module> [args...]         Run a registered command through the proxy
  gtkai mcp-scan                   List tools from all MCP servers, suggest passthrough prefixes
  gtkai gain                       Show token savings analytics
  gtkai filter install <mod@ver> [--replace]  Install an external filter module (go dependency)
  gtkai filter install-official <file> --core-version=<ver>  Install filters from official.json
  gtkai filter uninstall <id>      Remove an installed filter by full id
  gtkai filter list                List installed external filters (active marked)
  gtkai version                    Print version

Agents:
  claudecode, cursor, codex, opencode

Environment:
  GTK_MCP_PASSTHROUGH_PATTERNS  Comma-separated MCP tool patterns to skip filtering
                                 Example: hc_*,my_tool
`, version)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Printf("gtkai %s\n", version)

	case "hook-pre":
		agent, err := parseAgentFlag(os.Args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "gtkai hook-pre: %v\n", err)
			os.Exit(1)
		}
		bin, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "gtkai hook-pre: %v\n", err)
			os.Exit(1)
		}
		_, err = hook.RunPre(os.Stdin, os.Stdout, bin, agent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gtkai hook-pre: %v\n", err)
			os.Exit(1)
		}

	case "hook-post":
		agent, err := parseAgentFlag(os.Args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "gtkai hook-post: %v\n", err)
			os.Exit(1)
		}
		_, err = hook.Run(os.Stdin, os.Stdout, agent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gtkai hook-post: %v\n", err)
			os.Exit(1)
		}

	case "json-merge":
		if len(os.Args) != 3 || os.Args[2] == "" {
			fmt.Fprintln(os.Stderr, "usage: gtkai json-merge <file>")
			os.Exit(1)
		}
		changed, err := jsonmerge.MergeFile(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "gtkai json-merge: %v\n", err)
			os.Exit(1)
		}
		if changed {
			fmt.Println("updated")
		} else {
			fmt.Println("unchanged")
		}

	case "mcp-scan":
		if err := mcpscan.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "gtkai mcp-scan: %v\n", err)
			os.Exit(1)
		}

	case "gain":
		t, err := gain.Open()
		if err != nil {
			fmt.Fprintf(os.Stderr, "gtkai: cannot open gain db: %v\n", err)
			os.Exit(1)
		}
		defer t.Close()
		if err := gain.PrintSummary(t); err != nil {
			fmt.Fprintf(os.Stderr, "gtkai: %v\n", err)
			os.Exit(1)
		}

	case "filter":
		runFilter(os.Args[2:])

	default:
		if registry.Get(os.Args[1]) == nil && !filterregistry.HasActive(os.Args[1]) {
			fmt.Fprintf(os.Stderr, "gtkai: unknown command %q\n\n", os.Args[1])
			usage()
			os.Exit(1)
		}
		os.Exit(proxy.Run(os.Args[1], os.Args[2:]))
	}
}

func parseAgentFlag(args []string) (hook.Agent, error) {
	var agent hook.Agent
	seen := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--agent":
			if i+1 >= len(args) {
				return "", fmt.Errorf("--agent requires a value")
			}
			parsed, err := hook.ParseAgent(args[i+1])
			if err != nil {
				return "", err
			}
			agent = parsed
			seen = true
			i++
		case strings.HasPrefix(a, "--agent="):
			parsed, err := hook.ParseAgent(strings.TrimPrefix(a, "--agent="))
			if err != nil {
				return "", err
			}
			agent = parsed
			seen = true
		default:
			return "", fmt.Errorf("unknown argument %q", a)
		}
	}
	if !seen {
		return "", fmt.Errorf("--agent is required")
	}
	return agent, nil
}
