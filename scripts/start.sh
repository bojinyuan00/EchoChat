#!/usr/bin/env bash
# EchoChat 一键启动脚本
# 用法：
#   ./scripts/start.sh           # 启动全部（Docker 中间件 + Go 后端 + 前台 + 管理端）
#   ./scripts/start.sh --no-docker  # 仅启动应用层（假设 Docker 容器已在运行）
#   ./scripts/start.sh backend      # 仅启动指定服务（backend|frontend|admin|media|docker）
# 注意：不使用 set -e，避免单个服务启动失败中断其他服务；
# 失败时由 spawn_bg 内部打印日志末尾辅助排障
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RUN_DIR="$ROOT_DIR/.run"
LOG_DIR="$RUN_DIR/logs"
mkdir -p "$RUN_DIR" "$LOG_DIR"

COLOR_GREEN='\033[0;32m'
COLOR_YELLOW='\033[0;33m'
COLOR_RED='\033[0;31m'
COLOR_CYAN='\033[0;36m'
COLOR_RESET='\033[0m'

log_info()  { printf "${COLOR_CYAN}[INFO]${COLOR_RESET}  %s\n" "$*"; }
log_ok()    { printf "${COLOR_GREEN}[OK]${COLOR_RESET}    %s\n" "$*"; }
log_warn()  { printf "${COLOR_YELLOW}[WARN]${COLOR_RESET}  %s\n" "$*"; }
log_err()   { printf "${COLOR_RED}[ERR]${COLOR_RESET}   %s\n" "$*"; }

# 判断端口是否被占用
port_in_use() {
  lsof -iTCP:"$1" -sTCP:LISTEN -n -P >/dev/null 2>&1
}

# 将命令以后台方式启动，并记录 PID 与日志
# 启动失败时自动打印日志末尾帮助排障，但不中断其他服务的启动
spawn_bg() {
  local name="$1"; shift
  local workdir="$1"; shift
  local log_file="$LOG_DIR/$name.log"
  local pid_file="$RUN_DIR/$name.pid"
  (
    cd "$workdir"
    nohup "$@" >"$log_file" 2>&1 &
    echo $! >"$pid_file"
  )
  sleep 1
  local pid
  pid="$(cat "$pid_file" 2>/dev/null || echo '')"
  if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
    log_ok "$name 已启动 (PID=$pid)  日志: $log_file"
    return 0
  fi
  log_err "$name 启动失败，日志末尾："
  if [[ -f "$log_file" ]]; then
    echo "------------------------------------------------------------"
    tail -n 20 "$log_file"
    echo "------------------------------------------------------------"
  fi
  log_err "完整日志: $log_file"
  return 1
}

start_docker() {
  log_info "启动 Docker 中间件（Postgres/Redis/MinIO）..."
  cd "$ROOT_DIR/deploy"
  docker compose -f docker-compose.dev.yml up -d postgres redis minio
  log_ok "Docker 中间件已启动"
}

start_backend() {
  if port_in_use 8085; then
    log_warn "端口 8085 已被占用，跳过 Go 后端启动"
    return 0
  fi
  log_info "启动 Go 后端 (port 8085)..."
  spawn_bg "backend" "$ROOT_DIR/backend/go-service" go run cmd/server/main.go
}

start_frontend() {
  if port_in_use 5173; then
    log_warn "端口 5173 已被占用，跳过前台启动"
    return 0
  fi
  log_info "启动前台用户端 (port 5173)..."
  spawn_bg "frontend" "$ROOT_DIR/frontend" npm run dev:h5
}

start_admin() {
  if port_in_use 3100; then
    log_warn "端口 3100 已被占用，跳过管理端启动"
    return 0
  fi
  log_info "启动后台管理端 (port 3100)..."
  spawn_bg "admin" "$ROOT_DIR/admin" npm run dev
}

# 启动 Node 媒体服务器（mediasoup SFU，默认端口 3300）
# 首次启动若缺少 .env，则从 .env.example 拷贝一份，避免 config 校验失败
start_media() {
  local media_dir="$ROOT_DIR/media-server"
  if [[ ! -d "$media_dir" ]]; then
    log_warn "media-server 目录不存在，跳过"
    return 0
  fi
  if port_in_use 3300; then
    log_warn "端口 3300 已被占用，跳过媒体服务器启动"
    return 0
  fi
  if [[ ! -f "$media_dir/.env" && -f "$media_dir/.env.example" ]]; then
    log_info "media-server/.env 不存在，从 .env.example 复制"
    cp "$media_dir/.env.example" "$media_dir/.env"
  fi
  if [[ ! -d "$media_dir/node_modules" ]]; then
    log_warn "media-server 依赖未安装，请先在 media-server 目录执行 npm install"
    return 1
  fi
  log_info "启动媒体服务器 (port 3300)..."
  spawn_bg "media" "$media_dir" npm run dev
}

# 等待 vite/dev server 日志里出现 Network 行并提取其中的 LAN 地址
# 最长等待 8 秒；失败时返回空字符串（不阻塞启动流程）
wait_and_collect_network_urls() {
  local log_file="$1"
  local max_wait="${2:-8}"
  local i=0
  while (( i < max_wait )); do
    if grep -q "Network:" "$log_file" 2>/dev/null; then
      break
    fi
    sleep 1
    i=$((i + 1))
  done
  grep -Eo "http://[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+:[0-9]+/?" "$log_file" 2>/dev/null \
    | grep -v '127\.0\.0\.1' \
    | sort -u
}

print_summary() {
  printf "\n${COLOR_GREEN}=============== EchoChat 启动完成 ===============${COLOR_RESET}\n"
  printf "  前台用户端 (H5):   http://localhost:5173\n"
  printf "  后台管理端:        http://localhost:3100\n"
  printf "  Go 后端 API:       http://localhost:8085\n"
  printf "  媒体服务器 (SFU):  http://localhost:3300 (healthz: /healthz)\n"
  printf "  MinIO 控制台:      http://localhost:9001 (echochat / echochat123456)\n"

  # 前端 / 管理端 Vite dev server 启动时 host=0.0.0.0，
  # 会输出多条 Network 行（本机每块网卡一条），这里从日志提取展示
  local fe_log="$LOG_DIR/frontend.log"
  local admin_log="$LOG_DIR/admin.log"
  local fe_urls admin_urls
  fe_urls="$(wait_and_collect_network_urls "$fe_log" 8)"
  admin_urls="$(wait_and_collect_network_urls "$admin_log" 4)"

  if [[ -n "$fe_urls" || -n "$admin_urls" ]]; then
    printf "\n${COLOR_CYAN}局域网访问地址${COLOR_RESET}（同 WiFi/LAN 下其他设备可直接访问）：\n"
    if [[ -n "$fe_urls" ]]; then
      printf "  前台用户端：\n"
      while IFS= read -r line; do
        [[ -z "$line" ]] && continue
        printf "    %s\n" "$line"
      done <<< "$fe_urls"
    fi
    if [[ -n "$admin_urls" ]]; then
      printf "  后台管理端：\n"
      while IFS= read -r line; do
        [[ -z "$line" ]] && continue
        printf "    %s\n" "$line"
      done <<< "$admin_urls"
    fi
    printf "  提示：接口代理走前端 Vite，后端 API(8085) / 媒体(3300) 默认绑定 0.0.0.0，\n"
    printf "        如需从其他设备访问 API/媒体，需要在前端 .env 里把 baseURL 改成本机 LAN IP。\n"
  fi

  printf "\n  日志目录:          %s\n" "$LOG_DIR"
  printf "  PID 目录:          %s\n" "$RUN_DIR"
  printf "  查看状态:          ./scripts/status.sh\n"
  printf "  停止全部:          ./scripts/stop.sh\n"
  printf "${COLOR_GREEN}=================================================${COLOR_RESET}\n"
}

main() {
  local target="${1:-all}"
  local skip_docker=false
  if [[ "$target" == "--no-docker" ]]; then
    skip_docker=true
    target="all"
  fi

  case "$target" in
    all)
      $skip_docker || start_docker || log_err "Docker 中间件启动异常，请检查 Docker Desktop 是否运行"
      start_backend  || true
      start_media    || true
      start_frontend || true
      start_admin    || true
      print_summary
      ;;
    docker)   start_docker ;;
    backend)  start_backend ;;
    frontend) start_frontend ;;
    admin)    start_admin ;;
    media)    start_media ;;
    *)
      log_err "未知参数: $target"
      echo "用法: $0 [all|docker|backend|frontend|admin|media|--no-docker]"
      exit 1
      ;;
  esac
}

main "$@"
