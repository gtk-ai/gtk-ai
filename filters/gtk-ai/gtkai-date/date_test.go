package gtkai_date_test

import (
	"strings"
	"testing"

	gtkai_date "github.com/jmeiracorbal/gtk-ai/filters/gtk-ai/gtkai-date"
)

func module() *gtkai_date.Module { return &gtkai_date.Module{} }

// --- Rewrite ---

func TestRewriteNoArgs(t *testing.T) {
	m := module()
	got, ok := m.Rewrite(nil)
	if !ok {
		t.Fatal("expected rewrite when no format supplied")
	}
	if len(got) != 1 || !strings.HasPrefix(got[0], "+%") {
		t.Fatalf("unexpected rewritten args: %v", got)
	}
}

func TestRewriteWithFormatPassthrough(t *testing.T) {
	m := module()
	args := []string{"+%d/%m/%Y"}
	_, ok := m.Rewrite(args)
	if ok {
		t.Fatal("must not rewrite when format already present")
	}
}

func TestRewriteWithUtcFlag(t *testing.T) {
	m := module()
	// -u without a format: should still inject the ISO format.
	got, ok := m.Rewrite([]string{"-u"})
	if !ok {
		t.Fatal("expected rewrite with -u and no format")
	}
	hasISO := false
	for _, a := range got {
		if strings.HasPrefix(a, "+%") {
			hasISO = true
		}
	}
	if !hasISO {
		t.Fatalf("ISO format not injected, got: %v", got)
	}
}

func TestRewriteWithFormatAndFlag(t *testing.T) {
	m := module()
	// -u + explicit format: no rewrite.
	_, ok := m.Rewrite([]string{"-u", "+%s"})
	if ok {
		t.Fatal("must not rewrite when format already present alongside flags")
	}
}

// --- FilterOutput ---

func TestFilterOutputTrimsNewline(t *testing.T) {
	m := module()
	out := m.FilterOutput(nil, "2026-08-19T14:30:00Z\n\n", 0)
	if out != "2026-08-19T14:30:00Z\n" {
		t.Fatalf("unexpected: %q", out)
	}
}

func TestFilterOutputNonZeroExitPassthrough(t *testing.T) {
	m := module()
	out := m.FilterOutput(nil, "date: invalid option\n", 1)
	if out != "date: invalid option\n" {
		t.Fatalf("unexpected: %q", out)
	}
}

// --- ID ---

func TestID(t *testing.T) {
	if gtkai_date.ID != "gtk-ai/gtkai-date" {
		t.Fatalf("ID %q does not follow author/gtkai-<command> rule", gtkai_date.ID)
	}
}

// --- Name ---

func TestName(t *testing.T) {
	if module().Name() != "date" {
		t.Fatalf("Name() must return the shell command intercepted")
	}
}
