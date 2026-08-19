// Tests de integración end-to-end.
// Compilan el binario gtkai en un directorio temporal y lo ejecutan contra
// un repositorio git local simulado, verificando que la salida es compacta
// y que el exit code se propaga correctamente.
package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmeiracorbal/gtk-ai/internal/filterinstall"
	"github.com/jmeiracorbal/gtk-ai/internal/testhome"
)

// buildBinary compila gtkai en dir y devuelve la ruta al ejecutable.
func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gtkai")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = filepath.Join(moduleRoot(t), "cmd/gtkai")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

// moduleRoot sube desde el directorio del test hasta la raíz del módulo Go.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

// initRepo crea un repositorio git local con un commit inicial y un archivo sin seguimiento.
func initRepo(t *testing.T, dir string) {
	t.Helper()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	git("init", "-q")
	git("config", "user.email", "test@test")
	git("config", "user.name", "test")
	for _, name := range []string{"main.go", "util.go", "config.go"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("add", ".")
	git("commit", "-q", "-m", "init")
	// archivo sin seguimiento para que status no esté limpio
	if err := os.WriteFile(filepath.Join(dir, "draft.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// run ejecuta gtkai con los args dados en workDir y devuelve stdout+stderr y el exit code.
func run(t *testing.T, bin, workDir string, args ...string) (string, int) {
	return runHome(t, bin, workDir, t.TempDir(), args...)
}

func runHome(t *testing.T, bin, workDir, home string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "HOME="+home)
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("exec: %v", err)
		}
	}
	return string(out), code
}

func TestIntegrationGitStatus(t *testing.T) {
	bin := buildBinary(t)
	repo := t.TempDir()
	initRepo(t, repo)

	out, code := run(t, bin, repo, "git", "status")
	if code != 0 {
		t.Fatalf("exit %d, output:\n%s", code, out)
	}
	// la salida compacta del filtro debe contener el marcador de rama y "Untracked"
	if !strings.Contains(out, "* ") {
		t.Fatalf("expected branch line in compact output, got:\n%s", out)
	}
	if !strings.Contains(out, "Untracked") {
		t.Fatalf("expected Untracked group in compact output, got:\n%s", out)
	}
	// no debe aparecer la cabecera larga del formato por defecto
	if strings.Contains(out, "On branch") {
		t.Fatalf("verbose git status must not appear in filtered output, got:\n%s", out)
	}
}

func TestIntegrationGitStatusClean(t *testing.T) {
	bin := buildBinary(t)
	repo := t.TempDir()
	initRepo(t, repo)
	// eliminar el draft para que quede limpio
	if err := os.Remove(filepath.Join(repo, "draft.go")); err != nil {
		t.Fatal(err)
	}

	out, code := run(t, bin, repo, "git", "status")
	if code != 0 {
		t.Fatalf("exit %d, output:\n%s", code, out)
	}
	if !strings.Contains(out, "clean") {
		t.Fatalf("expected 'clean' in output for clean repo, got:\n%s", out)
	}
}

func TestIntegrationLs(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	// crear suficientes archivos para activar la lógica de compactación
	for i := 0; i < 35; i++ {
		name := filepath.Join(dir, strings.Repeat("x", 5)+string(rune('a'+i%26))+".go")
		if err := os.WriteFile(name, []byte("package p\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	out, code := run(t, bin, dir, "ls")
	if code != 0 {
		t.Fatalf("exit %d, output:\n%s", code, out)
	}
	// la salida compacta debe mencionar el total de archivos, no listarlos todos
	if !strings.Contains(out, "files") {
		t.Fatalf("expected compact ls with file count, got:\n%s", out)
	}
}

// installOfficialFilters installs filters from filters/official.json into HOME.
func installOfficialFilters(t *testing.T) string {
	t.Helper()
	home := testhome.Isolated(t)
	official := filepath.Join(moduleRoot(t), "filters/official.json")
	if _, err := filterinstall.InstallOfficial(official, "0.11.0-beta.2", ""); err != nil {
		t.Fatal(err)
	}
	return home
}

func TestIntegrationDateWithoutFilter(t *testing.T) {
	bin := buildBinary(t)
	_, code := run(t, bin, t.TempDir(), "date")
	if code == 0 {
		t.Fatal("date without installed filter must exit non-zero")
	}
}

func TestIntegrationDate(t *testing.T) {
	home := installOfficialFilters(t)
	bin := buildBinary(t)
	dir := t.TempDir()

	out, code := runHome(t, bin, dir, home, "date")
	if code != 0 {
		t.Fatalf("exit %d, output:\n%s", code, out)
	}
	trimmed := strings.TrimSpace(out)
	// la salida debe ser una sola línea en formato ISO-8601
	if strings.Count(trimmed, "\n") > 0 {
		t.Fatalf("expected single-line ISO timestamp, got:\n%s", out)
	}
	// verificar que tiene la forma YYYY-MM-DDTHH:MM:SSZ
	if len(trimmed) < 19 || trimmed[4] != '-' || trimmed[7] != '-' || trimmed[10] != 'T' {
		t.Fatalf("output does not look like ISO-8601: %q", trimmed)
	}
}

func TestIntegrationDateWithFormat(t *testing.T) {
	home := installOfficialFilters(t)
	bin := buildBinary(t)
	dir := t.TempDir()

	out, code := runHome(t, bin, dir, home, "date", "+%s")
	if code != 0 {
		t.Fatalf("exit %d, output:\n%s", code, out)
	}
	trimmed := strings.TrimSpace(out)
	// la salida debe ser un unix timestamp (solo dígitos)
	for _, c := range trimmed {
		if c < '0' || c > '9' {
			t.Fatalf("expected unix timestamp (digits only), got: %q", trimmed)
		}
	}
}

func TestIntegrationUnknownCommand(t *testing.T) {
	bin := buildBinary(t)
	_, code := run(t, bin, t.TempDir(), "echo", "hi")
	if code == 0 {
		t.Fatal("unknown command must exit non-zero")
	}
}

func TestIntegrationVersion(t *testing.T) {
	bin := buildBinary(t)
	out, code := run(t, bin, t.TempDir(), "version")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.HasPrefix(out, "gtkai ") {
		t.Fatalf("unexpected version output: %q", out)
	}
}

// TestIntegrationHookPreDate verifica el flujo completo del hook pre-tool-use
// para el comando date: el binario compilado lee un payload JSON y produce
// el JSON de reescritura correcto.
func TestIntegrationHookPreDate(t *testing.T) {
	home := installOfficialFilters(t)
	bin := buildBinary(t)
	payload := `{"tool_name":"Bash","tool_input":{"command":"date"}}`

	cmd := exec.Command(bin, "hook-pre", "--agent=claudecode")
	cmd.Stdin = strings.NewReader(payload)
	cmd.Env = append(os.Environ(), "HOME="+home)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("hook-pre: %v", err)
	}
	if !strings.Contains(string(out), "gtkai date") {
		t.Fatalf("hook-pre output does not contain 'gtkai date': %s", out)
	}
}

func TestIntegrationHookPreDateWithoutFilter(t *testing.T) {
	bin := buildBinary(t)
	payload := `{"tool_name":"Bash","tool_input":{"command":"date"}}`

	cmd := exec.Command(bin, "hook-pre", "--agent=claudecode")
	cmd.Stdin = strings.NewReader(payload)
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
	out, _ := cmd.Output()
	if strings.Contains(string(out), "gtkai date") {
		t.Fatalf("hook must not rewrite date without installed filter, got: %s", out)
	}
}

func TestIntegrationHookPreDateAlreadyProxied(t *testing.T) {
	bin := buildBinary(t)
	payload := `{"tool_name":"Bash","tool_input":{"command":"gtkai date"}}`

	cmd := exec.Command(bin, "hook-pre", "--agent=claudecode")
	cmd.Stdin = strings.NewReader(payload)
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir())
	out, _ := cmd.Output()
	// sin reescritura: el hook debe producir salida vacía
	if strings.Contains(string(out), "gtkai") {
		t.Fatalf("already proxied command must produce no rewrite output, got: %s", out)
	}
}
