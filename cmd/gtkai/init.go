package main

import (
	"fmt"
	"os"

	"github.com/jmeiracorbal/gtk-ai/internal/projectmarker"
)

func runInit() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gtkai init: %v\n", err)
		os.Exit(1)
	}
	root := projectmarker.ProjectRoot(cwd)

	if err := projectmarker.Create(root); err != nil {
		fmt.Fprintf(os.Stderr, "gtkai init: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("gtkai: marker written to %s/%s\n", root, projectmarker.MarkerName)
}
