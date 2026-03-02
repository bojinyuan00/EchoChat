# ============================================================
# EchoChat Go 后端服务 Dockerfile
# 多阶段构建：编译阶段 → 运行阶段（alpine 最小镜像）
# ============================================================

# --- 第一阶段：编译 ---
FROM golang:1.23-alpine AS builder

# 安装编译依赖（CGO 需要 gcc）
RUN apk add --no-cache gcc musl-dev

WORKDIR /build

# 先复制依赖文件利用 Docker 缓存层
COPY go.mod go.sum ./
RUN go mod download

# 复制全部源码并编译
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/echochat ./cmd/server

# --- 第二阶段：运行 ---
FROM alpine:3.19

# 安装运行时依赖（时区数据 + CA 证书）
RUN apk add --no-cache ca-certificates tzdata

# 设置时区为上海
ENV TZ=Asia/Shanghai

WORKDIR /app

# 从编译阶段复制二进制文件
COPY --from=builder /build/echochat .

# 复制配置文件目录
COPY --from=builder /build/config ./config

# 创建日志目录
RUN mkdir -p /app/logs

EXPOSE 8085

# 健康检查
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null http://localhost:8085/health || exit 1

CMD ["./echochat"]
