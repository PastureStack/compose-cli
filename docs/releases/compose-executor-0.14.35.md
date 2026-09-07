# Compose Executor v0.14.35

Support runtime, PID limits and GPU requests in the existing v1/v2 Compose path.
Accept `gpus: all`, the GPU request list form, and
`deploy.resources.reservations.devices`; reject unrelated deploy fields rather
than silently discarding them. Count and device IDs are mutually exclusive.

Preserve shared memory, CPU limits, device mappings and supplementary groups
through the Docker HostConfig-to-LaunchConfig API conversion. Propagate API
validation errors instead of returning a success-shaped empty configuration.

These options require compatible orchestration-engine and node-agent releases.
They do not add full Compose deploy support or GPU exclusive scheduling.
