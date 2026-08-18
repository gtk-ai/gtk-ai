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
		"plugin/scripts/gtkai-pre-tool-use.sh",
		"plugin/scripts/gtkai-post-tool-use.sh",
		"plugin/hooks/hooks.json",
		"scripts/claudecode/gtk-ai.md",
		"scripts/cursor/hooks/gtkai-pre-tool-use.sh",
		"scripts/cursor/hooks/gtkai-post-tool-use.sh",
		"scripts/cursor/rules/gtk-ai.mdc",
		"scripts/codex/hooks/gtkai-pre-tool-use.sh",
		"scripts/codex/AGENTS.md",
		"scripts/opencode/plugins/gtkai.ts",
		"scripts/opencode/AGENTS.md",
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
	pre, err := os.ReadFile(filepath.Join(root, "plugin/scripts/gtkai-pre-tool-use.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pre), "hook-pre --agent=claudecode") {
		t.Fatal("claude pre script must pass --agent=claudecode")
	}
	post, err := os.ReadFile(filepath.Join(root, "plugin/scripts/gtkai-post-tool-use.sh"))
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
		{"scripts/cursor/hooks/gtkai-pre-tool-use.sh", "hook-pre --agent=cursor"},
		{"scripts/cursor/hooks/gtkai-post-tool-use.sh", "hook-post --agent=cursor"},
		{"scripts/codex/hooks/gtkai-pre-tool-use.sh", "hook-pre --agent=codex"},
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
