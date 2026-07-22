#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
START_SCRIPT="$ROOT_DIR/scripts/start.sh"
STOP_SCRIPT="$ROOT_DIR/scripts/stop.sh"
FAILURES=0

pass() { printf '[PASS] %s\n' "$*"; }
fail() { printf '[FAIL] %s\n' "$*" >&2; FAILURES=$((FAILURES + 1)); }

case_body() {
  local script="$1"
  local pattern="$2"
  awk -v pattern="$pattern" '
    $0 ~ "^[[:space:]]*" pattern "\\)" { inside=1 }
    inside { print }
    inside && /;;[[:space:]]*$/ { exit }
  ' "$script"
}

assert_case_contains() {
  local script="$1" pattern="$2" expected="$3"
  if case_body "$script" "$pattern" | rg -q "$expected"; then
    pass "$pattern case contains $expected"
  else
    fail "$pattern case must contain $expected"
  fi
}

assert_case_not_contains() {
  local script="$1" pattern="$2" unexpected="$3"
  if case_body "$script" "$pattern" | rg -q "$unexpected"; then
    fail "$pattern case must not contain $unexpected"
  else
    pass "$pattern case does not contain $unexpected"
  fi
}

assert_case_contains "$STOP_SCRIPT" full stop_frontend
assert_case_contains "$STOP_SCRIPT" full stop_admin
assert_case_contains "$STOP_SCRIPT" full stop_full
assert_case_contains "$STOP_SCRIPT" '--all\|all' stop_full
assert_case_not_contains "$STOP_SCRIPT" '--all\|all' stop_docker

if rg -q '^START_FAILURES=0$' "$START_SCRIPT" && rg -q '^run_start_step\(\)' "$START_SCRIPT"; then
  pass 'start script aggregates failures'
else
  fail 'start script must aggregate failures with START_FAILURES and run_start_step'
fi

if rg -n 'start_(backend|media|frontend|admin)[[:space:]]+\|\|[[:space:]]+true' "$START_SCRIPT" >/dev/null; then
  fail 'start script must not swallow application failures with || true'
else
  pass 'start script does not swallow application failures'
fi

if (( FAILURES > 0 )); then
  printf '\nLifecycle contract failed with %d issue(s).\n' "$FAILURES" >&2
  exit 1
fi

printf '\nLifecycle contract passed.\n'
