package hook

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/jmeiracorbal/gtk-ai/internal/shell"
)

const stdinCap = 1 << 20

type preOutput struct {
	HookSpecificOutput preSpecific `json:"hookSpecificOutput"`
}

type preSpecific struct {
	HookEventName string         `json:"hookEventName"`
	UpdatedInput  map[string]any `json:"updatedInput"`
}

// RunPre reads a PreToolUse event, rewrites Bash commands to gtkai when a module matches.
// gtkaiBin is the binary path inserted into the rewritten command.
func RunPre(r io.Reader, w io.Writer, gtkaiBin string) (bool, error) {
	if gtkaiBin == "" {
		return false, fmt.Errorf("gtkai binary path is empty")
	}

	data, err := io.ReadAll(io.LimitReader(r, stdinCap+1))
	if err != nil {
		return false, fmt.Errorf("read stdin: %w", err)
	}
	if len(data) > stdinCap {
		return false, nil
	}
	if len(data) == 0 {
		return false, nil
	}

	var input hookInput
	if err := json.Unmarshal(data, &input); err != nil {
		return false, nil
	}
	if input.ToolName != "Bash" {
		return false, nil
	}

	var toolInput map[string]any
	if err := json.Unmarshal(input.ToolInput, &toolInput); err != nil {
		return false, nil
	}
	cmd, ok := toolInput["command"].(string)
	if !ok || cmd == "" {
		return false, nil
	}

	rewritten, changed := shell.Rewrite(cmd, gtkaiBin)
	if !changed {
		return false, nil
	}

	toolInput["command"] = rewritten
	out, err := json.Marshal(preOutput{
		HookSpecificOutput: preSpecific{
			HookEventName: "PreToolUse",
			UpdatedInput:  toolInput,
		},
	})
	if err != nil {
		return false, fmt.Errorf("marshal: %w", err)
	}
	_, err = fmt.Fprintln(w, string(out))
	return true, err
}
