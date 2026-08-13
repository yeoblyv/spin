package webserver

import (
	"errors"
	"regexp"
	"strings"
	"testing"
)

func TestRenderNginxSubstitutesAllPlaceholders(t *testing.T) {
	got, err := Render(Nginx, RenderParams{
		ServerName:  "blog.test",
		HTTPPort:    "80",
		ProjectRoot: "/srv/blog",
		AccessLog:   "/var/log/nginx/blog.test-access.log",
		ErrorLog:    "/var/log/nginx/blog.test-error.log",
		FastCGIPass: "unix:/var/run/php-fpm.sock",
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	for _, want := range []string{
		"server_name blog.test;",
		"listen 80;",
		"root /srv/blog/Public;",
		"access_log /var/log/nginx/blog.test-access.log;",
		"error_log  /var/log/nginx/blog.test-error.log;",
		"fastcgi_pass unix:/var/run/php-fpm.sock;",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Render(Nginx) = %q, want it to contain %q", got, want)
		}
	}
	assertNoUnresolvedPlaceholders(t, got)
}

func TestRenderApacheSubstitutesAllPlaceholders(t *testing.T) {
	got, err := Render(Apache, RenderParams{
		ServerName:  "blog.test",
		HTTPPort:    "80",
		ProjectRoot: "/srv/blog",
		AccessLog:   "/var/log/httpd/blog.test-access.log",
		ErrorLog:    "/var/log/httpd/blog.test-error.log",
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	for _, want := range []string{
		"ServerName blog.test",
		"<VirtualHost *:80>",
		"DocumentRoot /srv/blog/Public",
		"ErrorLog /var/log/httpd/blog.test-error.log",
		"CustomLog /var/log/httpd/blog.test-access.log combined",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Render(Apache) = %q, want it to contain %q", got, want)
		}
	}
	assertNoUnresolvedPlaceholders(t, got)
}

// assertNoUnresolvedPlaceholders checks for leftover {{TOKEN}}-shaped
// placeholders, not the literal string "{{" - the templates' own header
// comments use that syntax descriptively to explain the convention, which
// is expected to survive rendering unchanged.
func assertNoUnresolvedPlaceholders(t *testing.T, rendered string) {
	t.Helper()
	if placeholderPattern.MatchString(rendered) {
		t.Errorf("rendered output = %q, want no unresolved {{TOKEN}} placeholders", rendered)
	}
}

var placeholderPattern = regexp.MustCompile(`\{\{[A-Z_]+\}\}`)

func TestRenderUnknownKindErrors(t *testing.T) {
	if _, err := Render(Kind("caddy"), RenderParams{}); err == nil {
		t.Error("Render() error = nil, want an error for an unknown kind")
	}
}

func fakeEnvironment() Environment {
	return Environment{
		LookPath:   func(string) (string, error) { return "", errors.New("not found") },
		BrewPrefix: func() (string, bool) { return "", false },
		DirExists:  func(string) bool { return false },
		Glob:       func(string) ([]string, error) { return nil, nil },
		IsSocket:   func(string) bool { return false },
	}
}

func TestDetectPrefersNginx(t *testing.T) {
	env := fakeEnvironment()
	env.LookPath = func(file string) (string, error) {
		if file == "nginx" || file == "httpd" {
			return "/usr/bin/" + file, nil
		}
		return "", errors.New("not found")
	}

	got, err := Detect(env)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if got != Nginx {
		t.Errorf("Detect() = %q, want %q", got, Nginx)
	}
}

func TestDetectFallsBackToApacheHttpd(t *testing.T) {
	env := fakeEnvironment()
	env.LookPath = func(file string) (string, error) {
		if file == "httpd" {
			return "/usr/sbin/httpd", nil
		}
		return "", errors.New("not found")
	}

	got, err := Detect(env)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if got != Apache {
		t.Errorf("Detect() = %q, want %q", got, Apache)
	}
}

func TestDetectFallsBackToApache2(t *testing.T) {
	env := fakeEnvironment()
	env.LookPath = func(file string) (string, error) {
		if file == "apache2" {
			return "/usr/sbin/apache2", nil
		}
		return "", errors.New("not found")
	}

	got, err := Detect(env)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if got != Apache {
		t.Errorf("Detect() = %q, want %q", got, Apache)
	}
}

func TestDetectErrorsWhenNeitherInstalled(t *testing.T) {
	if _, err := Detect(fakeEnvironment()); err == nil {
		t.Error("Detect() error = nil, want an error when neither is installed")
	}
}

func TestConfDirPrefersBrewPrefix(t *testing.T) {
	env := fakeEnvironment()
	env.BrewPrefix = func() (string, bool) { return "/opt/homebrew", true }

	confDir, logDir, err := ConfDir(Nginx, env)
	if err != nil {
		t.Fatalf("ConfDir() error = %v", err)
	}
	if confDir != "/opt/homebrew/etc/nginx/servers" {
		t.Errorf("confDir = %q, want the Homebrew nginx servers directory", confDir)
	}
	if logDir != "/opt/homebrew/var/log/nginx" {
		t.Errorf("logDir = %q, want the Homebrew nginx log directory", logDir)
	}
}

func TestConfDirFallsBackToSystemNginxPath(t *testing.T) {
	env := fakeEnvironment()
	env.DirExists = func(path string) bool { return path == "/etc/nginx/conf.d" }

	confDir, logDir, err := ConfDir(Nginx, env)
	if err != nil {
		t.Fatalf("ConfDir() error = %v", err)
	}
	if confDir != "/etc/nginx/conf.d" || logDir != "/var/log/nginx" {
		t.Errorf("ConfDir() = (%q, %q), want the system nginx paths", confDir, logDir)
	}
}

func TestConfDirErrorsWhenNothingFound(t *testing.T) {
	if _, _, err := ConfDir(Nginx, fakeEnvironment()); err == nil {
		t.Error("ConfDir() error = nil, want an error when no nginx directory is found")
	}
	if _, _, err := ConfDir(Apache, fakeEnvironment()); err == nil {
		t.Error("ConfDir() error = nil, want an error when no Apache directory is found")
	}
}

func TestDetectPHPFPMSocketPrefersLiveBrewSocket(t *testing.T) {
	env := fakeEnvironment()
	env.BrewPrefix = func() (string, bool) { return "/opt/homebrew", true }
	env.IsSocket = func(path string) bool { return path == "/opt/homebrew/var/run/php-fpm.sock" }

	got := DetectPHPFPMSocket(env)
	want := "unix:/opt/homebrew/var/run/php-fpm.sock"
	if got != want {
		t.Errorf("DetectPHPFPMSocket() = %q, want %q", got, want)
	}
}

func TestDetectPHPFPMSocketFallsBackToRunPHPGlob(t *testing.T) {
	env := fakeEnvironment()
	env.Glob = func(pattern string) ([]string, error) {
		return []string{"/run/php/php8.3-fpm.sock"}, nil
	}
	env.IsSocket = func(path string) bool { return path == "/run/php/php8.3-fpm.sock" }

	got := DetectPHPFPMSocket(env)
	want := "unix:/run/php/php8.3-fpm.sock"
	if got != want {
		t.Errorf("DetectPHPFPMSocket() = %q, want %q", got, want)
	}
}

func TestDetectPHPFPMSocketFallsBackToTCP(t *testing.T) {
	got := DetectPHPFPMSocket(fakeEnvironment())
	if got != "127.0.0.1:9000" {
		t.Errorf("DetectPHPFPMSocket() = %q, want the TCP fallback", got)
	}
}
