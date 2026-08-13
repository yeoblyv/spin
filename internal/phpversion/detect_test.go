package phpversion

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadComposerConstraint(t *testing.T) {
	dir := t.TempDir()
	composer := `{"require": {"php": ">=8.2", "ext-mbstring": "*"}}`
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(composer), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := ReadComposerConstraint(dir)
	if err != nil {
		t.Fatalf("ReadComposerConstraint() error = %v", err)
	}
	if got != ">=8.2" {
		t.Errorf("ReadComposerConstraint() = %q, want %q", got, ">=8.2")
	}
}

func TestReadComposerConstraintMissingPHPEntry(t *testing.T) {
	dir := t.TempDir()
	composer := `{"require": {"ext-mbstring": "*"}}`
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(composer), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := ReadComposerConstraint(dir); err == nil {
		t.Error("ReadComposerConstraint() error = nil, want an error when no php entry is present")
	}
}

func TestReadComposerConstraintMissingFile(t *testing.T) {
	if _, err := ReadComposerConstraint(t.TempDir()); err == nil {
		t.Error("ReadComposerConstraint() error = nil, want an error when composer.json is missing")
	}
}

func TestDetectInstalled(t *testing.T) {
	env := Environment{
		LookPath:   func(string) (string, error) { return "/usr/bin/php", nil },
		RunVersion: func(string) (string, error) { return "8.2.10", nil },
	}

	got, err := DetectInstalled(env)
	if err != nil {
		t.Fatalf("DetectInstalled() error = %v", err)
	}
	want := Version{Major: 8, Minor: 2, Patch: 10}
	if got != want {
		t.Errorf("DetectInstalled() = %+v, want %+v", got, want)
	}
}

func TestDetectInstalledNoInterpreterOnPath(t *testing.T) {
	env := Environment{
		LookPath:   func(string) (string, error) { return "", errors.New("not found") },
		RunVersion: func(string) (string, error) { return "", errors.New("must not be called") },
	}

	if _, err := DetectInstalled(env); err == nil {
		t.Error("DetectInstalled() error = nil, want an error when no php is on PATH")
	}
}

func TestDetectInstalledRunVersionFails(t *testing.T) {
	env := Environment{
		LookPath:   func(string) (string, error) { return "/usr/bin/php", nil },
		RunVersion: func(string) (string, error) { return "", errors.New("boom") },
	}

	if _, err := DetectInstalled(env); err == nil {
		t.Error("DetectInstalled() error = nil, want an error when running php fails")
	}
}
