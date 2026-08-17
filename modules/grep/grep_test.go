package grep

import (
	"fmt"
	"strings"
	"testing"
)

func TestRewriteInjectsNH(t *testing.T) {
	m := &Module{}
	got, ok := m.Rewrite([]string{"foo", "."})
	if !ok {
		t.Fatal("expected rewrite")
	}
	if len(got) < 3 || got[0] != "-n" || got[1] != "-H" {
		t.Fatalf("got %v", got)
	}
}

func TestRewriteSkipsWhenPresent(t *testing.T) {
	m := &Module{}
	_, ok := m.Rewrite([]string{"-nH", "foo", "."})
	if ok {
		t.Fatal("already has -nH")
	}
}

func TestRewriteSkipsCount(t *testing.T) {
	m := &Module{}
	_, ok := m.Rewrite([]string{"-c", "foo"})
	if ok {
		t.Fatal("count mode must not rewrite")
	}
}

func TestFilterGroupsLikeRg(t *testing.T) {
	m := &Module{}
	var sb strings.Builder
	for i := 0; i < 15; i++ {
		for j := 0; j < 12; j++ {
			fmt.Fprintf(&sb, "src/file_%02d.go:%d:    return fmt.Errorf(\"error %d\")\n", i, j+1, j)
		}
	}
	raw := sb.String()
	out := m.FilterOutput(nil, raw)
	if out == raw {
		t.Fatal("expected grouped output")
	}
	if !strings.Contains(out, "180 matches") {
		t.Fatalf("got %s", out)
	}
	if !strings.Contains(out, "15 files") {
		t.Fatalf("got %s", out)
	}
}
