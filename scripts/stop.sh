#!/usr/bin/env bash
# EchoChat 一键停止脚本
# 用法：
#   ./scripts/stop.sh           # 停止应用层（Go 后端 + 前台 + 管理端），保留 Docker 容器
#   ./scripts/stop.sh --all     # 停止全部（含 Docker 中间件）
#   ./scripts/stop.sh backend   # 仅停止指定服务（backend|frontend|admin|media|docker）
# 不使用 set -e：即使某一项停止失败，其他项也要继续尝试
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RUN_DIR="$ROOT_DIR/.run"

COLOR_GREEN='\033[0;32m'
COLOR_YELLOW='\033[0;33m'
COLOR_RED='\033[0;31m'
COLOR_CYAN='\033[0;36m'
COLOR_RESET='\033[0m'

log_info()  { printf "${COLOR_CYAN}[INFO]${COLOR_RESET}  %s\n" "$*"; }
log_ok()    { printf "${COLOR_GREEN}[OK]${COLOR_RESET}    %s\n" "$*"; }
log_warn()  { printf "${COLOR_YELLOW}[WARN]${COLOR_RESET}  %s\n" "$*"; }

# 根据端口查找监听进程 PID
pids_by_port() {
  lsof -iTCP:"$1" -sTCP:LISTEN -n -P -t 2>/dev/null || true
}

# 终止 PID（先 TERM，2s 后仍存在则 KILL），同时杀掉其子进程组
kill_pid() {
  local pid="$1"
  [[ -z "$pid" ]] && return 0
  kill -0 "$pid" 2>/dev/null || return 0
  # 尝试杀死进程组（负 PID 表示进程组）
  kill -TERM -"$pid" 2>/dev/null || kill -TERM "$pid" 2>/dev/null || true
  for _ in 1 2 3 4; do
    kill -0 "$pid" 2>/dev/null || return 0
    sleep 0.5
  done
  kill -KILL -"$pid" 2>/dev/null || kill -KILL "$pid" 2>/dev/null || true
}

# 停止由 start.sh 启动的命名服务（以 .pid 文件 + 端口兜底）
stop_named() {
  local name="$1"
  local port="$2"
  local pid_file="$RUN_DIR/$name.pid"
  local killed=0

  if [[ -f "$pid_file" ]]; then
    local pid
    pid="$(cat "$pid_file")"
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
      log_info "停止 $name (PID=$pid, port=$port)..."
      kill_pid "$pid"
      killed=1
    fi
    rm -f "$pid_file"
  fi

  # 端口兜底：清理任何仍占用目标端口的进程
  local port_pids
  port_pids="$(pids_by_port "$port")"
  if [[ -n "$port_pids" ]]; then
    for p in $port_pids; do
      log_info "清理 $name 残留进程 (PID=$p, port=$port)..."
      kill_pid "$p"
      killed=1
    done
  fi

  if [[ $killed -eq 1 ]]; then
    log_ok "$name 已停止"
  else
    log_warn "$name 未在运行"
  fi
}

stop_backend()  { stop_named "backend"  8085; }
stop_frontend() { stop_named "frontend" 5173; }
stop_admin()    { stop_named "admin"    3100; }
stop_media()    { stop_named "media"    3300; }

stop_docker() {
  log_info "停止 Docker 中间件..."
  cd "$ROOT_DIR/deploy"
  docker compose -f docker-compose.dev.yml stop postgres redis minio || true
  log_ok "Docker 中间件已停止（数据卷保留）"
}

main() {
  local target="${1:-app}"

  case "$target" in
    app|"")
      stop_backend
      stop_media
      stop_frontend
      stop_admin
      log_ok "应用层服务已停止（Docker 中间件保持运行）"
      ;;
    --all|all)
      stop_backend
      stop_media
      stop_frontend
      stop_admin
      stop_docker
      log_ok "所有服务已停止"
      ;;
    backend)  stop_backend ;;
    frontend) stop_frontend ;;
    admin)    stop_admin ;;
    media)    stop_media ;;
    docker)   stop_docker ;;
    *)
      echo "用法: $0 [app|--all|backend|frontend|admin|media|docker]"
      exit 1
      ;;
  esac
}

main "$@"
