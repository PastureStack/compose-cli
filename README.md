# PastureStack Compose CLI

Compose CLI deploys Docker Compose workloads through the established stack and service API and also provides the compatible event executor used for stack create, upgrade, finish-upgrade, and rollback operations.

PastureStack is an independent community effort to preserve, audit, and modernize the Rancher 1.6 ecosystem. It is not affiliated with or endorsed by Rancher Labs or SUSE.

**Upstream:** [`rancher/rancher-compose-executor`](https://github.com/rancher/rancher-compose-executor). This GitHub fork preserves upstream history, authorship, dates, tags, licenses, and bundled dependency notices; PastureStack maintenance is consolidated into one commit after the preserved upstream boundary.

## Project status

The next source candidate, 0.14.45, replaces the three-second child-service
wait with bounded observation and progress replies for Stack rollback and
finish-upgrade. The event worker owns terminal errors; handler failures are
no longer swallowed. Each child action is dispatched once. Current published
artifacts remain 0.14.44 until the normal release checks complete.

The maintained source uses Ubuntu 26.04, Go 1.27.3, and Python 3.14.9 for integration-source verification. Ubuntu build packages, the base image, and Go archives are fixed by snapshot, exact version, and SHA-256. Go dependencies are managed by `go.mod` and a reproducible module vendor tree. AWS access uses AWS SDK for Go v2; container structures use the maintained Moby API module; the command parser uses urfave/cli v3. The abandoned AWS SDK v1, Rancher generated-client modules, event-subscriber module, `docker/libcompose`, and other GOPATH-era dependencies have been removed. The small Rancher 1.6 protocol boundary retained by this compatibility CLI is maintained and tested locally. The Dapper builder does not embed a Docker CLI or mount a Docker socket; Docker is required only on the host to create and run the isolated builder. The maintained binaries are `pasturestack-compose` and `compose-executor`; `platform-compose.yml` is the preferred companion file. Releases are built and verified from the preserved source history; this repository does not enable automated deployment.

The current published [v0.14.44](https://github.com/PastureStack/compose-cli/releases/tag/v0.14.44)
supports native rollback of a Stack service in `canceled-upgrade`, preserving
the existing wait and per-service action contract. Private Catalog preview and
delegated service creation retain their shared typed request and configuration
hash. Business maps, ordinary and legacy Compose hashing, SDK serialization,
and Engine authorization retain their existing behavior.

The original Go 1.27.3 CI passed two reproducible Linux/amd64 builds, race tests,
native compatibility, full security gates and CodeQL. All twenty immutable
public assets passed anonymous byte verification and native GitHub release
attestation. See the [published release note](https://github.com/PastureStack/compose-cli/releases/tag/v0.14.44).
Matching Server deployment and Stack/Catalog runtime acceptance are separate from
component qualification; a Stack reply alone does not prove workload completion.
See [the owner contract](docs/catalog-launch-owner.md).

## Configuration

Use `PLATFORM_URL`, `PLATFORM_ACCESS_KEY`, and `PLATFORM_SECRET_KEY`. Historical `RANCHER_*` and `CATTLE_*` settings remain compatibility fallbacks. Operator messages support `PASTURESTACK_LOCALE=en-US` and `zh-TW`.

The `--platform-file` flag selects the companion file. Existing `--rancher-file` and `rancher-compose.yml` inputs are accepted only as migration aliases.

## Build and test

From a Docker-capable Linux host:

```sh
make build
make test
make package
```

Release [v0.14.44](https://github.com/PastureStack/compose-cli/releases/tag/v0.14.44)
publishes `compose-executor-0.14.44-linux-amd64.gz`. Published tags and assets
remain immutable; a later recipe or documentation commit does not replace
their bytes or change their producer identity.

To rebuild the published source, check out `v0.14.44` and set
`VERSION_OVERRIDE=0.14.44` with its original recipe. Source changes alone are not
publication proof: verify the official release assets and their SHA-256.
Component publication and matching Server acceptance are separate evidence
boundaries. The Python integration suite must run against an
isolated compatible Server before integration; it must never target an
operator's live control plane.

Go 1.27.3 compiler archives are checksum-pinned for Linux/amd64 and Linux/arm64.
This release's artifact qualification covers Linux/amd64 only, not an ARM64
binary build.

The compatibility-server test requires an explicitly reviewed artifact URL through `PLATFORM_COMPAT_JAR_URL` and its exact SHA-256 through `PLATFORM_COMPAT_JAR_SHA256`; no artifact is downloaded by default. See [COMPATIBILITY.md](COMPATIBILITY.md), [SECURITY.md](SECURITY.md), and [ORIGIN.md](ORIGIN.md).

## License and attribution

The inherited project remains licensed under [Apache License 2.0](LICENSE). Copyright and attribution for inherited work and vendored dependencies remain with their respective authors and contributors. PastureStack contributors claim authorship only for their own changes.

## Toolchain security

Maintained Go builds select the patched Go 1.27.3 compiler without raising the existing Go language requirement. Ubuntu build/runtime stages apply the checksum-pinned `ca-certificates 20260816ubuntu0.26.04.1` security package after their frozen APT baseline; the final trust store and installed version are verified. Python tooling uses the patched pip/CA path; CI interpreter pins remain on the Python 3.14 series. Release artifacts must pass the repository's existing checks before publication.
# Python installer security rebuild

The disposable Dapper image rebuilds pip `26.2.2+pasturestack.2` from checksum-pinned official inputs using upstream vendoring and the original urllib3 patches. The reviewed changes use released urllib3 `2.8.0` and msgpack `1.2.3`; this is a local security rebuild, not a new official PyPA release. The expected wheel SHA-256 is `2aef7c341eef310645ab56455f5789de2819463b25fa70151954e1d208d2db36`. Ubuntu's Python ABI is retained. The image tests live pip, available ensurepip/new-venv seeds, virtualenv, streaming limits, and trusted/untrusted TLS. See `scripts/pip-security/README.md` for source and license evidence.


## Disposable builder security

CI retains the complete all-severity Dapper scan and SBOM. Available fixes,
unknown severity, secrets, missing binary analysis, and findings absent from the
exact official no-fix ledger fail the gate. Reviewed Linux API headers require
the fixed package/version/PURL/advisory set and an actual header-only payload
proof; this is not a kernel or runtime exemption. Ubuntu security micro-updates
apply only to already-installed packages without changing the Python ABI.

The published builder uses build-only `linux-libc-dev` at `7.0.0-42.42`
and the existing Brotli/PCRE2 libraries at their official fixed versions. The
current header-only scope is reviewed for that exact package; the previous
38.38 review remains historical evidence and cannot authorize the new version.

The focused `Builder full-severity security` workflow validates builder-only
maintenance without rebuilding or replacing published runtime assets. Raw distro pip metadata is retained alongside the actual replaced seed
paths, wheel hash, and existing narrow pip-vendor VEX evidence.
