// Tests de hook pre-tool-use para gtkai-date.
// Verifican que el filtro instalado provoca el rewrite correcto del comando `date`
// en todos los agentes soportados, y que los casos de passthrough se respetan.
package gtkai_date_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	// importar el filtro para forzar su registro via init()
	_ "github.com/jmeiracorbal/gtk-ai/filters/gtk-ai/gtkai-date"
	"github.com/jmeiracorbal/gtk-ai/internal/hook"
)

// rewrittenCommand extrae el campo command del JSON de salida del hook según el agente.
func rewrittenCommand(t *testing.T, agent hook.Agent, payload string) (string, bool) {
	t.Helper()
	var out bytes.Buffer
	ok, err := hook.RunPre(strings.NewReader(payload), &out, "gtkai", agent)
	if err != nil {
		t.Fatalf("RunPre: %v", err)
	}
	if !ok {
		return "", false
	}
	raw := out.Bytes()
	switch agent {
	case hook.AgentCursor:
		var r struct {
			UpdatedInput struct {
				Command string `json:"command"`
			} `json:"updated_input"`
		}
		if err := json.Unmarshal(raw, &r); err != nil {
			t.Fatalf("unmarshal cursor: %v", err)
		}
		return r.UpdatedInput.Command, true
	case hook.AgentOpenCode:
		var r struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal(raw, &r); err != nil {
			t.Fatalf("unmarshal opencode: %v", err)
		}
		return r.Command, true
	default:
		var r struct {
			HookSpecificOutput struct {
				UpdatedInput struct {
					Command string `json:"command"`
				} `json:"updatedInput"`
			} `json:"hookSpecificOutput"`
		}
		if err := json.Unmarshal(raw, &r); err != nil {
			t.Fatalf("unmarshal %s: %v", agent, err)
		}
		return r.HookSpecificOutput.UpdatedInput.Command, true
	}
}

// TestDateRewrittenInPreHook verifica que `date` se reescribe a `gtkai date`
// para todos los agentes.
func TestDateRewrittenInPreHook(t *testing.T) {
	agents := []struct {
		agent   hook.Agent
		payload string
	}{
		{
			hook.AgentClaudeCode,
			`{"tool_name":"Bash","tool_input":{"command":"date"}}`,
		},
		{
			hook.AgentCodex,
			`{"tool_name":"local_shell","tool_input":{"command":"date"}}`,
		},
		{
			hook.AgentCursor,
			`{"tool_name":"Shell","tool_input":{"command":"date","working_directory":"/proj"}}`,
		},
		{
			hook.AgentOpenCode,
			`{"tool_name":"bash","tool_input":{"command":"date"}}`,
		},
	}
	for _, tc := range agents {
		cmd, ok := rewrittenCommand(t, tc.agent, tc.payload)
		if !ok {
			t.Errorf("agent %s: expected rewrite, got passthrough", tc.agent)
			continue
		}
		if !strings.HasPrefix(cmd, "gtkai date") {
			t.Errorf("agent %s: rewritten command %q does not start with 'gtkai date'", tc.agent, cmd)
		}
	}
}

// TestDateWithFormatRewrittenToProxy verifica que `date '+%Y-%m-%d'` también se
// reescribe a `gtkai date`. El hook siempre reescribe cuando el módulo está registrado;
// es el proxy quien decide (consultando Rewrite del módulo) si inyecta un formato
// diferente o ejecuta el comando original. La delegación al proxy no se salta.
func TestDateWithFormatRewrittenToProxy(t *testing.T) {
	payload := `{"tool_name":"Bash","tool_input":{"command":"date '+%Y-%m-%d'"}}`
	cmd, ok := rewrittenCommand(t, hook.AgentClaudeCode, payload)
	if !ok {
		t.Fatal("date with format must still be rewritten to gtkai date by the hook")
	}
	if !strings.HasPrefix(cmd, "gtkai date") {
		t.Fatalf("unexpected rewritten command: %q", cmd)
	}
}

// TestDateAlreadyProxiedNotRewritten verifica que `gtkai date` no se vuelve a reescribir.
func TestDateAlreadyProxiedNotRewritten(t *testing.T) {
	payload := `{"tool_name":"Bash","tool_input":{"command":"gtkai date"}}`
	_, ok := rewrittenCommand(t, hook.AgentClaudeCode, payload)
	if ok {
		t.Fatal("already proxied command must not be rewritten again")
	}
}

// TestDateWithFlagRewritten verifica que `date -u` (sin formato) sí se reescribe.
func TestDateWithFlagRewritten(t *testing.T) {
	payload := `{"tool_name":"Bash","tool_input":{"command":"date -u"}}`
	cmd, ok := rewrittenCommand(t, hook.AgentClaudeCode, payload)
	if !ok {
		t.Fatal("date -u without format must be rewritten")
	}
	if !strings.HasPrefix(cmd, "gtkai date") {
		t.Fatalf("unexpected rewritten command: %q", cmd)
	}
}
