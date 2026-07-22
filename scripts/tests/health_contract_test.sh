#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
START_SCRIPT="$ROOT_DIR/scripts/start.sh"
STATUS_SCRIPT="$ROOT_DIR/scripts/status.sh"
FAILURES=0

pass() { printf '[PASS] %s\n' "$*"; }
fail() { printf '[FAIL] %s\n' "$*" >&2; FAILURES=$((FAILURES + 1)); }

function_body() {
  local name="$1"
  awk -v name="$name" '
    $0 ~ "^" name "\\(\\)" { inside=1 }
    inside { print }
    inside && /^}/ { exit }
  ' "$START_SCRIPT"
}

for expected in \
  'wait_for_http' \
  'http://127.0.0.1:8085/health' \
  'http://127.0.0.1:3300/readyz' \
  'https://127.0.0.1:5173' \
  'http://127.0.0.1:3100'; do
  if rg -Fq "$expected" "$START_SCRIPT"; then
    pass "start script contains $expected"
  else
    fail "start script must contain $expected"
  fi
done

for expected in 'http://127.0.0.1:8085/health' 'http://127.0.0.1:3300/readyz'; do
  if function_body start_full | rg -Fq "$expected"; then
    pass "full mode waits for $expected"
  else
    fail "full mode must wait for $expected"
  fi
done

for expected in '.State.Health.Status' 'STALE-PID' 'NOT-READY'; do
  if rg -Fq "$expected" "$STATUS_SCRIPT"; then
    pass "status script contains $expected"
  else
    fail "status script must contain $expected"
  fi
done

if rg -Fq 'ECHOCHAT_SCRIPT_SOURCE_ONLY' "$START_SCRIPT" && rg -q '^wait_for_http\(\)' "$START_SCRIPT"; then
  tmp_dir="$(mktemp -d)"
  server_pid=''
  cleanup() {
    [[ -n "$server_pid" ]] && kill "$server_pid" 2>/dev/null || true
    rm -rf "$tmp_dir"
  }
  trap cleanup EXIT

  /usr/bin/python3 - "$tmp_dir/port" >"$tmp_dir/server.log" 2>&1 <<'PY' &
import http.server
import sys

server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), http.server.SimpleHTTPRequestHandler)
with open(sys.argv[1], "w", encoding="utf-8") as handle:
    handle.write(str(server.server_address[1]))
server.serve_forever()
PY
  server_pid=$!

  # Homebrew Python 首次导入 http.server 可能超过 1 秒；按条件等待，最多 10 秒。
  for _ in $(seq 1 50); do
    [[ -s "$tmp_dir/port" ]] && break
    sleep 0.2
  done
  if [[ ! -s "$tmp_dir/port" ]]; then
    fail 'temporary HTTP server did not publish a port within 10 seconds'
  else
    port="$(cat "$tmp_dir/port")"

    ECHOCHAT_SCRIPT_SOURCE_ONLY=1 source "$START_SCRIPT"
    if wait_for_http "test server" "http://127.0.0.1:$port" "$server_pid" 3; then
      pass 'wait_for_http accepts a ready endpoint'
    else
      fail 'wait_for_http must accept a ready endpoint'
    fi

    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
    server_pid=''
    if wait_for_http "closed server" "http://127.0.0.1:$port" '' 2; then
      fail 'wait_for_http must reject an unreachable endpoint'
    else
      pass 'wait_for_http rejects an unreachable endpoint'
    fi
  fi
else
  fail 'start script must support source-only helper testing'
fi

if (( FAILURES > 0 )); then
  printf '\nHealth contract failed with %d issue(s).\n' "$FAILURES" >&2
  exit 1
fi

printf '\nHealth contract passed.\n'
