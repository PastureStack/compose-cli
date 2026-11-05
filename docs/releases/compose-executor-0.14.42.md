# Compose Executor v0.14.42

Catalog launches send the same business-only mutation DTOs that the human
approved in the private preview. The actual delegated SDK path now shares the
preview projection for Service, VolumeTemplate and nested launch configs;
empty service-link actions explicitly carry `serviceLinks: []`.

SDK response-envelope fields are removed only at typed DTO boundaries.
Metadata, labels, environment, driver options and other business maps retain
their exact keys and values, including keys named `id`, `type`, `links` or
`actions`. Ordinary and legacy SDK serialization stays byte-identical.

The Engine's exact-body checks, original actor/Key authorization, event proof,
origin checks and one-claim ledger remain unchanged. Component qualification
and publication do not establish matching Server/Catalog runtime acceptance.
