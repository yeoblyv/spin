package cli

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/yeoblyv/spin/internal/phpversion"
	"github.com/yeoblyv/spin/internal/platform"
	"github.com/yeoblyv/spin/internal/webserver"
)

func init() {
	registry = append(registry, command{
		name:        "doctor",
		description: "Check a site's registry entry, web server, PHP version, and local domain in one pass",
		run:         runDoctor,
	})
}

// doctorCheck is one diagnostic result: a pass/fail with human-readable
// detail, and - only meaningful when !ok - the exact command that fixes it.
type doctorCheck struct {
	name   string
	ok     bool
	detail string
	fix    string
}

// runDoctor prints every check from diagnose and exits non-zero if any
// failed.
func runDoctor(args []string, stdout, stderr io.Writer, version, commit string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	site := fs.String("site", "", "registered site name (defaults to the current directory or the default site)")
	hostsFile := fs.String("hosts-file", platform.HostsFilePath(), "hosts file to check the domain against")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	checks := diagnose(*site, *hostsFile, webserver.RealEnvironment(), phpversion.RealEnvironment())

	allOK := true
	for _, c := range checks {
		status := "PASS"
		if !c.ok {
			status = "FAIL"
			allOK = false
		}
		fmt.Fprintf(stdout, "[%s] %s: %s\n", status, c.name, c.detail)
		if !c.ok && c.fix != "" {
			fmt.Fprintf(stdout, "       run: %s\n", c.fix)
		}
	}

	if !allOK {
		return exitError
	}
	return exitOK
}

// diagnose runs every check spin doctor reports, in order: the site
// resolves in the registry, a web server is installed, the installed PHP
// satisfies the project's composer.json constraint, and the site's domain
// resolves via the hosts file. webEnv and phpEnv are injected so the whole
// pipeline is testable without depending on what is actually installed on
// the machine running it.
//
// A failed site-resolution check is terminal: without a resolved path,
// none of the later checks have anything to check.
func diagnose(siteFlag, hostsFile string, webEnv webserver.Environment, phpEnv phpversion.Environment) []doctorCheck {
	var checks []doctorCheck

	r, _, err := openRegistry()
	if err != nil {
		return append(checks, doctorCheck{name: "site", detail: err.Error()})
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}
	entry, err := r.ResolveSite(siteFlag, cwd)
	if err != nil {
		return append(checks, doctorCheck{name: "site", detail: err.Error(), fix: "spin site add"})
	}
	checks = append(checks, doctorCheck{name: "site", ok: true, detail: fmt.Sprintf("%s (%s)", entry.Name, entry.Path)})

	checks = append(checks, checkWebServer(webEnv))
	checks = append(checks, checkPHPVersion(entry.Path, phpEnv))
	checks = append(checks, checkDomain(entry.Domain, hostsFile))

	return checks
}

func checkWebServer(env webserver.Environment) doctorCheck {
	kind, err := webserver.Detect(env)
	if err != nil {
		return doctorCheck{name: "web server", detail: err.Error()}
	}
	return doctorCheck{name: "web server", ok: true, detail: fmt.Sprintf("%s detected", kind)}
}

func checkPHPVersion(projectDir string, env phpversion.Environment) doctorCheck {
	constraintText, err := phpversion.ReadComposerConstraint(projectDir)
	if err != nil {
		return doctorCheck{name: "php", detail: err.Error()}
	}

	constraint, err := phpversion.ParseConstraint(constraintText)
	if err != nil {
		return doctorCheck{name: "php", detail: err.Error()}
	}

	installed, err := phpversion.DetectInstalled(env)
	if err != nil {
		return doctorCheck{name: "php", detail: err.Error(), fix: fmt.Sprintf("install a PHP satisfying %q", constraintText)}
	}

	if !constraint.Satisfies(installed) {
		return doctorCheck{
			name:   "php",
			detail: fmt.Sprintf("installed PHP %s does not satisfy %q", installed, constraintText),
			fix:    fmt.Sprintf("install a PHP satisfying %q", constraintText),
		}
	}

	return doctorCheck{name: "php", ok: true, detail: fmt.Sprintf("PHP %s satisfies %q", installed, constraintText)}
}

func checkDomain(domain, hostsFile string) doctorCheck {
	if domain == "" {
		return doctorCheck{name: "domain", detail: "no domain assigned", fix: "spin site add --domain <name>.test (re-run to update an existing site)"}
	}
	if !domainInHostsFile(domain, hostsFile) {
		return doctorCheck{name: "domain", detail: fmt.Sprintf("%s is not in the hosts file", domain), fix: "spin hosts apply"}
	}
	return doctorCheck{name: "domain", ok: true, detail: fmt.Sprintf("%s resolves via the hosts file", domain)}
}
