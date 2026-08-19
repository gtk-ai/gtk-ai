// Package filterinstall downloads, validates, and installs external filter modules.
package filterinstall

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jmeiracorbal/gtk-ai/internal/filtermanifest"
	"github.com/jmeiracorbal/gtk-ai/internal/filterregistry"
	"github.com/jmeiracorbal/gtk-ai/internal/filtersubprocess"
	"github.com/jmeiracorbal/gtk-ai/internal/storage"
)

// Options configures filter installation.
type Options struct {
	Module      string
	Version     string
	CoreVersion string
	ReleaseRepo string
	Replace     bool // allow replacing another active filter for the same argv0
}

// ParseRef splits module@version. Both parts are required.
func ParseRef(ref string) (module, version string, err error) {
	if ref == "" {
		return "", "", fmt.Errorf("ref is empty")
	}
	i := strings.LastIndex(ref, "@")
	if i <= 0 || i == len(ref)-1 {
		return "", "", fmt.Errorf("ref must be module@version")
	}
	return ref[:i], ref[i+1:], nil
}

// Install downloads or builds the filter, validates gtkai.json, and registers it.
func Install(opts Options) (*filterregistry.Record, error) {
	if opts.Module == "" {
		return nil, fmt.Errorf("module is empty")
	}
	if opts.Version == "" {
		return nil, fmt.Errorf("version is empty")
	}
	if opts.CoreVersion == "" {
		return nil, fmt.Errorf("core version is empty")
	}

	platform := runtime.GOOS + "/" + runtime.GOARCH
	srcDir, binary, err := resolveSource(opts, platform)
	if err != nil {
		return nil, err
	}

	manifestPath := filepath.Join(srcDir, filtermanifest.ManifestFileName)
	manifest, err := filtermanifest.ParseFile(manifestPath)
	if err != nil {
		return nil, err
	}
	if err := manifest.Validate(opts.CoreVersion, platform); err != nil {
		return nil, err
	}
	if err := filtersubprocess.LivenessCheck(binary); err != nil {
		return nil, fmt.Errorf("liveness: %w", err)
	}

	db, err := filterregistry.Open()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := checkReplaceConflict(db, manifest.ID, manifest.Filters[0], opts.Replace); err != nil {
		return nil, err
	}

	destDir, err := installDir(manifest.ID)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, err
	}

	binName := filtersubprocess.BinaryNameFromID(manifest.ID)
	destBin := filepath.Join(destDir, binName)
	if err := copyFile(binary, destBin, 0o755); err != nil {
		return nil, err
	}
	destManifest := filepath.Join(destDir, filtermanifest.ManifestFileName)
	if err := copyFile(manifestPath, destManifest, 0o644); err != nil {
		return nil, err
	}

	rec := filterregistry.Record{
		ID:           manifest.ID,
		Module:       opts.Module,
		Version:      opts.Version,
		Argv0:        manifest.Filters[0],
		Contract:     manifest.Contract,
		BinaryPath:   destBin,
		ManifestPath: destManifest,
		InstalledAt:  time.Now(),
	}

	if err := db.Install(rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

func checkReplaceConflict(db *filterregistry.DB, id, argv0 string, replace bool) error {
	prev, err := db.Active(argv0)
	if err != nil {
		return err
	}
	if prev == nil || prev.ID == id {
		return nil
	}
	if !replace {
		return fmt.Errorf("active filter %s already handles %q; use --replace to install %s (or filter uninstall %s)", prev.ID, argv0, id, prev.ID)
	}
	fmt.Fprintf(os.Stderr, "replacing active filter %s with %s for command %q\n", prev.ID, id, argv0)
	return nil
}

func resolveSource(opts Options, platform string) (srcDir, binary string, err error) {
	if prebuilt, ok := tryPrebuilt(opts.Module, opts.Version, platform); ok {
		srcDir, err = fetchGoModule(opts.Module, opts.Version)
		if err != nil {
			return "", "", err
		}
		return srcDir, prebuilt, nil
	}
	srcDir, err = fetchGoModule(opts.Module, opts.Version)
	if err != nil {
		return "", "", err
	}
	binary, err = buildModule(opts.Module, opts.Version)
	if err != nil {
		return "", "", err
	}
	return srcDir, binary, nil
}

func tryPrebuilt(module, version, platform string) (path string, ok bool) {
	if module == "" {
		return "", false
	}
	repo := strings.TrimPrefix(module, "github.com/")
	binName := filepath.Base(module)
	osName, arch := splitPlatform(platform)
	tag := version
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s-%s-%s", repo, tag, binName, osName, arch)
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("gtkai-filter-prebuilt-%d", time.Now().UnixNano()))
	if err := downloadFile(url, tmp); err != nil {
		return "", false
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		os.Remove(tmp)
		return "", false
	}
	return tmp, true
}

func splitPlatform(platform string) (osName, arch string) {
	parts := strings.Split(platform, "/")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func execGo(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	return cmd.CombinedOutput()
}

func fetchGoModule(module, version string) (string, error) {
	modVersion := module + "@" + version
	out, err := execGo("", "mod", "download", "-json", modVersion)
	if err != nil {
		return "", fmt.Errorf("go mod download %s: %w", modVersion, err)
	}
	var info struct {
		Dir string `json:"Dir"`
	}
	if err := json.Unmarshal(out, &info); err != nil {
		return "", fmt.Errorf("parse go mod download json: %w", err)
	}
	if info.Dir == "" {
		return "", fmt.Errorf("go mod download returned empty dir for %s", modVersion)
	}
	return info.Dir, nil
}

func buildModule(module, version string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "gtkai-filter-build-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	init := exec.Command("go", "mod", "init", "gtkai-filter-build")
	init.Dir = tmpDir
	init.Env = os.Environ()
	if out, err := init.CombinedOutput(); err != nil {
		return "", fmt.Errorf("go mod init: %w\n%s", err, out)
	}
	modVersion := module + "@" + version
	get := exec.Command("go", "get", modVersion)
	get.Dir = tmpDir
	get.Env = os.Environ()
	if out, err := get.CombinedOutput(); err != nil {
		return "", fmt.Errorf("go get %s: %w\n%s", modVersion, err, out)
	}
	tmpBin := filepath.Join(os.TempDir(), fmt.Sprintf("gtkai-filter-bin-%d", time.Now().UnixNano()))
	pkg := module + "/cmd"
	if err := goBuildDir(tmpDir, pkg, tmpBin); err != nil {
		return "", err
	}
	return tmpBin, nil
}

func goBuildDir(dir, pkg, out string) error {
	cmd := exec.Command("go", "build", "-o", out, pkg)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build %s: %w\n%s", pkg, err, outBytes)
	}
	return nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func installDir(id string) (string, error) {
	base, err := storage.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "filters", id), nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// OfficialFilter is one entry in filters/official.json.
type OfficialFilter struct {
	Module  string `json:"module"`
	Version string `json:"version"`
}

// OfficialFilters is the list of filters install.sh installs by default.
type OfficialFilters struct {
	Filters []OfficialFilter `json:"filters"`
}

// LoadOfficial reads filters/official.json from path.
func LoadOfficial(path string) (OfficialFilters, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return OfficialFilters{}, err
	}
	var o OfficialFilters
	if err := json.Unmarshal(data, &o); err != nil {
		return OfficialFilters{}, err
	}
	if len(o.Filters) == 0 {
		return OfficialFilters{}, fmt.Errorf("no filters listed in %s", path)
	}
	for _, f := range o.Filters {
		if f.Module == "" || f.Version == "" {
			return OfficialFilters{}, fmt.Errorf("module and version are required in official filters")
		}
	}
	return o, nil
}

// InstallOfficial installs every filter listed in official.json.
func InstallOfficial(officialPath, coreVersion, releaseRepo string) ([]filterregistry.Record, error) {
	official, err := LoadOfficial(officialPath)
	if err != nil {
		return nil, err
	}
	var installed []filterregistry.Record
	for _, f := range official.Filters {
		opts := Options{
			Module:      f.Module,
			Version:     f.Version,
			CoreVersion: coreVersion,
			ReleaseRepo: releaseRepo,
			Replace:     true,
		}
		rec, err := Install(opts)
		if err != nil {
			return installed, fmt.Errorf("install %s@%s: %w", f.Module, f.Version, err)
		}
		installed = append(installed, *rec)
	}
	return installed, nil
}
