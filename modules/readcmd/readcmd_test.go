package readcmd

import (
	"strings"
	"testing"
)

func TestFilterSingleFile(t *testing.T) {
	m := &Module{name: "cat"}
	raw := strings.Repeat("// comment\nvar x = 1\n", 50)
	out := m.FilterOutput([]string{"main.go"}, raw, 0)
	if out == raw {
		t.Fatal("expected comment strip")
	}
	if strings.Contains(out, "// comment") {
		t.Fatalf("comments remain: %s", out)
	}
}

func TestFilterMultipleFilesPassthrough(t *testing.T) {
	m := &Module{name: "cat"}
	raw := "a\nb\n"
	if got := m.FilterOutput([]string{"a.go", "b.go"}, raw, 0); got != raw {
		t.Fatalf("multi-file must pass through, got %q", got)
	}
}

func TestFilePathSkipsFlags(t *testing.T) {
	if got := filePath([]string{"-n", "10", "src/main.go"}); got != "src/main.go" {
		t.Fatalf("got %q", got)
	}
}
