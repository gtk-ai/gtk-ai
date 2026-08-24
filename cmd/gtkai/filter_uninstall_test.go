package main_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/jmeiracorbal/gtk-ai/internal/testhome"
)

func TestFilterUninstallCLI(t *testing.T) {
	testhome.Isolated(t)

	// Install the local test date plugin.
	home := installTestDatePlugin(t)
	_ = home

	bin := buildBinary(t)
	cmd := exec.Command(bin, "filter", "uninstall", "gtk-ai/date")
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("filter uninstall: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "uninstalled gtk-ai/date") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestFilterListMarksActive(t *testing.T) {
	home := installTestDatePlugin(t)
	_ = home

	bin := buildBinary(t)
	cmd := exec.Command(bin, "filter", "list")
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, "gtk-ai/date") {
		t.Fatalf("expected filter in list: %s", text)
	}
	if !strings.Contains(text, "active") {
		t.Fatalf("expected active marker in list: %s", text)
	}
}
