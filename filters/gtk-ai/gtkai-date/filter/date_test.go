package filter_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmeiracorbal/gtk-ai/filters/gtk-ai/gtkai-date/filter"
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

func TestRewriteWithFormat(t *testing.T) {
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

func TestRewriteWithFlagAndFormat(t *testing.T) {
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

func TestFilterOutputPreservesOnSingleNewline(t *testing.T) {
	out := filter.FilterOutput(nil, "2026-08-19T14:30:00Z\n", 0)
	if out != "2026-08-19T14:30:00Z\n" {
		t.Fatalf("unexpected: %q", out)
	}
}

// --- ID ---

func TestID(t *testing.T) {
	if filter.ID != "gtk-ai/gtkai-date" {
		t.Fatalf("ID %q does not follow author/gtkai-<command> rule", filter.ID)
	}
}

// --- filter.json manifest ---

func TestManifest(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "filter.json"))
	if err != nil {
		t.Fatalf("read filter.json: %v", err)
	}

	var m struct {
		ID              string   `json:"id"`
		Filters         []string `json:"filters"`
		Version         string   `json:"version"`
		Platforms       []string `json:"platforms"`
		Contract        string   `json:"contract"`
		MinGtkaiVersion string   `json:"min_gtkai_version"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse filter.json: %v", err)
	}
	if m.ID != filter.ID {
		t.Fatalf("manifest id %q != code id %q", m.ID, filter.ID)
	}
	if len(m.Filters) != 1 || m.Filters[0] != "date" {
		t.Fatalf("unexpected filters: %v", m.Filters)
	}
	if m.Contract != "subprocess/v1" {
		t.Fatalf("unexpected contract: %q", m.Contract)
	}
	if m.Version == "" || m.MinGtkaiVersion == "" {
		t.Fatal("version fields must not be empty")
	}
	if len(m.Platforms) == 0 {
		t.Fatal("platforms must not be empty")
	}
}
