# Compose Executor v0.14.41 published

[v0.14.41](https://github.com/PastureStack/compose-cli/releases/tag/v0.14.41) is
published with the original official CI archive. All 20 public assets passed
actual anonymous GET, byte-count and SHA-256 verification. Matching Server565
runtime and designated Catalog workflows remain pending.

## Catalog request wire

The private preview producer uses a dedicated business-only link DTO:
`serviceLinks: []` is explicit for no links, while nonempty arrays preserve
native target IDs and aliases. Service/Volume skeletons and finalized child
request structs exclude anonymous typed SDK Resource envelopes, including
nested launch configurations. Business metadata, labels, environment and
log-driver option maps retain their keys and values; named Resource properties
are preserved.

Normal parser, conversion, service hash, secret-reference and sidekick owners
retain their behavior. Ordinary SDK/legacy serialization, Engine closed
validation and private protocol format version `1` remain unchanged.

## Published identity

| Identity | Value |
| --- | --- |
| Signed original producer / tag target | `ed3dbddda8a04522e6969fef12672bfb30f3d180` |
| Signed annotated tag | `v0.14.41` |
| Normal merged source | `d283a7787257770988d06ac770ca07b9783827dc` |
| Shared tested tree | `d6279b05f6e3ccc8f9ff28b1300979502f5079c8` |
| Linux/amd64 gzip SHA-256 | `a9f489651e1457beaf0e27e04d578ca4d0f66dd60998faa539a20f4da55b19d0` |
| Binary SHA-256 | `56536af89e4c95865550c9debd4926249fd5926da5468130bddd46c1b51867fa` |

The archive uses Go 1.27.3 with CGO disabled. The established deterministic
recipe uses `-buildvcs=false`; the signed producer, source-revision, source locks,
qualified tree and original archive bind the bytes. Later documentation commits
do not replace the producer, tag or assets. Linux/ARM64 product qualification
is not claimed.

## Validation scope

[Original Security CI](https://github.com/PastureStack/compose-cli/actions/runs/41329025852),
[Builder security](https://github.com/PastureStack/compose-cli/actions/runs/41329031497),
[CodeQL verification](https://github.com/PastureStack/compose-cli/actions/runs/41329025844)
and the PR CodeQL aggregate passed. Two complete official build/test/package
runs produced byte-identical archives. The source gate and two Linux native
compatibility runs passed with the existing pinned Engine281 JAR. These are
component compatibility checks, not Engine366 Catalog runtime acceptance.

The nine Catalog package cases passed race/coverage in both builds: three new
wire cases cover empty/nonempty links, typed nested requests and preserved
business maps and named Resource fields, and real marshaled finalize output. Six inherited
cases cover parsing, bounds, private inputs, references and closed protocol
behavior. The carried real Graph/NativeValidation proof has three offline PASS
checks for one service and zero volumes, with no resource dispatch or actor/Key
persistence test; product source is unchanged from that diagnostic candidate.

The [qualification](https://github.com/PastureStack/compose-cli/releases/download/v0.14.41/compose-executor-0.14.41-qualification.json)
and [security evidence](https://github.com/PastureStack/compose-cli/releases/download/v0.14.41/compose-executor-0.14.41-security-evidence.zip)
retain the original CI ZIP, source/product SBOMs, raw and effective scans,
exact installed Python vendor proof/VEX replay, header-only boundary and
unchanged full-severity builder gate. No product dependencies or toolchain pins
changed.

The builder ledger adds 99 newly observed MEDIUM findings for
`linux-libc-dev 7.0.0-38.38`. Canonical's pinned Ubuntu 26.04 records include
that installed version and list no fixed version; all 99 remain tracked as
`no_official_fix`, with review due 2026-11-09. These are build-only findings,
not fixed vulnerabilities or new `not_affected` statements.

Engine 0.183.366, Web Console 1.6.205 and bridge 0.2.7 are separately published
matching components. Server565 assembly/runtime and full Catalog/browser/role/
Key/cross-ID/weak-model/cleanup acceptance still need their own evidence.
