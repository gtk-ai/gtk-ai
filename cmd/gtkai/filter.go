package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/jmeiracorbal/gtk-ai/internal/filterinstall"
	"github.com/jmeiracorbal/gtk-ai/internal/filterregistry"
)

func runFilter(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: gtkai filter install <module@version> | install-official <official.json> | list")
		os.Exit(1)
	}
	switch args[0] {
	case "install":
		runFilterInstall(args[1:])
	case "install-official":
		runFilterInstallOfficial(args[1:])
	case "list":
		runFilterList()
	default:
		fmt.Fprintf(os.Stderr, "gtkai filter: unknown subcommand %q\n", args[0])
		os.Exit(1)
	}
}

func runFilterInstall(args []string) {
	if len(args) != 1 || args[0] == "" {
		fmt.Fprintln(os.Stderr, "usage: gtkai filter install <module@version>")
		os.Exit(1)
	}
	module, filterVer, err := filterinstall.ParseRef(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "gtkai filter install: %v\n", err)
		os.Exit(1)
	}
	rec, err := filterinstall.Install(filterinstall.Options{
		Module:      module,
		Version:     filterVer,
		CoreVersion: version,
		ReleaseRepo: releaseRepo(),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "gtkai filter install: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("installed %s@%s -> %s (%s)\n", rec.Module, rec.Version, rec.ID, rec.BinaryPath)
}

func runFilterInstallOfficial(args []string) {
	var officialPath string
	var coreVersion string
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--core-version":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "gtkai filter install-official: --core-version requires a value")
				os.Exit(1)
			}
			coreVersion = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--core-version="):
			coreVersion = strings.TrimPrefix(args[i], "--core-version=")
		case strings.HasPrefix(args[i], "-"):
			fmt.Fprintf(os.Stderr, "gtkai filter install-official: unknown flag %q\n", args[i])
			os.Exit(1)
		default:
			if officialPath != "" {
				fmt.Fprintln(os.Stderr, "gtkai filter install-official: too many arguments")
				os.Exit(1)
			}
			officialPath = args[i]
		}
	}
	if officialPath == "" {
		fmt.Fprintln(os.Stderr, "usage: gtkai filter install-official <official.json> --core-version=<ver>")
		os.Exit(1)
	}
	if coreVersion == "" {
		fmt.Fprintln(os.Stderr, "gtkai filter install-official: --core-version is required")
		os.Exit(1)
	}
	installed, err := filterinstall.InstallOfficial(officialPath, coreVersion, releaseRepo())
	if err != nil {
		fmt.Fprintf(os.Stderr, "gtkai filter install-official: %v\n", err)
		os.Exit(1)
	}
	for _, rec := range installed {
		fmt.Printf("installed %s@%s -> %s (%s)\n", rec.Module, rec.Version, rec.ID, rec.BinaryPath)
	}
}

func runFilterList() {
	db, err := filterregistry.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gtkai filter list: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	recs, err := db.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gtkai filter list: %v\n", err)
		os.Exit(1)
	}
	if len(recs) == 0 {
		fmt.Println("no filters installed")
		return
	}
	for _, rec := range recs {
		fmt.Printf("%s  %s@%s  argv0=%s  %s\n", rec.ID, rec.Module, rec.Version, rec.Argv0, rec.BinaryPath)
	}
}

func releaseRepo() string {
	if v := os.Getenv("GTKAI_RELEASE_REPO"); v != "" {
		return v
	}
	return "gtk-ai/gtk-ai"
}
