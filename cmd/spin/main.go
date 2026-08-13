// Command spin is the cross-platform installer, setup, and console
// replacement for Spider's PHP-based bin/spider entry point.
package main

import (
	"os"

	"github.com/yeoblyv/spin/internal/cli"
)

// version and commit are the module's only self-description: go.mod carries
// no version field for the module itself, so these are left at their
// development defaults here and overwritten at build time via
//
//	go build -ldflags "-X main.version=1.2.3 -X main.commit=<sha>"
//
// The git tag that triggers a release build is the single source of truth;
// see scripts/release for how it flows into this binary.
var (
	version = "dev"
	commit  = "none"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, version, commit))
}
