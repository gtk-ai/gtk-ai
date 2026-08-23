package plugininstall_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jmeiracorbal/gtk-ai/internal/plugininstall"
	"github.com/jmeiracorbal/gtk-ai/internal/pluginregistry"
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

func TestInstallConflictAbortsWithoutReplace(t *testing.T) {
	home := testhome.Isolated(t)

	db, err := pluginregistry.Open()
	if err != nil {
		t.Fatal(err)
	}
	existing := pluginregistry.Record{
		ID:           "acme/date",
		Module:       "github.com/acme/date",
		Version:      "v0.1.0",
		Argv0:        "date",
		Contract:     "subprocess/v1",
		BinaryPath:   filepath.Join(home, ".gtk-ai/filters/acme/date/date"),
		ManifestPath: filepath.Join(home, ".gtk-ai/filters/acme/date/gtkai.json"),
		InstalledAt:  time.Now().Add(-time.Hour),
	}
	if err := db.Install(existing); err != nil {
		t.Fatal(err)
	}
	db.Close()

	_, err = plugininstall.Install(plugininstall.Options{
		Module:      "github.com/gtk-ai/date",
		Version:     "v0.12.0",
		CoreVersion: "0.11.0",
	})
	if err == nil {
		t.Fatal("expected install to abort without --replace")
	}
	if !strings.Contains(err.Error(), "acme/date") || !strings.Contains(err.Error(), "--replace") {
		t.Fatalf("unexpected error: %v", err)
	}

	db, err = pluginregistry.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	active, err := db.Active("date")
	if err != nil {
		t.Fatal(err)
	}
	if active == nil || active.ID != "acme/date" {
		t.Fatalf("active filter must remain unchanged: %+v", active)
	}
}

func TestInstallConflictWithReplace(t *testing.T) {
	home := testhome.Isolated(t)

	db, err := pluginregistry.Open()
	if err != nil {
		t.Fatal(err)
	}
	existing := pluginregistry.Record{
		ID:           "acme/date",
		Module:       "github.com/acme/date",
		Version:      "v0.1.0",
		Argv0:        "date",
		Contract:     "subprocess/v1",
		BinaryPath:   filepath.Join(home, ".gtk-ai/filters/acme/date/date"),
		ManifestPath: filepath.Join(home, ".gtk-ai/filters/acme/date/gtkai.json"),
		InstalledAt:  time.Now().Add(-time.Hour),
	}
	if err := db.Install(existing); err != nil {
		t.Fatal(err)
	}
	db.Close()

	stderr := captureStderr(t, func() {
		if _, err := plugininstall.Install(plugininstall.Options{
			Module:      "github.com/gtk-ai/date",
			Version:     "v0.12.0",
			CoreVersion: "0.11.0",
			Replace:     true,
		}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(stderr, "replacing active filter") {
		t.Fatalf("expected replace notice on stderr, got %q", stderr)
	}

	db, err = pluginregistry.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	active, err := db.Active("date")
	if err != nil {
		t.Fatal(err)
	}
	if active == nil || active.ID != "gtk-ai/date" {
		t.Fatalf("active filter: %+v", active)
	}
	got, err := db.Get("acme/date")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("previous filter must remain installed but inactive")
	}
}

func TestUninstallRemovesInstallDir(t *testing.T) {
	testhome.Isolated(t)

	rec, err := plugininstall.Install(plugininstall.Options{
		Module:      "github.com/gtk-ai/date",
		Version:     "v0.12.0",
		CoreVersion: "0.11.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	installDir := filepath.Dir(rec.BinaryPath)
	if _, err := os.Stat(installDir); err != nil {
		t.Fatalf("install dir missing before uninstall: %v", err)
	}

	removed, err := plugininstall.Uninstall(rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if removed.ID != rec.ID {
		t.Fatalf("removed id %q", removed.ID)
	}
	if _, err := os.Stat(installDir); !os.IsNotExist(err) {
		t.Fatalf("install dir still present after uninstall: %v", err)
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
	if active != nil {
		t.Fatalf("expected no active filter, got %+v", active)
	}
}

func TestUninstallPromotesPreviousFilter(t *testing.T) {
	home := testhome.Isolated(t)

	db, err := pluginregistry.Open()
	if err != nil {
		t.Fatal(err)
	}
	older := pluginregistry.Record{
		ID:           "acme/date",
		Module:       "github.com/acme/date",
		Version:      "v0.1.0",
		Argv0:        "date",
		Contract:     "subprocess/v1",
		BinaryPath:   filepath.Join(home, ".gtk-ai/filters/acme/date/date"),
		ManifestPath: filepath.Join(home, ".gtk-ai/filters/acme/date/gtkai.json"),
		InstalledAt:  time.Now().Add(-time.Hour),
	}
	if err := db.Install(older); err != nil {
		t.Fatal(err)
	}
	db.Close()

	newer, err := plugininstall.Install(plugininstall.Options{
		Module:      "github.com/gtk-ai/date",
		Version:     "v0.12.0",
		CoreVersion: "0.11.0",
		Replace:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plugininstall.Uninstall(newer.ID); err != nil {
		t.Fatal(err)
	}

	db, err = pluginregistry.Open()
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
