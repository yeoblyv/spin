package cli

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func init() {
	registry = append(registry, command{
		name:        "console",
		description: "Start an interactive shell for running spin commands",
		run:         runConsole,
	})
}

// runConsole starts an interactive, line-at-a-time shell: each line entered
// is dispatched exactly like a top-level invocation of spin (so "init"
// typed at the prompt behaves identically to running "spin init"), until
// "exit"/"quit" or end of input.
func runConsole(args []string, stdout, stderr io.Writer, version, commit string) int {
	fs := flag.NewFlagSet("console", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "Spider project directory")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	printBanner(stdout, version, detectSpiderVersion(*dir))
	return repl(os.Stdin, stdout, stderr, version, commit)
}

// printBanner writes the shell's startup banner, in the style of a MySQL or
// Asterisk console: the running tool's own version and the version of the
// Spider project it is pointed at.
func printBanner(w io.Writer, spinVersion, spiderVersion string) {
	fmt.Fprintf(w, "SPIN/%s shell for Spider/%s\n", spinVersion, spiderVersion)
	fmt.Fprintln(w, `Type "help" for a list of commands, "exit" to leave.`)
	fmt.Fprintln(w)
}

// repl reads one command per line from in until "exit"/"quit" or EOF,
// dispatching each through Run exactly as if it were spin's top-level
// argument list. A failed command prints its exit code and keeps the
// session open - only "exit"/"quit"/EOF ends it.
func repl(in io.Reader, stdout, stderr io.Writer, version, commit string) int {
	scanner := bufio.NewScanner(in)
	for {
		fmt.Fprint(stdout, prompt())

		if !scanner.Scan() {
			fmt.Fprintln(stdout)
			return exitOK
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			return exitOK
		}

		if code := Run(strings.Fields(line), stdout, stderr, version, commit); code != exitOK {
			fmt.Fprintf(stderr, "(exit code %d)\n", code)
		}
	}
}

// promptColors are the four gradient stops from Spider's own logo - blue,
// green, yellow, red - as 24-bit ANSI foreground escape codes, applied one
// per bracket glyph below. This renders correctly in terminals with
// truecolor support (Terminal.app, iTerm2, most modern Linux/Windows
// terminals); on one without it, the escape codes are typically ignored and
// the brackets print uncolored rather than garbled.
var promptColors = [4]string{
	"\x1b[38;2;42;0;255m",  // #2a00ff
	"\x1b[38;2;0;255;0m",   // #00ff00
	"\x1b[38;2;255;255;0m", // #ffff00
	"\x1b[38;2;255;0;0m",   // #ff0000
}

const colorReset = "\x1b[0m"

// prompt renders spin's interactive prompt: "}}{{ ", each of the four
// brackets colored with one of promptColors, in order.
func prompt() string {
	glyphs := [4]string{"}", "}", "{", "{"}

	var b strings.Builder
	for i, glyph := range glyphs {
		b.WriteString(promptColors[i])
		b.WriteString(glyph)
	}
	b.WriteString(colorReset)
	b.WriteString(" ")
	return b.String()
}

// detectSpiderVersion looks for the version of the Spider project in dir,
// trying progressively less authoritative sources: the latest git tag,
// then composer.json's "version" field, then the newest release heading in
// CHANGELOG.md. It returns "unknown" if none of them yield a version -
// dir not being a Spider project at all is not this function's concern to
// report; callers that care already validate that elsewhere.
func detectSpiderVersion(dir string) string {
	if v := versionFromGitTag(dir); v != "" {
		return v
	}
	if v := versionFromComposerJSON(dir); v != "" {
		return v
	}
	if v := versionFromChangelog(dir); v != "" {
		return v
	}
	return "unknown"
}

// versionFromGitTag returns the most recent reachable git tag in dir, with
// a leading "v" stripped, or "" if dir has no tags or is not a git
// repository.
func versionFromGitTag(dir string) string {
	out, err := exec.Command("git", "-C", dir, "describe", "--tags", "--abbrev=0").Output()
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(string(out)), "v")
}

// versionFromComposerJSON returns dir/composer.json's "version" field, or
// "" if the file is missing, unparsable, or has no version set.
func versionFromComposerJSON(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "composer.json"))
	if err != nil {
		return ""
	}

	var parsed struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return ""
	}
	return strings.TrimSpace(parsed.Version)
}

// changelogHeading matches a Keep a Changelog release heading such as
// "## [1.0.0] - 2026-08-02", capturing the version - but not "[Unreleased]",
// which has no digits for \d+ to match.
var changelogHeading = regexp.MustCompile(`^## \[(\d+\.\d+\.\d+[^\]]*)\]`)

// versionFromChangelog returns the first (most recent) released version
// heading in dir/CHANGELOG.md, or "" if the file is missing or has none.
func versionFromChangelog(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(data), "\n") {
		if m := changelogHeading.FindStringSubmatch(line); m != nil {
			return m[1]
		}
	}
	return ""
}
