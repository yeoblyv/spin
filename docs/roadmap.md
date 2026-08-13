---
id: plan.roadmap
title: Development Plan & Capability Register
slug: /plan/roadmap
category: Planning
since: 0.1.0
status: draft
---

[README](../README.md) / **Roadmap**

# Development Plan & Capability Register

![status](https://img.shields.io/badge/status-living%20document-5b4fe0)
![milestones](https://img.shields.io/badge/milestones-parity%20%2F%20multi--project%20%2F%20release-2ea44f)

This is the single living plan for `spin`. It records, per capability area, **what exists today**, **what is missing**, **what it should become**, and the **requirements** to get there. Requirements use RFC 2119 keywords (**MUST**, **SHOULD**, **MAY**).

`spin` started as a single-project environment bootstrapper (`init`, `console`) and is pivoting to a native, local site manager: one tool, installed once, that owns every Spider project on the machine — project registry, local domains, web server and PHP runtime lifecycle — in the spirit of Laravel Herd, but native to Spider and without a Docker dependency in its core.

## On this page

- [How to read this](#how-to-read-this)
- [Capability matrix](#capability-matrix)
- [1. Environment bootstrap](#1-environment-bootstrap)
- [2. Project registry](#2-project-registry)
- [3. Local domains](#3-local-domains)
- [4. Web server orchestration](#4-web-server-orchestration)
- [5. PHP runtime management](#5-php-runtime-management)
- [6. Interactive console](#6-interactive-console)
- [7. Developer experience](#7-developer-experience)
- [8. Security & secrets](#8-security--secrets)
- [9. Optional services layer](#9-optional-services-layer)
- [10. Distribution & release engineering](#10-distribution--release-engineering)
- [11. Cross-platform compatibility](#11-cross-platform-compatibility)
- [12. Loose ends](#12-loose-ends)
- [Milestones](#milestones)
- [References](#references)

---

## How to read this

Each area is described with four fields:

- **Have** — what exists in the code today.
- **Missing** — the gap.
- **Target** — what it should become.
- **Requirements** — the actionable work (normative).

**Status** legend: `none` (nothing yet) · `partial` (some code) · `done`.
**Priority** legend: `P0` blocker · `P1` parity-critical · `P2` site-manager-critical · `P3` later/optional.
**Milestone** legend: **P** parity with `bin/spider` + setup scripts · **M** multi-project site manager · **R** first public release.

## Capability matrix

| # | Area | Status | Priority | Milestone |
|---|---|---|---|---|
| 1 | Environment bootstrap (`init`) | done | — | P |
| 2 | Project registry | none | P1 | M |
| 3 | Local domains | none | P1 | M |
| 4 | Web server orchestration | none | P1 | P |
| 5 | PHP runtime management | none | P1 | M |
| 6 | Interactive console | partial | P2 | P |
| 7 | Developer experience | partial | P2 | M |
| 8 | Security & secrets | partial | P1 | P |
| 9 | Optional services layer | none | P3 | 1.x |
| 10 | Distribution & release engineering | partial | P1 | R |
| 11 | Cross-platform compatibility | partial | P1 | R |
| 12 | Loose ends (open decisions) | partial | P0 | P |

---

## 1. Environment bootstrap

**Have.** `spin init`: copies `.env.example` to `.env` when missing, generates `APP_KEY` when empty, never touches an existing file or key. Ships with `flag`-based `--dir`.

**Missing.** Awareness of the project registry ([§2](#2-project-registry)) — today `init` only ever knows about the one directory it was pointed at.

**Target.** `init` keeps working exactly as it does standalone, and additionally registers the project when run inside a directory that isn't already known.

**Requirements.**

- **SHOULD** call into the registry ([§2](#2-project-registry)) after a successful bootstrap, so `spin init` is sufficient to both prepare and register a project in one step.
- **MUST** keep working with no registry present at all (bare `.env`/`APP_KEY` bootstrap stays a valid standalone use case, e.g. in CI).

**Priority** — · **Milestone** P.

## 2. Project registry

**Have.** Nothing. Every command that needs a project directory takes it via a per-invocation `--dir` flag; there is no memory between runs.

**Missing.** A persistent list of known Spider projects (path, chosen local domain, PHP version, web server, last-used state) and commands to manage it.

**Target.** `spin` knows about every Spider project on the machine without being told the path each time; other commands default to "the project for the current directory" or "the project selected via `spin use`" instead of requiring `--dir`.

**Requirements.**

- **MUST** provide `spin site add [path]`, `spin site list`, `spin site remove <name>`, and `spin site use <name>` (or equivalent verbs), backed by a single config file (e.g. `~/.config/spin/registry.json` on Linux/macOS, `%APPDATA%\spin\registry.json` on Windows — path resolution follows [§11](#11-cross-platform-compatibility)).
- **MUST** resolve "current project" in this order: an explicit `--site`/`--dir` flag, then the current working directory if it is a known or bootstrappable Spider project, then the `use`-selected default; commands **MUST** fail with a clear error rather than guess when none apply.
- **MUST** validate a registered path still looks like a Spider project (has `composer.json` naming `yeoblyv/spider` or a `bootstrap.php`) at read time, and mark stale entries rather than silently erroring.
- **SHOULD** store per-project metadata needed by [§3](#3-local-domains)–[§5](#5-php-runtime-management) (domain, PHP version, assigned ports/sockets) in the same registry entry, so those areas have one source of truth.

**Priority** P1 · **Milestone** M.

## 3. Local domains

**Have.** Nothing. The working pattern established during local nginx setup was a manually chosen `server_name` and, where needed, a manual `/etc/hosts` edit.

**Missing.** Any automated local-domain handling; a decision on the TLD convention (e.g. `.test`, per [RFC 2606][R5]'s reserved-for-testing space).

**Target.** Adding a project to the registry gives it a working `https://<name>.test` (or configured TLD) address with no manual hosts-file or resolver editing.

**Requirements.**

- **MUST** default new sites to a reserved TLD (`.test`) per [RFC 2606][R5], overridable per project.
- **MUST**, on macOS, install a per-TLD resolver file under `/etc/resolver/` pointed at a local DNS responder — this is the same mechanism Herd/Valet use and avoids editing `/etc/hosts` per site.
- **MUST**, on Linux, either integrate with `systemd-resolved` (a stub domain pointed at a local responder) where present, or fall back to managed `/etc/hosts` entries with clearly delimited `# spin:begin`/`# spin:end` markers so `spin` only ever touches its own block.
- **MUST**, on Windows (no per-domain resolver mechanism), manage `%WinDir%\System32\drivers\etc\hosts` with the same delimited-block approach.
- **MUST** require elevated privileges only for the specific step that needs them (writing the resolver file or hosts block), and **MUST** state clearly, before prompting, why elevation is needed.
- **SHOULD** provision locally-trusted TLS for each domain (a local CA installed once into the OS/browser trust store, then a leaf certificate per site) rather than serving local sites over plain HTTP.

**Priority** P1 · **Milestone** M.

## 4. Web server orchestration

**Have.** `deploy/nginx/spider.conf.template`, `deploy/apache/spider.conf.template`, and `scripts/setup-nginx.sh`/`setup-apache.sh` on the Spider side — portable, placeholder-driven, but external to `spin`, one-shot, and single-project (they render and install one vhost and stop).

**Missing.** Any of this logic inside `spin` itself; start/stop/reload lifecycle management; concurrent multi-project vhost management; automatic port/socket allocation to avoid collisions between sites running at once.

**Target.** `spin` renders, installs, and manages the web server config for every registered project — the setup scripts are absorbed and retired.

**Requirements.**

- **MUST** port the template-rendering logic from `setup-nginx.sh`/`setup-apache.sh` into `spin` natively (same `{{PLACEHOLDER}}` templates, embedded via `go:embed` rather than shipped as separate shell scripts).
- **MUST** detect the web server already installed (nginx vs Apache vs neither) the same way the shell scripts do today, and **MUST** fail with actionable install instructions rather than silently doing nothing when neither is present.
- **MUST** allocate a free upstream PHP-FPM socket/port per project automatically so multiple sites can run concurrently without manual coordination — this is a direct requirement of the registry becoming multi-project ([§2](#2-project-registry)).
- **MUST** provide `spin site up`/`spin site down` (or equivalent) to start/stop a project's web server + PHP-FPM pairing, and reload the web server config without a full restart where the underlying server supports it.
- **SHOULD** detect drift (a project's rendered vhost no longer matches what the registry expects, e.g. edited by hand or by another tool) and offer to re-render rather than overwrite silently.

Once this ships, `scripts/setup-nginx.sh` and `scripts/setup-apache.sh` in the Spider repo become candidates for the deprecation path already noted in their `# Temporary: superseded by spin once it ships.` comment.

**Priority** P1 · **Milestone** P.

## 5. PHP runtime management

**Have.** Nothing. The current workflow assumes a single, already-installed, already-running PHP-FPM that `setup-nginx.sh` merely locates (`$(brew --prefix)/var/run/php-fpm.sock` and equivalents).

**Missing.** Any handling of a project needing a specific PHP version, or of two projects on the machine needing different versions concurrently.

**Target.** `spin` reads a project's PHP version constraint (from `composer.json`) and ensures a matching PHP-FPM is running for it, isolated from other projects' versions.

**Requirements.**

- **MUST** read the `php` constraint from the project's `composer.json` and surface a clear error (not a silent fallback) when no installed PHP satisfies it.
- **SHOULD** manage per-version PHP-FPM pools so two registered projects requiring different PHP versions can run at the same time, each behind its own socket.
- **MAY** offer to install a missing PHP version via the platform's package manager (Homebrew on macOS, the system package manager on Linux) rather than requiring the user to do so manually first — this is explicitly a "nice to have," not a blocker, since it touches system package state and **MUST** always ask before installing anything.

**Priority** P1 · **Milestone** M.

## 6. Interactive console

**Have.** `spin console`: a `mysql`/`asterisk -r`-style REPL: banner reporting both `spin`'s and the target Spider project's version (detected from git tag, then `composer.json`, then `CHANGELOG.md`), each line dispatched through the same `Run` path as a top-level invocation, colored `}}{{ ` prompt matching the Spider logo's four gradient stops.

**Missing.** Registry awareness (currently takes `--dir` like every other command); command history/line-editing (a bare `bufio.Scanner` today, no arrow-key recall); tab completion of registered project names or subcommands.

**Target.** The console is the primary interactive entry point into a multi-project `spin`: `spin console` alone drops into the shell for the current/default project, and commands typed inside it can address any registered project.

**Requirements.**

- **MUST** resolve its target project through the same rules as every other command ([§2](#2-project-registry)) once the registry exists, instead of only `--dir`.
- **SHOULD** add line-editing and history (e.g. via a small readline-equivalent library) so the shell is comfortable for repeated use, not just scriptable input.
- **MAY** add tab completion for subcommands and registered project names.

**Priority** P2 · **Milestone** P.

## 7. Developer experience

**Have.** `--version`/`--help`/`version`/`help` at the top level; per-command `-h` via the standard `flag` package; consistent, testable `stdout`/`stderr` separation and exit codes (`0`/`1`/`2`) throughout.

**Missing.** A diagnostic/health-check command; shell completions; update notifications; log tailing for a running project's web server/PHP-FPM.

**Target.** `spin` is pleasant to live in day to day: it tells you what's wrong in one command, updates itself without ceremony, and gets out of your way in the shell.

**Requirements.**

- **MUST** provide `spin doctor` (or equivalent): checks the current project's registry entry, web server presence, PHP version match, and local-domain resolution in one pass, and reports each with a clear pass/fail and, on failure, the exact next command to run.
- **SHOULD** provide shell completion scripts (bash/zsh/fish, PowerShell) generated from the command registry so they can never drift out of sync with the actual command set.
- **SHOULD** check the latest published release against the running binary's own `--version` output and print a one-line notice (never auto-update without being asked).
- **SHOULD** provide `spin logs <site>` to tail a project's web server and PHP-FPM error logs together, prefixed by source.
- **MUST** keep every error message actionable: state what failed and what command (if any) fixes it, following the pattern already used in `init.go`'s error messages.

**Priority** P2 · **Milestone** M.

## 8. Security & secrets

**Have.** `APP_KEY` generation via `crypto/rand`, base64-encoded, matching Spider's own convention; `.env` written with `0o600` permissions; no secret value is ever logged to `stdout`/`stderr`.

**Missing.** A documented policy for the registry file and any rendered web-server config (both may contain filesystem paths but never secrets by design — this **MUST** stay true as new fields are added); a story for locally-trusted TLS material ([§3](#3-local-domains)) that avoids storing a CA private key world-readable.

**Target.** Nothing `spin` writes to disk is ever more permissive than it needs to be, and nothing it prints ever includes a secret.

**Requirements.**

- **MUST** write the registry file and any local CA/certificate private key with owner-only permissions (`0o600` files, `0o700` directories), matching the existing `.env` convention.
- **MUST** never write an `APP_KEY`, database credential, or other secret value to `stdout`, `stderr`, or any log file `spin` produces.
- **MUST** treat the local CA's private key (once TLS provisioning exists, [§3](#3-local-domains)) as sensitive: generated once per machine, never transmitted, never included in diagnostics output (`spin doctor` **MUST** report that a CA exists, never its contents).

**Priority** P1 · **Milestone** P.

## 9. Optional services layer

**Have.** Nothing, and nothing planned for `spin`'s core — the core orchestrates PHP-FPM and the web server natively, no container runtime required.

**Missing.** A way to run auxiliary services a project might want locally (Redis, a mail-catcher, a search engine) without making Docker a requirement for everyone.

**Target.** Projects that want these services can opt in; projects that don't never see Docker mentioned, installed, or required.

**Requirements.**

- **MAY** add an opt-in `spin services` layer, backed by Docker Compose, purely additive to the native core — **MUST NOT** become a dependency of [§2](#2-project-registry)–[§8](#8-security--secrets).
- **MUST** keep this out of the parity ([§4](#4-web-server-orchestration)) and multi-project ([§2](#2-project-registry)) milestones entirely; it is post-release scope.

**Priority** P3 · **Milestone** 1.x.

## 10. Distribution & release engineering

**Have.** `.github/workflows/ci.yml` (gofmt/`go vet`/`go test -race` on push/PR); `.github/workflows/release.yml` (cross-compiles darwin/linux/windows × amd64/arm64 from one runner on a `v*` tag push, embeds version/commit via `-ldflags`, verifies the linux/amd64 binary's own `--version` output against the tag, publishes SHA-256 checksums via `softprops/action-gh-release`); version is git-tag-only, `go.mod` carries no version field.

**Missing.** A public GitHub repository and a first tagged release to point the checksum-verified download flow at; the Spider-side installer that pulls a released `spin` binary on demand ([README](../README.md)'s stated model).

**Target.** A tagged `v0.1.0` (or first agreed version) published with checksummed binaries for every supported OS/architecture, and a working download-and-verify path from a fresh Spider project.

**Requirements.**

- **MUST** publish the `yeoblyv/spin` repository and push the existing local history (pending the author-email amend noted in [§12](#12-loose-ends)).
- **MUST** cut a first tagged release once [§1](#1-environment-bootstrap) and [§6](#6-interactive-console) are the only shipped commands, or once enough of [§2](#2-project-registry)–[§5](#5-php-runtime-management) lands to justify it — this is an owner call, not an engineering gate.
- **MUST** build the Spider-side installer (referenced in `README.md`'s "Relationship to Spider" section) that resolves OS/arch, downloads the matching binary and checksum, verifies it, and installs it — this supersedes the deferred submodule/committed-binary options already ruled out.
- **MUST** confirm the chosen package name is actually free in each target channel before committing to it, not just check the GitHub repository name. The bare name `spin` is **already taken** in the official Debian/Ubuntu archives by an unrelated formal-verification tool ([R6]) present in `bullseye` through `sid`; a submission to those official archives would need a distinct source-package name (e.g. `spin-cli`, `spider-spin`). This does not block a self-hosted APT repository, where the source-package name is this project's own choice and the installed command can still be `spin` regardless.
- **SHOULD** ship a `.deb` build (via `nfpm` or `goreleaser`, as an additional step alongside the existing cross-compiled binaries in `release.yml`) and a small self-hosted APT repository (e.g. published from a GitHub Pages branch), so `apt install` works immediately without waiting on any official-archive review.
- **MAY** pursue inclusion in the official Debian/Ubuntu archives later. That path is independent of this project's own release cadence, is reviewed by people outside this project, and is not a release blocker — see [§12](#12-loose-ends) for why it takes as long as it does.

**Priority** P1 · **Milestone** R.

## 11. Cross-platform compatibility

**Have.** CI cross-compiles all six OS/architecture combinations from one Linux runner (relying on Go's deterministic cross-compilation rather than a three-OS build matrix, per the toolchain exception this project follows); real runtime testing to date has only happened on macOS.

**Missing.** Any real-machine verification on Linux and Windows of the orchestration features ([§3](#3-local-domains)–[§5](#5-php-runtime-management)) — these touch OS-specific mechanisms (`/etc/resolver`, `systemd-resolved`, Windows `hosts`, package managers) that cross-compilation alone cannot validate, since they only execute at runtime on their target OS.
**Target.** Every capability in this document behaves correctly on Linux, macOS, and Windows, not just builds for them.

**Requirements.**

- **MUST** run `go test`/`go vet`/`gofmt -l .` on `ubuntu-latest`, `macos-latest`, and `windows-latest` in CI once any OS-specific code lands (today's pure-Go scaffold does not yet need this, but [§3](#3-local-domains)–[§5](#5-php-runtime-management) will).
- **MUST** isolate every OS-specific branch (resolver mechanism, hosts-file path, package manager, config directory location) behind a single small internal package per concern, so the platform-specific code is easy to find, test, and extend.
- **MUST** verify each OS-specific feature on real hardware/VMs for that OS at least once before it is considered done — cross-compiled builds that have never executed are not a substitute for this.

**Priority** P1 · **Milestone** R.

## 12. Loose ends

Items already identified elsewhere that must not be dropped.

- **`yeoblyv/spin` GitHub repository does not exist yet.** Two local commits (`feat: scaffold project`, `feat(cli): add init command for .env bootstrap`) are ready to push once created.
- **Author email on those two commits still needs amending** to the project's current canonical address before the first push (local-only rewrite, safe pre-push; the local git config for future commits is already correct).
- **`spin console`'s underlying changes are uncommitted** pending a green `go build`/`go vet`/`gofmt -l .`/`go test ./... -v` run — **MUST NOT** be committed before that gate is confirmed green, per this project's own commit-on-green-gate rule.
- **`bin/spider` deprecation timing** is intentionally not scheduled here: per Spider's own `UPGRADING.md` policy (deprecate in a minor release only once the replacement is available, remove in a major release), that step happens once `spin` reaches real feature parity ([§4](#4-web-server-orchestration) done, milestone **P** reached) — it is a Spider-repo decision, not a `spin`-repo one.
- **Local-CA / TLS provisioning mechanism ([§3](#3-local-domains))** is stated as a direction but not yet designed in detail — needs its own short design pass (which OS trust-store APIs to use, certificate lifetime/rotation) before [§3](#3-local-domains) requirements are actionable.
- **Docker's role stays fixed at "optional, non-core"** ([§9](#9-optional-services-layer)) — re-litigating this is out of scope unless the core native approach demonstrably fails on a platform.
- **Official Debian/Ubuntu archive inclusion ([§10](#10-distribution--release-engineering)) is a separate, slow, human-reviewed track**, not something this project controls the timeline of: it requires a distinct package name (`spin` is taken there), a Debian Developer sponsor to review and upload the package, and a manual ftpmaster review of licensing/policy compliance before it clears the archive's NEW queue. Track record requirements alone (six months of prior packaging activity before a sponsor will typically take on a new package) put this well past the **R** milestone. The self-hosted APT repository above is what actually unblocks `apt install` on this project's own timeline.

**Priority** P0 · **Milestone** P.

## Milestones

| Milestone | Definition | Gated by |
|---|---|---|
| **Parity-ready (P)** | `spin` fully replaces `bin/spider` + `setup-nginx.sh`/`setup-apache.sh` for one project at a time. | Environment bootstrap; web server orchestration; interactive console; security & secrets baseline; loose ends closed. |
| **Multi-project-ready (M)** | The site-manager vision: many Spider projects, one tool, no manual coordination. | Project registry; local domains; PHP runtime management; developer experience (`doctor`, completions). |
| **Public release (R)** | First public, checksummed, installable release. | Distribution & release engineering; cross-platform compatibility verified on real machines for every shipped feature. |
| **Post-release (1.x)** | Depth and optional capabilities. | Optional services layer; anything deferred from the areas above as **MAY**. |

## References

- [R5]: https://www.rfc-editor.org/rfc/rfc2606 "RFC 2606 — Reserved Top Level DNS Names"
- [R6]: https://packages.debian.org/search?keywords=spin "Debian Package Search — spin (formal-verification tool, unrelated to this project)"
- Spider's `UPGRADING.md` — deprecation policy governing when `bin/spider` is deprecated and removed.
- Spider's `docs/roadmap.md` and `docs/execution-plan.md` — capability-register and execution-plan format this document follows.
