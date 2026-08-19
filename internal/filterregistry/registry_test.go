package filterregistry_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jmeiracorbal/gtk-ai/internal/filterregistry"
)

func TestInstallAndActive(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	db, err := filterregistry.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rec := filterregistry.Record{
		ID:           "gtk-ai/gtkai-date",
		Module:       "github.com/gtk-ai/gtkai-date",
		Version:      "v0.10.1",
		Argv0:        "date",
		Contract:     "subprocess/v1",
		BinaryPath:   filepath.Join(home, "gtkai-date"),
		ManifestPath: filepath.Join(home, "gtkai.json"),
		InstalledAt:  time.Now(),
	}
	if err := db.Install(rec); err != nil {
		t.Fatal(err)
	}
	active, err := db.Active("date")
	if err != nil {
		t.Fatal(err)
	}
	if active == nil || active.ID != rec.ID {
		t.Fatalf("active filter: %+v", active)
	}
}
