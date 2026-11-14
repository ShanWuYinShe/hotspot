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
./scripts/seed-dev.sh   # 注入开发测试数据（Postgres + SeaweedFS，幂等）
```

启动后访问 `http://localhost:8888`（nginx 网关统一反代前端、API 与 WebSocket）。

开发环境不内置 API mock，所有请求走真实后端；测试账号与种子数据说明见 [AGENTS.md](AGENTS.md)「开发种子数据」。

## 版本号管理与发布

版本号以 git tag 为准（`vX.Y.Z`，全项目共用一个 tag 序列），`git describe --tags --always` 自动生成人类可读版本串，git commit sha 做精确回溯锚点。版本指纹贯穿发布链路，线上出问题时可逐层核对：

| 层 | 查看方式 |
|----|----------|
| Go 后端 | `POST /public/v1/system/health` 响应的 `version`/`commit`/`build_time`；容器启动日志首行 |
| Web 前端 | `GET /version.json`（构建产物静态文件）；设置页「关于 HotSpot」徽标对用户可见 |
| Docker 镜像 | `IMAGE_TAG` 即版本串；OCI label（`org.opencontainers.image.revision`）含完整 commit |
| 鸿蒙客户端 | 设置页「关于」行显示 `AppScope/app.json5` 的 `versionName`（发版约定见 `harmonyapp/AGENTS.md`） |

发布为**产物分发模式**：tag 触发 Release 工作流构建前后端镜像（注入版本指纹）并附 `.sha256` 挂 GitHub Release，
本地 `./scripts/deploy-production.sh vX.Y.Z` 完成校验和核验 → 配置导出 → 服务器 `docker load` → OCI 指纹核对 →
数据库快照 → 滚动更新 → 三级版本校验，失败自动回滚。

发布门禁（硬性）：打 tag 前 `origin/main` CI 必须全绿（含 `Docker Build + Runtime Smoke` 与 `Go Backend (Postgres Integration)`）
——生产镜像构建与容器内验证由 CI 在 PR/main 上覆盖，本地不构建生产镜像；鸿蒙端改动需本地 hvigor 构建确认
（见 `AGENTS.md`「发布门禁」）。

回滚：`./scripts/rollback-production.sh --previous`（或指定版本）——复用同一条部署链路，只回滚镜像、不回滚数据。

热修复、发布门禁清单与异常处置见 [docs/release-runbook.md](docs/release-runbook.md)。

## 文档

- [AGENTS.md](AGENTS.md) — 协作规范
- [spec/](spec/) — API 契约
