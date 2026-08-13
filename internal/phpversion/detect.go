package phpversion

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ReadComposerConstraint reads the "php" entry from projectDir's
// composer.json "require" block.
func ReadComposerConstraint(projectDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(projectDir, "composer.json"))
	if err != nil {
		return "", fmt.Errorf("reading composer.json: %w", err)
	}

	var parsed struct {
		Require map[string]string `json:"require"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", fmt.Errorf("parsing composer.json: %w", err)
	}

	constraint, ok := parsed.Require["php"]
	if !ok || strings.TrimSpace(constraint) == "" {
		return "", fmt.Errorf(`composer.json has no "php" entry under "require"`)
	}
	return constraint, nil
}

// Environment isolates the interpreter-detection side effect (running
// "php --version") behind an injectable seam, so tests never depend on
// what is actually installed on the machine running them.
type Environment struct {
	LookPath   func(file string) (string, error)
	RunVersion func(binary string) (string, error)
}

// RealEnvironment wires Environment to the actual host machine's php
// binary, if any.
func RealEnvironment() Environment {
	return Environment{
		LookPath: exec.LookPath,
		RunVersion: func(binary string) (string, error) {
			out, err := exec.Command(binary, "-r", "echo PHP_VERSION;").Output()
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(out)), nil
		},
	}
}

// DetectInstalled returns the version of the php interpreter on PATH, or
// an error if none is installed.
func DetectInstalled(env Environment) (Version, error) {
	path, err := env.LookPath("php")
	if err != nil {
		return Version{}, fmt.Errorf("no php interpreter found on PATH")
	}

	raw, err := env.RunVersion(path)
	if err != nil {
		return Version{}, fmt.Errorf("running %s: %w", path, err)
	}

	v, err := ParseVersion(raw)
	if err != nil {
		return Version{}, fmt.Errorf("parsing PHP version output %q: %w", raw, err)
	}
	return v, nil
}
