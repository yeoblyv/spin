package cli

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/yeoblyv/spin/internal/platform"
	reg "github.com/yeoblyv/spin/internal/registry"
)

// defaultTLD is the reserved top-level domain (RFC 2606) a newly added site
// is assigned under when --domain is not given.
const defaultTLD = "test"

func init() {
	registry = append(registry, command{
		name:        "site",
		description: "Manage the registry of known Spider projects",
		run:         runSite,
	})
}

// runSite dispatches to the site subcommands: add, list, remove, use.
func runSite(args []string, stdout, stderr io.Writer, version, commit string) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "spin: site: expected a subcommand (add, list, remove, use)")
		return exitUsage
	}

	sub, rest := args[0], args[1:]
	switch sub {
	case "add":
		return runSiteAdd(rest, stdout, stderr)
	case "list":
		return runSiteList(rest, stdout, stderr)
	case "remove":
		return runSiteRemove(rest, stdout, stderr)
	case "use":
		return runSiteUse(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "spin: site: unknown subcommand %q\n", sub)
		return exitUsage
	}
}

// registryPath returns the path spin's registry file lives at, under the
// OS-appropriate configuration directory (internal/platform).
func registryPath() (string, error) {
	dir, err := platform.ConfigDir()
	if err != nil {
		return "", fmt.Errorf("locating config directory: %w", err)
	}
	return filepath.Join(dir, "registry.json"), nil
}

// openRegistry loads the registry file, returning both the registry and the
// path it should be saved back to.
func openRegistry() (*reg.Registry, string, error) {
	path, err := registryPath()
	if err != nil {
		return nil, "", err
	}
	r, err := reg.Load(path)
	if err != nil {
		return nil, "", fmt.Errorf("loading registry: %w", err)
	}
	return r, path, nil
}

func runSiteAdd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("site add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	name := fs.String("name", "", "registry name for this project (defaults to the directory name)")
	domain := fs.String("domain", "", "local domain for this project (defaults to <name>."+defaultTLD+")")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	path := "."
	if fs.NArg() > 0 {
		path = fs.Arg(0)
	}

	r, regPath, err := openRegistry()
	if err != nil {
		fmt.Fprintf(stderr, "spin: site: %v\n", err)
		return exitError
	}

	entry, err := r.Add(*name, path)
	if err != nil {
		fmt.Fprintf(stderr, "spin: site: %v\n", err)
		return exitError
	}

	resolvedDomain := *domain
	if resolvedDomain == "" {
		resolvedDomain = entry.Name + "." + defaultTLD
	} else if strings.ContainsAny(resolvedDomain, " \t\n") {
		fmt.Fprintf(stderr, "spin: site: domain %q must not contain whitespace\n", resolvedDomain)
		return exitError
	}
	if err := r.SetDomain(entry.Name, resolvedDomain); err != nil {
		fmt.Fprintf(stderr, "spin: site: %v\n", err)
		return exitError
	}

	if err := r.Save(regPath); err != nil {
		fmt.Fprintf(stderr, "spin: site: %v\n", err)
		return exitError
	}

	fmt.Fprintf(stdout, "registered %q at %s (domain: %s)\n", entry.Name, entry.Path, resolvedDomain)
	return exitOK
}

func runSiteList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("site list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	r, _, err := openRegistry()
	if err != nil {
		fmt.Fprintf(stderr, "spin: site: %v\n", err)
		return exitError
	}

	entries := r.List()
	if len(entries) == 0 {
		fmt.Fprintln(stdout, "no sites registered - run \"spin site add\" from a Spider project")
		return exitOK
	}

	defaultEntry, hasDefault := r.Default()
	for _, e := range entries {
		marker := " "
		if hasDefault && e.Name == defaultEntry.Name {
			marker = "*"
		}
		status := "ok"
		if reg.Stale(e) {
			status = "stale"
		}
		domain := e.Domain
		if domain == "" {
			domain = "-"
		}
		fmt.Fprintf(stdout, "%s %-20s %-8s %-20s %s\n", marker, e.Name, status, domain, e.Path)
	}
	return exitOK
}

func runSiteRemove(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("site remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "spin: site: remove requires exactly one site name")
		return exitUsage
	}
	name := fs.Arg(0)

	r, regPath, err := openRegistry()
	if err != nil {
		fmt.Fprintf(stderr, "spin: site: %v\n", err)
		return exitError
	}

	if err := r.Remove(name); err != nil {
		fmt.Fprintf(stderr, "spin: site: %v\n", err)
		return exitError
	}

	if err := r.Save(regPath); err != nil {
		fmt.Fprintf(stderr, "spin: site: %v\n", err)
		return exitError
	}

	fmt.Fprintf(stdout, "removed %q\n", name)
	return exitOK
}

func runSiteUse(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("site use", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "spin: site: use requires exactly one site name")
		return exitUsage
	}
	name := fs.Arg(0)

	r, regPath, err := openRegistry()
	if err != nil {
		fmt.Fprintf(stderr, "spin: site: %v\n", err)
		return exitError
	}

	if err := r.Use(name); err != nil {
		fmt.Fprintf(stderr, "spin: site: %v\n", err)
		return exitError
	}

	if err := r.Save(regPath); err != nil {
		fmt.Fprintf(stderr, "spin: site: %v\n", err)
		return exitError
	}

	fmt.Fprintf(stdout, "now using %q\n", name)
	return exitOK
}
