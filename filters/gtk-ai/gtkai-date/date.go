// Package gtkai_date filters `date` output for coding agents.
//
// Contract:
//   - id:      gtk-ai/gtkai-date
//   - filters: date
//
// Rewrite: when the agent calls `date` without a format argument (+%...),
// this filter injects `+%Y-%m-%dT%H:%M:%SZ` so the output is a compact
// ISO-8601 timestamp instead of the verbose locale-specific string.
// If the user already supplied a format, no rewrite is performed.
//
// FilterOutput: passes the output through unchanged — the rewrite already
// constrains the format; no post-processing is needed.
package gtkai_date

import (
	"strings"

	"github.com/jmeiracorbal/gtk-ai/internal/registry"
)

const (
	// ID is the full filter identity following the author/gtkai-<command> rule.
	ID = "gtk-ai/gtkai-date"

	// isoFmt is the injected format when none is provided.
	isoFmt = "+%Y-%m-%dT%H:%M:%SZ"
)

func init() {
	registry.Register(&Module{})
}

// Module implements registry.Module for the `date` command.
type Module struct{}

func (m *Module) Name() string { return "date" }

// Rewrite injects a compact ISO-8601 format when the caller did not provide
// a format argument. Returns no change when a format is already present.
func (m *Module) Rewrite(args []string) ([]string, bool) {
	for _, a := range args {
		if strings.HasPrefix(a, "+") {
			return nil, false
		}
	}
	out := make([]string, len(args)+1)
	copy(out, args)
	out[len(args)] = isoFmt
	return out, true
}

// FilterOutput passes the output through; the rewrite already constrains
// the format to a single compact line.
func (m *Module) FilterOutput(args []string, output string, exitCode int) string {
	return strings.TrimRight(output, "\n") + "\n"
}

func (m *Module) TokensBefore(output string) int {
	return registry.EstimateTokens(output)
}

func (m *Module) TokensAfter(filtered string) int {
	return registry.EstimateTokens(filtered)
}
