// Package filtersubprocess invokes external filter binaries using subprocess/v1.
package filtersubprocess

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/jmeiracorbal/gtk-ai/internal/registry"
)

const livenessTimeout = 500 * time.Millisecond

type request struct {
	Operation string   `json:"operation"`
	Args      []string `json:"args"`
	Output    string   `json:"output"`
	ExitCode  int      `json:"exit_code"`
}

type response struct {
	Args    []string `json:"args"`
	Changed bool     `json:"changed"`
	Output  string   `json:"output"`
}

// Module adapts an external filter binary to registry.Module.
type Module struct {
	binary string
	name   string
}

// NewModule returns a subprocess-backed module for argv0 name.
func NewModule(name, binaryPath string) *Module {
	return &Module{name: name, binary: binaryPath}
}

func (m *Module) Name() string { return m.name }

func (m *Module) Rewrite(args []string) ([]string, bool) {
	resp, err := m.call("rewrite", args, "", 0)
	if err != nil || !resp.Changed {
		return nil, false
	}
	return resp.Args, true
}

func (m *Module) FilterOutput(args []string, output string, exitCode int) string {
	resp, err := m.call("filter_output", args, output, exitCode)
	if err != nil {
		return output
	}
	return resp.Output
}

func (m *Module) TokensBefore(output string) int  { return registry.EstimateTokens(output) }
func (m *Module) TokensAfter(filtered string) int { return registry.EstimateTokens(filtered) }

func (m *Module) call(operation string, args []string, output string, exitCode int) (response, error) {
	req := request{Operation: operation, Args: args, Output: output, ExitCode: exitCode}
	payload, err := json.Marshal(req)
	if err != nil {
		return response{}, err
	}
	cmd := exec.Command(m.binary)
	cmd.Stdin = bytes.NewReader(payload)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return response{}, fmt.Errorf("filter binary: %w", err)
	}
	var resp response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		return response{}, fmt.Errorf("decode filter response: %w", err)
	}
	return resp, nil
}

// LivenessCheck verifies the binary responds to rewrite within 500ms.
func LivenessCheck(binaryPath string) error {
	if binaryPath == "" {
		return fmt.Errorf("binary path is empty")
	}
	done := make(chan error, 1)
	go func() {
		mod := NewModule("probe", binaryPath)
		_, err := mod.call("rewrite", nil, "", 0)
		done <- err
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(livenessTimeout):
		return fmt.Errorf("liveness check exceeded %s", livenessTimeout)
	}
}

// BinaryNameFromID derives the installed binary filename from a filter id.
func BinaryNameFromID(id string) string {
	for i := len(id) - 1; i >= 0; i-- {
		if id[i] == '/' {
			return id[i+1:]
		}
	}
	return id
}
