// Module git: filters `git` subcommand output for Claude Code.
// Handles: diff, log, status, branch.
package git

import (
	"fmt"
	"strings"

	"github.com/jmeiracorbal/gtk-ai/internal/registry"
)

const (
	maxDiffLines   = 300
	maxLogEntries  = 50
	maxStatusLines = 100
)

func init() {
	registry.Register(&Module{})
}

type Module struct{}

func (m *Module) Name() string { return "git" }

func (m *Module) Rewrite(args []string) ([]string, bool) {
	globals, sub, rest, ok := splitGitArgs(args)
	if !ok || sub != "status" {
		return nil, false
	}
	if !usesCompactStatusPath(rest) {
		return nil, false
	}
	out := append(append([]string{}, globals...), "status", "--porcelain", "-b")
	return out, true
}

func (m *Module) FilterOutput(args []string, output string) string {
	_, sub, _, ok := splitGitArgs(args)
	if !ok {
		return output
	}
	return FilterOutputWithArgs(sub, output)
}

// FilterOutputWithArgs filters git output based on the subcommand.
func FilterOutputWithArgs(subcommand, output string) string {
	switch subcommand {
	case "diff":
		return filterDiff(output)
	case "log":
		return filterLog(output)
	case "status":
		return filterStatus(output)
	case "branch":
		return filterBranch(output)
	default:
		return output
	}
}

func filterDiff(output string) string {
	lines := strings.Split(output, "\n")
	if len(lines) <= maxDiffLines {
		return output
	}
	var sb strings.Builder
	for _, l := range lines[:maxDiffLines] {
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("... +%d lines truncated (use git diff <file> for specific files)\n", len(lines)-maxDiffLines))
	return sb.String()
}

func filterLog(output string) string {
	// Each log entry starts with "commit "
	entries := splitLogEntries(output)
	if len(entries) <= maxLogEntries {
		return output
	}
	var sb strings.Builder
	for _, e := range entries[:maxLogEntries] {
		sb.WriteString(e)
	}
	sb.WriteString(fmt.Sprintf("... +%d commits truncated\n", len(entries)-maxLogEntries))
	return sb.String()
}

func filterStatus(output string) string {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return "nothing to commit, working tree clean\n"
	}

	if !isPorcelainStatus(lines) {
		if len(lines) > maxStatusLines {
			return strings.Join(lines[:maxStatusLines], "\n") + fmt.Sprintf("\n... +%d lines\n", len(lines)-maxStatusLines)
		}
		return output
	}

	var branch string
	var modified, untracked, staged []string
	for _, l := range lines {
		if strings.HasPrefix(l, "##") {
			branch = strings.TrimSpace(strings.TrimPrefix(l, "##"))
			continue
		}
		if len(l) < 2 {
			continue
		}
		xy := l[:2]
		path := ""
		if len(l) > 2 {
			path = strings.TrimSpace(l[2:])
		}
		switch {
		case xy == "??":
			untracked = append(untracked, path)
		case xy[0] != ' ' && xy[0] != '?':
			staged = append(staged, path)
			if xy[1] != ' ' && xy[1] != '?' {
				modified = append(modified, path)
			}
		case xy[1] != ' ' && xy[1] != '?':
			modified = append(modified, path)
		}
	}

	var sb strings.Builder
	if branch != "" {
		sb.WriteString("* ")
		sb.WriteString(branch)
		sb.WriteString("\n")
	}
	if len(staged) > 0 {
		sb.WriteString(fmt.Sprintf("Staged (%d): %s\n", len(staged), strings.Join(staged, ", ")))
	}
	if len(modified) > 0 {
		sb.WriteString(fmt.Sprintf("Modified (%d): %s\n", len(modified), strings.Join(modified, ", ")))
	}
	if len(untracked) > 0 {
		if len(untracked) > 10 {
			sb.WriteString(fmt.Sprintf("Untracked (%d): %s ... +%d more\n", len(untracked), strings.Join(untracked[:10], ", "), len(untracked)-10))
		} else {
			sb.WriteString(fmt.Sprintf("Untracked (%d): %s\n", len(untracked), strings.Join(untracked, ", ")))
		}
	}
	if len(staged)+len(modified)+len(untracked) == 0 {
		if branch != "" {
			sb.WriteString("clean — nothing to commit\n")
			return sb.String()
		}
		return output
	}
	return sb.String()
}

func isPorcelainStatus(lines []string) bool {
	sawEntry := false
	for _, l := range lines {
		if l == "" {
			continue
		}
		if strings.HasPrefix(l, "##") {
			continue
		}
		if len(l) < 2 || !isPorcelainCode(l[0]) || !isPorcelainCode(l[1]) {
			return false
		}
		sawEntry = true
	}
	return sawEntry || hasBranchHeader(lines)
}

func hasBranchHeader(lines []string) bool {
	for _, l := range lines {
		if strings.HasPrefix(l, "##") {
			return true
		}
	}
	return false
}

func isPorcelainCode(c byte) bool {
	switch c {
	case ' ', 'M', 'A', 'D', 'R', 'C', 'U', 'T', '?', '!':
		return true
	default:
		return false
	}
}

func splitGitArgs(args []string) (globals []string, sub string, rest []string, ok bool) {
	i := 0
	for i < len(args) {
		a := args[i]
		switch {
		case a == "-C" || a == "-c" || a == "--git-dir" || a == "--work-tree":
			if i+1 >= len(args) {
				return nil, "", nil, false
			}
			globals = append(globals, a, args[i+1])
			i += 2
		case strings.HasPrefix(a, "--git-dir=") || strings.HasPrefix(a, "--work-tree="):
			globals = append(globals, a)
			i++
		case a == "--no-pager" || a == "--no-optional-locks" || a == "--bare" || a == "--literal-pathspecs":
			globals = append(globals, a)
			i++
		default:
			if strings.HasPrefix(a, "-") {
				return nil, "", nil, false
			}
			return globals, a, args[i+1:], true
		}
	}
	return globals, "", nil, false
}

func usesCompactStatusPath(args []string) bool {
	if len(args) == 0 {
		return true
	}
	sawBranch := false
	for _, a := range args {
		switch a {
		case "-b", "--branch":
			sawBranch = true
		case "-sb", "-bs":
			return true
		case "-s", "--short":
		default:
			return false
		}
	}
	return sawBranch
}

func filterBranch(output string) string {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	var branches []string
	var current string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if strings.HasPrefix(l, "* ") {
			current = strings.TrimPrefix(l, "* ")
		} else {
			// Skip remote-tracking branches to reduce noise
			if !strings.Contains(l, "->") {
				branches = append(branches, l)
			}
		}
	}
	var sb strings.Builder
	if current != "" {
		sb.WriteString(fmt.Sprintf("current: %s\n", current))
	}
	if len(branches) > 0 {
		sb.WriteString(fmt.Sprintf("local (%d): %s\n", len(branches), strings.Join(branches, ", ")))
	}
	return sb.String()
}

func splitLogEntries(output string) []string {
	var entries []string
	var current strings.Builder
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "commit ") && current.Len() > 0 {
			entries = append(entries, current.String())
			current.Reset()
		}
		current.WriteString(line)
		current.WriteString("\n")
	}
	if current.Len() > 0 {
		entries = append(entries, current.String())
	}
	return entries
}

func (m *Module) TokensBefore(output string) int {
	return registry.EstimateTokens(output)
}

func (m *Module) TokensAfter(filtered string) int {
	return registry.EstimateTokens(filtered)
}
