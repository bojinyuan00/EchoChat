#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

test -x "$ROOT_DIR/scripts/verify-baseline.sh"
"$ROOT_DIR/scripts/verify-baseline.sh"

rg -q 'stop_frontend' "$ROOT_DIR/scripts/stop.sh"
rg -q 'stop_admin' "$ROOT_DIR/scripts/stop.sh"
rg -q 'ECHOCHAT_DATABASE_PASSWORD' "$ROOT_DIR/deploy/docker-compose.dev.yml"
rg -q 'ECHOCHAT_REDIS_PASSWORD' "$ROOT_DIR/deploy/docker-compose.dev.yml"
rg -q 'ECHOCHAT_MINIO_SECRET_KEY' "$ROOT_DIR/deploy/docker-compose.dev.yml"
