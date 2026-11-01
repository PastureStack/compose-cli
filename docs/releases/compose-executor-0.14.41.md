# Compose Executor v0.14.41 source candidate

Status: unpublished source candidate. The latest published component remains
[v0.14.40](https://github.com/PastureStack/compose-cli/releases/tag/v0.14.40).

The private Catalog preview producer previously serialized generic SDK Resource
envelopes into mutation inputs. Empty link lists were also omitted. A strict
Engine rejected the resulting link action before native child preflight.

The producer now uses a dedicated business-only link DTO with explicit
`serviceLinks: []` for no links, retaining native target IDs and aliases.
Service/Volume skeletons and finalized child request structs exclude the typed
Resource envelope, including nested launch configs. Metadata, labels,
environment and log-driver option maps retain their keys and values; this is
not a recursive deletion of similarly named business fields.

Normal parser, conversion, service hash, secret-reference and sidekick owners
are unchanged. Ordinary SDK/legacy serialization and Engine closed validation
are unchanged. The private protocol retains format version `1`.

The release workflow version, artifact name and binary-version expectation
are synchronized to 0.14.41. No module, vendor, toolchain or compatibility JAR
pin changes. Official producer CI, race/security/CodeQL gates, original artifact
qualification, publication and matching Server runtime acceptance remain
required. Offline component checks do not establish native launch completion
or full Catalog/role/weak-model acceptance.
