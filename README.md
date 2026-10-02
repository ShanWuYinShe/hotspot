# HotSpot

基于地理位置的社交应用。围绕位置组织附近的人、群组与生活瞬间，提供发现、聊天、地图等能力。

## 功能

- **附近发现**：按位置浏览附近的人、群组、生活瞬间
- **社交**：关注、好友、私聊、群聊、评论、点赞、举报
- **地图**：基于地理位置的浏览与位置选择
- **搜索**：用户、群组、内容搜索
- **实时消息**：WebSocket 推送的聊天与通知
- **管理后台**：用户与内容管理

## 子项目

| 子项目 | 说明 | 技术栈 |
|--------|------|--------|
| `backend-go/` | Go 主后端（当前活跃） | Go / Chi / PostgreSQL / S2 |
| `frontend/` | Web 主前端 | SvelteKit / Svelte 5 / TypeScript / Tailwind CSS |
| `harmonyapp/` | HarmonyOS 客户端 | HarmonyOS ArkTS |
| `spec/` | API 契约（只读权威） | — |

## 快速开始

```bash
cp .env.example .env
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d
```

启动后访问 `http://localhost:8888`（nginx 网关统一反代前端、API 与 WebSocket）。

## 版本号管理与发布

版本号以 git tag 为准（`vX.Y.Z`，全项目共用一个 tag 序列），`git describe --tags --always` 自动生成人类可读版本串，git commit sha 做精确回溯锚点。版本指纹贯穿发布链路，线上出问题时可逐层核对：

| 层 | 查看方式 |
|----|----------|
| Go 后端 | `POST /public/v1/system/health` 响应的 `version`/`commit`/`build_time`；容器启动日志首行 |
| Web 前端 | `GET /version.json`（构建产物静态文件）；设置页「关于 HotSpot」徽标对用户可见 |
| Docker 镜像 | `IMAGE_TAG` 即版本串；OCI label（`org.opencontainers.image.revision`）含完整 commit |
| 鸿蒙客户端 | 设置页「关于」行显示 `AppScope/app.json5` 的 `versionName`（发版约定见 `harmonyapp/AGENTS.md`） |

发布为**产物分发模式**，服务器不接触源码、不构建：

0. 本地 Docker 验证（硬性前置，链路见 `AGENTS.md`「发布门禁」：dev 联调 → 集成测试 → 生产镜像构建 → 版本指纹核对），通过后才允许打 tag；
1. 打 tag 并推送（`git tag vX.Y.Z && git push origin vX.Y.Z`，一次一个）；
2. CI「Release」工作流先过发布门禁（tag 必须是 `vX.Y.Z` 且落在 `origin/main` 上、该提交 CI 全绿），再构建前后端镜像（注入版本指纹），打包并生成 `.sha256` 校验和，挂到同名 GitHub Release；
3. 本地执行 `./scripts/deploy-production.sh vX.Y.Z`：校验发布包校验和 → 从该 tag 导出部署配置（配置与镜像同源）→ 服务器 `docker load` → 核对镜像 OCI 指纹 → 数据库快照 → 滚动更新 → 后端/前端/网关三级版本校验，失败自动回滚上一版本；结束后回收旧镜像并写入部署审计日志。

回滚：`./scripts/rollback-production.sh --previous`（或指定版本）——复用同一条部署链路，只回滚镜像、不回滚数据。

热修复、发布门禁清单与异常处置见 [docs/release-runbook.md](docs/release-runbook.md)。

## 文档

- [AGENTS.md](AGENTS.md) — 协作规范
- [spec/](spec/) — API 契约
