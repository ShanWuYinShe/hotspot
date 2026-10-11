# ── HotSpot Frontend (SvelteKit SPA) ────────────────────────────────
# 多阶段构建：生产镜像只保留 adapter-static 产物，通过 compose 共享 volume 挂给 nginx。

# 基础镜像版本:默认值以本仓 deploy/versions.env 为准,由 workwork 的
# scripts/sync-versions.sh 按跨仓主本校验/刷写(此前用内联 :- 默认值,无法被治理)
ARG BUN_VERSION=1.3.14
ARG ALPINE_VERSION=3.24.1

# Stage 1: Build
FROM oven/bun:${BUN_VERSION}-alpine AS build
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
FROM alpine:${ALPINE_VERSION}

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

# 非 root 运行：uid/gid 显式固定为 100:101——deploy-production.sh 迁移存量
# 卷属主时依赖该值（存量 hotspot_frontend_build 卷属主为 root，容器降权前
# 需 docker run alpine chown 100:101 一次性迁移，脚本已内置、幂等）。
# 预建 /build 并 chown：命名卷首挂载时 Docker 会复制挂载点属主，新部署的
# 卷直接以 app 属主初始化，容器内无需特权操作。
RUN addgroup -S -g 101 app && adduser -S -u 100 -G app app \
    && mkdir -p /build && chown app:app /build

USER app

# 每次容器启动（= 每次换镜像重建）把新版产物铺进静态卷，静态卷自刷新，无需
# 部署脚本手工 tar 拷贝；随后保持容器存活（healthcheck 依赖）。
#
# 关键：不再先 rm -rf 清空卷再拷贝——nginx 只读挂载同一卷，清空到拷贝完成之间
# 的请求会撞上「index.html 不存在」，表现为部署瞬间的白屏/404（而部署脚本的
# 版本校验发生在拷贝完成之后，照样报成功）。改为三步安全序：
#   1) 先同步除入口文件外的资源（同名覆盖；旧 index.html 与旧 chunk 都还在，
#      这期间到达的请求仍由旧版本正常响应）；
#   2) index.html / version.json 用「写临时文件 + mv」原子替换，读者不会拿到
#      半截文件（mv 与 cp 同目录同一文件系统内为原子重命名）；
#   3) 清理上一代遗留资源：不在本版清单内、且 mtime 超过 24h 的文件——宽限期
#      让还停留在旧页面的客户端仍能取到旧 chunk，也保证卷不会无限增长。
# 任一步失败（磁盘满、tar 出错）容器退出，卷内仍是上一版完整产物：宁可保持
# 旧版本可用，也不留半新半旧的卷。
CMD ["sh", "-c", "set -e; SRC=/srv/dist; DEST=/build; mkdir -p \"$DEST\"; ( cd \"$SRC\" && tar cf - --exclude=./index.html --exclude=./version.json . ) | ( cd \"$DEST\" && tar xf - ); if [ -f \"$SRC/version.json\" ]; then cp \"$SRC/version.json\" \"$DEST/.version.json.tmp\" && mv -f \"$DEST/.version.json.tmp\" \"$DEST/version.json\"; fi; cp \"$SRC/index.html\" \"$DEST/.index.html.tmp\" && mv -f \"$DEST/.index.html.tmp\" \"$DEST/index.html\"; ( cd \"$SRC\" && find . -type f | sed 's|^\\./||' ) > /tmp/keep.txt; find \"$DEST\" -type f -mmin +1440 | while read -r f; do rel=\"${f#$DEST/}\"; grep -qxF \"$rel\" /tmp/keep.txt || rm -f \"$f\"; done; find \"$DEST\" -type d -empty -exec rmdir {} + 2>/dev/null || true; exec sleep infinity"]
