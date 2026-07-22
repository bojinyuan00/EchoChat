#!/usr/bin/env bash
# Verify source-derived API counts, documentation links, and deployment mappings.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
FAILURES=0

ok() { printf '[OK]   %s\n' "$*"; }
fail() {
  printf '[FAIL] %s\n' "$*" >&2
  FAILURES=$((FAILURES + 1))
}

check_file() {
  local path="$1"
  if [[ -f "$ROOT_DIR/$path" ]]; then
    ok "$path exists"
  else
    fail "$path is missing"
  fi
}

check_text() {
  local path="$1"
  local expected="$2"
  if rg -Fq "$expected" "$ROOT_DIR/$path" 2>/dev/null; then
    ok "$path contains current baseline: $expected"
  else
    fail "$path is missing current baseline: $expected"
  fi
}

check_count() {
  local label="$1"
  local actual="$2"
  local expected="$3"
  if [[ "$actual" == "$expected" ]]; then
    ok "$label = $expected"
  else
    fail "$label expected $expected, got $actual"
  fi
}

route_count() {
  rg -c 'authed\.(GET|POST|PUT|DELETE|PATCH)\(' "$1" 2>/dev/null || echo 0
}

check_markdown_links() {
  local links_file
  links_file="$(mktemp)"
  trap 'rm -f "$links_file"' RETURN

  local files=()
  while IFS= read -r file; do
    files+=("$ROOT_DIR/$file")
  done < <(cd "$ROOT_DIR" && rg --files README.md docs -g '*.md' -g '!docs/superpowers/**')

  perl -ne '
    while (/\[[^\]]+\]\(([^)]+)\)/g) {
      my $target = $1;
      $target =~ s/^<|>$//g;
      $target =~ s/\s+"[^"]*"$//;
      print "$ARGV\t$target\n";
    }
  ' "${files[@]}" >"$links_file"

  local source target clean resolved
  local before="$FAILURES"
  while IFS=$'\t' read -r source target; do
    case "$target" in
      ''|'#'*|http://*|https://*|mailto:*|data:*|javascript:*) continue ;;
    esac
    clean="${target%%#*}"
    clean="${clean%%\?*}"
    [[ -z "$clean" ]] && continue
    if [[ "$clean" == /* ]]; then
      resolved="$ROOT_DIR$clean"
    else
      resolved="$(dirname "$source")/$clean"
    fi
    if [[ ! -e "$resolved" ]]; then
      fail "broken Markdown link: ${source#"$ROOT_DIR/"} -> $target"
    fi
  done <"$links_file"

  if [[ "$FAILURES" == "$before" ]]; then
    ok "local Markdown links resolve"
  fi
  rm -f "$links_file"
  trap - RETURN
}

check_file "docs/api/frontend/group.md"
check_file "docs/api/admin/group.md"

check_count "IM REST routes" \
  "$(route_count "$ROOT_DIR/backend/go-service/app/im/router.go")" 9
check_count "group REST routes" \
  "$(route_count "$ROOT_DIR/backend/go-service/app/group/router.go")" 19
check_count "meeting REST routes" \
  "$(route_count "$ROOT_DIR/backend/go-service/app/meeting/router.go")" 12

meeting_client_events="$(awk '
  /var MeetingWSClientEvents = \[\]string\{/ { inside=1; next }
  inside && /^}/ { inside=0 }
  inside && /MeetingWSEvent/ { count++ }
  END { print count + 0 }
' "$ROOT_DIR/backend/go-service/app/constants/meeting.go")"
meeting_all_events="$(rg -c '^\s*MeetingWSEvent[A-Za-z]+\s*=' "$ROOT_DIR/backend/go-service/app/constants/meeting.go")"
meeting_server_events="$(rg -c '^\s*MeetingWSEvent.*// (S→C|双向)' "$ROOT_DIR/backend/go-service/app/constants/meeting.go")"
check_count "meeting C-to-S events" "$meeting_client_events" 9
check_count "meeting S-to-C events" "$meeting_server_events" 8
check_count "meeting unique events" "$meeting_all_events" 16

media_lifecycle_routes="$(rg -c '\bapp\.(get|post|put|delete|patch)\(' "$ROOT_DIR"/media-server/src/routes/*.ts | awk -F: '{ total += $2 } END { print total + 0 }')"
media_info_routes="$(rg -c "app\.get\('/internal/info'" "$ROOT_DIR/media-server/src/app.ts")"
media_health_routes="$(rg -c "app\.get\('/(healthz|readyz)'" "$ROOT_DIR/media-server/src/app.ts")"
check_count "media lifecycle routes" "$media_lifecycle_routes" 10
check_count "media internal info routes" "$media_info_routes" 1
check_count "media public health routes" "$media_health_routes" 2

check_text docs/api/README.md '即时通讯（9 个 API）'
check_text docs/api/README.md '群聊管理（19 个 API）'
check_text docs/api/README.md '会议（12 REST + 16 WS 事件）'
check_text README.md '12 个 REST 接口 + 16 个 WebSocket 事件'
check_text media-server/README.md '当前实现：10 个 `/internal/v1` 生命周期接口'

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
  if ! rg -q "^[[:space:]]+$mapping:" "$ROOT_DIR/deploy/docker-compose.dev.yml"; then
    fail "Compose go-service environment is missing $mapping"
  fi
done

check_markdown_links

if (( FAILURES > 0 )); then
  printf '\nBaseline verification failed with %d issue(s).\n' "$FAILURES" >&2
  exit 1
fi

printf '\nBaseline verification passed.\n'
