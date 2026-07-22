#!/usr/bin/env bash
# EchoChat 一键启动脚本
# 用法：
#   ./scripts/start.sh              # 启动应用全部（Docker 中间件 + Go/前端/管理端，本地进程跑）
#   ./scripts/start.sh --no-docker  # 仅启动应用层（假设 Docker 容器已在运行）
#   ./scripts/start.sh backend      # 仅启动指定服务（backend|frontend|admin|media|docker）
#   ./scripts/start.sh full         # Phase 2e-2 Task 14：docker compose 全栈启动（含 media-server 容器）
#                                   # 会读 deploy/.env（如不存在提示 copy example），公网则用 deploy-public.sh
# 注意：不使用 set -e，确保一个服务失败后仍能尝试启动其余服务；
# main 会累计失败数并以非 0 退出，避免输出虚假的“启动完成”。
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RUN_DIR="$ROOT_DIR/.run"
LOG_DIR="$RUN_DIR/logs"
DEPLOY_DIR="$ROOT_DIR/deploy"
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

START_FAILURES=0

run_start_step() {
  local label="$1"
  shift
  if "$@"; then
    return 0
  fi
  log_err "$label 启动失败"
  START_FAILURES=$((START_FAILURES + 1))
  return 0
}

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
  if ! docker compose -f docker-compose.dev.yml up -d postgres redis minio; then
    log_err "Docker 中间件启动命令执行失败"
    return 1
  fi
  log_ok "Docker 中间件已启动"
}

# 确保 deploy/.env 存在：若缺失则从 .env.local.example 拷贝（等价于本机 Demo 默认值）
# 公网部署走 scripts/deploy-public.sh 单独校验，不经此函数
ensure_env_file() {
  local env_file="$DEPLOY_DIR/.env"
  if [[ -f "$env_file" ]]; then
    return 0
  fi
  local tmpl="$DEPLOY_DIR/.env.local.example"
  if [[ -f "$tmpl" ]]; then
    cp "$tmpl" "$env_file"
    log_warn "deploy/.env 不存在，已从 .env.local.example 复制（本机 Demo 默认值）"
  else
    log_warn "deploy/.env 和 .env.local.example 都不存在，compose 将使用内置默认值"
  fi
}

# Phase 2e-2 Task 14：docker compose 全栈启动
# 默认只拉起非 public profile 的服务（5 个：postgres/redis/minio/go-service/media-server）
# coturn 走 public profile，需要用 scripts/deploy-public.sh
start_full() {
  ensure_env_file
  log_info "docker compose 全栈启动（postgres + redis + minio + go-service + media-server）..."
  cd "$DEPLOY_DIR"
  if ! docker compose -f docker-compose.dev.yml up -d --build; then
    log_err "docker compose 启动失败，请检查上方错误输出"
    return 1
  fi
  log_ok "全栈服务已发起启动，等待 healthcheck..."
  # 等待最多 180 秒让 media-server 进入 healthy
  local waited=0
  while (( waited < 180 )); do
    local health
    health="$(docker inspect -f '{{.State.Health.Status}}' echochat-media-server 2>/dev/null || echo 'starting')"
    case "$health" in
      healthy)
        log_ok "media-server 已 healthy（$waited 秒）"
        break
        ;;
      unhealthy)
        log_err "media-server unhealthy，请查看日志：docker logs echochat-media-server"
        return 1
        ;;
    esac
    sleep 3
    waited=$((waited + 3))
  done
  if (( waited >= 180 )); then
    log_warn "等待 media-server healthy 超过 180 秒，可能仍在 mediasoup worker 初始化中，请手动验证"
  fi
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
# 同时匹配 http:// 和 https://（frontend Vite 启用了 basicSsl + https，admin 是 http）
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
  grep -Eo "https?://[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+:[0-9]+/?" "$log_file" 2>/dev/null \
    | grep -v '127\.0\.0\.1' \
    | sort -u
}

# 探测本机活跃 LAN IPv4，规则与 media-server/src/utils/network.ts 中的 detectLanIpv4() 完全一致：
#   1. 排除回环、link-local、各类虚拟网卡（utun/bridge/vmnet/docker/br-/veth/awdl/anpi 等）
#   2. 优先级：192.168.x → 10.x → 172.16-31.x
#   3. 取第一个候选；探不到时返回空字符串
detect_lan_ip() {
  ifconfig 2>/dev/null | awk '
    /^[a-zA-Z0-9]+:/ {
      iface = $1
      sub(":", "", iface)
      lower = tolower(iface)
      virt = 0
      if (lower ~ /^(lo|docker|br-|veth|utun|awdl|llw|gif|stf|anpi|ap[0-9]|bridge|vmnet|vnic|vboxnet|tun|tap)/) virt = 1
    }
    /[ \t]inet [0-9]/ {
      if (virt) next
      ip = $2
      if (ip == "127.0.0.1") next
      if (ip ~ /^169\.254\./) next
      if (ip ~ /^192\.168\./) { print "0\t" ip; next }
      if (ip ~ /^10\./)        { print "1\t" ip; next }
      if (ip ~ /^172\.(1[6-9]|2[0-9]|3[0-1])\./) { print "2\t" ip; next }
      print "3\t" ip
    }
  ' | sort -k1,1n | head -n1 | awk '{print $2}'
}

# 提取 media-server/.env 中 MEDIASOUP_ANNOUNCED_IP 的当前生效值（仅用于摘要展示一致性）
read_announced_ip() {
  local env_file="$ROOT_DIR/media-server/.env"
  [[ -f "$env_file" ]] || { echo ""; return; }
  grep -E '^MEDIASOUP_ANNOUNCED_IP=' "$env_file" | head -n1 | cut -d= -f2- | tr -d '"' | tr -d "'" | xargs
}

print_summary() {
  local lan_ip
  lan_ip="$(detect_lan_ip)"
  local announced_ip
  announced_ip="$(read_announced_ip)"

  printf "\n${COLOR_GREEN}=============== EchoChat 启动完成 ===============${COLOR_RESET}\n"
  printf "${COLOR_CYAN}本机访问地址${COLOR_RESET}（仅当前电脑可访问）：\n"
  printf "  前台用户端 (H5):   https://localhost:5173  ${COLOR_YELLOW}(自签证书，浏览器需点击\"高级 → 继续访问\")${COLOR_RESET}\n"
  printf "  后台管理端:        http://localhost:3100\n"
  printf "  Go 后端 API:       http://localhost:8085\n"
  printf "  媒体服务器 (SFU):  http://localhost:3300  (healthz: /healthz)\n"
  printf "  MinIO 控制台:      http://localhost:9001  (echochat / echochat123456)\n"

  # 前端 / 管理端 Vite dev server 启动时 host=0.0.0.0，
  # 会输出多条 Network 行（本机每块网卡一条），这里从日志提取展示
  local fe_log="$LOG_DIR/frontend.log"
  local admin_log="$LOG_DIR/admin.log"
  local fe_urls admin_urls
  fe_urls="$(wait_and_collect_network_urls "$fe_log" 8)"
  admin_urls="$(wait_and_collect_network_urls "$admin_log" 4)"

  if [[ -n "$lan_ip" || -n "$fe_urls" || -n "$admin_urls" ]]; then
    printf "\n${COLOR_CYAN}局域网访问地址${COLOR_RESET}（同 WiFi/LAN 下手机/平板/其他电脑可直接访问）：\n"

    # 优先用 vite 实际打印的 URL（Network 行），更可靠；vite 没启动好则 fallback 到 detect_lan_ip
    if [[ -n "$fe_urls" ]]; then
      printf "  前台用户端：\n"
      while IFS= read -r line; do
        [[ -z "$line" ]] && continue
        printf "    %s\n" "$line"
      done <<< "$fe_urls"
    elif [[ -n "$lan_ip" ]]; then
      printf "  前台用户端：    https://%s:5173  ${COLOR_YELLOW}(HTTPS + 自签证书，移动端首次访问需点信任)${COLOR_RESET}\n" "$lan_ip"
    fi

    if [[ -n "$admin_urls" ]]; then
      printf "  后台管理端：\n"
      while IFS= read -r line; do
        [[ -z "$line" ]] && continue
        printf "    %s\n" "$line"
      done <<< "$admin_urls"
    elif [[ -n "$lan_ip" ]]; then
      printf "  后台管理端：    http://%s:3100\n" "$lan_ip"
    fi

    if [[ -n "$lan_ip" ]]; then
      printf "  Go 后端 API：   http://%s:8085  ${COLOR_YELLOW}(前端会自动通过 Vite 反代访问，无需直连)${COLOR_RESET}\n" "$lan_ip"
      printf "  媒体服务器：    http://%s:3300  ${COLOR_YELLOW}(供后端调用，移动端不直接访问)${COLOR_RESET}\n" "$lan_ip"
    fi

    printf "  ${COLOR_CYAN}提示${COLOR_RESET}：移动端浏览器只需打开\"前台用户端\"地址即可，API/WebSocket/媒体均通过 Vite 反代自动转发。\n"
    printf "        WebRTC 音视频会议依赖 mediasoup 通告 IP，必须与下方一致；不一致 dev 模式下会自动覆盖。\n"
  fi

  # mediasoup ANNOUNCED_IP 与当前 LAN IP 一致性检查（避免再发生"IP 漂移导致音视频不通"）
  if [[ -n "$lan_ip" ]]; then
    printf "\n${COLOR_CYAN}WebRTC 音视频通告 IP${COLOR_RESET}：\n"
    printf "  当前 LAN IP:                 %s\n" "$lan_ip"
    if [[ -z "$announced_ip" ]]; then
      printf "  media-server ANNOUNCED_IP:   ${COLOR_GREEN}(空，启动时自动探测)${COLOR_RESET}\n"
    elif [[ "$announced_ip" == "$lan_ip" ]]; then
      printf "  media-server ANNOUNCED_IP:   ${COLOR_GREEN}%s ✓ 与当前 LAN 一致${COLOR_RESET}\n" "$announced_ip"
    else
      printf "  media-server ANNOUNCED_IP:   ${COLOR_YELLOW}%s ⚠ 与当前 LAN 不一致${COLOR_RESET}\n" "$announced_ip"
      printf "    → dev 模式下 media-server 启动时会自动覆盖为 %s（看 logs/media.log 中 AUTO-OVERRIDING 行）\n" "$lan_ip"
      printf "    → 永久消除该提示：将 media-server/.env 的 MEDIASOUP_ANNOUNCED_IP 留空 或 改为 %s\n" "$lan_ip"
    fi
  fi

  printf "\n  日志目录:          %s\n" "$LOG_DIR"
  printf "  PID 目录:          %s\n" "$RUN_DIR"
  printf "  查看状态:          ./scripts/status.sh\n"
  printf "  停止全部:          ./scripts/stop.sh\n"
  printf "${COLOR_GREEN}=================================================${COLOR_RESET}\n"
}

finish_start() {
  if (( START_FAILURES > 0 )); then
    log_err "启动流程结束，但有 $START_FAILURES 个服务失败；请查看上方错误与 .run/logs"
    return 1
  fi
  print_summary
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
      if ! $skip_docker; then
        run_start_step "Docker 中间件" start_docker
      fi
      run_start_step "Go 后端" start_backend
      run_start_step "媒体服务器" start_media
      run_start_step "前台用户端" start_frontend
      run_start_step "后台管理端" start_admin
      finish_start
      ;;
    docker)   start_docker ;;
    backend)  start_backend ;;
    frontend) start_frontend ;;
    admin)    start_admin ;;
    media)    start_media ;;
    full)
      run_start_step "Docker Compose 全栈" start_full
      # full 模式下仍用本地进程跑前台 + 管理端（热更体验更好，符合开发机场景）
      run_start_step "前台用户端" start_frontend
      run_start_step "后台管理端" start_admin
      finish_start
      ;;
    *)
      log_err "未知参数: $target"
      echo "用法: $0 [all|docker|backend|frontend|admin|media|full|--no-docker]"
      echo ""
      echo "模式说明："
      echo "  all       本地进程启动应用 + docker 只起中间件（默认，开发常用）"
      echo "  full      docker compose 全栈启动（含 media-server 容器，接近生产形态）"
      echo "  公网部署  使用 scripts/deploy-public.sh（校验 ANNOUNCED_IP + 启动 coturn）"
      exit 1
      ;;
  esac
}

main "$@"
