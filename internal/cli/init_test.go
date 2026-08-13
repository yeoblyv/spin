package cli

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInit_copiesExampleAndGeneratesKey(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env.example"), "APP_ENV=dev\nAPP_KEY=\nLOG_LEVEL=debug\n")

	var stdout, stderr strings.Builder
	code := runInit([]string{"--dir", dir}, &stdout, &stderr, "1.2.3", "abc123")

	if code != exitOK {
		t.Fatalf("runInit() code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	env := readFile(t, filepath.Join(dir, ".env"))
	if !strings.Contains(env, "APP_ENV=dev") {
		t.Errorf(".env = %q, want it to contain the copied APP_ENV line", env)
	}

	key := extractAppKey(t, env)
	if key == "" {
		t.Fatal(".env APP_KEY is still empty, want a generated value")
	}
	assertValidAppKey(t, key)
}

func TestRunInit_leavesExistingEnvUntouched(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env.example"), "APP_KEY=\n")
	writeFile(t, filepath.Join(dir, ".env"), "APP_KEY=already-here\nCUSTOM=value\n")

	var stdout, stderr strings.Builder
	code := runInit([]string{"--dir", dir}, &stdout, &stderr, "1.2.3", "abc123")

	if code != exitOK {
		t.Fatalf("runInit() code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	env := readFile(t, filepath.Join(dir, ".env"))
	if env != "APP_KEY=already-here\nCUSTOM=value\n" {
		t.Errorf(".env = %q, want it unchanged since it already existed", env)
	}
}

func TestRunInit_leavesExistingAppKeyUntouched(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env.example"), "APP_KEY=\n")
	// .env doesn't exist yet, so it gets copied from .env.example - but the
	// example itself already carries a real-looking key here, and that
	// must survive the copy without being regenerated.
	writeFile(t, filepath.Join(dir, ".env.example"), "APP_KEY=pre-existing-key\n")

	var stdout, stderr strings.Builder
	code := runInit([]string{"--dir", dir}, &stdout, &stderr, "1.2.3", "abc123")

	if code != exitOK {
		t.Fatalf("runInit() code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	env := readFile(t, filepath.Join(dir, ".env"))
	if env != "APP_KEY=pre-existing-key\n" {
		t.Errorf(".env = %q, want the pre-existing key preserved", env)
	}
}

func TestRunInit_registersProjectInSiteRegistry(t *testing.T) {
	isolateRegistry(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env.example"), "APP_KEY=\n")
	writeFile(t, filepath.Join(dir, "bootstrap.php"), "<?php\n")

	var stdout, stderr strings.Builder
	code := runInit([]string{"--dir", dir}, &stdout, &stderr, "1.2.3", "abc123")

	if code != exitOK {
		t.Fatalf("runInit() code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "registered project") {
		t.Errorf("stdout = %q, want it to report the project was registered", stdout.String())
	}

	var listOut, listErr strings.Builder
	if code := runSite([]string{"list"}, &listOut, &listErr, "1.2.3", "abc123"); code != exitOK {
		t.Fatalf("site list code = %d, want %d; stderr = %q", code, exitOK, listErr.String())
	}
	if !strings.Contains(listOut.String(), dir) {
		t.Errorf("site list output = %q, want it to include %q", listOut.String(), dir)
	}
}

func TestRunInit_doesNotRegisterNonSpiderDirectory(t *testing.T) {
	isolateRegistry(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".env.example"), "APP_KEY=\n")

	var stdout, stderr strings.Builder
	code := runInit([]string{"--dir", dir}, &stdout, &stderr, "1.2.3", "abc123")

	if code != exitOK {
		t.Fatalf("runInit() code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}
	if strings.Contains(stdout.String(), "registered project") {
		t.Errorf("stdout = %q, want no registration message for a non-Spider directory", stdout.String())
	}
}

func TestRunInit_errorsWithoutEnvExample(t *testing.T) {
	dir := t.TempDir()

	var stdout, stderr strings.Builder
	code := runInit([]string{"--dir", dir}, &stdout, &stderr, "1.2.3", "abc123")

	if code != exitError {
		t.Fatalf("runInit() code = %d, want %d", code, exitError)
	}
	if !strings.Contains(stderr.String(), ".env.example") {
		t.Errorf("stderr = %q, want it to mention .env.example", stderr.String())
	}
}

func TestGenerateAppKey_producesDistinct32ByteKeys(t *testing.T) {
	a, err := generateAppKey()
	if err != nil {
		t.Fatalf("generateAppKey() error = %v", err)
	}
	b, err := generateAppKey()
	if err != nil {
		t.Fatalf("generateAppKey() error = %v", err)
	}

	if a == b {
		t.Error("generateAppKey() produced the same value twice in a row")
	}
	assertValidAppKey(t, a)
	assertValidAppKey(t, b)
}

func assertValidAppKey(t *testing.T, key string) {
	t.Helper()

	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		t.Fatalf("APP_KEY %q is not valid base64: %v", key, err)
	}
	if len(decoded) != 32 {
		t.Errorf("APP_KEY %q decodes to %d bytes, want 32", key, len(decoded))
	}
}

func extractAppKey(t *testing.T, env string) string {
	t.Helper()

	for _, line := range strings.Split(env, "\n") {
		if strings.HasPrefix(line, "APP_KEY=") {
			return strings.TrimPrefix(line, "APP_KEY=")
		}
	}
	t.Fatalf("env %q has no APP_KEY line", env)
	return ""
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(contents)
}
