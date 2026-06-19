# Security

## Scope

This document covers security considerations for `ciscocollector` and instructions for responsible disclosure.

---

## Known security posture

### SSH host key verification

`ciscocollector` currently disables SSH host key verification (`InsecureIgnoreHostKey`). This means the tool does not verify that it is connecting to the intended device, and a machine-in-the-middle attack is possible if the management network is untrusted.

**Mitigation:** run `ciscocollector` exclusively from a dedicated management host or jump server on a network segment where rogue devices cannot impersonate production equipment. Do not use this tool over untrusted or public networks.

### Credentials

Credentials are prompted at runtime and are never written to disk, environment variables, or log files. The password prompt does not echo input to the terminal.

Credentials are held in memory for the duration of the run and are not cached between invocations.

### Output files

Output files contain raw CLI text from your network devices. This data may include IP addresses, routing tables, VLAN configurations, and other operational details that are sensitive in your environment.

- Store output directories on a host with appropriate access controls.
- Restrict read access to authorized personnel only.
- Apply your organization's data retention policy to the output folder before sharing or archiving.

### Parallel connections

The default worker pool opens up to 10 concurrent SSH sessions. Adjust `--workers` to stay within any concurrent-session limits enforced by your environment.

---

## Responsible disclosure

If you discover a security vulnerability in `ciscocollector`, please report it privately before disclosing it publicly.

**GitHub:** open a [Security Advisory](https://github.com/skhell/cisco-go-collector/security/advisories/new) on this repository (preferred for tracking).

Please include:

- A description of the vulnerability and its impact
- Steps to reproduce or a proof-of-concept
- The version of `ciscocollector` affected

You can expect an acknowledgement within 7 business days. Fixes for confirmed vulnerabilities are prioritized for the next release.

---

## Out of scope

The following are considered out of scope for security reports:

- Vulnerabilities in upstream dependencies (report those to the respective project)
- Issues that require physical access to the management host
- Denial-of-service against network devices via excessive command execution
