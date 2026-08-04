# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.6.0] - 2026-08-04

### Added
- `--install` flag in `arcup` CLI to copy default configuration template to `~/.arcup.yaml`.
- Embedded default configuration using Go `go:embed` feature.
- OS (`linux`, `darwin`) and Architecture (`amd64`, `arm64`) auto-detection in `install.sh`.
- Automatic dependency prompt for installing `rclone` in `install.sh`.
- Pre-install checks in `install.sh` for compression algorithms (`zstd`, `gzip`, `zip`) with warnings if missing from system `$PATH`.

### Changed
- Target installation directory in `install.sh` moved to `/usr/local/bin/arcup`.
- Updated installer repository URL to `Dlcuy22/arcup`.
