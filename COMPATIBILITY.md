# Compatibility Contract

## 0.14.42 published: identical approved and delegated mutation DTOs

Private preview and proof-bearing HTTP mutations now share the typed SDK
request projection, including nested launch configurations and explicit empty
service links. Business maps keep their keys and values; ordinary and legacy
SDK bytes remain unchanged. Engine exact-body authorization is not relaxed.
Original official CI, two reproducible packages, all 20 anonymous assets and
native GitHub release attestation passed. Matching Server runtime remains pending.
The builder retains the original 22 tools and Go1.27.3; its selected official
security overlay preserves installed package names and the Python ABI.

## 0.14.41 published: private Catalog request wire

The private Catalog preview producer emits business-only mutation inputs.
`setServiceLinks` always includes an array, including `[]` for no links, and
retains native target IDs and aliases. Service/Volume creation skeletons and
finalized child request structs exclude the generic SDK Resource envelope,
including nested launch configs. Business maps such as metadata, labels,
environment and log-driver options retain their exact keys and values.

This changes only the private preview serialization. Format version `1`,
normal parser/DTO owners, ordinary CLI/SDK serialization and Engine closed
validation remain unchanged. Official Security/Builder/CodeQL gates, two
byte-identical packages and all 20 anonymously read public assets passed.
Producer `ed3dbddda8a04522e6969fef12672bfb30f3d180` is the signed v0.14.41 tag
target; merged source `d283a7787257770988d06ac770ca07b9783827dc` shares tree
`d6279b05f6e3ccc8f9ff28b1300979502f5079c8`.

Three new wire cases and six inherited Catalog cases passed official race
coverage in both builds. Two Linux native compatibility runs use the existing
pinned Engine281 JAR; they do not establish Engine366 Catalog runtime acceptance.
Matching Server565/runtime acceptance remains pending. See the
[published note](docs/releases/compose-executor-0.14.41.md).

The migration preserves Docker Compose parsing, stack event names, generated stack fields, service labels, catalog question data, upgrade and rollback payloads, and compatibility API behavior.

The server-private mode introduced in v0.14.40 remains
`compose-executor.real --catalog-launch-preview`, with fixed inspect/finalize
JSON contracts and bounded stdin/stdout. Existing parser, conversion, service
factory, launch-config and secret-reference owners produce the exact graph;
there is no second renderer or model-supplied URL/file/command interface.
Typed templates use the reviewed bounded subset and freeze the approved bytes.

Only a trusted Engine event may carry `catalogLaunchDelegation`. Its event-local
client uses the fixed delegation header for schema/read/transform/child calls;
missing or invalid proof cannot fall back to service FULL. Control progress and
replies retain the original trusted service channel, and unmarked ordinary
events retain their original client and API behavior. The matching Engine must recheck
the original actor/Key, source/plan/Stack/event/graph, destination read/create/update
and references on each request. Completion needs its committed ledger, full
intended links and normal deployment readiness, not only a Stack reply.

KVM, unsupported provider/host/build/file/dependency/network graphs, external or
cross-Stack links and unsupported typed inputs fail closed in this private mode;
they remain pending typed adapters rather than completed UI features. Component
source/unit and official CI evidence do not establish matching Server runtime.
Official component gates and original public assets passed verification;
matching Server565 runtime/browser/role/weak-model acceptance remains pending.
Use Catalog 0.20.17 and the matching Engine 0.183.366 private-proxy/delegation boundary;
an older Engine without those guards is not a compatible launch gateway.

The archived external Compose implementation is no longer a build dependency. The conversion, collection-copy, logger, and Compose configuration-hash contracts still required by this compatibility client are implemented locally and covered by behavior tests. This does not expand or silently reinterpret the accepted Compose document format.

`platform-compose.yml`, `PLATFORM_URL`, `PLATFORM_ACCESS_KEY`, and `PLATFORM_SECRET_KEY` are preferred. The historical companion filename, legacy CLI alias, `RANCHER_*` and `CATTLE_*` variables, `io.rancher.*` labels, generated `RancherCompose` and `RancherClient` types, legacy service-image identifiers, and vendored `github.com/rancher/*` paths remain only as wire, data, or dependency contracts.

At startup the maintained executor also removes an active `rancher-compose-executor` event-handler registration before installing the canonical `compose-executor` handler. This exact historical identifier is retained only as a one-way upgrade cleanup key. Without that cleanup, an upgraded database can dispatch each stack action to both executors and leave upgrades waiting on the stale handler.

Operator messages support `en-US` and `zh-TW`. Compose documents, catalog questions, API payloads, stack names, labels, and remote errors are not translated.

Before release, validate create, up, render, upgrade, finish-upgrade, rollback, service hashing, sidekicks, load balancers, secrets, volumes, legacy companion-file fallback, and real event execution.
