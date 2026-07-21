# Compatibility Contract

The migration preserves Docker Compose parsing, stack event names, generated stack fields, service labels, catalog question data, upgrade and rollback payloads, and compatibility API behavior.

`platform-compose.yml`, `PLATFORM_URL`, `PLATFORM_ACCESS_KEY`, and `PLATFORM_SECRET_KEY` are preferred. The historical companion filename, legacy CLI alias, `RANCHER_*` and `CATTLE_*` variables, `io.rancher.*` labels, generated `RancherCompose` and `RancherClient` types, legacy service-image identifiers, and vendored `github.com/rancher/*` paths remain only as wire, data, or dependency contracts.

At startup the maintained executor also removes an active `rancher-compose-executor` event-handler registration before installing the canonical `compose-executor` handler. This exact historical identifier is retained only as a one-way upgrade cleanup key. Without that cleanup, an upgraded database can dispatch each stack action to both executors and leave upgrades waiting on the stale handler.

Operator messages support `en-US` and `zh-TW`. Compose documents, catalog questions, API payloads, stack names, labels, and remote errors are not translated.

Before release, validate create, up, render, upgrade, finish-upgrade, rollback, service hashing, sidekicks, load balancers, secrets, volumes, legacy companion-file fallback, and real event execution.
