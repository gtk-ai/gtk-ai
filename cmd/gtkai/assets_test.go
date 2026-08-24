package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShippedAgentAssets(t *testing.T) {
	root := filepath.Join("..", "..")
	files := []string{
		"integrations/claude/scripts/gtkai-pre-tool-use.sh",
		"integrations/claude/scripts/gtkai-post-tool-use.sh",
		"integrations/claude/hooks/hooks.json",
		"integrations/cursor/hooks/gtkai-pre-tool-use.sh",
		"integrations/cursor/hooks/gtkai-post-tool-use.sh",
		"integrations/cursor/rules/gtk-ai.mdc",
		"integrations/codex/hooks/gtkai-pre-tool-use.sh",
		"integrations/codex/AGENTS.md",
		"integrations/opencode/plugins/gtkai.ts",
		"integrations/opencode/AGENTS.md",
		"scripts/claudecode/gtk-ai.md",
	}
	for _, rel := range files {
		path := filepath.Join(root, rel)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing shipped file %s: %v", rel, err)
		}
	}
}

func TestClaudePluginScriptsPassAgent(t *testing.T) {
	root := filepath.Join("..", "..")
	pre, err := os.ReadFile(filepath.Join(root, "integrations/claude/scripts/gtkai-pre-tool-use.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pre), "hook-pre --agent=claudecode") {
		t.Fatal("claude pre script must pass --agent=claudecode")
	}
	post, err := os.ReadFile(filepath.Join(root, "integrations/claude/scripts/gtkai-post-tool-use.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(post), "hook-post --agent=claudecode") {
		t.Fatal("claude post script must pass --agent=claudecode")
	}
}

func TestCursorAndCodexScriptsPassAgent(t *testing.T) {
	root := filepath.Join("..", "..")
	cases := []struct {
		path   string
		needle string
	}{
		{"integrations/cursor/hooks/gtkai-pre-tool-use.sh", "hook-pre --agent=cursor"},
		{"integrations/cursor/hooks/gtkai-post-tool-use.sh", "hook-post --agent=cursor"},
		{"integrations/codex/hooks/gtkai-pre-tool-use.sh", "hook-pre --agent=codex"},
	}
	for _, tc := range cases {
		data, err := os.ReadFile(filepath.Join(root, tc.path))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), tc.needle) {
			t.Errorf("%s: missing %q", tc.path, tc.needle)
		}
	}
}
