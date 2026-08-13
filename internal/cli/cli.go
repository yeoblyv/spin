// Package cli implements spin's command dispatch: argument parsing, the
// version/help output, and the registry subcommands attach to as they are
// added.
package cli

import (
	"fmt"
	"io"
)

// exit codes follow the common Unix convention: 0 for success, 1 for a
// command that ran but failed, 2 for a usage error (unknown/missing
// arguments) - mirrored by Go's own flag package.
const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

// command is a single spin subcommand. version and commit are threaded
// through explicitly rather than read from a package-level variable, so a
// subcommand's dependencies are visible in its signature.
type command struct {
	name        string
	description string
	run         func(args []string, stdout, stderr io.Writer, version, commit string) int
}

// registry lists every subcommand spin currently supports, in the order
// they appear in help output. It starts empty; commands are added here as
// they are ported from bin/spider or introduced natively.
var registry []command

// Run parses args and dispatches to the matching subcommand, or handles
// version/help directly. It never calls os.Exit itself, so it stays
// testable: callers pass the process's real streams and forward the
// returned code to os.Exit.
func Run(args []string, stdout, stderr io.Writer, version, commit string) int {
	if len(args) == 0 {
		printUsage(stdout)
		return exitUsage
	}

	switch args[0] {
	case "--version", "-v", "version":
		fmt.Fprintf(stdout, "spin %s (%s)\n", version, commit)
		return exitOK
	case "--help", "-h", "help":
		printUsage(stdout)
		return exitOK
	}

	for _, cmd := range registry {
		if cmd.name == args[0] {
			return cmd.run(args[1:], stdout, stderr, version, commit)
		}
	}

	fmt.Fprintf(stderr, "spin: unknown command %q\n", args[0])
	printUsage(stderr)
	return exitUsage
}

// printUsage writes the top-level command list to w.
func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: spin <command> [arguments]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	if len(registry) == 0 {
		fmt.Fprintln(w, "  (none yet)")
	}
	for _, cmd := range registry {
		fmt.Fprintf(w, "  %-12s %s\n", cmd.name, cmd.description)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  version      Print the spin version")
	fmt.Fprintln(w, "  help         Show this message")
}
