# ==================================================
# EchoChat media-server Dockerfile（多阶段构建）
# 基础镜像：node:20-bookworm-slim（mediasoup 构建需要 python3 + g++）
# ==================================================

# ---------------- Stage 1: builder ----------------
FROM node:20-bookworm-slim AS builder

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      python3 \
      python3-pip \
      build-essential \
      pkg-config \
      ca-certificates \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY package.json package-lock.json* ./
RUN --mount=type=cache,target=/root/.npm \
    if [ -f package-lock.json ]; then npm ci; else npm install; fi

COPY tsconfig.json tsconfig.build.json ./
COPY src ./src

RUN npm run build

# 裁掉 devDependencies，只保留运行时依赖
RUN npm prune --omit=dev

# ---------------- Stage 2: runtime ----------------
FROM node:20-bookworm-slim AS runtime

ENV NODE_ENV=production \
    LOG_PRETTY=false \
    HTTP_HOST=0.0.0.0 \
    HTTP_PORT=3300

# mediasoup-worker 是原生二进制，运行时需要 libstdc++ 等基础库，slim 默认已包含
# 健康检查需要 curl
RUN apt-get update \
 && apt-get install -y --no-install-recommends curl ca-certificates \
 && rm -rf /var/lib/apt/lists/* \
 && useradd --system --create-home --shell /usr/sbin/nologin mediasoup

WORKDIR /app

COPY --from=builder --chown=mediasoup:mediasoup /app/node_modules ./node_modules
COPY --from=builder --chown=mediasoup:mediasoup /app/dist ./dist
COPY --chown=mediasoup:mediasoup package.json ./

USER mediasoup

EXPOSE 3300
EXPOSE 40000-40199/udp
EXPOSE 40000-40199/tcp

HEALTHCHECK --interval=10s --timeout=3s --start-period=15s --retries=3 \
  CMD curl -fsS http://127.0.0.1:${HTTP_PORT:-3300}/healthz || exit 1

CMD ["node", "dist/app.js"]
