package filterinstall_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jmeiracorbal/gtk-ai/internal/filterinstall"
	"github.com/jmeiracorbal/gtk-ai/internal/filterregistry"
)

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func TestInstallLocalGtkaiDate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	localDir := filepath.Join(moduleRoot(t), "filters/gtk-ai/gtkai-date")
	rec, err := filterinstall.Install(filterinstall.Options{
		Module:      "github.com/gtk-ai/gtkai-date",
		Version:     "v0.10.1",
		CoreVersion: "0.10.0",
		LocalDir:    localDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.ID != "gtk-ai/gtkai-date" {
		t.Fatalf("id %q", rec.ID)
	}
	if rec.Argv0 != "date" {
		t.Fatalf("argv0 %q", rec.Argv0)
	}
	if _, err := os.Stat(rec.BinaryPath); err != nil {
		t.Fatalf("binary missing: %v", err)
	}
	if _, err := os.Stat(rec.ManifestPath); err != nil {
		t.Fatalf("manifest missing: %v", err)
	}

	db, err := filterregistry.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	active, err := db.Active("date")
	if err != nil {
		t.Fatal(err)
	}
	if active == nil || active.BinaryPath != rec.BinaryPath {
		t.Fatalf("registry active: %+v", active)
	}
}

func TestInstallOfficialLocal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := moduleRoot(t)
	official := filepath.Join(root, "filters/official.json")
	localRoot := filepath.Join(root, "filters/gtk-ai")
	installed, err := filterinstall.InstallOfficial(official, "0.10.0", "", localRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(installed) != 1 {
		t.Fatalf("installed %d filters", len(installed))
	}
}
