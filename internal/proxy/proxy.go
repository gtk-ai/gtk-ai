// Package proxy runs a registered module against a real command.
package proxy

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jmeiracorbal/gtk-ai/internal/registry"
	"github.com/jmeiracorbal/gtk-ai/internal/text"
	"github.com/jmeiracorbal/gtk-ai/modules/gain"
)

// Run executes name with args, filters stdout, records gain, and returns the child exit code.
func Run(name string, args []string) int {
	mod := registry.Get(name)
	if mod == nil {
		fmt.Fprintf(os.Stderr, "gtkai: unknown command %q\n", name)
		return 1
	}

	execArgs := args
	rewritten, didRewrite := mod.Rewrite(args)
	if didRewrite {
		execArgs = rewritten
	}

	start := time.Now()

	var rawOut, execOut, execErr string
	var execCode int
	var execFail error

	if didRewrite {
		rawOut, _, _, _ = runCmd(name, args)
		execOut, execErr, execCode, execFail = runCmd(name, execArgs)
	} else {
		execOut, execErr, execCode, execFail = runCmd(name, execArgs)
		rawOut = execOut
	}

	elapsed := time.Since(start)

	stripped := text.StripANSI(execOut)
	filtered := mod.FilterOutput(execArgs, stripped)
	shown := registry.NeverWorse(rawOut, filtered)

	if _, werr := os.Stdout.WriteString(shown); werr != nil {
		fmt.Fprintf(os.Stderr, "gtkai: write stdout: %v\n", werr)
	}
	if execErr != "" {
		_, _ = os.Stderr.WriteString(execErr)
	}

	label := name
	if len(args) > 0 {
		label = name + " " + strings.Join(args, " ")
	}
	recordGain(label, rawOut, shown, elapsed)

	if execFail != nil {
		fmt.Fprintf(os.Stderr, "gtkai: %v\n", execFail)
		return 1
	}
	return execCode
}

func runCmd(name string, args []string) (stdout, stderr string, code int, err error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	code = 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
			err = nil
		}
	}
	return outBuf.String(), errBuf.String(), code, err
}

func recordGain(cmd, raw, shown string, elapsed time.Duration) {
	t, err := gain.Open()
	if err != nil {
		return
	}
	defer t.Close()
	_ = t.Record(cmd, registry.EstimateTokens(raw), registry.EstimateTokens(shown), elapsed)
}
