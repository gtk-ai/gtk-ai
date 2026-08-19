package gtkai_date_test

import (
	"strings"
	"testing"

	gtkai_date "github.com/jmeiracorbal/gtk-ai/filters/gtk-ai/gtkai-date"
	"github.com/jmeiracorbal/gtk-ai/filters/gtk-ai/gtkai-date/filter"
	"github.com/jmeiracorbal/gtk-ai/internal/filtermanifest"
)

// --- Rewrite ---

func TestRewriteNoArgs(t *testing.T) {
	got, ok := filter.Rewrite(nil)
	if !ok {
		t.Fatal("expected rewrite when no format supplied")
	}
	if len(got) != 1 || !strings.HasPrefix(got[0], "+%") {
		t.Fatalf("unexpected rewritten args: %v", got)
	}
}

func TestRewriteWithFormatPassthrough(t *testing.T) {
	_, ok := filter.Rewrite([]string{"+%d/%m/%Y"})
	if ok {
		t.Fatal("must not rewrite when format already present")
	}
}

func TestRewriteWithUtcFlag(t *testing.T) {
	got, ok := filter.Rewrite([]string{"-u"})
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
	_, ok := filter.Rewrite([]string{"-u", "+%s"})
	if ok {
		t.Fatal("must not rewrite when format already present alongside flags")
	}
}

// --- FilterOutput ---

func TestFilterOutputTrimsNewline(t *testing.T) {
	out := filter.FilterOutput(nil, "2026-08-19T14:30:00Z\n\n", 0)
	if out != "2026-08-19T14:30:00Z\n" {
		t.Fatalf("unexpected: %q", out)
	}
}

func TestFilterOutputNonZeroExitPassthrough(t *testing.T) {
	out := filter.FilterOutput(nil, "date: invalid option\n", 1)
	if out != "date: invalid option\n" {
		t.Fatalf("unexpected: %q", out)
	}
}

// --- ID ---

func TestID(t *testing.T) {
	if gtkai_date.ID != "gtk-ai/gtkai-date" {
		t.Fatalf("ID %q does not follow author/gtkai-<command> rule", gtkai_date.ID)
	}
	if filter.ID != gtkai_date.ID {
		t.Fatalf("filter.ID %q != package ID %q", filter.ID, gtkai_date.ID)
	}
}

// --- filter.json manifest ---

func TestFilterManifest(t *testing.T) {
	m, err := filtermanifest.ParseFile("filter.json")
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != gtkai_date.ID {
		t.Fatalf("manifest id %q != code id %q", m.ID, gtkai_date.ID)
	}
	if len(m.Filters) != 1 || m.Filters[0] != "date" {
		t.Fatalf("unexpected filters list: %v", m.Filters)
	}
	if m.Contract != "subprocess/v1" {
		t.Fatalf("unexpected contract: %q", m.Contract)
	}
	if m.GtkaiCoreVersion.Version == "" {
		t.Fatal("gtkai-core-version.version must not be empty")
	}
	switch m.GtkaiCoreVersion.Constraint {
	case "min", "exact":
	default:
		t.Fatalf("gtkai-core-version.constraint must be min or exact, got %q", m.GtkaiCoreVersion.Constraint)
	}
	if len(m.Platforms) == 0 {
		t.Fatal("platforms must not be empty")
	}
	if err := m.ValidateGtkaiCoreVersion("0.10.0"); err != nil {
		t.Fatalf("running gtkai 0.10.0 must satisfy manifest: %v", err)
	}
}
