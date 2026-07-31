# ── HotSpot Frontend (SvelteKit SPA) ────────────────────────────────
# 多阶段构建：生产镜像只保留 adapter-static 产物，通过 compose 共享 volume 挂给 nginx。

# Stage 1: Build
FROM oven/bun:${BUN_VERSION:-1.3.14}-alpine AS build
WORKDIR /app
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile
COPY . .
# SPA 构建期无服务端渲染，PUBLIC_* 变量在客户端运行时读取（经 nginx 同源），无需注入
RUN bun run build

# Stage 2: Minimal production image（仅持有 SPA 静态产物）
FROM alpine:${ALPINE_VERSION:-3.22}
COPY --from=build /app/build/ /build/
