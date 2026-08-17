package git

import (
	"reflect"
	"strings"
	"testing"
)

func TestRewriteStatusInjectsPorcelain(t *testing.T) {
	m := &Module{}
	got, ok := m.Rewrite([]string{"status"})
	if !ok {
		t.Fatal("expected rewrite")
	}
	want := []string{"status", "--porcelain", "-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestRewriteStatusKeepsGlobals(t *testing.T) {
	m := &Module{}
	got, ok := m.Rewrite([]string{"-C", "/tmp", "status"})
	if !ok {
		t.Fatal("expected rewrite")
	}
	want := []string{"-C", "/tmp", "status", "--porcelain", "-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestRewriteStatusSkipsExplicitFormat(t *testing.T) {
	m := &Module{}
	_, ok := m.Rewrite([]string{"status", "--porcelain"})
	if ok {
		t.Fatal("user --porcelain must not be replaced")
	}
	_, ok = m.Rewrite([]string{"status", "--long"})
	if ok {
		t.Fatal("user --long must not be replaced")
	}
}

func TestRewriteNonStatus(t *testing.T) {
	m := &Module{}
	_, ok := m.Rewrite([]string{"diff"})
	if ok {
		t.Fatal("diff must not inject status flags")
	}
}

func TestRewriteIncompleteGlobal(t *testing.T) {
	m := &Module{}
	_, ok := m.Rewrite([]string{"-C"})
	if ok {
		t.Fatal("incomplete -C must not rewrite")
	}
}

func TestFilterStatusPorcelainBranch(t *testing.T) {
	raw := "## main...origin/main\n M a.go\nA  b.go\n?? c.go\n"
	out := filterStatus(raw)
	if !strings.Contains(out, "* main...origin/main") {
		t.Fatalf("expected branch, got %q", out)
	}
	if !strings.Contains(out, "Staged") || !strings.Contains(out, "Modified") || !strings.Contains(out, "Untracked") {
		t.Fatalf("expected groups, got %q", out)
	}
}

func TestFilterStatusPorcelainClean(t *testing.T) {
	raw := "## main\n"
	out := filterStatus(raw)
	if !strings.Contains(out, "* main") || !strings.Contains(out, "clean") {
		t.Fatalf("got %q", out)
	}
}
