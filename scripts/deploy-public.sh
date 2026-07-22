#!/usr/bin/env bash
# ============================================================
# EchoChat 公网部署脚本（Phase 2e-2 Task 14）
# ------------------------------------------------------------
# 职责：
#   1. 校验 deploy/.env 内关键参数（ANNOUNCED_IP / 密码替换等）
#   2. 给出防火墙/安全组端口放通 checklist（本地用 nc 探测）
#   3. 启动 docker compose --profile public（含 media-server + coturn）
#
# 不做的事：
#   - 不替你改 iptables / ufw（不同云/OS 命令不一，避免误操作）
#   - 不替你改 hosts / DNS（域名解析请预先完成）
#
# 用法：
#   cp deploy/.env.public.example deploy/.env
#   vim deploy/.env    # 按 checklist 替换所有 _REPLACE_WITH_*_
#   ./scripts/deploy-public.sh
# ============================================================
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DEPLOY_DIR="$ROOT_DIR/deploy"
ENV_FILE="$DEPLOY_DIR/.env"
COMPOSE_FILE="$DEPLOY_DIR/docker-compose.dev.yml"

COLOR_GREEN='\033[0;32m'
COLOR_YELLOW='\033[0;33m'
COLOR_RED='\033[0;31m'
COLOR_CYAN='\033[0;36m'
COLOR_RESET='\033[0m'

log_info()  { printf "${COLOR_CYAN}[INFO]${COLOR_RESET}  %s\n" "$*"; }
log_ok()    { printf "${COLOR_GREEN}[OK]${COLOR_RESET}    %s\n" "$*"; }
log_warn()  { printf "${COLOR_YELLOW}[WARN]${COLOR_RESET}  %s\n" "$*"; }
log_err()   { printf "${COLOR_RED}[ERR]${COLOR_RESET}   %s\n" "$*"; }

# 计数器
ERR_COUNT=0
WARN_COUNT=0
err()   { log_err "$*";  ERR_COUNT=$((ERR_COUNT + 1)); }
warn()  { log_warn "$*"; WARN_COUNT=$((WARN_COUNT + 1)); }

# ------------------------------------------------------------
# Step 1：env 文件校验
# ------------------------------------------------------------
step_validate_env() {
  log_info "=== Step 1/5: 校验 deploy/.env ==="

  if [[ ! -f "$ENV_FILE" ]]; then
    err "deploy/.env 不存在"
    log_info "请先执行：cp deploy/.env.public.example deploy/.env 并填写关键参数"
    return 1
  fi

  # shellcheck disable=SC1090
  set -a
  source "$ENV_FILE"
  set +a

  # DEPLOY_MODE 必须是 public（防止误把 local 的 env 复制过来）
  if [[ "${DEPLOY_MODE:-}" != "public" ]]; then
    err "deploy/.env 的 DEPLOY_MODE 必须为 public"
  fi

  if [[ "${GO_SERVER_MODE:-}" != "release" ]]; then
    err "GO_SERVER_MODE 必须为 release"
  fi

  if [[ -z "${WS_ALLOWED_ORIGINS:-}" ]]; then
    err "WS_ALLOWED_ORIGINS 不能为空"
  elif [[ "$WS_ALLOWED_ORIGINS" == _REPLACE_WITH_* ]]; then
    err "WS_ALLOWED_ORIGINS 仍是占位符，请填写实际 HTTPS Origin"
  else
    local origin
    IFS=',' read -r -a origins <<<"$WS_ALLOWED_ORIGINS"
    for origin in "${origins[@]}"; do
      if [[ "$origin" != https://* ]]; then
        err "WS_ALLOWED_ORIGINS 中每一项都必须以 https:// 开头"
        break
      fi
    done
  fi

  # MEDIASOUP_ANNOUNCED_IP 必填
  if [[ -z "${MEDIASOUP_ANNOUNCED_IP:-}" ]]; then
    err "MEDIASOUP_ANNOUNCED_IP 未设置，远端浏览器将无法建连 mediasoup"
    log_info "请填写服务器的公网 IP，例：MEDIASOUP_ANNOUNCED_IP=203.0.113.42"
  else
    log_ok "MEDIASOUP_ANNOUNCED_IP=${MEDIASOUP_ANNOUNCED_IP}"
  fi

  # 检查占位符没被替换
  local placeholders=(
    POSTGRES_PASSWORD
    REDIS_PASSWORD
    MINIO_ROOT_PASSWORD
    JWT_SECRET
    MEDIA_INTERNAL_TOKEN
    TURN_USERNAME
    TURN_PASSWORD
  )
  for var in "${placeholders[@]}"; do
    local val="${!var:-}"
    if [[ -z "$val" ]]; then
      err "$var 为空"
    elif [[ "$val" == _REPLACE_WITH_* ]]; then
      err "$var 仍是占位符，请替换为真实值"
    fi
  done

  # JWT_SECRET 长度
  if [[ -n "${JWT_SECRET:-}" && ${#JWT_SECRET} -lt 32 ]]; then
    warn "JWT_SECRET 长度仅 ${#JWT_SECRET} 字符，推荐 >= 64 字符（openssl rand -hex 32）"
  fi

  log_ok "Redis 密码将由 Compose 同时注入 redis-server、健康检查和 Go 客户端"

  # TURN 开关一致性
  if [[ "${TURN_ENABLED:-false}" != "true" ]]; then
    warn "TURN_ENABLED=${TURN_ENABLED:-false}，公网部署下建议开启 TURN 作为对称 NAT fallback"
  fi

  if (( ERR_COUNT > 0 )); then
    log_err "env 校验失败：$ERR_COUNT 个错误，$WARN_COUNT 个警告"
    return 1
  fi
  log_ok "env 校验通过（$WARN_COUNT 个警告）"
}

# ------------------------------------------------------------
# Step 2：防火墙/端口占用检查（本机视角）
# ------------------------------------------------------------
# 说明：云安全组需要在云控制台放通，本脚本只能检测本机 iptables + 当前占用
step_check_ports() {
  log_info "=== Step 2/5: 本机端口占用检查 ==="

  local ports=(
    "8085/tcp|Go backend API"
    "3300/tcp|media-server (internal, 不建议暴露公网)"
    "40000/udp|mediasoup RTC (端口范围起点)"
    "40199/udp|mediasoup RTC (端口范围终点)"
  )

  if [[ "${TURN_ENABLED:-false}" == "true" ]]; then
    ports+=(
      "${TURN_LISTEN_PORT:-3478}/udp|coturn STUN/TURN"
      "${TURN_LISTEN_PORT:-3478}/tcp|coturn STUN/TURN"
      "${TURN_TLS_PORT:-5349}/tcp|coturn TLS"
    )
  fi

  for entry in "${ports[@]}"; do
    local port="${entry%%|*}"
    local desc="${entry##*|}"
    local port_num="${port%/*}"
    local proto="${port##*/}"
    if [[ "$proto" == "tcp" ]]; then
      if lsof -iTCP:"$port_num" -sTCP:LISTEN -n -P >/dev/null 2>&1; then
        warn "TCP:$port_num 已被占用（${desc}），compose 启动可能失败"
      else
        log_ok "TCP:$port_num 空闲（${desc}）"
      fi
    else
      # UDP 只做提示，不中断
      log_ok "UDP:$port_num 待 docker compose 绑定（${desc}）"
    fi
  done

  printf "\n${COLOR_YELLOW}⚠️  云安全组/iptables 放通清单${COLOR_RESET}（本脚本无法自动执行，请手动确认）：\n"
  printf "    TCP:8085                   Go backend API\n"
  printf "    UDP:40000-40199            mediasoup RTC 流量\n"
  printf "    TCP:40000-40199            mediasoup ICE-TCP fallback\n"
  if [[ "${TURN_ENABLED:-false}" == "true" ]]; then
    printf "    UDP:${TURN_LISTEN_PORT:-3478}                    coturn STUN/TURN\n"
    printf "    TCP:${TURN_LISTEN_PORT:-3478}                    coturn STUN/TURN\n"
    printf "    TCP:${TURN_TLS_PORT:-5349}                    coturn TLS\n"
    printf "    UDP:${TURN_MIN_PORT:-49160}-${TURN_MAX_PORT:-49200}            coturn 中继流量\n"
  fi
  echo ""
}

# ------------------------------------------------------------
# Step 3：Docker 环境自检
# ------------------------------------------------------------
step_check_docker() {
  log_info "=== Step 3/5: 现有 Docker 环境自检 ==="

  if ! command -v docker >/dev/null 2>&1; then
    err "未检测到 docker 命令"
    return 1
  fi
  log_ok "docker: $(docker --version)"

  if ! docker info >/dev/null 2>&1; then
    err "docker daemon 无法连接（sudo systemctl start docker?）"
    return 1
  fi
  log_ok "docker daemon 可访问"

  if ! docker compose version >/dev/null 2>&1; then
    err "未检测到 docker compose（需 Compose V2 plugin）"
    return 1
  fi
  log_ok "docker compose: $(docker compose version | head -n1)"
}

# ------------------------------------------------------------
# Step 4：渲染后的 Compose 配置预检（只读，不创建容器）
# ------------------------------------------------------------
step_validate_compose() {
  log_info "=== Step 4/5: 渲染 Compose 配置 ==="
  if ! docker compose --env-file "$ENV_FILE" --profile public -f "$COMPOSE_FILE" config --quiet; then
    err "Compose 配置渲染失败，拒绝启动任何容器"
    return 1
  fi
  log_ok "Compose 配置渲染通过"
}

# ------------------------------------------------------------
# Step 4：启动 public profile
# ------------------------------------------------------------
step_launch() {
  log_info "=== Step 5/5: 启动 docker compose (public profile) ==="

  if (( ERR_COUNT > 0 )); then
    log_err "前面校验有错误，拒绝启动（先修正后再跑脚本）"
    return 1
  fi

  cd "$DEPLOY_DIR"
  if [[ "${TURN_ENABLED:-false}" == "true" ]]; then
    log_info "TURN_ENABLED=true，使用 --profile public 启动（含 coturn）"
    docker compose --env-file "$ENV_FILE" --profile public -f "$COMPOSE_FILE" up -d --build
  else
    log_info "TURN_ENABLED=false，跳过 coturn（仅默认服务）"
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d --build
  fi

  log_ok "启动指令已发出，等待 healthcheck（最多 180 秒）..."
  local waited=0
  while (( waited < 180 )); do
    local health
    health="$(docker inspect -f '{{.State.Health.Status}}' echochat-media-server 2>/dev/null || echo 'starting')"
    if [[ "$health" == "healthy" ]]; then
      log_ok "media-server healthy（$waited 秒）"
      break
    fi
    if [[ "$health" == "unhealthy" ]]; then
      log_err "media-server unhealthy，查看日志：docker logs echochat-media-server"
      return 1
    fi
    sleep 3
    waited=$((waited + 3))
  done
  if (( waited >= 180 )); then
    log_err "等待 media-server healthy 超过 180 秒"
    return 1
  fi

  printf "\n${COLOR_GREEN}=============== 公网部署完成 ===============${COLOR_RESET}\n"
  printf "  Go 后端 API:    http://${MEDIASOUP_ANNOUNCED_IP}:8085\n"
  printf "  media-server:   http://127.0.0.1:3300 (仅内网，勿对外暴露)\n"
  if [[ "${TURN_ENABLED:-false}" == "true" ]]; then
    printf "  coturn TURN:    ${MEDIASOUP_ANNOUNCED_IP}:${TURN_LISTEN_PORT:-3478}\n"
  fi
  printf "\n  查看状态:       docker compose -f deploy/docker-compose.dev.yml ps\n"
  printf "  查看日志:       docker compose -f deploy/docker-compose.dev.yml logs -f [service]\n"
  printf "  停止全栈:       ./scripts/stop.sh full\n"
  printf "${COLOR_GREEN}============================================${COLOR_RESET}\n"
}

main() {
  step_validate_env || exit 1
  step_check_ports   # Step 2，不会 return 1，只收集 warn
  step_check_docker || exit 1   # Step 3
  step_validate_compose || exit 1 # Step 4，只读预检
  # 二次确认（至少暴露 1 个明显错误时已退出，这里只对警告放行）
  if [[ "${1:-}" != "--yes" ]] && (( WARN_COUNT > 0 )); then
    printf "\n${COLOR_YELLOW}上述 %d 个警告仅提示，不阻塞启动。继续部署？(y/N): ${COLOR_RESET}" "$WARN_COUNT"
    read -r confirm
    case "$confirm" in
      y|Y|yes|YES) ;;
      *) log_info "已取消部署"; exit 0 ;;
    esac
  fi
  step_launch
}

main "$@"
