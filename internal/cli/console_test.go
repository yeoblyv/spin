package cli

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrompt_matchesLogoColoredBracketsExactly(t *testing.T) {
	// Each glyph is immediately preceded by its own color code, so the
	// bracket characters are not contiguous in the output - this checks
	// the exact byte sequence prompt() is expected to build, rather than
	// a substring that the interleaved escape codes would break.
	want := promptColors[0] + "}" +
		promptColors[1] + "}" +
		promptColors[2] + "{" +
		promptColors[3] + "{" +
		colorReset + " "

	if got := prompt(); got != want {
		t.Errorf("prompt() = %q, want %q", got, want)
	}
}

func TestPrompt_glyphsAppearInBracketOrder(t *testing.T) {
	got := prompt()
	stripped := got
	for _, color := range promptColors {
		stripped = strings.ReplaceAll(stripped, color, "")
	}
	stripped = strings.ReplaceAll(stripped, colorReset, "")

	if want := "}}{{ "; stripped != want {
		t.Errorf("prompt() with color codes stripped = %q, want %q", stripped, want)
	}
}

func TestPrintBanner_reportsBothVersions(t *testing.T) {
	var out strings.Builder

	printBanner(&out, "1.2.3", "4.5.6")

	want := "SPIN/1.2.3 shell for Spider/4.5.6\n"
	if !strings.HasPrefix(out.String(), want) {
		t.Errorf("printBanner() wrote %q, want it to start with %q", out.String(), want)
	}
}

func TestRepl_dispatchesCommandsUntilExit(t *testing.T) {
	in := strings.NewReader("version\nbogus-command\nexit\n")
	var stdout, stderr strings.Builder

	code := repl(in, &stdout, &stderr, "1.2.3", "abc123")

	if code != exitOK {
		t.Fatalf("repl() code = %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout.String(), "spin 1.2.3 (abc123)") {
		t.Errorf("stdout = %q, want it to contain the version output from the \"version\" line", stdout.String())
	}
	if !strings.Contains(stderr.String(), "bogus-command") {
		t.Errorf("stderr = %q, want it to mention the unknown command from the \"bogus-command\" line", stderr.String())
	}
}

func TestRepl_endsCleanlyOnEOFWithoutExitCommand(t *testing.T) {
	in := strings.NewReader("version\n")
	var stdout, stderr strings.Builder

	code := repl(in, &stdout, &stderr, "1.2.3", "abc123")

	if code != exitOK {
		t.Fatalf("repl() code = %d, want %d", code, exitOK)
	}
}

func TestRepl_skipsBlankLines(t *testing.T) {
	in := strings.NewReader("\n\nversion\n\nexit\n")
	var stdout, stderr strings.Builder

	code := repl(in, &stdout, &stderr, "1.2.3", "abc123")

	if code != exitOK {
		t.Fatalf("repl() code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "spin 1.2.3 (abc123)") {
		t.Errorf("stdout = %q, want it to contain the version output", stdout.String())
	}
}

func TestDetectSpiderVersion_prefersGitTagOverComposerAndChangelog(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "composer.json"), `{"version": "0.0.1"}`)
	writeFile(t, filepath.Join(dir, "CHANGELOG.md"), "## [Unreleased]\n\n## [9.9.9] - 2020-01-01\n")

	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	writeFile(t, filepath.Join(dir, "README.md"), "test\n")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "initial")
	runGit(t, dir, "tag", "v2.5.0")

	if got := detectSpiderVersion(dir); got != "2.5.0" {
		t.Errorf("detectSpiderVersion() = %q, want %q", got, "2.5.0")
	}
}

func TestDetectSpiderVersion_fallsBackToComposerJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "composer.json"), `{"name": "yeoblyv/spider", "version": "1.4.2"}`)

	if got := detectSpiderVersion(dir); got != "1.4.2" {
		t.Errorf("detectSpiderVersion() = %q, want %q", got, "1.4.2")
	}
}

func TestDetectSpiderVersion_fallsBackToChangelog(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "CHANGELOG.md"), "## [Unreleased]\n\n### Added\n- something\n\n## [1.0.0] - 2026-08-02\n")

	if got := detectSpiderVersion(dir); got != "1.0.0" {
		t.Errorf("detectSpiderVersion() = %q, want %q", got, "1.0.0")
	}
}

func TestDetectSpiderVersion_unknownWhenNothingMatches(t *testing.T) {
	dir := t.TempDir()

	if got := detectSpiderVersion(dir); got != "unknown" {
		t.Errorf("detectSpiderVersion() = %q, want %q", got, "unknown")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
