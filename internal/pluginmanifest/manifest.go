// Package pluginmanifest parses and validates external plugin manifests (gtkai.json).
package pluginmanifest

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"golang.org/x/mod/semver"
)

var idRegex = regexp.MustCompile(`^[a-z0-9_-]+/[a-z0-9_-]+$`)

// availableContracts lists the plugin communication protocols this version of gtkai supports.
// Add new entries here when a new protocol (e.g. grpc/v1) is implemented.
var availableContracts = []string{"stdin/v1"}

func contractSupported(c string) bool {
	for _, v := range availableContracts {
		if v == c {
			return true
		}
	}
	return false
}

// Validate checks manifest fields, platform, and core version compatibility.
func (m *Manifest) Validate(runningGtkai, platform string) error {
	if !idRegex.MatchString(m.ID) {
		return fmt.Errorf("id %q does not match naming rule", m.ID)
	}
	if strings.TrimSpace(m.Command) == "" {
		return fmt.Errorf("command must not be empty")
	}
	if !contractSupported(m.Contract) {
		return fmt.Errorf("unsupported contract %q (supported: %s)", m.Contract, strings.Join(availableContracts, ", "))
	}
	if !semver.IsValid(normalizeSemver(m.GtkaiCoreVersion.Version)) {
		return fmt.Errorf("gtkai-core-version.version %q is not valid semver", m.GtkaiCoreVersion.Version)
	}
	if m.GtkaiCoreVersion.Constraint != "min" && m.GtkaiCoreVersion.Constraint != "exact" {
		return fmt.Errorf("unknown constraint %q", m.GtkaiCoreVersion.Constraint)
	}
	if !platformListed(m.Platforms, platform) {
		return fmt.Errorf("platform %q not in manifest platforms", platform)
	}
	return m.ValidateGtkaiCoreVersion(runningGtkai)
}

func platformListed(platforms []string, platform string) bool {
	for _, p := range platforms {
		if p == platform {
			return true
		}
	}
	return false
}

// ManifestFileName is the required manifest filename at the root of a filter repository.
const ManifestFileName = "gtkai.json"

// Manifest is the required gtkai.json schema for external plugins.
type Manifest struct {
	ID               string           `json:"id"`
	Command          string           `json:"command"`
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
		if !satisfiesMin(runningGtkai, m.GtkaiCoreVersion.Version) {
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

func satisfiesMin(running, required string) bool {
	runningNorm := normalizeSemver(running)
	requiredNorm := normalizeSemver(required)
	if semver.Compare(runningNorm, requiredNorm) >= 0 {
		return true
	}
	// pre-release of the required base satisfies min (0.11.0-beta.2 >= min 0.11.0)
	if semver.Prerelease(runningNorm) != "" && semver.Prerelease(requiredNorm) == "" {
		if i := strings.Index(runningNorm, "-"); i > 0 {
			base := runningNorm[:i]
			if semver.Compare(base, requiredNorm) >= 0 {
				return true
			}
		}
	}
	return false
}

func normalizeSemver(v string) string {
	if strings.HasPrefix(v, "v") || strings.HasPrefix(v, "V") {
		return "v" + strings.TrimPrefix(strings.TrimPrefix(v, "V"), "v")
	}
	return "v" + v
}
