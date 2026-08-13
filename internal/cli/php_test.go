package cli

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeoblyv/spin/internal/phpversion"
)

func newComposerProjectDir(t *testing.T, phpConstraint string) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "bootstrap.php"), "<?php\n")
	composer := `{"require": {"php": "` + phpConstraint + `"}}`
	writeFile(t, filepath.Join(dir, "composer.json"), composer)
	return dir
}

func fakePHPEnvironment(installed string) phpversion.Environment {
	return phpversion.Environment{
		LookPath:   func(string) (string, error) { return "/usr/bin/php", nil },
		RunVersion: func(string) (string, error) { return installed, nil },
	}
}

func TestCheckPHPSatisfiedConstraint(t *testing.T) {
	dir := newComposerProjectDir(t, ">=8.2")

	var stdout, stderr strings.Builder
	code := checkPHP([]string{"--dir", dir}, &stdout, &stderr, fakePHPEnvironment("8.3.0"))

	if code != exitOK {
		t.Fatalf("checkPHP() code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "8.3.0") {
		t.Errorf("stdout = %q, want it to mention the installed version", stdout.String())
	}
}

func TestCheckPHPUnsatisfiedConstraint(t *testing.T) {
	dir := newComposerProjectDir(t, ">=8.2")

	var stdout, stderr strings.Builder
	code := checkPHP([]string{"--dir", dir}, &stdout, &stderr, fakePHPEnvironment("8.1.0"))

	if code != exitError {
		t.Fatalf("checkPHP() code = %d, want %d", code, exitError)
	}
	if !strings.Contains(stderr.String(), "does not satisfy") {
		t.Errorf("stderr = %q, want it to explain the mismatch", stderr.String())
	}
}

func TestCheckPHPNoInterpreterInstalled(t *testing.T) {
	dir := newComposerProjectDir(t, ">=8.2")
	env := phpversion.Environment{
		LookPath:   func(string) (string, error) { return "", errors.New("not found") },
		RunVersion: func(string) (string, error) { return "", errors.New("must not be called") },
	}

	var stdout, stderr strings.Builder
	code := checkPHP([]string{"--dir", dir}, &stdout, &stderr, env)

	if code != exitError {
		t.Fatalf("checkPHP() code = %d, want %d", code, exitError)
	}
	if !strings.Contains(stderr.String(), ">=8.2") {
		t.Errorf("stderr = %q, want it to mention the required constraint", stderr.String())
	}
}

func TestCheckPHPMissingComposerJSON(t *testing.T) {
	dir := t.TempDir()

	var stdout, stderr strings.Builder
	code := checkPHP([]string{"--dir", dir}, &stdout, &stderr, fakePHPEnvironment("8.3.0"))

	if code != exitError {
		t.Fatalf("checkPHP() code = %d, want %d", code, exitError)
	}
}

func TestCheckPHPUsesRegistryWhenNoDirGiven(t *testing.T) {
	isolateRegistry(t)
	dir := newComposerProjectDir(t, ">=8.2")

	var stdout, stderr strings.Builder
	if code := runSite([]string{"add", "--name", "blog", dir}, &stdout, &stderr, "1.2.3", "abc123"); code != exitOK {
		t.Fatalf("site add code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	stdout.Reset()
	code := checkPHP([]string{"--site", "blog"}, &stdout, &stderr, fakePHPEnvironment("8.3.0"))
	if code != exitOK {
		t.Fatalf("checkPHP() code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}
}
