// gtk-ai — PreToolUse proxy and PostToolUse filter for Claude Code.
// Binary: gtkai
package main

import (
	"fmt"
	"os"

	"github.com/jmeiracorbal/gtk-ai/internal/hook"
	"github.com/jmeiracorbal/gtk-ai/internal/proxy"
	"github.com/jmeiracorbal/gtk-ai/internal/registry"
	"github.com/jmeiracorbal/gtk-ai/modules/gain"
	"github.com/jmeiracorbal/gtk-ai/modules/mcpscan"

	_ "github.com/jmeiracorbal/gtk-ai/modules/cargo"
	_ "github.com/jmeiracorbal/gtk-ai/modules/find"
	_ "github.com/jmeiracorbal/gtk-ai/modules/go"
	_ "github.com/jmeiracorbal/gtk-ai/modules/git"
	_ "github.com/jmeiracorbal/gtk-ai/modules/grep"
	_ "github.com/jmeiracorbal/gtk-ai/modules/ls"
	_ "github.com/jmeiracorbal/gtk-ai/modules/readcmd"
	_ "github.com/jmeiracorbal/gtk-ai/modules/rg"
	_ "github.com/jmeiracorbal/gtk-ai/modules/tree"
)

const version = "0.7.0"

func usage() {
	fmt.Fprintf(os.Stderr, `gtkai %s

Usage:
  gtkai hook-pre             PreToolUse hook — rewrites Bash commands to gtkai
  gtkai hook-post            PostToolUse hook — reads stdin, writes filtered output
  gtkai <module> [args...]   Run a registered command through the proxy
  gtkai mcp-scan             List tools from all MCP servers, suggest passthrough prefixes
  gtkai gain                 Show token savings analytics
  gtkai version              Print version

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
		bin, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "gtkai hook-pre: %v\n", err)
			os.Exit(1)
		}
		_, err = hook.RunPre(os.Stdin, os.Stdout, bin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gtkai hook-pre: %v\n", err)
			os.Exit(1)
		}

	case "hook-post":
		_, err := hook.Run(os.Stdin, os.Stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gtkai hook-post: %v\n", err)
			os.Exit(1)
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

	default:
		if registry.Get(os.Args[1]) == nil {
			fmt.Fprintf(os.Stderr, "gtkai: unknown command %q\n\n", os.Args[1])
			usage()
			os.Exit(1)
		}
		os.Exit(proxy.Run(os.Args[1], os.Args[2:]))
	}
}
