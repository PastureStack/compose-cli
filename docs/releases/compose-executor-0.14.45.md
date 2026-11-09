# Compose Executor v0.14.45

Stack rollback and finish-upgrade no longer abandon a transitioning service
after six half-second polls. Each child is observed for up to five minutes.
While waiting, progress replies keep the original Engine event alive through
its existing keepalive contract. The executor does not repeat the Stack
handler or reissue an already dispatched child action.

The old WithTimeout adapter swallowed every handler error. Handlers now return
their original errors to the event worker, which alone publishes the terminal
error reply. Timeout, reload, child-transition and progress-publish failures
remain failures rather than a missing completion response. Active children,
previous Stack environment restoration and native service actions are preserved.

Deterministic-clock regressions cover children settling beyond both the old
three-second limit and the Engine's ordinary event timeout, periodic original-
event keepalive, exactly-once rollback/finish dispatch, active-child isolation,
the bounded deadline and failure propagation. No wall-clock sleep determines
test success. Existing Catalog delegation and control-channel tests remain in
the affected-package checks. The toolchain, dependency graph and security gates
are unchanged; normal release qualification still requires reproducible builds,
race tests, native compatibility, full security evidence and CodeQL.

Matching Server runtime acceptance is separate from component qualification.
A successful Stack reply alone does not prove workload readiness or data
preservation. Keep the previous qualified Server image for rollback. Restoring
0.14.44 also restores its short-wait/error-suppression limitation; do not delete
workload data, volumes or an unresolved operation to recover it.
