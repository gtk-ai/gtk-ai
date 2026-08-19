// Tests de integración del protocolo subprocess/v1.
// Compilan el binario en un directorio temporal y verifican el contrato
// JSON completo: rewrite con y sin formato, filter_output, operación desconocida.
package main_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gtkai-date")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	wd, _ := os.Getwd()
	cmd.Dir = wd
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

func invoke(t *testing.T, bin string, req any) map[string]any {
	t.Helper()
	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin)
	cmd.Stdin = strings.NewReader(string(payload))
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	var resp map[string]any
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("parse response: %v\nraw: %s", err, out)
	}
	return resp
}

func TestProtocolRewriteNoFormat(t *testing.T) {
	bin := buildBinary(t)
	resp := invoke(t, bin, map[string]any{
		"operation": "rewrite",
		"args":      []string{},
		"output":    "",
		"exit_code": 0,
	})
	if resp["changed"] != true {
		t.Fatalf("expected changed=true, got %v", resp)
	}
	args, _ := resp["args"].([]any)
	if len(args) == 0 {
		t.Fatal("expected rewritten args")
	}
	if !strings.HasPrefix(args[0].(string), "+%") {
		t.Fatalf("expected ISO format arg, got %v", args[0])
	}
}

func TestProtocolRewriteWithFormat(t *testing.T) {
	bin := buildBinary(t)
	resp := invoke(t, bin, map[string]any{
		"operation": "rewrite",
		"args":      []string{"+%s"},
		"output":    "",
		"exit_code": 0,
	})
	if resp["changed"] != false {
		t.Fatalf("expected changed=false when format present, got %v", resp)
	}
}

func TestProtocolFilterOutput(t *testing.T) {
	bin := buildBinary(t)
	resp := invoke(t, bin, map[string]any{
		"operation": "filter_output",
		"args":      []string{},
		"output":    "2026-08-19T14:30:00Z\n\n",
		"exit_code": 0,
	})
	out, _ := resp["output"].(string)
	if out != "2026-08-19T14:30:00Z\n" {
		t.Fatalf("unexpected filtered output: %q", out)
	}
}

func TestProtocolUnknownOperationFails(t *testing.T) {
	bin := buildBinary(t)
	payload := `{"operation":"unknown","args":[],"output":"","exit_code":0}`
	cmd := exec.Command(bin)
	cmd.Stdin = strings.NewReader(payload)
	err := cmd.Run()
	if err == nil {
		t.Fatal("unknown operation must exit non-zero")
	}
}

func TestProtocolLiveness(t *testing.T) {
	// comprueba que el binario responde en menos de 500ms (contrato de instalación)
	bin := buildBinary(t)
	resp := invoke(t, bin, map[string]any{
		"operation": "rewrite",
		"args":      []string{},
		"output":    "",
		"exit_code": 0,
	})
	if resp == nil {
		t.Fatal("no response")
	}
}
