// Package filtermanifest parses and validates external filter manifests (filter.json).
package filtermanifest

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/mod/semver"
)

// Manifest is the required filter.json schema for subprocess/v1 filters.
type Manifest struct {
	ID               string           `json:"id"`
	Filters          []string         `json:"filters"`
	Platforms        []string         `json:"platforms"`
	Contract         string           `json:"contract"`
	GtkaiCoreVersion GtkaiCoreVersion `json:"gtkai-core-version"`
}

// GtkaiCoreVersion declares which gtkai core versions may run this filter.
type GtkaiCoreVersion struct {
	Version    string `json:"version"`
	Constraint string `json:"constraint"`
}

// ParseFile reads and unmarshals filter.json at path.
func ParseFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read filter.json: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse filter.json: %w", err)
	}
	return &m, nil
}

// ValidateGtkaiCoreVersion checks that runningGtkai satisfies the manifest constraint.
func (m *Manifest) ValidateGtkaiCoreVersion(runningGtkai string) error {
	switch m.GtkaiCoreVersion.Constraint {
	case "min":
		if semver.Compare(normalizeSemver(runningGtkai), normalizeSemver(m.GtkaiCoreVersion.Version)) < 0 {
			return fmt.Errorf("gtkai %s < required min %s", runningGtkai, m.GtkaiCoreVersion.Version)
		}
	case "exact":
		if runningGtkai != m.GtkaiCoreVersion.Version {
			return fmt.Errorf("gtkai %s != required exact %s", runningGtkai, m.GtkaiCoreVersion.Version)
		}
	default:
		return fmt.Errorf("unknown constraint %q", m.GtkaiCoreVersion.Constraint)
	}
	return nil
}

func normalizeSemver(v string) string {
	if strings.HasPrefix(v, "v") || strings.HasPrefix(v, "V") {
		return "v" + strings.TrimPrefix(strings.TrimPrefix(v, "V"), "v")
	}
	return "v" + v
}
