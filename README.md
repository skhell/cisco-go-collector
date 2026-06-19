# Cisco Go collector

[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/skhell/cisco-go-collector/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/skhell/cisco-go-collector)](https://goreportcard.com/report/github.com/skhell/cisco-go-collector)
[![Release](https://img.shields.io/github/v/release/skhell/cisco-go-collector)](https://github.com/skhell/cisco-go-collector/releases)

A simple portable and fast CLI tool that connects to Cisco NX-OS and IOS-XE devices over SSH, runs commands defined in a CSV file and saves the output as organized plain-text files.

Built for engineers and architects who need repeatable CLI evidence collection without installing Python and operate complicate scripts, dial with Ansible or any other framework overhead.


## Why

During migrations, pre/post-checks, and RCA sessions, collecting the same command outputs manually across many devices is slow, inconsistent and error-prone specially under pressure. Most automation frameworks solve this at the cost of a setup complexity.

`ciscocollector` keeps the workflow super minimal:

1. Define devices and commands in a CSV file.
2. Run a single binary `ciscocollector filename.csv`.
3. Collect organized, timestamped text output.

The CSV is the operational input, the binary handles SSH, the output folder is the evidence.


## Features

- Single static binary no extra programs to install
- CSV-driven: devices, platforms and commands in one file
- Fast parallel execution across devices (configurable worker pool)
- Supports initially Cisco NX-OS and IOS-XE/IOS
- Automatic paging disablement (`terminal length 0`)
- VRF-aware ping and traceroute with platform-correct syntax
- Session recovery via Ctrl+C on command timeout
- Output organized by datacenter/room/rack/hostname with UTC timestamps
- Live spinner progress on TTY, clean line output when piped


## Installation

Download the latest release for your platform from the [Releases](https://github.com/skhell/cisco-go-collector/releases) page.

### Homebrew

```sh
brew tap skhell/cisco-go-collector
brew install ciscocollector
```

### Linux packages

`.deb` and `.rpm` packages are available on the [Releases](https://github.com/skhell/cisco-go-collector/releases) page.

### Windows

`.exe` package is available on the [Releases](https://github.com/skhell/cisco-go-collector/releases) page.

### Build from source

```sh
git clone https://github.com/skhell/cisco-go-collector.git
cd cisco-go-collector
go build -o ciscocollector ./cmd/ciscocollector
```

## Usage

```sh
ciscocollector [options] devices.csv

Options:
  --out       string   base output directory (default "output")
  --workers   int      number of devices processed in parallel (default 10)
  --port      int      SSH port (default 22)
  --timeout   duration per-command and connection timeout (default 30s)
  --version            print version information and exit
```

### Examples

```sh
# Basic run
ciscocollector devices.csv

# Custom output directory and higher parallelism
ciscocollector --out /tmp/audit --workers 20 devices.csv

# Non-standard SSH port and longer timeout
ciscocollector --port 2222 --timeout 60s devices.csv

# Version info
ciscocollector --version
```

The tool prompts for username and password at startup. The password is read without echo.


## CSV format

The CSV file drives everything, each row maps one command to one device.

**Required columns:** `datacenter`, `room`, `rack`, `hostname`, `ip`, `platform`, `category`, `command`

**Optional columns:** `target_ip`, `vrf`


| datacenter | room | rack | hostname | ip | platform | category | command | target_ip | vrf |
|---|---|---|---|---|---|---|---|---|---|
|DC1|ROOM-A|12|MPX01|10.10.10.100|nx-os|common|show clock||
|DC1|ROOM-A|12|MPX01|10.10.10.100|nx-os|common|show vlan brief||
|DC1|ROOM-A|12|MPX01|10.10.10.100|nx-os|common|show pbr static summary||
|DC1|ROOM-A|12|MPX01|10.10.10.100|nx-os|specialized|show ip route ospf-xxx vrf all||
|DC1|ROOM-A|12|MPX01|10.10.10.100|nx-os|specialized|show ip ospf neighbor vrf all||
|DC1|ROOM-B|58|MPX02|10.10.30.100|ios-xe|connectivity|traceroute|10.0.0.1|VRFNAME|
|DC1|ROOM-B|58|MPX02|10.10.30.100|ios-xe|connectivity|ping|10.0.0.1|VRFNAME|
|DC1|ROOM-B|60|MPX03|10.10.20.100|nx-os|connectivity|traceroute|1.1.1.1||
|DC1|ROOM-B|60|MPX03|10.10.20.100|nx-os|connectivity|ping|1.1.1.1||

**Platform values:** `nx-os` (or `nxos`), `ios-xe`, `ios`

**Category values:** `common`, `specialized`, or any label. Only `specialized` creates a subdirectory.

**ping / traceroute rows:** set `command` to `ping` or `traceroute`, fill `target_ip`, and optionally `vrf`. The tool builds the correct platform syntax automatically.


## Output structure

Results are saved under `<out>/<UTC-timestamp>/<datacenter>/<room>/<rack>/<hostname>/`.

```
output/
+-- 20260619-143022/
    +-- DC1/
        +-- ROOM-A/
            +-- CORE/
                +-- MPX01/
                |   +-- show_clock.txt
                |   +-- show_vlan_brief.txt
                |   +-- show_pbr_static_summary.txt
                |   +-- specialized/
                |       +-- show_ip_route_ospf_xxx_vrf_all.txt
                |       +-- show_ip_ospf_neighbor_vrf_all.txt
                +-- MPX02/
                    +-- traceroute_vrf_vrfname_10_0_0_1.txt
                    +-- ping_vrf_vrfname_10_0_0_1.txt
```

Each `.txt` file contains the raw CLI output for that command, with the echo and device prompt stripped.


## Use cases

- Pre/post-migration configuration snapshots
- Multi-device command collection for RCA documentation
- Quick audit evidence gathering across DC networks

## Scope

`ciscocollector` is not intended to replace Ansible, Nornir, Netmiko, pyATS, or Cisco NSO. It covers a specific and deliberate operational need: fast, repeatable Cisco CLI output collection from a plain CSV file, with no dependencies and no setup.


## Security

See [SECURITY.md](SECURITY.md) for full details and responsible disclosure instructions.


## Feedback

- Star the project on [GitHub](https://github.com/skhell/cisco-go-collector)
- Report bugs or request features in [Issues](https://github.com/skhell/cisco-go-collector/issues)
- [Buy me a coffee](https://buymeacoffee.com/skhell), or a snack for my Schnauzer Tyson via [GitHub Sponsors](https://github.com/sponsors/skhell)
