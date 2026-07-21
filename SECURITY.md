# Security Policy

## Supported state

This repository is under migration review and is not release-ready.

## Security boundaries

- API credentials, Compose secrets, registry credentials, build contexts, and event payloads are sensitive.
- The optional compatibility server must come from an explicitly reviewed URL and is never fetched by default.
- Build and deployment operations can execute container images and access local files referenced by Compose input.
- Do not commit credentials, private Compose files, production event payloads, or private artifact URLs.

## Reporting

Report suspected vulnerabilities through this repository's private security advisory channel. Do not include live credentials or private Compose data in a public issue.
