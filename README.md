# spin

![status](https://img.shields.io/badge/status-early--scaffold-9e9e9e)
![go](https://img.shields.io/badge/go-%3E%3D1.23-00ADD8)
![platforms](https://img.shields.io/badge/platforms-Linux%20%C2%B7%20macOS%20%C2%B7%20Windows-2ea44f)

A single cross-platform binary for installing, configuring, and running [Spider](https://github.com/yeoblyv/spider) projects — the eventual replacement for Spider's PHP-based `bin/spider` console entry point, plus the setup tooling currently living in Spider's `scripts/setup-nginx.sh` and `scripts/setup-apache.sh`.

## Project status

This is an early scaffold: the command dispatcher and `--version` output exist; no subcommands are ported yet. Spider's `bin/spider` remains the working console tool in the meantime — see [Spider's `UPGRADING.md`](https://github.com/yeoblyv/spider/blob/main/UPGRADING.md) for the deprecation policy `bin/spider` will eventually follow once `spin` reaches parity.

## Requirements

- Go `>= 1.23` (build-time only; released binaries have no runtime dependency)

## Building from source

```bash
git clone https://github.com/yeoblyv/spin.git
cd spin
go build -o bin/spin ./cmd/spin
./bin/spin --version
```

## Testing

```bash
go test ./...
go vet ./...
gofmt -l .
```

`gofmt -l .` should print nothing; any listed file is not correctly formatted (`gofmt -w .` fixes it in place).

## Relationship to Spider

`spin` is versioned and released independently of Spider itself. A Spider project pulls a released `spin` binary on demand — matched to the running OS/architecture and verified against its published checksum — rather than vendoring or committing one; see Spider's own deployment tooling for the installer side of that flow.

## Author

**Yehor Oblyvantsov**
Email: xineraman8@gmail.com · GitHub: [@yeoblyv](https://github.com/yeoblyv) · Web: [oblyvantsov.net](https://oblyvantsov.net/)
