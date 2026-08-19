package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmeiracorbal/gtk-ai/internal/filterinstall"
	"github.com/jmeiracorbal/gtk-ai/internal/testhome"
)

func TestFilterUninstallCLI(t *testing.T) {
	testhome.Isolated(t)
	root := moduleRoot(t)
	official := filepath.Join(root, "filters/official.json")
	installed, err := filterinstall.InstallOfficial(official, "0.11.0-beta.2", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(installed) != 1 {
		t.Fatalf("installed %d filters", len(installed))
	}

	bin := buildBinary(t)
	cmd := exec.Command(bin, "filter", "uninstall", installed[0].ID)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("filter uninstall: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "uninstalled "+installed[0].ID) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestFilterListMarksActive(t *testing.T) {
	home := testhome.Isolated(t)

	if _, err := filterinstall.Install(filterinstall.Options{
		Module:      "github.com/gtk-ai/date",
		Version:     "v0.12.0",
		CoreVersion: "0.11.0-beta.2",
	}); err != nil {
		t.Fatal(err)
	}

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
	_ = home
}
