# Changelog

All notable changes to this project are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `spin init`: bootstraps a Spider project's `.env` from `.env.example` and generates `APP_KEY` when empty.
- `spin console`: interactive REPL dispatching each line through the same command path as a top-level invocation.
- Top-level `--version`/`--help`/`version`/`help` handling and command dispatch.
