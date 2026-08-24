package pluginmanifest_test

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/jmeiracorbal/gtk-ai/internal/pluginmanifest"
)

func TestValidateGtkaiCoreVersionMinPass(t *testing.T) {
	m := &pluginmanifest.Manifest{
		GtkaiCoreVersion: pluginmanifest.GtkaiCoreVersion{
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
	m := &pluginmanifest.Manifest{
		GtkaiCoreVersion: pluginmanifest.GtkaiCoreVersion{
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
	m := &pluginmanifest.Manifest{
		GtkaiCoreVersion: pluginmanifest.GtkaiCoreVersion{
			Version:    "0.10.0",
			Constraint: "exact",
		},
	}
	if err := m.ValidateGtkaiCoreVersion("0.10.0"); err != nil {
		t.Fatal(err)
	}
}

func TestValidateGtkaiCoreVersionExactFail(t *testing.T) {
	m := &pluginmanifest.Manifest{
		GtkaiCoreVersion: pluginmanifest.GtkaiCoreVersion{
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
	m := &pluginmanifest.Manifest{
		GtkaiCoreVersion: pluginmanifest.GtkaiCoreVersion{
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

func TestValidateCommandEmpty(t *testing.T) {
	m := &pluginmanifest.Manifest{
		ID:       "gtk-ai/date",
		Command:  "",
		Contract: "subprocess/v1",
		GtkaiCoreVersion: pluginmanifest.GtkaiCoreVersion{
			Version:    "0.11.0",
			Constraint: "min",
		},
		Platforms: []string{"linux/amd64"},
	}
	err := m.Validate("0.11.0", "linux/amd64")
	if err == nil {
		t.Fatal("expected error for empty command")
	}
	if !strings.Contains(err.Error(), "command must not be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateGtkaiCoreVersionMinPrereleaseBase(t *testing.T) {
	m := &pluginmanifest.Manifest{
		GtkaiCoreVersion: pluginmanifest.GtkaiCoreVersion{
			Version:    "0.11.0",
			Constraint: "min",
		},
	}
	if err := m.ValidateGtkaiCoreVersion("0.11.0-beta.2"); err != nil {
		t.Fatal(err)
	}
}

func TestValidateGtkaiCoreVersionMinPrereleaseBelowBase(t *testing.T) {
	m := &pluginmanifest.Manifest{
		GtkaiCoreVersion: pluginmanifest.GtkaiCoreVersion{
			Version:    "0.11.0",
			Constraint: "min",
		},
	}
	err := m.ValidateGtkaiCoreVersion("0.10.0-beta.1")
	if err == nil {
		t.Fatal("expected error for pre-release below required base")
	}
}

func TestParseGtkaiDateManifest(t *testing.T) {
	dir, err := downloadModuleDir("github.com/gtk-ai/date@v0.12.0")
	if err != nil {
		t.Fatal(err)
	}
	m, err := pluginmanifest.ParseFile(dir + "/gtkai.json")
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "gtk-ai/date" {
		t.Fatalf("id %q", m.ID)
	}
	if m.Command != "date" {
		t.Fatalf("command %q", m.Command)
	}
	if m.GtkaiCoreVersion.Constraint != "min" {
		t.Fatalf("constraint %q", m.GtkaiCoreVersion.Constraint)
	}
	if err := m.ValidateGtkaiCoreVersion("0.11.0"); err != nil {
		t.Fatal(err)
	}
}

func downloadModuleDir(ref string) (string, error) {
	out, err := exec.Command("go", "mod", "download", "-json", ref).Output()
	if err != nil {
		return "", err
	}
	var info struct {
		Dir string `json:"Dir"`
	}
	if err := json.Unmarshal(out, &info); err != nil {
		return "", err
	}
	if info.Dir == "" {
		return "", fmt.Errorf("go mod download returned empty dir for %s", ref)
	}
	return info.Dir, nil
}
