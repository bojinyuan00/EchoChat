#!/usr/bin/env bash
# ==============================================================================
# EchoChat 开发环境【首次】初始化脚本
# ------------------------------------------------------------------------------
# ✅ 适用时机：
#   - 开发者首次 clone 仓库后
#   - 删除过所有 Docker 卷 / 需要重建数据库时
#   - 中间件容器长时间未使用、需要做一次性健康验证时
#
# ❌ 非适用时机（请改用 scripts/start.sh）：
#   - 每天日常启停（开机/关机）
#   - 仅想拉起应用层（Go 后端 + 前台 + 管理端）
#
# 职责：
#   1. 检查 Docker / Docker Compose 是否安装
#   2. 启动 PostgreSQL + Redis + MinIO 容器（docker-compose up -d）
#   3. 通过重试式健康检查（pg_isready / redis ping / MinIO /health/live）确认就绪
#   4. 输出基础设施连接信息与默认管理员账号
#
# 日常启停请使用：
#   ./scripts/start.sh    # 一键启动全部服务
#   ./scripts/stop.sh     # 一键停止应用层
#   ./scripts/status.sh   # 查看端口/PID/容器状态
# ==============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DEPLOY_DIR="$PROJECT_ROOT/deploy"

echo "========================================="
echo "  EchoChat 开发环境【首次】搭建"
echo "========================================="

if ! command -v docker &> /dev/null; then
    echo "❌ Docker 未安装，请先安装 Docker Desktop"
    exit 1
fi

if ! docker compose version &> /dev/null; then
    echo "❌ Docker Compose 不可用，请确保 Docker Desktop 已正确安装"
    exit 1
fi

echo "✅ Docker 和 Docker Compose 已就绪"

echo ""
echo "🚀 启动 PostgreSQL / Redis / MinIO ..."
cd "$DEPLOY_DIR"
docker compose -f docker-compose.dev.yml up -d postgres redis minio

echo ""
echo "⏳ 等待服务健康检查通过 ..."
sleep 3

MAX_RETRIES=30

# PostgreSQL
RETRY=0
until docker exec echochat-postgres pg_isready -U echochat > /dev/null 2>&1; do
    RETRY=$((RETRY + 1))
    if [ $RETRY -ge $MAX_RETRIES ]; then
        echo "❌ PostgreSQL 启动超时"
        exit 1
    fi
    sleep 1
done
echo "✅ PostgreSQL 已就绪 (localhost:5432)"

# Redis
RETRY=0
until docker exec echochat-redis redis-cli ping > /dev/null 2>&1; do
    RETRY=$((RETRY + 1))
    if [ $RETRY -ge $MAX_RETRIES ]; then
        echo "❌ Redis 启动超时"
        exit 1
    fi
    sleep 1
done
echo "✅ Redis 已就绪 (localhost:6379)"

# MinIO：使用 /minio/health/live 端点（容器内 curl 检查）
RETRY=0
until docker exec echochat-minio curl -fs http://localhost:9000/minio/health/live > /dev/null 2>&1; do
    RETRY=$((RETRY + 1))
    if [ $RETRY -ge $MAX_RETRIES ]; then
        echo "❌ MinIO 启动超时"
        exit 1
    fi
    sleep 1
done
echo "✅ MinIO 已就绪 (API: localhost:9000 | Console: localhost:9001)"

echo ""
echo "========================================="
echo "  ✅ 开发环境搭建完成！"
echo "========================================="
echo ""
echo "  PostgreSQL:   localhost:5432"
echo "    数据库:     echochat"
echo "    用户名:     echochat"
echo "    密码:       echochat_dev_2026"
echo ""
echo "  Redis:        localhost:6379"
echo ""
echo "  MinIO API:    http://localhost:9000"
echo "  MinIO 控制台: http://localhost:9001"
echo "    账号:       echochat / echochat123456"
echo ""
echo "  管理员账号:   admin / admin123456"
echo ""
echo "  👉 下一步：运行 ./scripts/start.sh 启动应用层服务"
echo "========================================="
