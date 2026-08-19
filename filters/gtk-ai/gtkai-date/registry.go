// Package gtkai_date wires the filter logic into the gtk-ai module registry.
// This file is only compiled as part of the gtk-ai monorepo — it is not
// present in the standalone gtk-ai/gtkai-date repository.
package gtkai_date

import (
	"github.com/jmeiracorbal/gtk-ai/filters/gtk-ai/gtkai-date/filter"
	"github.com/jmeiracorbal/gtk-ai/internal/registry"
)

// ID re-exports the filter identity so callers that import this package
// directly (e.g. tests) can read it without importing the sub-package.
const ID = filter.ID

func init() {
	registry.Register(&module{})
}

type module struct{}

func (m *module) Name() string { return "date" }

func (m *module) Rewrite(args []string) ([]string, bool) {
	return filter.Rewrite(args)
}

func (m *module) FilterOutput(args []string, output string, exitCode int) string {
	return filter.FilterOutput(args, output, exitCode)
}

func (m *module) TokensBefore(output string) int {
	return registry.EstimateTokens(output)
}

func (m *module) TokensAfter(filtered string) int {
	return registry.EstimateTokens(filtered)
}
