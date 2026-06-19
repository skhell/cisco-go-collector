# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-06-19

### Added

- CSV-driven device and command configuration with required columns: `datacenter`, `room`, `rack`, `hostname`, `ip`, `platform`, `category`, `command`
- Optional CSV columns: `target_ip`, `vrf` for ping and traceroute including VRF-aware
- SSH execution via interactive shell (Cisco NX-OS and IOS-XE/IOS)
- Automatic terminal paging disablement (`terminal length 0`) on connect
- Platform-aware command building for ping and traceroute:
  - NX-OS: `ping <ip> [vrf <vrf>]`
  - IOS/IOS-XE: `ping [vrf <vrf>] <ip>`
- Parallel device processing with configurable worker pool (`--workers`, default 10)
- UTC-timestamped output directories (`<out>/<YYYYMMDD-HHMMSS>/`)
- Output tree organized by datacenter, room, rack, and hostname
- `specialized` category creates a subdirectory under the hostname folder
- Command filenames sanitized to lowercase alphanumeric slugs
- Live spinner progress tracker on TTY stderr; plain `[n/N] done` lines when piped
- Session recovery: sends Ctrl+C on command timeout and attempts to reclaim the prompt
- `--version` flag with author and license notice
- `--out`, `--port`, `--timeout`, `--workers` flags accepted
- CSV path accepted as a positional argument: `ciscocollector devices.csv`
- Single static binary with no external runtime required (CGO disabled)
- GoReleaser configuration for Linux, macOS, and Windows (amd64 and arm64)
- `.deb` and `.rpm` package targets for Linux
- Homebrew formula via `skhell/homebrew-cisco-go-collector` tap
- GitHub Actions release workflow triggered on `v*` tags

[Unreleased]: https://github.com/skhell/cisco-go-collector/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/skhell/cisco-go-collector/releases/tag/v0.1.0
