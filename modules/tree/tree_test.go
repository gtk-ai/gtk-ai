package tree

import (
	"fmt"
	"strings"
	"testing"
)

func TestFilterLargeTree(t *testing.T) {
	m := &Module{}
	var sb strings.Builder
	for i := 0; i < 50; i++ {
		fmt.Fprintf(&sb, "    |-- file_%02d.go\n", i)
	}
	raw := sb.String()
	out := m.FilterOutput(nil, raw, 0)
	if out == raw {
		t.Fatal("expected compact tree")
	}
	if !strings.Contains(out, "50 entries") {
		t.Fatalf("got %q", out)
	}
}

func TestFilterSmallUnchanged(t *testing.T) {
	m := &Module{}
	raw := ".\n`-- main.go\n"
	if got := m.FilterOutput(nil, raw, 0); got != raw {
		t.Fatalf("got %q", got)
	}
}
