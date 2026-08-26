package main

import (
	"fmt"
	"os"

	"github.com/jmeiracorbal/gtk-ai/internal/projectmarker"
)

func runInit() {
	dir := "."
	for _, arg := range os.Args[2:] {
		if arg == "--help" || arg == "-h" {
			fmt.Fprintln(os.Stderr, "usage: gtkai init [--path=<dir>]")
			return
		}
		if len(arg) > 7 && arg[:7] == "--path=" {
			dir = arg[7:]
		}
	}

	abs, err := absInitPath(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gtkai init: %v\n", err)
		os.Exit(1)
	}
	root := projectmarker.ProjectRoot(abs)

	if err := projectmarker.Create(root); err != nil {
		fmt.Fprintf(os.Stderr, "gtkai init: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("gtkai: marker written to %s/%s\n", root, projectmarker.MarkerName)
}

func absInitPath(dir string) (string, error) {
	if dir == "." {
		return os.Getwd()
	}
	return dir, nil
}
