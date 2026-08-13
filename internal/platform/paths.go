// Package platform isolates the OS-specific filesystem conventions spin
// depends on behind a single small package per concern, so platform-specific
// code stays easy to find, test, and extend as more of it is needed.
package platform

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// ConfigDir returns the directory spin stores its persistent configuration
// in: $XDG_CONFIG_HOME/spin (or ~/.config/spin) on Linux and macOS,
// %APPDATA%\spin on Windows.
func ConfigDir() (string, error) {
	return configDir(runtime.GOOS, os.Getenv, os.UserHomeDir)
}

// configDir is ConfigDir's testable core: goos, getenv, and homeDir are
// injected so both branches are exercised in unit tests regardless of the
// host running them.
func configDir(goos string, getenv func(string) string, homeDir func() (string, error)) (string, error) {
	if goos == "windows" {
		base := getenv("APPDATA")
		if base == "" {
			return "", errors.New("%APPDATA% is not set")
		}
		return filepath.Join(base, "spin"), nil
	}

	if base := getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "spin"), nil
	}

	home, err := homeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "spin"), nil
}

// HostsFilePath returns the OS hosts file spin manages local-domain entries
// in: /etc/hosts on Linux and macOS, %WinDir%\System32\drivers\etc\hosts on
// Windows.
func HostsFilePath() string {
	return hostsFilePath(runtime.GOOS, os.Getenv)
}

// hostsFilePath is HostsFilePath's testable core: goos and getenv are
// injected so both branches are exercised in unit tests regardless of the
// host running them.
func hostsFilePath(goos string, getenv func(string) string) string {
	if goos != "windows" {
		return "/etc/hosts"
	}

	winDir := getenv("WINDIR")
	if winDir == "" {
		winDir = getenv("SystemRoot")
	}
	if winDir == "" {
		winDir = `C:\Windows`
	}
	return filepath.Join(winDir, "System32", "drivers", "etc", "hosts")
}
