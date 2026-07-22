#!/usr/bin/env bash
# EchoChat 一键停止脚本
# 用法：
#   ./scripts/stop.sh           # 停止应用层（Go 后端 + 前台 + 管理端 + 媒体），保留 Docker 容器
#   ./scripts/stop.sh --all     # 停止全部本地进程 + compose 全栈（数据卷保留）
#   ./scripts/stop.sh full      # 停止 full 模式：本地前端/管理端 + compose 全栈
#   ./scripts/stop.sh backend   # 仅停止指定服务（backend|frontend|admin|media|docker）
# 不使用 set -e：即使某一项停止失败，其他项也要继续尝试
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RUN_DIR="$ROOT_DIR/.run"
DEPLOY_DIR="$ROOT_DIR/deploy"

COLOR_GREEN='\033[0;32m'
COLOR_YELLOW='\033[0;33m'
COLOR_RED='\033[0;31m'
COLOR_CYAN='\033[0;36m'
COLOR_RESET='\033[0m'

log_info()  { printf "${COLOR_CYAN}[INFO]${COLOR_RESET}  %s\n" "$*"; }
log_ok()    { printf "${COLOR_GREEN}[OK]${COLOR_RESET}    %s\n" "$*"; }
log_warn()  { printf "${COLOR_YELLOW}[WARN]${COLOR_RESET}  %s\n" "$*"; }
log_err()   { printf "${COLOR_RED}[ERR]${COLOR_RESET}   %s\n" "$*"; }

STOP_FAILURES=0

run_stop_step() {
  local label="$1"
  shift
  if "$@"; then
    return 0
  fi
  log_err "$label 停止失败"
  STOP_FAILURES=$((STOP_FAILURES + 1))
  return 0
}

finish_stop() {
  local success_message="$1"
  if (( STOP_FAILURES > 0 )); then
    log_err "停止流程结束，但有 $STOP_FAILURES 个步骤失败"
    return 1
  fi
  log_ok "$success_message"
}

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
  if ! docker compose -f docker-compose.dev.yml stop postgres redis minio; then
    return 1
  fi
  log_ok "Docker 中间件已停止（数据卷保留）"
}

# Phase 2e-2 Task 14：停止 docker compose 全栈（含 media-server、coturn）
# 使用 --profile public 确保即使 coturn 在 public profile 下运行也能被停掉
stop_full() {
  log_info "停止 docker compose 全栈（含 media-server / coturn）..."
  cd "$DEPLOY_DIR"
  if ! docker compose -f docker-compose.dev.yml --profile public stop; then
    return 1
  fi
  log_ok "全栈服务已停止（数据卷保留）"
}

main() {
  local target="${1:-app}"

  case "$target" in
    app|"")
      run_stop_step "Go 后端" stop_backend
      run_stop_step "媒体服务器" stop_media
      run_stop_step "前台用户端" stop_frontend
      run_stop_step "后台管理端" stop_admin
      finish_stop "应用层服务已停止（Docker 中间件保持运行）"
      ;;
    --all|all)
      run_stop_step "Go 后端" stop_backend
      run_stop_step "媒体服务器" stop_media
      run_stop_step "前台用户端" stop_frontend
      run_stop_step "后台管理端" stop_admin
      run_stop_step "Docker Compose 全栈" stop_full
      finish_stop "所有本地进程与 Compose 服务已停止（数据卷保留）"
      ;;
    backend)  stop_backend ;;
    frontend) stop_frontend ;;
    admin)    stop_admin ;;
    media)    stop_media ;;
    docker)   stop_docker ;;
    full)
      run_stop_step "前台用户端" stop_frontend
      run_stop_step "后台管理端" stop_admin
      run_stop_step "Docker Compose 全栈" stop_full
      finish_stop "full 模式服务已停止（数据卷保留）"
      ;;
    *)
      echo "用法: $0 [app|--all|backend|frontend|admin|media|docker|full]"
      exit 1
      ;;
  esac
}

main "$@"
