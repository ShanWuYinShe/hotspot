# ── HotSpot Frontend (SvelteKit SPA) ────────────────────────────────
# 多阶段构建：生产镜像只保留 adapter-static 产物，通过 compose 共享 volume 挂给 nginx。

# Stage 1: Build
FROM oven/bun:${BUN_VERSION:-1.3.14}-alpine AS build
WORKDIR /app
# .npmrc 指定 npmmirror 源（bun.lock 亦为 npmmirror），避免回落 npmjs.org
COPY package.json bun.lock .npmrc ./
# package.json 的 prepare 脚本在 bun install 时执行，必须先于源码拷贝到位；
# 容器内无 git，脚本会按其兼容性约定静默跳过
COPY scripts/setup-git-hooks.sh ./scripts/
RUN bun install --frozen-lockfile
COPY . .
# SPA 构建期无服务端渲染，PUBLIC_* 变量在客户端运行时读取（经 nginx 同源），无需注入
RUN bun run build

# Stage 2: Minimal production image（仅持有 SPA 静态产物）
FROM alpine:${ALPINE_VERSION:-3.24.1}
COPY --from=build /app/build/ /build/
# 保持容器运行：命名卷首次挂载时从镜像复制产物（healthcheck 依赖存活容器）
CMD ["sh", "-c", "sleep infinity"]
