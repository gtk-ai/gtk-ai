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
		printFilterUsage()
		os.Exit(1)
	}
	switch args[0] {
	case "install":
		runFilterInstall(args[1:])
	case "install-official":
		runFilterInstallOfficial(args[1:])
	case "uninstall":
		runFilterUninstall(args[1:])
	case "list":
		runFilterList()
	default:
		fmt.Fprintf(os.Stderr, "gtkai filter: unknown subcommand %q\n", args[0])
		printFilterUsage()
		os.Exit(1)
	}
}

func printFilterUsage() {
	fmt.Fprintln(os.Stderr, "usage: gtkai filter install <module@version> [--replace] | install-official <official.json> | uninstall <id> | list")
}

func runFilterInstall(args []string) {
	var replace bool
	var ref string
	for _, arg := range args {
		switch {
		case arg == "--replace":
			replace = true
		case strings.HasPrefix(arg, "-"):
			fmt.Fprintf(os.Stderr, "gtkai filter install: unknown flag %q\n", arg)
			os.Exit(1)
		default:
			if ref != "" {
				fmt.Fprintln(os.Stderr, "usage: gtkai filter install <module@version> [--replace]")
				os.Exit(1)
			}
			ref = arg
		}
	}
	if ref == "" {
		fmt.Fprintln(os.Stderr, "usage: gtkai filter install <module@version> [--replace]")
		os.Exit(1)
	}
	module, filterVer, err := filterinstall.ParseRef(ref)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gtkai filter install: %v\n", err)
		os.Exit(1)
	}
	rec, err := filterinstall.Install(filterinstall.Options{
		Module:      module,
		Version:     filterVer,
		CoreVersion: version,
		ReleaseRepo: releaseRepo(),
		Replace:     replace,
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

func runFilterUninstall(args []string) {
	if len(args) != 1 || args[0] == "" {
		fmt.Fprintln(os.Stderr, "usage: gtkai filter uninstall <id>")
		os.Exit(1)
	}
	rec, err := filterinstall.Uninstall(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "gtkai filter uninstall: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("uninstalled %s (argv0=%s)\n", rec.ID, rec.Argv0)
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
	activeByArgv0, err := activeFilterIDs(db, recs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gtkai filter list: %v\n", err)
		os.Exit(1)
	}
	for _, rec := range recs {
		tag := ""
		if activeByArgv0[rec.Argv0] == rec.ID {
			tag = "  active"
		}
		fmt.Printf("%s  %s@%s  argv0=%s%s  %s\n", rec.ID, rec.Module, rec.Version, rec.Argv0, tag, rec.BinaryPath)
	}
}

func activeFilterIDs(db *filterregistry.DB, recs []filterregistry.Record) (map[string]string, error) {
	seen := make(map[string]struct{})
	out := make(map[string]string)
	for _, rec := range recs {
		if _, ok := seen[rec.Argv0]; ok {
			continue
		}
		seen[rec.Argv0] = struct{}{}
		active, err := db.Active(rec.Argv0)
		if err != nil {
			return nil, err
		}
		if active != nil {
			out[rec.Argv0] = active.ID
		}
	}
	return out, nil
}

func releaseRepo() string {
	if v := os.Getenv("GTKAI_RELEASE_REPO"); v != "" {
		return v
	}
	return "gtk-ai/gtk-ai"
}
