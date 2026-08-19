// Package storage resolves the gtk-ai data directory (~/.gtk-ai).
package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// Dir returns ~/.gtk-ai and ensures it exists.
func Dir() (string, error) {
	home := os.Getenv("HOME")
	if home == "" {
		return "", fmt.Errorf("HOME is not set")
	}
	dir := filepath.Join(home, ".gtk-ai")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
