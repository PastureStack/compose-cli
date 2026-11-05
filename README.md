# PastureStack Compose CLI

Compose CLI deploys Docker Compose workloads through the established stack and service API and also provides the compatible event executor used for stack create, upgrade, finish-upgrade, and rollback operations.

PastureStack is an independent community effort to preserve, audit, and modernize the Rancher 1.6 ecosystem. It is not affiliated with or endorsed by Rancher Labs or SUSE.

**Upstream:** [`rancher/rancher-compose-executor`](https://github.com/rancher/rancher-compose-executor). This GitHub fork preserves upstream history, authorship, dates, tags, licenses, and bundled dependency notices; PastureStack maintenance is consolidated into one commit after the preserved upstream boundary.

## Project status

The maintained source uses Ubuntu 26.04, Go 1.27.3, and Python 3.14.9 for integration-source verification. Ubuntu build packages, the base image, and Go archives are fixed by snapshot, exact version, and SHA-256. Go dependencies are managed by `go.mod` and a reproducible module vendor tree. AWS access uses AWS SDK for Go v2; container structures use the maintained Moby API module; the command parser uses urfave/cli v3. The abandoned AWS SDK v1, Rancher generated-client modules, event-subscriber module, `docker/libcompose`, and other GOPATH-era dependencies have been removed. The small Rancher 1.6 protocol boundary retained by this compatibility CLI is maintained and tested locally. The Dapper builder does not embed a Docker CLI or mount a Docker socket; Docker is required only on the host to create and run the isolated builder. The maintained binaries are `pasturestack-compose` and `compose-executor`; `platform-compose.yml` is the preferred companion file. Releases are built and verified from the preserved source history; this repository does not enable automated deployment.

The current published [v0.14.41](https://github.com/PastureStack/compose-cli/releases/tag/v0.14.41) fixes
the private Catalog launch wire format: mutation DTOs contain business fields,
and an empty link action explicitly carries `serviceLinks: []`. Native link
targets and aliases are preserved. Generic API resource links/actions are
excluded from these request structs; ordinary CLI and SDK requests retain
their behavior. Original CI, reproducible Linux/amd64 packaging and all twenty
public assets passed verification. See the [release note](docs/releases/compose-executor-0.14.41.md)
for signed producer identity and scoped evidence.

The private two-phase Catalog materializer shares normal service, volume,
launch-config, secret-reference, sidekick and hash owners. Matching Engine
0.183.366 owns actor/Key limits, human approval, private inputs and the exact
child-request ledger. Web Console 1.6.205, bridge 0.2.7 and Compose 0.14.41 are published;
matching Server565 runtime/browser/role/weak-model acceptance remains pending.
See [the owner contract](docs/catalog-launch-owner.md); a Stack reply alone does
not prove workload completion.

The 0.14.42 source candidate aligns delegated child requests with that approved
preview using one typed DTO projection; ordinary/legacy SDK bytes and Engine
exact-body authorization stay unchanged. Runtime acceptance remains pending.

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

Release [v0.14.41](https://github.com/PastureStack/compose-cli/releases/tag/v0.14.41)
publishes `compose-executor-0.14.41-linux-amd64.gz`. Published tags and assets
remain immutable; a later recipe or documentation commit does not replace
their bytes or change their producer identity.

To rebuild the published source, check out `v0.14.41` and set
`VERSION_OVERRIDE=0.14.41` with its original recipe. Source changes alone are not
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

## October 2026 security maintenance

Maintained Go builds select the patched Go 1.27.3 compiler without raising the existing Go language requirement. Ubuntu build/runtime stages apply the checksum-pinned `ca-certificates 20260816ubuntu0.26.04.1` security package after their frozen APT baseline; the final trust store and installed version are verified. Python tooling uses the patched pip/CA path; CI interpreter pins remain on the Python 3.14 series. Release artifacts must pass the repository's existing checks before publication.
# Python installer security rebuild

The disposable Dapper image rebuilds pip `26.2.2+pasturestack.1` from checksum-pinned official inputs using upstream vendoring and the original urllib3 patches. Only urllib3 changes to released `2.8.0`; this is a local security rebuild, not a new official PyPA release. The expected wheel SHA-256 is `be5f63d537bbe611dc422fabb6740a44b5997888b592888df5091727f023802a`. Ubuntu's Python ABI is retained. The image tests live pip, available ensurepip/new-venv seeds, virtualenv, streaming limits, and trusted/untrusted TLS. See `scripts/pip-security/README.md` for source and license evidence.


## Disposable builder security

CI retains the complete all-severity Dapper scan and SBOM. Available fixes,
unknown severity, secrets, missing binary analysis, and findings absent from the
exact official no-fix ledger fail the gate. Reviewed Linux API headers require
the fixed package/version/PURL/advisory set and an actual header-only payload
proof; this is not a kernel or runtime exemption. Ubuntu security micro-updates
apply only to already-installed packages without changing the Python ABI.

The release tracks 99 newly observed MEDIUM findings for build-only
`linux-libc-dev 7.0.0-38.38`. Canonical lists no official fixed version; review is
due 2026-11-09. These remain tracked findings, with the gate unchanged.

The focused `Builder full-severity security` workflow validates builder-only
maintenance without rebuilding or replacing published runtime assets. Raw distro pip metadata is retained alongside the actual replaced seed
paths, wheel hash, and existing narrow pip-vendor VEX evidence.
