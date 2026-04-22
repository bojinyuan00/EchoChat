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

check_port() {
  local name="$1"
  local port="$2"
  local url="$3"
  local pid
  pid="$(lsof -iTCP:"$port" -sTCP:LISTEN -n -P -t 2>/dev/null | head -n1 || true)"
  if [[ -n "$pid" ]]; then
    printf "  %-22s ${COLOR_GREEN}● RUNNING${COLOR_RESET}  PID=%-8s  %s\n" "$name (:$port)" "$pid" "$url"
  else
    printf "  %-22s ${COLOR_RED}○ STOPPED${COLOR_RESET}  %-12s  %s\n" "$name (:$port)" "" "$url"
  fi
}

check_container() {
  local name="$1"
  local port="$2"
  local status
  status="$(docker inspect -f '{{.State.Status}}' "$name" 2>/dev/null || echo 'missing')"
  case "$status" in
    running)
      printf "  %-22s ${COLOR_GREEN}● RUNNING${COLOR_RESET}  container=%s\n" "$name (:$port)" "$name"
      ;;
    missing)
      printf "  %-22s ${COLOR_YELLOW}? NOT-CREATED${COLOR_RESET}\n" "$name (:$port)"
      ;;
    *)
      printf "  %-22s ${COLOR_RED}○ %s${COLOR_RESET}\n" "$name (:$port)" "$status"
      ;;
  esac
}

echo ""
echo "===== EchoChat 应用层 ====="
check_port "Go 后端"       8085 "http://localhost:8085"
check_port "媒体服务器 SFU" 3300 "http://localhost:3300/healthz"
check_port "前台用户端"    5173 "http://localhost:5173"
check_port "后台管理端"    3100 "http://localhost:3100"

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
