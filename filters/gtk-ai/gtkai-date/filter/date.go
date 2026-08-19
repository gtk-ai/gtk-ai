// Package filter implements the gtk-ai/gtkai-date filter logic.
//
// Contract:
//   - id:      gtk-ai/gtkai-date
//   - filters: date
//
// Rewrite: when `date` is called without a format argument (+%...),
// injects +%Y-%m-%dT%H:%M:%SZ so the output is a compact ISO-8601
// timestamp instead of the verbose locale-specific string.
// If the caller already supplied a format, no rewrite is performed.
//
// FilterOutput: passes output through unchanged — the rewrite already
// constrains the format to a single compact line.
package filter

import "strings"

const (
	// ID is the full filter identity following the author/gtkai-<command> rule.
	ID = "gtk-ai/gtkai-date"

	// isoFmt is the injected format when none is provided by the caller.
	isoFmt = "+%Y-%m-%dT%H:%M:%SZ"
)

// Rewrite returns the rewritten args and true when no format is present.
// Returns nil, false when a format argument (+%...) is already in args.
func Rewrite(args []string) ([]string, bool) {
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

// FilterOutput trims trailing newlines.
// The rewrite already constrains the format to a single compact line,
// so no further transformation is needed.
func FilterOutput(_ []string, output string, _ int) string {
	return strings.TrimRight(output, "\n") + "\n"
}
