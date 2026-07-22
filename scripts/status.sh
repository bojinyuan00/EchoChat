#!/usr/bin/env bash
# EchoChat 服务状态查看脚本
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RUN_DIR="$ROOT_DIR/.run"

COLOR_GREEN='\033[0;32m'
COLOR_RED='\033[0;31m'
COLOR_YELLOW='\033[0;33m'
COLOR_RESET='\033[0m'

check_app() {
  local name="$1"
  local port="$2"
  local url="$3"
  local pid_file="$4"
  local listener_pid tracked_pid
  listener_pid="$(lsof -iTCP:"$port" -sTCP:LISTEN -n -P -t 2>/dev/null | head -n1 || true)"
  tracked_pid="$(cat "$pid_file" 2>/dev/null || true)"

  if [[ -n "$listener_pid" ]]; then
    if curl --insecure --fail --silent --max-time 2 "$url" >/dev/null 2>&1; then
      printf "  %-22s ${COLOR_GREEN}● RUNNING${COLOR_RESET}   PID=%-8s HTTP=ready      %s\n" "$name (:$port)" "$listener_pid" "$url"
    else
      printf "  %-22s ${COLOR_YELLOW}! NOT-READY${COLOR_RESET} PID=%-8s HTTP=failed     %s\n" "$name (:$port)" "$listener_pid" "$url"
    fi
    return
  fi

  if [[ -n "$tracked_pid" ]]; then
    if kill -0 "$tracked_pid" 2>/dev/null; then
      printf "  %-22s ${COLOR_YELLOW}! NOT-READY${COLOR_RESET} PID=%-8s no-listener     %s\n" "$name (:$port)" "$tracked_pid" "$url"
    else
      printf "  %-22s ${COLOR_YELLOW}! STALE-PID${COLOR_RESET} PID=%-8s dead            %s\n" "$name (:$port)" "$tracked_pid" "$url"
    fi
    return
  fi

  printf "  %-22s ${COLOR_RED}○ STOPPED${COLOR_RESET}   %-12s                 %s\n" "$name (:$port)" "" "$url"
}

check_container() {
  local name="$1"
  local port="$2"
  local status
  status="$(docker inspect -f '{{.State.Status}}|{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$name" 2>/dev/null || echo 'missing|missing')"
  local state="${status%%|*}"
  local health="${status#*|}"
  case "$state" in
    running)
      case "$health" in
        healthy)
          printf "  %-22s ${COLOR_GREEN}● RUNNING${COLOR_RESET}   health=healthy       container=%s\n" "$name (:$port)" "$name"
          ;;
        starting)
          printf "  %-22s ${COLOR_YELLOW}! RUNNING${COLOR_RESET}   health=starting      container=%s\n" "$name (:$port)" "$name"
          ;;
        unhealthy)
          printf "  %-22s ${COLOR_RED}! UNHEALTHY${COLOR_RESET} health=unhealthy     container=%s\n" "$name (:$port)" "$name"
          ;;
        *)
          printf "  %-22s ${COLOR_GREEN}● RUNNING${COLOR_RESET}   health=no-healthcheck container=%s\n" "$name (:$port)" "$name"
          ;;
      esac
      ;;
    missing)
      printf "  %-22s ${COLOR_YELLOW}? NOT-CREATED${COLOR_RESET}\n" "$name (:$port)"
      ;;
    *)
      printf "  %-22s ${COLOR_RED}○ %s${COLOR_RESET} health=%s\n" "$name (:$port)" "$state" "$health"
      ;;
  esac
}

echo ""
echo "===== EchoChat 应用层 ====="
check_app "Go 后端"       8085 "http://127.0.0.1:8085/health" "$RUN_DIR/backend.pid"
check_app "媒体服务器 SFU" 3300 "http://127.0.0.1:3300/readyz" "$RUN_DIR/media.pid"
check_app "前台用户端"    5173 "https://127.0.0.1:5173" "$RUN_DIR/frontend.pid"
check_app "后台管理端"    3100 "http://127.0.0.1:3100" "$RUN_DIR/admin.pid"

echo ""
echo "===== Docker 中间件 ====="
check_container "echochat-postgres"     5432
check_container "echochat-redis"        6379
check_container "echochat-minio"        9000

echo ""
echo "===== 容器化应用（docker compose full） ====="
check_container "echochat-go-service"    8085
check_container "echochat-media-server"  3300
check_container "echochat-coturn"        3478

echo ""
echo "日志目录：$RUN_DIR/logs"
echo "compose 日志：docker compose -f deploy/docker-compose.dev.yml logs -f [service]"
echo ""
