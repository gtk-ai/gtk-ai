// Package filtermanifest parses and validates external filter manifests (gtkai.json).
package filtermanifest

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/mod/semver"
)

// ManifestFileName is the required manifest filename at the root of a filter repository.
const ManifestFileName = "gtkai.json"

// Manifest is the required gtkai.json schema for subprocess/v1 filters.
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

// ParseFile reads and unmarshals gtkai.json at path.
func ParseFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read gtkai.json: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse gtkai.json: %w", err)
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
