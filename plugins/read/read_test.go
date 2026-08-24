package read

import (
	"strings"
	"testing"
)

func TestFilterBlockComments(t *testing.T) {
	raw := "package main\n/* block\n * line\n */\nvar x = 1\n"
	filtered, changed := FilterContent("main.go", raw)
	if !changed {
		t.Fatal("expected block comment removal")
	}
	if strings.Contains(filtered, "block") {
		t.Fatalf("block leaked: %q", filtered)
	}
	if !strings.Contains(filtered, "var x = 1") {
		t.Fatalf("code missing: %q", filtered)
	}
}

func TestFilterLineComments(t *testing.T) {
	raw := "package main\n// comment\nvar x = 1\n"
	filtered, changed := FilterContent("main.go", raw)
	if !changed || strings.Contains(filtered, "// comment") {
		t.Fatalf("got %q", filtered)
	}
}
