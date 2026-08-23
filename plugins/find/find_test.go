package find

import (
	"fmt"
	"strings"
	"testing"
)

func TestFilterGroupsByDirectory(t *testing.T) {
	m := &Module{}
	var sb strings.Builder
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&sb, "./src/module_%02d/handler.go\n", i)
		fmt.Fprintf(&sb, "./src/module_%02d/util.go\n", i)
	}
	raw := sb.String()
	out := m.FilterOutput(nil, raw, 0)
	if out == raw {
		t.Fatal("expected grouped output")
	}
	if !strings.Contains(out, "80 files") {
		t.Fatalf("missing file count: %s", out)
	}
	if !strings.Contains(out, ".go(80)") {
		t.Fatalf("missing ext: %s", out)
	}
	if strings.Count(out, "handler.go") > maxShown {
		t.Fatalf("did not cap shown files: %s", out)
	}
}

func TestFilterSmallUnchanged(t *testing.T) {
	m := &Module{}
	raw := "./main.go\n./go.mod\n"
	if got := m.FilterOutput(nil, raw, 0); got != raw {
		t.Fatalf("got %q", got)
	}
}

func TestFilterEmptyUnchanged(t *testing.T) {
	m := &Module{}
	if got := m.FilterOutput(nil, "", 0); got != "" {
		t.Fatalf("got %q", got)
	}
}
