# Compose Executor v0.14.40 candidate

Status: unpublished verification candidate. The current immutable component is
[v0.14.39](https://github.com/PastureStack/compose-cli/releases/tag/v0.14.39).
The existing Go 1.27.3 build/test/race/reproducibility/security/CodeQL gates remain
required. Matching Engine/Web/Server launch acceptance is not yet established.

The fixed private inspect/finalize mode produces frozen Compose and exact
service/volume/links DTOs through the normal owner functions. It accepts bounded
server-private JSON, not a URL, local file, service credential or arbitrary
command. Finalize requires complete transformed launch configs and verified
secret references; unknown, missing, extra or unsupported graph shapes fail
closed. The normal CLI parser and unmarked executor behavior remain in place.

The trusted launch event creates only an event-local proof-bearing API client.
It binds the exact event and Stack; shared service clients and control replies
are unchanged. The Engine must validate original live account/Key permissions,
revision/revocation, claimed plan, Stack/event/graph and each normal child
request. It must commit business mutation plus ledger before responding and
verify full intended links and deployment readiness before reporting success.
The proof, private Compose, answers and credentials must not enter public tools,
logs or browser storage.

No Go module/vendor/toolchain or compatibility-artifact pin is changed. Existing
ordinary compatibility tests remain required; the old compatibility Server does
not establish new delegated Catalog launch acceptance. Unsupported typed graph
features remain partial, and published 0.14.39 bytes are never overwritten.
