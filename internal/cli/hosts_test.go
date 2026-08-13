package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunHostsApplyWritesRegisteredDomains(t *testing.T) {
	isolateRegistry(t)
	dir := newSpiderProjectDir(t, "blog")

	var stdout, stderr strings.Builder
	if code := runSite([]string{"add", "--name", "blog", dir}, &stdout, &stderr, "1.2.3", "abc123"); code != exitOK {
		t.Fatalf("site add code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	hostsPath := filepath.Join(t.TempDir(), "hosts")
	writeFile(t, hostsPath, "127.0.0.1 localhost\n")

	stdout.Reset()
	code := runHosts([]string{"apply", "--hosts-file", hostsPath}, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitOK {
		t.Fatalf("hosts apply code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	content := readFile(t, hostsPath)
	if !strings.Contains(content, "blog.test") {
		t.Errorf("hosts file = %q, want it to contain blog.test", content)
	}
	if !strings.Contains(content, "127.0.0.1 localhost") {
		t.Errorf("hosts file = %q, want the pre-existing line preserved", content)
	}
}

func TestRunHostsApplyDryRunDoesNotWrite(t *testing.T) {
	isolateRegistry(t)
	dir := newSpiderProjectDir(t, "blog")

	var stdout, stderr strings.Builder
	if code := runSite([]string{"add", "--name", "blog", dir}, &stdout, &stderr, "1.2.3", "abc123"); code != exitOK {
		t.Fatalf("site add code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	hostsPath := filepath.Join(t.TempDir(), "hosts")
	writeFile(t, hostsPath, "127.0.0.1 localhost\n")

	stdout.Reset()
	code := runHosts([]string{"apply", "--hosts-file", hostsPath, "--dry-run"}, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitOK {
		t.Fatalf("hosts apply --dry-run code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "blog.test") {
		t.Errorf("stdout = %q, want the rendered block to contain blog.test", stdout.String())
	}

	content := readFile(t, hostsPath)
	if content != "127.0.0.1 localhost\n" {
		t.Errorf("hosts file = %q, want it unchanged by a dry run", content)
	}
}

func TestRunHostsRemoveStripsBlock(t *testing.T) {
	hostsPath := filepath.Join(t.TempDir(), "hosts")
	writeFile(t, hostsPath, "127.0.0.1 localhost\n# spin:begin - managed by spin, do not edit by hand\n127.0.0.1 blog.test\n# spin:end\n")

	var stdout, stderr strings.Builder
	code := runHosts([]string{"remove", "--hosts-file", hostsPath}, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitOK {
		t.Fatalf("hosts remove code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	content := readFile(t, hostsPath)
	if strings.Contains(content, "blog.test") {
		t.Errorf("hosts file = %q, want the block removed", content)
	}
	if !strings.Contains(content, "127.0.0.1 localhost") {
		t.Errorf("hosts file = %q, want the pre-existing line preserved", content)
	}
}

func TestRunHostsApplyPermissionDeniedReportsClearError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses POSIX permission checks")
	}

	hostsPath := filepath.Join(t.TempDir(), "hosts")
	writeFile(t, hostsPath, "127.0.0.1 localhost\n")
	// os.WriteFile on an existing file needs write permission on the file
	// itself, not its directory - unlike creating a new entry.
	if err := os.Chmod(hostsPath, 0o400); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}

	isolateRegistry(t)

	var stdout, stderr strings.Builder
	code := runHosts([]string{"apply", "--hosts-file", hostsPath}, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitError {
		t.Fatalf("hosts apply code = %d, want %d", code, exitError)
	}
	if !strings.Contains(stderr.String(), "elevated privileges") {
		t.Errorf("stderr = %q, want it to explain that elevated privileges are needed", stderr.String())
	}
}

func TestRunHostsNoSubcommandIsUsageError(t *testing.T) {
	var stdout, stderr strings.Builder
	code := runHosts(nil, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitUsage {
		t.Fatalf("hosts (no args) code = %d, want %d", code, exitUsage)
	}
}
