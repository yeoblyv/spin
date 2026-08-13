package cli

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/yeoblyv/spin/internal/phpversion"
)

func init() {
	registry = append(registry, command{
		name:        "php",
		description: "Check the PHP interpreter on PATH against a project's composer.json constraint",
		run:         runPHP,
	})
}

// runPHP reads the target project's composer.json "php" constraint and
// reports whether the php interpreter on PATH satisfies it, surfacing a
// clear, actionable error rather than a silent fallback when it does not.
func runPHP(args []string, stdout, stderr io.Writer, version, commit string) int {
	return checkPHP(args, stdout, stderr, phpversion.RealEnvironment())
}

// checkPHP is runPHP's testable core: env is injected so tests never
// depend on what PHP interpreter, if any, is actually installed on the
// machine running them.
func checkPHP(args []string, stdout, stderr io.Writer, env phpversion.Environment) int {
	fs := flag.NewFlagSet("php", flag.ContinueOnError)
	fs.SetOutput(stderr)
	site := fs.String("site", "", "registered site name (defaults to the current directory or the default site)")
	dir := fs.String("dir", "", "project directory (bypasses the site registry entirely)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	projectDir := *dir
	if projectDir == "" {
		r, _, err := openRegistry()
		if err != nil {
			fmt.Fprintf(stderr, "spin: php: %v\n", err)
			return exitError
		}
		cwd, err := os.Getwd()
		if err != nil {
			cwd = ""
		}
		entry, err := r.ResolveSite(*site, cwd)
		if err != nil {
			fmt.Fprintf(stderr, "spin: php: %v\n", err)
			return exitError
		}
		projectDir = entry.Path
	}

	constraintText, err := phpversion.ReadComposerConstraint(projectDir)
	if err != nil {
		fmt.Fprintf(stderr, "spin: php: %v\n", err)
		return exitError
	}
	constraint, err := phpversion.ParseConstraint(constraintText)
	if err != nil {
		fmt.Fprintf(stderr, "spin: php: %v\n", err)
		return exitError
	}

	installed, err := phpversion.DetectInstalled(env)
	if err != nil {
		fmt.Fprintf(stderr, "spin: php: %v - install a PHP satisfying %q first\n", err, constraintText)
		return exitError
	}

	if !constraint.Satisfies(installed) {
		fmt.Fprintf(stderr, "spin: php: installed PHP %s does not satisfy %q\n", installed, constraintText)
		return exitError
	}

	fmt.Fprintf(stdout, "PHP %s satisfies %q\n", installed, constraintText)
	return exitOK
}
