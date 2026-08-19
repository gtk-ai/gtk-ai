package filtermanifest_test

import (
	"strings"
	"testing"

	"github.com/jmeiracorbal/gtk-ai/internal/filtermanifest"
)

func TestValidateGtkaiCoreVersionMinPass(t *testing.T) {
	m := &filtermanifest.Manifest{
		GtkaiCoreVersion: filtermanifest.GtkaiCoreVersion{
			Version:    "0.10.0",
			Constraint: "min",
		},
	}
	if err := m.ValidateGtkaiCoreVersion("0.10.0"); err != nil {
		t.Fatal(err)
	}
	if err := m.ValidateGtkaiCoreVersion("0.11.0"); err != nil {
		t.Fatal(err)
	}
}

func TestValidateGtkaiCoreVersionMinFail(t *testing.T) {
	m := &filtermanifest.Manifest{
		GtkaiCoreVersion: filtermanifest.GtkaiCoreVersion{
			Version:    "0.10.0",
			Constraint: "min",
		},
	}
	err := m.ValidateGtkaiCoreVersion("0.9.0")
	if err == nil {
		t.Fatal("expected error for version below min")
	}
	if !strings.Contains(err.Error(), "< required min") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateGtkaiCoreVersionExactPass(t *testing.T) {
	m := &filtermanifest.Manifest{
		GtkaiCoreVersion: filtermanifest.GtkaiCoreVersion{
			Version:    "0.10.0",
			Constraint: "exact",
		},
	}
	if err := m.ValidateGtkaiCoreVersion("0.10.0"); err != nil {
		t.Fatal(err)
	}
}

func TestValidateGtkaiCoreVersionExactFail(t *testing.T) {
	m := &filtermanifest.Manifest{
		GtkaiCoreVersion: filtermanifest.GtkaiCoreVersion{
			Version:    "0.10.0",
			Constraint: "exact",
		},
	}
	err := m.ValidateGtkaiCoreVersion("0.11.0")
	if err == nil {
		t.Fatal("expected error for exact mismatch")
	}
	if !strings.Contains(err.Error(), "!= required exact") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateGtkaiCoreVersionUnknownConstraint(t *testing.T) {
	m := &filtermanifest.Manifest{
		GtkaiCoreVersion: filtermanifest.GtkaiCoreVersion{
			Version:    "0.10.0",
			Constraint: "latest",
		},
	}
	err := m.ValidateGtkaiCoreVersion("0.10.0")
	if err == nil {
		t.Fatal("expected error for unknown constraint")
	}
	if !strings.Contains(err.Error(), `unknown constraint "latest"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseGtkaiDateManifest(t *testing.T) {
	m, err := filtermanifest.ParseFile("../../filters/gtk-ai/gtkai-date/filter.json")
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "gtk-ai/gtkai-date" {
		t.Fatalf("id %q", m.ID)
	}
	if m.GtkaiCoreVersion.Constraint != "min" {
		t.Fatalf("constraint %q", m.GtkaiCoreVersion.Constraint)
	}
	if err := m.ValidateGtkaiCoreVersion("0.10.0"); err != nil {
		t.Fatal(err)
	}
}
