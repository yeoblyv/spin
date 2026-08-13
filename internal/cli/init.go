package cli

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	reg "github.com/yeoblyv/spin/internal/registry"
)

func init() {
	registry = append(registry, command{
		name:        "init",
		description: "Copy .env.example to .env and generate APP_KEY if missing",
		run:         runInit,
	})
}

// runInit bootstraps a freshly cloned Spider project's environment file: it
// copies .env.example to .env if .env does not exist yet, then generates a
// fresh APP_KEY if the one in .env is empty. Neither step ever overwrites
// an existing value - a real .env or a real key already in place is always
// left untouched. version and commit are unused here but are part of every
// command's signature; see the command type in cli.go.
func runInit(args []string, stdout, stderr io.Writer, version, commit string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "project directory to initialize")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	envPath := filepath.Join(*dir, ".env")
	examplePath := filepath.Join(*dir, ".env.example")

	if err := ensureEnvFile(envPath, examplePath, stdout); err != nil {
		fmt.Fprintf(stderr, "spin: init: %v\n", err)
		return exitError
	}

	if err := ensureAppKey(envPath, stdout); err != nil {
		fmt.Fprintf(stderr, "spin: init: %v\n", err)
		return exitError
	}

	registerProject(*dir, stdout)

	return exitOK
}

// registerProject adds dir to the site registry when it looks like a
// Spider project and isn't already known, so a single "spin init" both
// prepares and registers a project. Every failure here is silently
// non-fatal: init's own job already succeeded, and a bare .env/APP_KEY
// bootstrap with no registry present at all remains a valid standalone use
// case (e.g. in CI).
func registerProject(dir string, stdout io.Writer) {
	absDir, err := filepath.Abs(dir)
	if err != nil || !reg.IsSpiderProject(absDir) {
		return
	}

	path, err := registryPath()
	if err != nil {
		return
	}

	r, err := reg.Load(path)
	if err != nil {
		return
	}
	if _, exists := r.FindByPath(absDir); exists {
		return
	}

	entry, err := r.Add("", absDir)
	if err != nil {
		return
	}
	if err := r.SetDomain(entry.Name, entry.Name+"."+defaultTLD); err != nil {
		return
	}
	if err := r.Save(path); err != nil {
		return
	}

	fmt.Fprintf(stdout, "registered project %q in the site registry\n", entry.Name)
}

// ensureEnvFile copies examplePath to envPath when envPath does not exist
// yet. It refuses to run at all when examplePath is missing, since that
// means the target directory is not a Spider project.
func ensureEnvFile(envPath, examplePath string, stdout io.Writer) error {
	if _, err := os.Stat(envPath); err == nil {
		fmt.Fprintf(stdout, "%s already exists, leaving it untouched\n", envPath)
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking %s: %w", envPath, err)
	}

	contents, err := os.ReadFile(examplePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s not found - run this from a Spider project's root", examplePath)
		}
		return fmt.Errorf("reading %s: %w", examplePath, err)
	}

	if err := os.WriteFile(envPath, contents, 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", envPath, err)
	}

	fmt.Fprintf(stdout, "copied %s to %s\n", examplePath, envPath)
	return nil
}

// ensureAppKey generates a fresh APP_KEY - base64-encoded, 32 cryptographically
// random bytes, matching Spider's own `base64_encode(random_bytes(32))`
// convention - and writes it into envPath's APP_KEY line, but only when
// that line is currently empty. An already-set key is never touched: it may
// already protect real encrypted data.
func ensureAppKey(envPath string, stdout io.Writer) error {
	lines, keyLine, err := readAppKeyLine(envPath)
	if err != nil {
		return err
	}
	if keyLine == -1 {
		fmt.Fprintf(stdout, "no APP_KEY line found in %s, skipping\n", envPath)
		return nil
	}

	if value := strings.TrimSpace(strings.TrimPrefix(lines[keyLine], "APP_KEY=")); value != "" {
		fmt.Fprintf(stdout, "APP_KEY is already set, leaving it untouched\n")
		return nil
	}

	key, err := generateAppKey()
	if err != nil {
		return fmt.Errorf("generating APP_KEY: %w", err)
	}

	lines[keyLine] = "APP_KEY=" + key
	if err := os.WriteFile(envPath, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", envPath, err)
	}

	fmt.Fprintf(stdout, "generated a new APP_KEY\n")
	return nil
}

// generateAppKey returns 32 cryptographically random bytes, base64-encoded -
// the same format Spider's own quickstart documents generating via
// `php -r "echo base64_encode(random_bytes(32));"`.
func generateAppKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

// readAppKeyLine reads envPath and returns its lines along with the index
// of the line starting with "APP_KEY=", or -1 if there is none.
func readAppKeyLine(envPath string) ([]string, int, error) {
	contents, err := os.ReadFile(envPath)
	if err != nil {
		return nil, -1, fmt.Errorf("reading %s: %w", envPath, err)
	}

	lines := strings.Split(string(contents), "\n")
	// A trailing newline produces one empty trailing element; drop it so
	// writing the file back with a single appended "\n" round-trips exactly
	// instead of accumulating a blank line on every run.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	for i, line := range lines {
		if strings.HasPrefix(line, "APP_KEY=") {
			return lines, i, nil
		}
	}
	return lines, -1, nil
}
