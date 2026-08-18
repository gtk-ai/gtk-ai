package hook

import "fmt"

// Agent is a supported coding-agent integration.
type Agent string

const (
	AgentClaudeCode Agent = "claudecode"
	AgentCursor     Agent = "cursor"
	AgentCodex      Agent = "codex"
	AgentOpenCode   Agent = "opencode"
)

// ParseAgent returns a known agent or an error. s must be one of the supported names.
func ParseAgent(s string) (Agent, error) {
	if s == "" {
		return "", fmt.Errorf("agent is empty")
	}
	switch Agent(s) {
	case AgentClaudeCode, AgentCursor, AgentCodex, AgentOpenCode:
		return Agent(s), nil
	default:
		return "", fmt.Errorf("unknown agent %q (want claudecode, cursor, codex, opencode)", s)
	}
}

func isShellTool(name string) bool {
	switch name {
	case "Bash", "Shell", "bash", "shell",
		"local_shell", "container_exec", "exec_command", "shell_command":
		return true
	default:
		return false
	}
}
