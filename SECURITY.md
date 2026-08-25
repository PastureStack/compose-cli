# Security Policy

## Supported state

This repository is under migration review and is not release-ready.

## Security boundaries

- API credentials, Compose secrets, registry credentials, build contexts, and event payloads are sensitive.
- The compatibility server must come from an explicitly reviewed URL and is verified against the required exact SHA-256 before execution; it is never fetched by default.
- Remote in-memory Compose input cannot read executor-host files through `env_file`, `extends`, or secret file references.
- Local CLI file references are limited to the selected Compose-file directories and are opened through a root-scoped filesystem handle that rejects traversal and symbolic-link escape.
- CI permits only unfixed `linux-libc-dev` findings from the disposable builder,
  generates the corresponding OpenVEX evidence from that scan, rejects any
  fixed or different finding, rejects kernel image or module packages, and
  separately scans the shipped CGO-disabled binary.
- [`security/openvex.json`](security/openvex.json) records the reviewed `GO-2026-5932` applicability decision. The advisory is limited to the discontinued `golang.org/x/crypto/openpgp` package. This project uses maintained `bcrypt` and `scrypt` packages through Sprig; the source gate enumerates the complete package graph and fails if `openpgp` ever becomes a dependency. The VEX product identity is pinned to the resolved x/crypto version and must change with it.
- Do not commit credentials, private Compose files, production event payloads, or private artifact URLs.

## Reporting

Report suspected vulnerabilities through this repository's private security advisory channel. Do not include live credentials or private Compose data in a public issue.
