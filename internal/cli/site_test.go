package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolateRegistry points spin's config directory at a fresh temp directory
// for the duration of the test, so tests never touch the real machine's
// registry file.
func isolateRegistry(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
}

func newSpiderProjectDir(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", dir, err)
	}
	writeFile(t, filepath.Join(dir, "bootstrap.php"), "<?php\n")
	return dir
}

func TestRunSiteAddAndList(t *testing.T) {
	isolateRegistry(t)
	dir := newSpiderProjectDir(t, "blog")

	var stdout, stderr strings.Builder
	code := runSite([]string{"add", dir}, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitOK {
		t.Fatalf("site add code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	stdout.Reset()
	code = runSite([]string{"list"}, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitOK {
		t.Fatalf("site list code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "blog") {
		t.Errorf("site list output = %q, want it to mention %q", stdout.String(), "blog")
	}
}

func TestRunSiteAddRejectsNonSpiderDirectory(t *testing.T) {
	isolateRegistry(t)
	dir := t.TempDir()

	var stdout, stderr strings.Builder
	code := runSite([]string{"add", dir}, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitError {
		t.Fatalf("site add code = %d, want %d", code, exitError)
	}
}

func TestRunSiteRemove(t *testing.T) {
	isolateRegistry(t)
	dir := newSpiderProjectDir(t, "blog")

	var stdout, stderr strings.Builder
	if code := runSite([]string{"add", "--name", "blog", dir}, &stdout, &stderr, "1.2.3", "abc123"); code != exitOK {
		t.Fatalf("site add code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	stdout.Reset()
	code := runSite([]string{"remove", "blog"}, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitOK {
		t.Fatalf("site remove code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	stdout.Reset()
	code = runSite([]string{"list"}, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitOK {
		t.Fatalf("site list code = %d, want %d", code, exitOK)
	}
	if strings.Contains(stdout.String(), "blog") {
		t.Errorf("site list output = %q, want it to no longer mention removed site %q", stdout.String(), "blog")
	}
}

func TestRunSiteRemoveUnknownNameErrors(t *testing.T) {
	isolateRegistry(t)

	var stdout, stderr strings.Builder
	code := runSite([]string{"remove", "ghost"}, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitError {
		t.Fatalf("site remove code = %d, want %d", code, exitError)
	}
}

func TestRunSiteUseMarksDefaultInList(t *testing.T) {
	isolateRegistry(t)
	dir := newSpiderProjectDir(t, "blog")

	var stdout, stderr strings.Builder
	if code := runSite([]string{"add", "--name", "blog", dir}, &stdout, &stderr, "1.2.3", "abc123"); code != exitOK {
		t.Fatalf("site add code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	stdout.Reset()
	if code := runSite([]string{"use", "blog"}, &stdout, &stderr, "1.2.3", "abc123"); code != exitOK {
		t.Fatalf("site use code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	stdout.Reset()
	code := runSite([]string{"list"}, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitOK {
		t.Fatalf("site list code = %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout.String(), "* blog") {
		t.Errorf("site list output = %q, want the default site marked with \"*\"", stdout.String())
	}
}

func TestRunSiteNoSubcommandIsUsageError(t *testing.T) {
	isolateRegistry(t)

	var stdout, stderr strings.Builder
	code := runSite(nil, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitUsage {
		t.Fatalf("site (no args) code = %d, want %d", code, exitUsage)
	}
}

func TestRunSiteUnknownSubcommandIsUsageError(t *testing.T) {
	isolateRegistry(t)

	var stdout, stderr strings.Builder
	code := runSite([]string{"bogus"}, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitUsage {
		t.Fatalf("site bogus code = %d, want %d", code, exitUsage)
	}
}

func TestRunSiteRegisteredThroughTopLevelDispatch(t *testing.T) {
	isolateRegistry(t)

	var stdout, stderr strings.Builder
	code := Run([]string{"site", "list"}, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitOK {
		t.Fatalf("Run([site list]) code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "no sites registered") {
		t.Errorf("stdout = %q, want the empty-registry message", stdout.String())
	}
}
