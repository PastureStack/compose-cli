# Compatibility Contract

The migration preserves Docker Compose parsing, stack event names, generated stack fields, service labels, catalog question data, upgrade and rollback payloads, and compatibility API behavior.

The v0.14.40 candidate adds one server-private mode,
`compose-executor.real --catalog-launch-preview`, with fixed inspect/finalize
JSON contracts and bounded stdin/stdout. Existing parser, conversion, service
factory, launch-config and secret-reference owners produce the exact graph;
there is no second renderer or model-supplied URL/file/command interface.
Typed templates use the reviewed bounded subset and freeze the approved bytes.

Only a trusted Engine event may carry `catalogLaunchDelegation`. Its event-local
client uses the fixed delegation header for schema/read/transform/child calls;
missing or invalid proof cannot fall back to service FULL. Control progress and
replies retain the original trusted service channel, and unmarked ordinary
events retain their original client and API behavior. The Engine must recheck
the original actor/Key, source/plan/Stack/event/graph, destination create/update
and references on each request. Completion needs its committed ledger, full
intended links and normal deployment readiness, not only a Stack reply.

KVM, unsupported provider/host/build/file/dependency/network graphs, external or
cross-Stack links and unsupported typed inputs fail closed in this private mode;
they remain pending typed adapters rather than completed UI features. Component
source/unit and official CI evidence do not establish matching Server runtime.

The archived external Compose implementation is no longer a build dependency. The conversion, collection-copy, logger, and Compose configuration-hash contracts still required by this compatibility client are implemented locally and covered by behavior tests. This does not expand or silently reinterpret the accepted Compose document format.

`platform-compose.yml`, `PLATFORM_URL`, `PLATFORM_ACCESS_KEY`, and `PLATFORM_SECRET_KEY` are preferred. The historical companion filename, legacy CLI alias, `RANCHER_*` and `CATTLE_*` variables, `io.rancher.*` labels, generated `RancherCompose` and `RancherClient` types, legacy service-image identifiers, and vendored `github.com/rancher/*` paths remain only as wire, data, or dependency contracts.

At startup the maintained executor also removes an active `rancher-compose-executor` event-handler registration before installing the canonical `compose-executor` handler. This exact historical identifier is retained only as a one-way upgrade cleanup key. Without that cleanup, an upgraded database can dispatch each stack action to both executors and leave upgrades waiting on the stale handler.

Operator messages support `en-US` and `zh-TW`. Compose documents, catalog questions, API payloads, stack names, labels, and remote errors are not translated.

Before release, validate create, up, render, upgrade, finish-upgrade, rollback, service hashing, sidekicks, load balancers, secrets, volumes, legacy companion-file fallback, and real event execution.
