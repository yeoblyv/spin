package platform

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestConfigDirWindowsUsesAppData(t *testing.T) {
	getenv := func(key string) string {
		if key == "APPDATA" {
			return `C:\Users\eve\AppData\Roaming`
		}
		return ""
	}
	homeDir := func() (string, error) { return "", errors.New("must not be called") }

	got, err := configDir("windows", getenv, homeDir)
	if err != nil {
		t.Fatalf("configDir() error = %v", err)
	}

	want := filepath.Join(`C:\Users\eve\AppData\Roaming`, "spin")
	if got != want {
		t.Errorf("configDir() = %q, want %q", got, want)
	}
}

func TestConfigDirWindowsMissingAppDataErrors(t *testing.T) {
	getenv := func(string) string { return "" }
	homeDir := func() (string, error) { return "", errors.New("must not be called") }

	if _, err := configDir("windows", getenv, homeDir); err == nil {
		t.Error("configDir() error = nil, want an error when %APPDATA% is unset")
	}
}

func TestConfigDirUnixPrefersXDGConfigHome(t *testing.T) {
	getenv := func(key string) string {
		if key == "XDG_CONFIG_HOME" {
			return "/home/eve/.xdgconfig"
		}
		return ""
	}
	homeDir := func() (string, error) { return "", errors.New("must not be called") }

	for _, goos := range []string{"linux", "darwin"} {
		got, err := configDir(goos, getenv, homeDir)
		if err != nil {
			t.Fatalf("configDir(%q) error = %v", goos, err)
		}

		want := filepath.Join("/home/eve/.xdgconfig", "spin")
		if got != want {
			t.Errorf("configDir(%q) = %q, want %q", goos, got, want)
		}
	}
}

func TestConfigDirUnixFallsBackToHomeConfig(t *testing.T) {
	getenv := func(string) string { return "" }
	homeDir := func() (string, error) { return "/home/eve", nil }

	for _, goos := range []string{"linux", "darwin"} {
		got, err := configDir(goos, getenv, homeDir)
		if err != nil {
			t.Fatalf("configDir(%q) error = %v", goos, err)
		}

		want := filepath.Join("/home/eve", ".config", "spin")
		if got != want {
			t.Errorf("configDir(%q) = %q, want %q", goos, got, want)
		}
	}
}

func TestConfigDirUnixPropagatesHomeDirError(t *testing.T) {
	getenv := func(string) string { return "" }
	homeDir := func() (string, error) { return "", errors.New("no home directory") }

	if _, err := configDir("linux", getenv, homeDir); err == nil {
		t.Error("configDir() error = nil, want the propagated homeDir error")
	}
}

func TestHostsFilePathUnix(t *testing.T) {
	getenv := func(string) string { return "" }

	for _, goos := range []string{"linux", "darwin"} {
		if got, want := hostsFilePath(goos, getenv), "/etc/hosts"; got != want {
			t.Errorf("hostsFilePath(%q) = %q, want %q", goos, got, want)
		}
	}
}

func TestHostsFilePathWindowsUsesWinDir(t *testing.T) {
	getenv := func(key string) string {
		if key == "WINDIR" {
			return `C:\Windows`
		}
		return ""
	}

	got := hostsFilePath("windows", getenv)
	want := filepath.Join(`C:\Windows`, "System32", "drivers", "etc", "hosts")
	if got != want {
		t.Errorf("hostsFilePath() = %q, want %q", got, want)
	}
}

func TestHostsFilePathWindowsFallsBackToSystemRoot(t *testing.T) {
	getenv := func(key string) string {
		if key == "SystemRoot" {
			return `C:\WINNT`
		}
		return ""
	}

	got := hostsFilePath("windows", getenv)
	want := filepath.Join(`C:\WINNT`, "System32", "drivers", "etc", "hosts")
	if got != want {
		t.Errorf("hostsFilePath() = %q, want %q", got, want)
	}
}

func TestHostsFilePathWindowsFallsBackToDefault(t *testing.T) {
	getenv := func(string) string { return "" }

	got := hostsFilePath("windows", getenv)
	want := filepath.Join(`C:\Windows`, "System32", "drivers", "etc", "hosts")
	if got != want {
		t.Errorf("hostsFilePath() = %q, want %q", got, want)
	}
}
