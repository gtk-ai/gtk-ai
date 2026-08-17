package ls

import (
	"fmt"
	"strings"
	"testing"
)

func TestRewriteInjectsLa(t *testing.T) {
	m := &Module{}
	got, ok := m.Rewrite(nil)
	if !ok {
		t.Fatal("expected rewrite")
	}
	if len(got) != 1 || got[0] != "-l" {
		t.Fatalf("got %v", got)
	}
}

func TestRewriteAlreadyLa(t *testing.T) {
	m := &Module{}
	_, ok := m.Rewrite([]string{"-la"})
	if ok {
		t.Fatal("ls -la must not rewrite")
	}
}

func TestRewriteStripsDuplicateFlags(t *testing.T) {
	m := &Module{}
	got, ok := m.Rewrite([]string{"src"})
	if !ok {
		t.Fatal("expected rewrite")
	}
	if len(got) != 2 || got[0] != "-l" || got[1] != "src" {
		t.Fatalf("got %v", got)
	}
}

func TestFilterNamesCompact(t *testing.T) {
	m := &Module{}
	var sb strings.Builder
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&sb, "handler_%02d.go\n", i)
	}
	raw := sb.String()
	out := m.FilterOutput(nil, raw)
	if out == raw {
		t.Fatal("expected compact listing")
	}
	if !strings.Contains(out, "40 files") {
		t.Fatalf("got %q", out)
	}
	if strings.Count(out, "handler_") > 10 {
		t.Fatalf("should sample names, got %q", out)
	}
}

func TestFilterLongListing(t *testing.T) {
	m := &Module{}
	var sb strings.Builder
	sb.WriteString("total 48\n")
	sb.WriteString("drwxr-xr-x 2 user staff 64 Jan 1 12:00 .\n")
	sb.WriteString("drwxr-xr-x 2 user staff 64 Jan 1 12:00 ..\n")
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&sb, "-rw-r--r-- 1 user staff 1234 Jan 1 12:00 file_%02d.go\n", i)
	}
	sb.WriteString("drwxr-xr-x 2 user staff 64 Jan 1 12:00 src\n")
	sb.WriteString("drwxr-xr-x 2 user staff 64 Jan 1 12:00 node_modules\n")
	raw := sb.String()
	out := m.FilterOutput([]string{"-l"}, raw)
	if strings.Contains(out, "node_modules") {
		t.Fatalf("noise dir leaked: %s", out)
	}
	if !strings.Contains(out, "src/") {
		t.Fatalf("missing dir: %s", out)
	}
	if !strings.Contains(out, ".go") {
		t.Fatalf("missing ext: %s", out)
	}
	if strings.Contains(out, "drwx") {
		t.Fatalf("raw perms leaked: %s", out)
	}
}

func TestFilterSmallUnchanged(t *testing.T) {
	m := &Module{}
	raw := "main.go\ngo.mod\n"
	if got := m.FilterOutput(nil, raw); got != raw {
		t.Fatalf("small listing must stay raw, got %q", got)
	}
}

func TestExtraEnv(t *testing.T) {
	m := &Module{}
	got := m.ExtraEnv(nil)
	if len(got) != 1 || got[0] != "LC_ALL=C" {
		t.Fatalf("got %v", got)
	}
}
