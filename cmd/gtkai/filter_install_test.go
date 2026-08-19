package main_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmeiracorbal/gtk-ai/internal/filterinstall"
	"github.com/jmeiracorbal/gtk-ai/internal/proxy"
)

func captureProxyStdout(t *testing.T, fn func() int) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	code := fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	if code != 0 {
		t.Fatalf("exit %d, out %q", code, buf.String())
	}
	return buf.String()
}

func TestFilterInstallOfficialAndProxyDate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := moduleRoot(t)
	official := filepath.Join(root, "filters/official.json")
	localRoot := filepath.Join(root, "filters/gtk-ai")
	if _, err := filterinstall.InstallOfficial(official, "0.10.0", "", localRoot); err != nil {
		t.Fatal(err)
	}

	out := captureProxyStdout(t, func() int { return proxy.Run("date", nil) })
	if len(out) < 20 {
		t.Fatalf("expected compact ISO date, got %q", out)
	}
	if out[4] != '-' || out[7] != '-' {
		t.Fatalf("expected ISO-8601 format, got %q", out)
	}
}
