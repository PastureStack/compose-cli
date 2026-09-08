# Compose Executor v0.14.36

Preserve mapping-style `environment` and `labels` values after the YAML v3
parser migration. Nested service maps are now preprocessed and interpolated as
structured values instead of being flattened into a single Go string.

This restores stack create and upgrade requests that use object-form
environment variables or labels, including the PastureStack infrastructure
catalog templates. List-form values and the hardware options added in v0.14.35
remain unchanged.
