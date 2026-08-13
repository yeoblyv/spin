package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/yeoblyv/spin/internal/hosts"
	"github.com/yeoblyv/spin/internal/platform"
)

func init() {
	registry = append(registry, command{
		name:        "hosts",
		description: "Manage the spin-owned block of local-domain entries in the hosts file",
		run:         runHosts,
	})
}

// runHosts dispatches to the hosts subcommands: apply, remove.
func runHosts(args []string, stdout, stderr io.Writer, version, commit string) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "spin: hosts: expected a subcommand (apply, remove)")
		return exitUsage
	}

	sub, rest := args[0], args[1:]
	switch sub {
	case "apply":
		return runHostsApply(rest, stdout, stderr)
	case "remove":
		return runHostsRemove(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "spin: hosts: unknown subcommand %q\n", sub)
		return exitUsage
	}
}

// runHostsApply writes every registered site's domain into the hosts
// file's spin-owned block, pointed at 127.0.0.1.
func runHostsApply(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("hosts apply", flag.ContinueOnError)
	fs.SetOutput(stderr)
	path := fs.String("hosts-file", platform.HostsFilePath(), "hosts file to update")
	dryRun := fs.Bool("dry-run", false, "print the block that would be written instead of writing it")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	r, _, err := openRegistry()
	if err != nil {
		fmt.Fprintf(stderr, "spin: hosts: %v\n", err)
		return exitError
	}

	entries := map[string]string{}
	for _, e := range r.List() {
		if e.Domain != "" {
			entries[e.Domain] = "127.0.0.1"
		}
	}

	if *dryRun {
		fmt.Fprint(stdout, hosts.Render(entries))
		return exitOK
	}

	if err := hosts.Apply(*path, entries); err != nil {
		reportHostsError(stderr, *path, err)
		return exitError
	}

	fmt.Fprintf(stdout, "updated %s with %d domain(s)\n", *path, len(entries))
	return exitOK
}

// runHostsRemove strips spin's block from the hosts file entirely.
func runHostsRemove(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("hosts remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	path := fs.String("hosts-file", platform.HostsFilePath(), "hosts file to update")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	if err := hosts.Clear(*path); err != nil {
		reportHostsError(stderr, *path, err)
		return exitError
	}

	fmt.Fprintf(stdout, "removed spin's block from %s\n", *path)
	return exitOK
}

// reportHostsError surfaces a permission-denied write with the specific
// next step to take (elevate and re-run) rather than a bare OS error,
// since writing the hosts file is the one step in this command that
// actually needs elevated privileges.
func reportHostsError(stderr io.Writer, path string, err error) {
	if errors.Is(err, os.ErrPermission) {
		fmt.Fprintf(stderr, "spin: hosts: permission denied writing %s - re-run with elevated privileges, e.g. \"sudo spin hosts apply\"\n", path)
		return
	}
	fmt.Fprintf(stderr, "spin: hosts: %v\n", err)
}
