package cli

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeoblyv/spin/internal/webserver"
)

func fakeWebserverEnvironment(installed bool) webserver.Environment {
	return webserver.Environment{
		LookPath: func(file string) (string, error) {
			if installed && file == "nginx" {
				return "/usr/bin/nginx", nil
			}
			return "", errors.New("not found")
		},
		BrewPrefix: func() (string, bool) { return "", false },
		DirExists:  func(string) bool { return false },
		Glob:       func(string) ([]string, error) { return nil, nil },
		IsSocket:   func(string) bool { return false },
	}
}

func TestDiagnoseAllPass(t *testing.T) {
	isolateRegistry(t)
	dir := newSpiderProjectDir(t, "blog")
	writeFile(t, filepath.Join(dir, "composer.json"), `{"require": {"php": ">=8.2"}}`)

	var stdout, stderr strings.Builder
	if code := runSite([]string{"add", "--name", "blog", dir}, &stdout, &stderr, "1.2.3", "abc123"); code != exitOK {
		t.Fatalf("site add code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	hostsPath := filepath.Join(t.TempDir(), "hosts")
	writeFile(t, hostsPath, "127.0.0.1 blog.test\n")

	checks := diagnose("blog", hostsPath, fakeWebserverEnvironment(true), fakePHPEnvironment("8.3.0"))

	if len(checks) != 4 {
		t.Fatalf("len(diagnose()) = %d, want 4", len(checks))
	}
	for _, c := range checks {
		if !c.ok {
			t.Errorf("check %q failed unexpectedly: %s", c.name, c.detail)
		}
	}
}

func TestDiagnoseUnknownSiteIsTerminal(t *testing.T) {
	isolateRegistry(t)

	checks := diagnose("ghost", "/nonexistent", fakeWebserverEnvironment(true), fakePHPEnvironment("8.3.0"))

	if len(checks) != 1 {
		t.Fatalf("len(diagnose()) = %d, want 1 (site resolution is terminal)", len(checks))
	}
	if checks[0].name != "site" || checks[0].ok {
		t.Errorf("checks[0] = %+v, want a failing site check", checks[0])
	}
	if checks[0].fix == "" {
		t.Error("checks[0].fix is empty, want the next command to run")
	}
}

func TestDiagnoseFlagsMissingWebServer(t *testing.T) {
	isolateRegistry(t)
	dir := newSpiderProjectDir(t, "blog")
	writeFile(t, filepath.Join(dir, "composer.json"), `{"require": {"php": ">=8.2"}}`)

	var stdout, stderr strings.Builder
	if code := runSite([]string{"add", "--name", "blog", dir}, &stdout, &stderr, "1.2.3", "abc123"); code != exitOK {
		t.Fatalf("site add code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	checks := diagnose("blog", "/nonexistent", fakeWebserverEnvironment(false), fakePHPEnvironment("8.3.0"))

	webCheck := findCheck(t, checks, "web server")
	if webCheck.ok {
		t.Error("web server check ok = true, want false when neither nginx nor Apache is detected")
	}
}

func TestDiagnoseFlagsPHPMismatch(t *testing.T) {
	isolateRegistry(t)
	dir := newSpiderProjectDir(t, "blog")
	writeFile(t, filepath.Join(dir, "composer.json"), `{"require": {"php": ">=8.2"}}`)

	var stdout, stderr strings.Builder
	if code := runSite([]string{"add", "--name", "blog", dir}, &stdout, &stderr, "1.2.3", "abc123"); code != exitOK {
		t.Fatalf("site add code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	checks := diagnose("blog", "/nonexistent", fakeWebserverEnvironment(true), fakePHPEnvironment("8.1.0"))

	phpCheck := findCheck(t, checks, "php")
	if phpCheck.ok {
		t.Error("php check ok = true, want false for an unsatisfied constraint")
	}
	if phpCheck.fix == "" {
		t.Error("php check fix is empty, want the next command to run")
	}
}

func TestDiagnoseFlagsMissingDomainInHosts(t *testing.T) {
	isolateRegistry(t)
	dir := newSpiderProjectDir(t, "blog")
	writeFile(t, filepath.Join(dir, "composer.json"), `{"require": {"php": ">=8.2"}}`)

	var stdout, stderr strings.Builder
	if code := runSite([]string{"add", "--name", "blog", dir}, &stdout, &stderr, "1.2.3", "abc123"); code != exitOK {
		t.Fatalf("site add code = %d, want %d; stderr = %q", code, exitOK, stderr.String())
	}

	emptyHosts := filepath.Join(t.TempDir(), "hosts")
	writeFile(t, emptyHosts, "127.0.0.1 localhost\n")

	checks := diagnose("blog", emptyHosts, fakeWebserverEnvironment(true), fakePHPEnvironment("8.3.0"))

	domainCheck := findCheck(t, checks, "domain")
	if domainCheck.ok {
		t.Error("domain check ok = true, want false when the domain is not in the hosts file")
	}
	if domainCheck.fix != "spin hosts apply" {
		t.Errorf("domain check fix = %q, want %q", domainCheck.fix, "spin hosts apply")
	}
}

func TestRunDoctorExitsNonZeroOnFailure(t *testing.T) {
	isolateRegistry(t)

	var stdout, stderr strings.Builder
	code := runDoctor([]string{"--site", "ghost"}, &stdout, &stderr, "1.2.3", "abc123")
	if code != exitError {
		t.Fatalf("runDoctor() code = %d, want %d", code, exitError)
	}
	if !strings.Contains(stdout.String(), "[FAIL]") {
		t.Errorf("stdout = %q, want it to contain a FAIL line", stdout.String())
	}
}

func findCheck(t *testing.T, checks []doctorCheck, name string) doctorCheck {
	t.Helper()
	for _, c := range checks {
		if c.name == name {
			return c
		}
	}
	t.Fatalf("no check named %q in %+v", name, checks)
	return doctorCheck{}
}
