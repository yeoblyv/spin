package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSiteUpWritesNginxConfig(t *testing.T) {
	isolateRegistry(t)
	dir := newSpiderProjectDir(t, "blog")
	confDir := t.TempDir()

	var stdout, stderr strings.Builder
	if code := runSite([]string{"add", "--name", "blog", dir}, &stdout, &stderr, "1.2.3", "abc123"); code != exitOK {
		t.Fatalf("site add code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	stdout.Reset()
	code := runSite([]string{"up", "--site", "blog", "--server", "nginx", "--conf-dir", confDir}, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitOK {
		t.Fatalf("site up code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	outPath := filepath.Join(confDir, "blog.test.conf")
	content := readFile(t, outPath)
	if !strings.Contains(content, "server_name blog.test;") {
		t.Errorf("rendered config = %q, want it to set server_name blog.test", content)
	}
	if !strings.Contains(content, "root "+dir+"/Public;") {
		t.Errorf("rendered config = %q, want root under %s", content, dir)
	}
}

func TestRunSiteUpWritesApacheConfig(t *testing.T) {
	isolateRegistry(t)
	dir := newSpiderProjectDir(t, "blog")
	confDir := t.TempDir()

	var stdout, stderr strings.Builder
	if code := runSite([]string{"add", "--name", "blog", dir}, &stdout, &stderr, "1.2.3", "abc123"); code != exitOK {
		t.Fatalf("site add code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	stdout.Reset()
	code := runSite([]string{"up", "--site", "blog", "--server", "apache", "--conf-dir", confDir}, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitOK {
		t.Fatalf("site up code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	outPath := filepath.Join(confDir, "blog.test.conf")
	content := readFile(t, outPath)
	if !strings.Contains(content, "ServerName blog.test") {
		t.Errorf("rendered config = %q, want it to set ServerName blog.test", content)
	}
}

func TestRunSiteUpUnknownSiteErrors(t *testing.T) {
	isolateRegistry(t)

	var stdout, stderr strings.Builder
	code := runSite([]string{"up", "--site", "ghost", "--server", "nginx", "--conf-dir", t.TempDir()}, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitError {
		t.Fatalf("site up code = %d, want %d", code, exitError)
	}
}

func TestRunSiteDownRemovesConfig(t *testing.T) {
	isolateRegistry(t)
	dir := newSpiderProjectDir(t, "blog")
	confDir := t.TempDir()

	var stdout, stderr strings.Builder
	if code := runSite([]string{"add", "--name", "blog", dir}, &stdout, &stderr, "1.2.3", "abc123"); code != exitOK {
		t.Fatalf("site add code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}
	stdout.Reset()
	if code := runSite([]string{"up", "--site", "blog", "--server", "nginx", "--conf-dir", confDir}, &stdout, &stderr, "1.2.3", "abc123"); code != exitOK {
		t.Fatalf("site up code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	outPath := filepath.Join(confDir, "blog.test.conf")
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("Stat(%s) error = %v, want the config to exist after site up", outPath, err)
	}

	stdout.Reset()
	code := runSite([]string{"down", "--site", "blog", "--server", "nginx", "--conf-dir", confDir}, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitOK {
		t.Fatalf("site down code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Errorf("Stat(%s) error = %v, want the config removed after site down", outPath, err)
	}
}

func TestRunSiteDownMissingConfigIsNotAnError(t *testing.T) {
	isolateRegistry(t)
	dir := newSpiderProjectDir(t, "blog")
	confDir := t.TempDir()

	var stdout, stderr strings.Builder
	if code := runSite([]string{"add", "--name", "blog", dir}, &stdout, &stderr, "1.2.3", "abc123"); code != exitOK {
		t.Fatalf("site add code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	stdout.Reset()
	code := runSite([]string{"down", "--site", "blog", "--server", "nginx", "--conf-dir", confDir}, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitOK {
		t.Fatalf("site down code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}
}
