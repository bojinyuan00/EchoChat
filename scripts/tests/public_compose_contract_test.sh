#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/deploy/docker-compose.dev.yml"
PUBLIC_ENV="$ROOT_DIR/deploy/.env.public.example"
DEPLOY_SCRIPT="$ROOT_DIR/scripts/deploy-public.sh"
FAILURES=0

pass() { printf '[PASS] %s\n' "$*"; }
fail() { printf '[FAIL] %s\n' "$*" >&2; FAILURES=$((FAILURES + 1)); }

for mapping in \
  ECHOCHAT_SERVER_MODE \
  ECHOCHAT_SERVER_WS_ALLOWED_ORIGINS \
  ECHOCHAT_DATABASE_USER \
  ECHOCHAT_DATABASE_PASSWORD \
  ECHOCHAT_DATABASE_DBNAME \
  ECHOCHAT_REDIS_PASSWORD \
  ECHOCHAT_MINIO_ACCESS_KEY \
  ECHOCHAT_MINIO_SECRET_KEY \
  ECHOCHAT_MINIO_BUCKET; do
  if rg -q "^[[:space:]]+$mapping:" "$COMPOSE_FILE"; then
    pass "Compose maps $mapping"
  else
    fail "Compose must map $mapping"
  fi
done

for expected in \
  'GO_SERVER_MODE=release' \
  'WS_ALLOWED_ORIGINS=_REPLACE_WITH_HTTPS_ORIGINS_' \
  'MINIO_BUCKET=echochat'; do
  if rg -Fq "$expected" "$PUBLIC_ENV"; then
    pass "public env contains $expected"
  else
    fail "public env must contain $expected"
  fi
done

if rg -Fq 'config --quiet' "$DEPLOY_SCRIPT"; then
  pass 'deploy script validates rendered Compose config'
else
  fail 'deploy script must validate rendered Compose config before launch'
fi

if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' EXIT
  env_file="$tmp_dir/public.env"
  rendered="$tmp_dir/rendered.json"
  cat >"$env_file" <<'ENV'
DEPLOY_MODE=public
POSTGRES_DB=contract_db
POSTGRES_USER=contract_user
POSTGRES_PASSWORD=contract_pg_secret
REDIS_PASSWORD=contract_redis_secret
MINIO_ROOT_USER=contract_minio_user
MINIO_ROOT_PASSWORD=contract_minio_secret
MINIO_BUCKET=contract_bucket
GO_SERVER_MODE=release
WS_ALLOWED_ORIGINS=https://chat.example.test,https://admin.example.test
JWT_SECRET=contract_jwt_secret
MEDIA_INTERNAL_TOKEN=contract_media_token
MEDIASOUP_ANNOUNCED_IP=203.0.113.10
TURN_USERNAME=contract_turn_user
TURN_PASSWORD=contract_turn_secret
ENV

  if docker compose --env-file "$env_file" -f "$COMPOSE_FILE" --profile public config --format json >"$rendered"; then
    if /usr/bin/python3 - "$rendered" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    services = json.load(handle)["services"]

go_env = services["go-service"]["environment"]
expected = {
    "ECHOCHAT_SERVER_MODE": "release",
    "ECHOCHAT_SERVER_WS_ALLOWED_ORIGINS": "https://chat.example.test,https://admin.example.test",
    "ECHOCHAT_DATABASE_USER": "contract_user",
    "ECHOCHAT_DATABASE_PASSWORD": "contract_pg_secret",
    "ECHOCHAT_DATABASE_DBNAME": "contract_db",
    "ECHOCHAT_REDIS_PASSWORD": "contract_redis_secret",
    "ECHOCHAT_MINIO_ACCESS_KEY": "contract_minio_user",
    "ECHOCHAT_MINIO_SECRET_KEY": "contract_minio_secret",
    "ECHOCHAT_MINIO_BUCKET": "contract_bucket",
    "ECHOCHAT_JWT_SECRET": "contract_jwt_secret",
    "ECHOCHAT_MEDIA_SERVER_INTERNAL_TOKEN": "contract_media_token",
}
for key, value in expected.items():
    actual = go_env.get(key)
    if actual != value:
        raise SystemExit(f"go-service {key}: expected {value!r}, got {actual!r}")

redis = services["redis"]
if redis.get("environment", {}).get("REDIS_PASSWORD") != "contract_redis_secret":
    raise SystemExit("Redis password was not propagated")
command = " ".join(redis.get("command", [])) if isinstance(redis.get("command"), list) else str(redis.get("command"))
health_test = " ".join(redis.get("healthcheck", {}).get("test", []))
if "--requirepass" not in command:
    raise SystemExit("Redis command does not enable requirepass conditionally")
if "REDISCLI_AUTH" not in health_test:
    raise SystemExit("Redis healthcheck does not authenticate")
PY
    then
      pass 'rendered Compose propagates all public values'
    else
      fail 'rendered Compose public value propagation is incomplete'
    fi
  else
    fail 'docker compose could not render the public profile'
  fi
else
  printf '[SKIP] Docker Compose unavailable; static assertions still executed.\n'
fi

if (( FAILURES > 0 )); then
  printf '\nPublic Compose contract failed with %d issue(s).\n' "$FAILURES" >&2
  exit 1
fi

printf '\nPublic Compose contract passed.\n'
