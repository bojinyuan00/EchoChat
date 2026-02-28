#!/bin/bash
# EchoChat 开发环境一键搭建脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DEPLOY_DIR="$PROJECT_ROOT/deploy"

echo "========================================="
echo "  EchoChat 开发环境搭建"
echo "========================================="

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null; then
    echo "❌ Docker 未安装，请先安装 Docker Desktop"
    exit 1
fi

# 检查 Docker Compose 是否可用
if ! docker compose version &> /dev/null; then
    echo "❌ Docker Compose 不可用，请确保 Docker Desktop 已正确安装"
    exit 1
fi

echo "✅ Docker 和 Docker Compose 已就绪"

# 启动基础设施服务
echo ""
echo "🚀 启动 PostgreSQL 和 Redis ..."
cd "$DEPLOY_DIR"
docker compose -f docker-compose.dev.yml up -d

# 等待服务就绪
echo ""
echo "⏳ 等待服务健康检查通过 ..."
sleep 3

# 检查 PostgreSQL
MAX_RETRIES=30
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

# 检查 Redis
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

echo ""
echo "========================================="
echo "  ✅ 开发环境搭建完成！"
echo "========================================="
echo ""
echo "  PostgreSQL: localhost:5432"
echo "    数据库: echochat"
echo "    用户名: echochat"
echo "    密码:   echochat_dev_2026"
echo ""
echo "  Redis: localhost:6379"
echo ""
echo "  管理员账号: admin / admin123456"
echo "========================================="
