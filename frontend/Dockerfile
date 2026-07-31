# ── HotSpot Frontend (React SPA with Next.js) ────────────────────
# 多阶段构建：生产镜像只保留 static export 产物，通过 compose 共享 volume 挂给 nginx。

# Stage 1: Build
FROM oven/bun:${BUN_VERSION:-1.3.14}-alpine AS build
WORKDIR /app
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile
COPY . .
ARG NEXT_PUBLIC_AMAP_KEY
ARG NEXT_PUBLIC_AMAP_SECURITY_CODE
ARG NEXT_PUBLIC_AMAP_SERVICE_HOST
ENV NEXT_PUBLIC_AMAP_KEY=${NEXT_PUBLIC_AMAP_KEY}
ENV NEXT_PUBLIC_AMAP_SECURITY_CODE=${NEXT_PUBLIC_AMAP_SECURITY_CODE}
ENV NEXT_PUBLIC_AMAP_SERVICE_HOST=${NEXT_PUBLIC_AMAP_SERVICE_HOST}
RUN bun run build

# Stage 2: Minimal production image（仅持有 static export 产物）
FROM alpine:${ALPINE_VERSION:-3.22}
COPY --from=build /app/out/ /build/
