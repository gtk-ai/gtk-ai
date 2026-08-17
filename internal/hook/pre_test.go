package hook_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/jmeiracorbal/gtk-ai/internal/hook"
)

func TestPreRewritesGitStatus(t *testing.T) {
	payload := `{"tool_name":"Bash","tool_input":{"command":"git status","description":"show status"}}`
	var out bytes.Buffer
	ok, err := hook.RunPre(strings.NewReader(payload), &out, "gtkai")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected rewrite")
	}
	var result struct {
		HookSpecificOutput struct {
			HookEventName string `json:"hookEventName"`
			UpdatedInput  struct {
				Command     string `json:"command"`
				Description string `json:"description"`
			} `json:"updatedInput"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.HookSpecificOutput.HookEventName != "PreToolUse" {
		t.Fatalf("event %q", result.HookSpecificOutput.HookEventName)
	}
	if result.HookSpecificOutput.UpdatedInput.Command != "gtkai git status" {
		t.Fatalf("command %q", result.HookSpecificOutput.UpdatedInput.Command)
	}
	if result.HookSpecificOutput.UpdatedInput.Description != "show status" {
		t.Fatalf("description dropped: %q", result.HookSpecificOutput.UpdatedInput.Description)
	}
}

func TestPreLeavesEcho(t *testing.T) {
	payload := `{"tool_name":"Bash","tool_input":{"command":"echo hi"}}`
	var out bytes.Buffer
	ok, err := hook.RunPre(strings.NewReader(payload), &out, "gtkai")
	if err != nil {
		t.Fatal(err)
	}
	if ok || out.Len() != 0 {
		t.Fatalf("echo must pass through, ok=%v out=%q", ok, out.String())
	}
}

func TestPreLeavesGtkai(t *testing.T) {
	payload := `{"tool_name":"Bash","tool_input":{"command":"gtkai git status"}}`
	var out bytes.Buffer
	ok, err := hook.RunPre(strings.NewReader(payload), &out, "gtkai")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("already proxied command must not rewrite")
	}
}

func TestPreEmptyBin(t *testing.T) {
	_, err := hook.RunPre(strings.NewReader(`{"tool_name":"Bash","tool_input":{"command":"git status"}}`), &bytes.Buffer{}, "")
	if err == nil {
		t.Fatal("empty gtkai path must fail")
	}
}

func TestPreIgnoresRead(t *testing.T) {
	payload := `{"tool_name":"Read","tool_input":{"file_path":"x.go"}}`
	var out bytes.Buffer
	ok, err := hook.RunPre(strings.NewReader(payload), &out, "gtkai")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("Read must not be rewritten")
	}
}

func TestPreRewritesRegisteredBash(t *testing.T) {
	cases := []string{"ls -la", "find . -name '*.go'", "grep -n foo .", "rg foo"}
	for _, cmd := range cases {
		payload := fmt.Sprintf(`{"tool_name":"Bash","tool_input":{"command":%q}}`, cmd)
		var out bytes.Buffer
		ok, err := hook.RunPre(strings.NewReader(payload), &out, "gtkai")
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatalf("%q: expected rewrite", cmd)
		}
		if !strings.Contains(out.String(), "gtkai "+strings.Fields(cmd)[0]) {
			t.Fatalf("%q: rewritten command missing gtkai prefix: %s", cmd, out.String())
		}
	}
}

func TestPostSkipsGtkaiCommand(t *testing.T) {
	modified, _ := runHook(t, bashPayload("gtkai git status", strings.Repeat("M  file.go\n", 40)))
	if modified {
		t.Fatal("post-hook must not filter gtkai proxy output")
	}
}
