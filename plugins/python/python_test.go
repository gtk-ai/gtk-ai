package python

import (
	"strings"
	"testing"
)

func TestIsPytestInvocation(t *testing.T) {
	if !IsPytestInvocation([]string{"-m", "pytest", "-v"}) {
		t.Fatal("expected pytest via -m")
	}
	if IsPytestInvocation([]string{"-m", "pip", "install", "pytest"}) {
		t.Fatal("pip must not match")
	}
}

func TestFilterOutputNonPytestPassthrough(t *testing.T) {
	m := &Module{name: "python3"}
	raw := "Hello from python\n"
	out := m.FilterOutput([]string{"-c", "print('Hello from python')"}, raw, 0)
	if out != raw {
		t.Fatalf("got %q", out)
	}
}

func TestFilterOutputPytestUsesFilter(t *testing.T) {
	m := &Module{name: "python3"}
	raw := "collected 1 items\n============================== 1 passed in 0.01s ==============================\n"
	out := m.FilterOutput([]string{"-m", "pytest"}, raw, 0)
	if !strings.Contains(out, "1 passed in 0.01s") {
		t.Fatalf("got %q", out)
	}
}
