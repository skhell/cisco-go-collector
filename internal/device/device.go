// Package device loads the CSV inventory and builds the per-device command list,
// including platform-aware syntax for ping and traceroute with optional VRF support.
package device

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

type Command struct {
	Category string
	Command  string // full command string sent to the device
	Filename string // stem used for the output file (before .txt); may differ from Command
}

type Device struct {
	Datacenter string
	Room       string
	Rack       string
	Hostname   string
	IP         string
	Platform   string
	Commands   []Command
}

func Load(path string) ([]*Device, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	col := make(map[string]int, len(header))
	for i, name := range header {
		col[name] = i
	}
	for _, required := range []string{"datacenter", "room", "rack", "hostname", "ip", "platform", "category", "command"} {
		if _, ok := col[required]; !ok {
			return nil, fmt.Errorf("File is missing required column %q in %s", required, path)
		}
	}

	// target_ip and vrf are optional columns.
	_, hasTargetIP := col["target_ip"]
	_, hasVRF := col["vrf"]

	order := make([]string, 0)
	byHostname := make(map[string]*Device)

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read row: %w", err)
		}

		hostname := row[col["hostname"]]
		verb := row[col["command"]]
		if hostname == "" || verb == "" {
			continue
		}

		d, ok := byHostname[hostname]
		if !ok {
			d = &Device{
				Datacenter: row[col["datacenter"]],
				Room:       row[col["room"]],
				Rack:       row[col["rack"]],
				Hostname:   hostname,
				IP:         row[col["ip"]],
				Platform:   row[col["platform"]],
			}
			byHostname[hostname] = d
			order = append(order, hostname)
		}

		var targetIP, vrf string
		if hasTargetIP {
			targetIP = strings.TrimSpace(row[col["target_ip"]])
		}
		if hasVRF {
			vrf = strings.TrimSpace(row[col["vrf"]])
		}

		cmd := buildCommand(verb, targetIP, vrf, d.Platform)
		cmd.Category = row[col["category"]]
		d.Commands = append(d.Commands, cmd)
	}

	devices := make([]*Device, 0, len(order))
	for _, hostname := range order {
		devices = append(devices, byHostname[hostname])
	}
	return devices, nil
}

// buildCommand constructs a Command, handling ping/traceroute specially when
// target_ip is set. Platform syntax:
//
//	nx-os:           ping <ip> [vrf <vrf>]
//	ios / ios-xe:    ping [vrf <vrf>] <ip>
func buildCommand(verb, targetIP, vrf, platform string) Command {
	if targetIP == "" {
		// Regular show/exec command - filename matches the command.
		return Command{Category: "", Command: verb, Filename: verb}
	}

	lcVerb := strings.ToLower(verb)
	if lcVerb != "ping" && lcVerb != "traceroute" {
		// Unrecognised verb with a target, treat as a regular command.
		return Command{Command: verb, Filename: verb}
	}

	var fullCmd string
	if strings.EqualFold(platform, "nx-os") || strings.EqualFold(platform, "nxos") {
		// NX-OS: ping <ip> [vrf <vrf>]
		if vrf != "" {
			fullCmd = fmt.Sprintf("%s %s vrf %s", lcVerb, targetIP, vrf)
		} else {
			fullCmd = fmt.Sprintf("%s %s", lcVerb, targetIP)
		}
	} else {
		// IOS / IOS-XE: ping [vrf <vrf>] <ip>
		if vrf != "" {
			fullCmd = fmt.Sprintf("%s vrf %s %s", lcVerb, vrf, targetIP)
		} else {
			fullCmd = fmt.Sprintf("%s %s", lcVerb, targetIP)
		}
	}

	// Filename is always verb-first with vrf before ip so it is consistent
	// across platforms: ping_1_1_1_1 or ping_vrf_mgmt_1_1_1_1.
	var filename string
	if vrf != "" {
		filename = fmt.Sprintf("%s vrf %s %s", lcVerb, vrf, targetIP)
	} else {
		filename = fmt.Sprintf("%s %s", lcVerb, targetIP)
	}

	return Command{Command: fullCmd, Filename: filename}
}
