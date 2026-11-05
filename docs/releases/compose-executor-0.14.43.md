# Compose Executor v0.14.43 — release candidate

This candidate is not published. The current published release remains
[v0.14.42](https://github.com/PastureStack/compose-cli/releases/tag/v0.14.42).

Catalog preview and delegated creation now share one hash owner. It projects
only typed SDK response envelopes before hashing and hashes the owned Stack
binding symbolically, while the actual request retains its exact bound Stack.
This keeps the approved body stable across the native transform response and
Stack creation. The Engine's exact-body checks and authorization are unchanged.

Business metadata, labels, environment, ports, volumes and security settings
remain intact. Empty hash-output maps are restored without changing user data.

Ordinary and legacy Compose hash and serialization paths remain unchanged.
An explicit ordinary Compose `up` or `upgrade` that later takes over a service
created with the Catalog hash may detect one initial hash difference. This
does not trigger an automatic update or relax any permission check.

Local focused source tests are not matching Server/Catalog runtime acceptance.
Publication requires the original CI artifact and component-bound security
qualification; no runtime result is claimed by this note.
