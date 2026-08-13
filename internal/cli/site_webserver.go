package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yeoblyv/spin/internal/platform"
	reg "github.com/yeoblyv/spin/internal/registry"
	"github.com/yeoblyv/spin/internal/webserver"
)

// runSiteUp renders and installs the target site's web server
// configuration, mirroring setup-nginx.sh/setup-apache.sh but ported
// natively and driven by the site registry instead of positional
// arguments. It never reloads or restarts the web server itself - like the
// scripts it replaces, it prints the exact command to do so.
func runSiteUp(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("site up", flag.ContinueOnError)
	fs.SetOutput(stderr)
	site := fs.String("site", "", "registered site name (defaults to the current directory or the default site)")
	port := fs.String("port", "80", "HTTP port to listen on")
	serverFlag := fs.String("server", "", "web server to target: nginx or apache (auto-detected when empty)")
	confDirFlag := fs.String("conf-dir", "", "configuration directory to write the vhost into (auto-detected when empty)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	entry, kind, env, err := resolveSiteAndServer(*site, *serverFlag)
	if err != nil {
		fmt.Fprintf(stderr, "spin: site: %v\n", err)
		return exitError
	}
	if entry.Domain == "" {
		fmt.Fprintf(stderr, "spin: site: %q has no domain assigned - run \"spin site add\" again or set one with \"spin site domain\"\n", entry.Name)
		return exitError
	}

	confDir, logDir, err := resolveConfDir(*confDirFlag, kind, env)
	if err != nil {
		fmt.Fprintf(stderr, "spin: site: %v\n", err)
		return exitError
	}

	params := webserver.RenderParams{
		ServerName:  entry.Domain,
		HTTPPort:    *port,
		ProjectRoot: entry.Path,
		AccessLog:   filepath.Join(logDir, entry.Domain+"-access.log"),
		ErrorLog:    filepath.Join(logDir, entry.Domain+"-error.log"),
		FastCGIPass: webserver.DetectPHPFPMSocket(env),
	}

	rendered, err := webserver.Render(kind, params)
	if err != nil {
		fmt.Fprintf(stderr, "spin: site: %v\n", err)
		return exitError
	}

	if err := os.MkdirAll(confDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "spin: site: creating %s: %v\n", confDir, err)
		return exitError
	}
	outPath := filepath.Join(confDir, entry.Domain+".conf")
	if err := os.WriteFile(outPath, []byte(rendered), 0o644); err != nil {
		fmt.Fprintf(stderr, "spin: site: writing %s: %v\n", outPath, err)
		return exitError
	}

	fmt.Fprintf(stdout, "wrote %s\n", outPath)
	printReloadHint(stdout, kind, *port, env)
	if !domainInHosts(entry.Domain) {
		fmt.Fprintf(stdout, "%s is not yet in the hosts file - run \"spin hosts apply\" to add it\n", entry.Domain)
	}
	return exitOK
}

// runSiteDown removes the target site's installed web server
// configuration. Like runSiteUp, it never reloads or restarts the web
// server itself.
func runSiteDown(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("site down", flag.ContinueOnError)
	fs.SetOutput(stderr)
	site := fs.String("site", "", "registered site name (defaults to the current directory or the default site)")
	serverFlag := fs.String("server", "", "web server to target: nginx or apache (auto-detected when empty)")
	confDirFlag := fs.String("conf-dir", "", "configuration directory the vhost was written into (auto-detected when empty)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	entry, kind, env, err := resolveSiteAndServer(*site, *serverFlag)
	if err != nil {
		fmt.Fprintf(stderr, "spin: site: %v\n", err)
		return exitError
	}
	if entry.Domain == "" {
		fmt.Fprintf(stderr, "spin: site: %q has no domain assigned\n", entry.Name)
		return exitError
	}

	confDir, _, err := resolveConfDir(*confDirFlag, kind, env)
	if err != nil {
		fmt.Fprintf(stderr, "spin: site: %v\n", err)
		return exitError
	}

	outPath := filepath.Join(confDir, entry.Domain+".conf")
	if err := os.Remove(outPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stdout, "%s was not present\n", outPath)
			return exitOK
		}
		fmt.Fprintf(stderr, "spin: site: removing %s: %v\n", outPath, err)
		return exitError
	}

	fmt.Fprintf(stdout, "removed %s\n", outPath)
	printReloadHint(stdout, kind, "80", env)
	return exitOK
}

// resolveSiteAndServer resolves both the target registry entry and the web
// server kind to act on, sharing the same logic between "site up" and
// "site down".
func resolveSiteAndServer(siteFlag, serverFlag string) (reg.Entry, webserver.Kind, webserver.Environment, error) {
	r, _, err := openRegistry()
	if err != nil {
		return reg.Entry{}, "", webserver.Environment{}, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}
	entry, err := r.ResolveSite(siteFlag, cwd)
	if err != nil {
		return reg.Entry{}, "", webserver.Environment{}, err
	}

	env := webserver.RealEnvironment()
	kind := webserver.Kind(serverFlag)
	if kind == "" {
		kind, err = webserver.Detect(env)
		if err != nil {
			return reg.Entry{}, "", webserver.Environment{}, err
		}
	}

	return entry, kind, env, nil
}

// resolveConfDir returns override, if set, paired with itself as the log
// directory (an explicit override means webserver.ConfDir's own log-path
// convention no longer applies); otherwise it auto-detects both via
// webserver.ConfDir.
func resolveConfDir(override string, kind webserver.Kind, env webserver.Environment) (confDir, logDir string, err error) {
	if override != "" {
		return override, override, nil
	}
	return webserver.ConfDir(kind, env)
}

// printReloadHint prints the exact command to reload kind's web server,
// matching setup-nginx.sh/setup-apache.sh's own convention of never
// reloading automatically - only ever telling the operator what to run.
func printReloadHint(w io.Writer, kind webserver.Kind, port string, env webserver.Environment) {
	_, isBrew := env.BrewPrefix()

	switch kind {
	case webserver.Nginx:
		fmt.Fprintln(w, "Reload nginx with:")
		switch {
		case isBrew:
			fmt.Fprintln(w, "  brew services restart nginx")
		case isPrivilegedPort(port):
			fmt.Fprintf(w, "  sudo nginx -s reload   # port %s is privileged, needs root\n", port)
		default:
			fmt.Fprintln(w, "  sudo nginx -s reload")
		}
	case webserver.Apache:
		fmt.Fprintln(w, "Reload Apache with:")
		if isBrew {
			fmt.Fprintln(w, "  brew services restart httpd")
		} else {
			fmt.Fprintln(w, "  sudo systemctl reload apache2")
		}
	}
}

func isPrivilegedPort(port string) bool {
	p, err := strconv.Atoi(port)
	return err == nil && p < 1024
}

// domainInHosts reports whether domain already appears anywhere in the OS
// hosts file - a best-effort reminder, not a guarantee it resolves.
func domainInHosts(domain string) bool {
	return domainInHostsFile(domain, platform.HostsFilePath())
}

// domainInHostsFile is domainInHosts's testable core: path is injected so
// tests never depend on the real machine's hosts file.
func domainInHostsFile(domain, path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), domain)
}
