# Compose Executor v0.14.44

Stack rollback now recognizes the native `canceled-upgrade` service state.
After the existing wait completes, it calls the native Service rollback action
for `upgraded` and `canceled-upgrade` services. Active services remain untouched;
an upgrading service must settle through the existing wait before dispatch.

The real HTTP-client regression covers canceled, upgraded, active and
upgrading-to-settled children. Native action ordering, Stack replies, errors and
timeouts retain their existing behavior. The matching Engine owns restoring
eligible previous configurations and validating the approved Stack snapshot.
The AI layer does not add direct Service requests or rewrite configuration.

Builds retain Go 1.27.3, the existing Go language requirement and module/vendor
graph. The disposable builder uses the checksum-pinned
`26.2.2+pasturestack.2` pip rebuild. Native Debian package version/release
matching remains exact; raw findings, official no-fix tracking and full-severity
security checks remain available. The compatibility JAR uses its official asset
API with the original complete SHA-256 check.

Release qualification requires the original CI artifact, two byte-identical
Linux/amd64 builds, race tests, compatibility checks, full security evidence and
CodeQL. Published assets and their source/attestation records are immutable.
Component publication is separate from matching Server and Stack runtime
acceptance; a successful Stack reply alone does not establish workload readiness.

Keep the previous qualified component and matching Server rollback image when
upgrading. Restoring an older executor also restores its canceled-upgrade
limitation. Do not remove workload data or volumes as a rollback shortcut.
