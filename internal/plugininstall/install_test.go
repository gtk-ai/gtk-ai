package plugininstall_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jmeiracorbal/gtk-ai/internal/plugininstall"
	"github.com/jmeiracorbal/gtk-ai/internal/pluginregistry"
	"github.com/jmeiracorbal/gtk-ai/internal/testhome"
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

func TestInstallGtkaiDateRemote(t *testing.T) {
	testhome.Isolated(t)

	rec, err := plugininstall.Install(plugininstall.Options{
		Module:      "github.com/gtk-ai/date",
		Version:     "v0.12.0",
		CoreVersion: "0.11.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.ID != "gtk-ai/date" {
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

	db, err := pluginregistry.Open()
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

func TestInstallMarketplace(t *testing.T) {
	testhome.Isolated(t)

	root := moduleRoot(t)
	catalog := filepath.Join(root, "marketplace.json")
	installed, err := plugininstall.InstallMarketplace(catalog, "0.11.0-beta.2", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(installed) != 1 {
		t.Fatalf("installed %d filters", len(installed))
	}
}
