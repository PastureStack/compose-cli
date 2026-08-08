#!/bin/bash
set -euo pipefail

: "${RC16_COMPOSE_LATEST_BUCKET:?set RC16_COMPOSE_LATEST_BUCKET to the maintained compose/latest bucket}"

target="${RC16_COMPOSE_LATEST_BUCKET%/}"

gsutil -m rsync -r dist/artifacts/latest/ "${target}"
