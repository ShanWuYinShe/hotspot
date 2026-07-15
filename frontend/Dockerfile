# ── Stage 1: Build SvelteKit (adapter-static) ──────────────────────
FROM oven/bun:${BUN_VERSION:-1.3.14}-alpine AS builder

WORKDIR /app

# 先拷贝依赖清单，利用 Docker 层缓存
COPY package.json bun.lock* ./
RUN bun install --frozen-lockfile

# 再拷贝源码（.dockerignore 应排除 node_modules / build / .svelte-kit）
COPY . .

# Vite 构建期需要的高德地图配置（构建时注入，产物内联）
ARG VITE_AMAP_KEY
ARG VITE_AMAP_SECURITY_CODE
ARG VITE_AMAP_SERVICE_HOST
ENV VITE_AMAP_KEY=${VITE_AMAP_KEY} \
    VITE_AMAP_SECURITY_CODE=${VITE_AMAP_SECURITY_CODE} \
    VITE_AMAP_SERVICE_HOST=${VITE_AMAP_SERVICE_HOST}

RUN bun run build

# ── Stage 2: 产物占位容器 ──────────────────────────────────────────
# 仅持有 build 产物供网关 nginx 通过共享 volume 挂载，不再自带 nginx。
# 用 tail 保持容器存活（compose depends_on 需要服务 running）。
FROM alpine:${ALPINE_VERSION:-3.22}

COPY --from=builder /app/build /build

CMD ["tail", "-f", "/dev/null"]
