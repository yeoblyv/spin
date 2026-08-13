// Package webserver renders and locates nginx/Apache virtual host
// configuration for a Spider project, porting the logic that previously
// lived in Spider's setup-nginx.sh and setup-apache.sh into spin natively.
package webserver

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed templates/nginx.conf.template
var nginxTemplate string

//go:embed templates/apache.conf.template
var apacheTemplate string

// Kind identifies which web server a rendered configuration targets.
type Kind string

const (
	Nginx  Kind = "nginx"
	Apache Kind = "apache"
)

// RenderParams are the placeholder values substituted into a template.
// FastCGIPass is only meaningful for Nginx: Apache's template has no
// fastcgi_pass equivalent, since PHP handling there is Apache's own module
// or a separately configured proxy.
type RenderParams struct {
	ServerName  string
	HTTPPort    string
	ProjectRoot string
	AccessLog   string
	ErrorLog    string
	FastCGIPass string
}

// Render substitutes params into kind's template.
func Render(kind Kind, params RenderParams) (string, error) {
	var tmpl string
	switch kind {
	case Nginx:
		tmpl = nginxTemplate
	case Apache:
		tmpl = apacheTemplate
	default:
		return "", fmt.Errorf("unknown web server kind %q", kind)
	}

	replacer := strings.NewReplacer(
		"{{SERVER_NAME}}", params.ServerName,
		"{{HTTP_PORT}}", params.HTTPPort,
		"{{PROJECT_ROOT}}", params.ProjectRoot,
		"{{ACCESS_LOG}}", params.AccessLog,
		"{{ERROR_LOG}}", params.ErrorLog,
		"{{FASTCGI_PASS}}", params.FastCGIPass,
	)
	return replacer.Replace(tmpl), nil
}

// Environment isolates every OS-and-tooling-specific lookup detection
// depends on, so it can run against a fake in tests instead of the real
// machine's installed software.
type Environment struct {
	LookPath   func(file string) (string, error)
	BrewPrefix func() (prefix string, ok bool)
	DirExists  func(path string) bool
	Glob       func(pattern string) ([]string, error)
	IsSocket   func(path string) bool
}

// RealEnvironment wires Environment to the actual host machine.
func RealEnvironment() Environment {
	return Environment{
		LookPath: exec.LookPath,
		BrewPrefix: func() (string, bool) {
			out, err := exec.Command("brew", "--prefix").Output()
			if err != nil {
				return "", false
			}
			return strings.TrimSpace(string(out)), true
		},
		DirExists: func(path string) bool {
			info, err := os.Stat(path)
			return err == nil && info.IsDir()
		},
		Glob: filepath.Glob,
		IsSocket: func(path string) bool {
			info, err := os.Lstat(path)
			return err == nil && info.Mode()&os.ModeSocket != 0
		},
	}
}

// Detect reports which web server is installed, checking nginx before
// Apache to match setup-nginx.sh/setup-apache.sh's own precedent (a
// machine with both installed renders nginx first). It fails with
// actionable install guidance when neither is present.
func Detect(env Environment) (Kind, error) {
	if _, err := env.LookPath("nginx"); err == nil {
		return Nginx, nil
	}
	if _, err := env.LookPath("httpd"); err == nil {
		return Apache, nil
	}
	if _, err := env.LookPath("apache2"); err == nil {
		return Apache, nil
	}
	return "", errors.New(`neither nginx nor Apache is installed - install one first, e.g. "brew install nginx" or "brew install httpd" on macOS, or your distribution's nginx/apache2 package on Linux`)
}

// ConfDir returns the directory a rendered vhost for kind should be written
// into, and the directory its access/error logs live in. Unlike
// setup-apache.sh's httpd-vhosts.conf-append approach, spin writes one file
// per site for both web servers - simpler to add, remove, and reason about
// than editing markers inside a shared file, and it is what nginx's own
// servers/ convention already does.
func ConfDir(kind Kind, env Environment) (confDir, logDir string, err error) {
	switch kind {
	case Nginx:
		if prefix, ok := env.BrewPrefix(); ok {
			return filepath.Join(prefix, "etc", "nginx", "servers"), filepath.Join(prefix, "var", "log", "nginx"), nil
		}
		if env.DirExists("/etc/nginx/conf.d") {
			return "/etc/nginx/conf.d", "/var/log/nginx", nil
		}
		return "", "", errors.New("could not locate an nginx configuration directory automatically - set one explicitly with --conf-dir")
	case Apache:
		if prefix, ok := env.BrewPrefix(); ok {
			return filepath.Join(prefix, "etc", "httpd", "extra", "spin-sites"), filepath.Join(prefix, "var", "log", "httpd"), nil
		}
		if env.DirExists("/etc/apache2/sites-available") {
			return "/etc/apache2/sites-available", "/var/log/apache2", nil
		}
		return "", "", errors.New("could not locate an Apache configuration directory automatically - set one explicitly with --conf-dir")
	default:
		return "", "", fmt.Errorf("unknown web server kind %q", kind)
	}
}

// DetectPHPFPMSocket returns a fastcgi_pass-style target for a live PHP-FPM
// socket, mirroring setup-nginx.sh's own heuristic: a Homebrew socket if
// verified live, else the first live socket under /run/php, else the plain
// TCP fallback every fresh install listens on by default.
func DetectPHPFPMSocket(env Environment) string {
	if prefix, ok := env.BrewPrefix(); ok {
		candidate := filepath.Join(prefix, "var", "run", "php-fpm.sock")
		if env.IsSocket(candidate) {
			return "unix:" + candidate
		}
	}

	if matches, err := env.Glob("/run/php/*.sock"); err == nil {
		for _, m := range matches {
			if env.IsSocket(m) {
				return "unix:" + m
			}
		}
	}

	return "127.0.0.1:9000"
}
