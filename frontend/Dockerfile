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

# 版本指纹（容器内无 .git，必须由部署链路经 build args 注入环境变量：
# docker-compose.yml ← scripts/deploy-production.sh；不传时兜底 dev）。
# write-version-info.mjs 会把指纹写入 static/version.json 随产物发布。
ARG GIT_VERSION=dev
ARG GIT_COMMIT=
ARG GIT_BUILD_TIME=
ENV GIT_VERSION=${GIT_VERSION} \
    GIT_COMMIT=${GIT_COMMIT} \
    GIT_BUILD_TIME=${GIT_BUILD_TIME}
# 高德 Web key（构建期烘焙进静态产物；geo-service.ts 经 import.meta.env 读取，
# 为空则自动走 OSM 降级。key 打包后可被提取，配额与告警见 .env.example）
ARG VITE_AMAP_KEY=
ENV VITE_AMAP_KEY=${VITE_AMAP_KEY:-}
# SPA 构建期无服务端渲染，PUBLIC_* 变量在客户端运行时读取（经 nginx 同源），无需注入
RUN bun run build

# Stage 2: Minimal production image（仅持有 SPA 静态产物）
FROM alpine:${ALPINE_VERSION:-3.24.1}

ARG GIT_VERSION=dev
ARG GIT_COMMIT=
ARG GIT_BUILD_TIME=
# OCI 标准镜像标签：docker inspect 即可核对镜像对应的源码版本
LABEL org.opencontainers.image.version=${GIT_VERSION} \
      org.opencontainers.image.revision=${GIT_COMMIT} \
      org.opencontainers.image.created=${GIT_BUILD_TIME}
# 产物在镜像内固定放 /srv/dist（不放在 /build——那是 compose 命名卷挂载点，
# 挂载会遮住镜像内容，这正是「卷非空时新产物不生效」坑的根源）。
COPY --from=build /app/build/ /srv/dist/

# 每次容器启动（= 每次换镜像重建）先把卷清空再灌入新产物：静态卷自刷新，
# 无需部署脚本再做手工 tar 拷贝；随后保持容器存活（healthcheck 依赖）。
CMD ["sh", "-c", "rm -rf /build/* /build/.[!.]* /build/..?* 2>/dev/null; cp -r /srv/dist/. /build/ && exec sleep infinity"]
