package filterinstall_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jmeiracorbal/gtk-ai/internal/filterinstall"
	"github.com/jmeiracorbal/gtk-ai/internal/filterregistry"
	"github.com/jmeiracorbal/gtk-ai/internal/testhome"
)

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	fn()
	_ = w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	return buf.String()
}

func TestInstallConflictWarning(t *testing.T) {
	home := testhome.Isolated(t)

	db, err := filterregistry.Open()
	if err != nil {
		t.Fatal(err)
	}
	existing := filterregistry.Record{
		ID:           "acme/gtkai-date",
		Module:       "github.com/acme/gtkai-date",
		Version:      "v0.1.0",
		Argv0:        "date",
		Contract:     "subprocess/v1",
		BinaryPath:   filepath.Join(home, ".gtk-ai/filters/acme/gtkai-date/acme-gtkai-date"),
		ManifestPath: filepath.Join(home, ".gtk-ai/filters/acme/gtkai-date/gtkai.json"),
		InstalledAt:  time.Now().Add(-time.Hour),
	}
	if err := db.Install(existing); err != nil {
		t.Fatal(err)
	}
	db.Close()

	stderr := captureStderr(t, func() {
		if _, err := filterinstall.Install(filterinstall.Options{
			Module:      "github.com/gtk-ai/gtkai-date",
			Version:     "v0.10.1",
			CoreVersion: "0.10.0",
		}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(stderr, "warning:") {
		t.Fatalf("expected conflict warning on stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, "acme/gtkai-date") || !strings.Contains(stderr, "gtk-ai/gtkai-date") {
		t.Fatalf("expected both filter ids in warning, got %q", stderr)
	}

	db, err = filterregistry.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	active, err := db.Active("date")
	if err != nil {
		t.Fatal(err)
	}
	if active == nil || active.ID != "gtk-ai/gtkai-date" {
		t.Fatalf("active filter: %+v", active)
	}
}

func TestUninstallRemovesInstallDir(t *testing.T) {
	testhome.Isolated(t)

	rec, err := filterinstall.Install(filterinstall.Options{
		Module:      "github.com/gtk-ai/gtkai-date",
		Version:     "v0.10.1",
		CoreVersion: "0.10.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	installDir := filepath.Dir(rec.BinaryPath)
	if _, err := os.Stat(installDir); err != nil {
		t.Fatalf("install dir missing before uninstall: %v", err)
	}

	removed, err := filterinstall.Uninstall(rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if removed.ID != rec.ID {
		t.Fatalf("removed id %q", removed.ID)
	}
	if _, err := os.Stat(installDir); !os.IsNotExist(err) {
		t.Fatalf("install dir still present after uninstall: %v", err)
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
	if active != nil {
		t.Fatalf("expected no active filter, got %+v", active)
	}
}

func TestUninstallPromotesPreviousFilter(t *testing.T) {
	home := testhome.Isolated(t)

	db, err := filterregistry.Open()
	if err != nil {
		t.Fatal(err)
	}
	older := filterregistry.Record{
		ID:           "acme/gtkai-date",
		Module:       "github.com/acme/gtkai-date",
		Version:      "v0.1.0",
		Argv0:        "date",
		Contract:     "subprocess/v1",
		BinaryPath:   filepath.Join(home, ".gtk-ai/filters/acme/gtkai-date/acme-gtkai-date"),
		ManifestPath: filepath.Join(home, ".gtk-ai/filters/acme/gtkai-date/gtkai.json"),
		InstalledAt:  time.Now().Add(-time.Hour),
	}
	if err := db.Install(older); err != nil {
		t.Fatal(err)
	}
	db.Close()

	newer, err := filterinstall.Install(filterinstall.Options{
		Module:      "github.com/gtk-ai/gtkai-date",
		Version:     "v0.10.1",
		CoreVersion: "0.10.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := filterinstall.Uninstall(newer.ID); err != nil {
		t.Fatal(err)
	}

	db, err = filterregistry.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	active, err := db.Active("date")
	if err != nil {
		t.Fatal(err)
	}
	if active == nil || active.ID != older.ID {
		t.Fatalf("active filter: %+v", active)
	}
}
