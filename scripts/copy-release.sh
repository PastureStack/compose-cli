#!/bin/bash
set -euo pipefail

: "${RC16_COMPOSE_RELEASE_BUCKET:?set RC16_COMPOSE_RELEASE_BUCKET to the maintained compose release bucket}"

target="${RC16_COMPOSE_RELEASE_BUCKET%/}"

gsutil -m cp -r dist/artifacts/v* "${target}"
