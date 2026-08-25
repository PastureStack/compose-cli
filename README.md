# PastureStack Compose CLI

Compose CLI deploys Docker Compose workloads through the established stack and service API and also provides the compatible event executor used for stack create, upgrade, finish-upgrade, and rollback operations.

PastureStack is an independent community effort to preserve, audit, and modernize the Rancher 1.6 ecosystem. It is not affiliated with or endorsed by Rancher Labs or SUSE.

**Upstream:** [`rancher/rancher-compose-executor`](https://github.com/rancher/rancher-compose-executor). This GitHub fork preserves upstream history, authorship, dates, tags, licenses, and bundled dependency notices; PastureStack maintenance is consolidated into one commit after the preserved upstream boundary.

## Project status

The maintained release uses Ubuntu 26.04, Go 1.27.0, and Python 3.14.7 for integration-source verification. Ubuntu build packages, the base image, and Go archives are fixed by snapshot, exact version, and SHA-256. Go dependencies are managed by `go.mod` and a reproducible module vendor tree. AWS access uses AWS SDK for Go v2; container structures use the maintained Moby API module; the command parser uses urfave/cli v3. The abandoned AWS SDK v1, Rancher generated-client modules, event-subscriber module, `docker/libcompose`, and other GOPATH-era dependencies have been removed. The small Rancher 1.6 protocol boundary retained by this compatibility CLI is maintained and tested locally. The Dapper builder does not embed a Docker CLI or mount a Docker socket; Docker is required only on the host to create and run the isolated builder. The maintained binaries are `pasturestack-compose` and `compose-executor`; `platform-compose.yml` is the preferred companion file. Releases are built and verified from the preserved source history; this repository does not enable automated deployment.

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

Set `VERSION_OVERRIDE=0.14.33` for the current maintenance candidate. Packaging produces the deterministic, versioned `compose-executor-0.14.33-linux-amd64.gz` asset for a matching future `PastureStack/server` GitHub Release. The Python integration suite must run against an isolated compatible Server before integration; it must never target an operator's live control plane.

The compatibility-server test requires an explicitly reviewed artifact URL through `PLATFORM_COMPAT_JAR_URL` and its exact SHA-256 through `PLATFORM_COMPAT_JAR_SHA256`; no artifact is downloaded by default. See [COMPATIBILITY.md](COMPATIBILITY.md), [SECURITY.md](SECURITY.md), and [ORIGIN.md](ORIGIN.md).

## License and attribution

The inherited project remains licensed under [Apache License 2.0](LICENSE). Copyright and attribution for inherited work and vendored dependencies remain with their respective authors and contributors. PastureStack contributors claim authorship only for their own changes.
