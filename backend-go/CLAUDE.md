# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in `backend-go/`.

## 这个后端做什么
这是 HotSpot 的 Go / chi 后端重构版本。当前已经不只是认证、用户和地图，而是按模块化单体结构逐步覆盖：auth、users、groups、private chats/messages、group messages、moments/comments、notifications、media、search、feed、map、reports、share、admin、system，以及 websocket/realtime。地理相关逻辑统一走 S2，不使用 PostGIS。

## 当前代码组织
- `cmd/server/main.go`：程序入口。
- `internal/bootstrap/`：依赖装配、模块注册、HTTP/WS 路由组装。
- `internal/platform/`：平台层能力：`auth`、`httpx`、`idempotency`、`persistence`、`cache`、`realtime`、`geo`、`observability`、`appconfig` 等。
- `internal/modules/`：业务模块。每个模块内部维持 `handler -> service -> repo` 分层，并逐步引入 `persistence-backed` adapter。
- `migrations/`：数据库迁移。
- `test/`：集成测试与后续 PG/Redis harness 的扩展入口。

## 关键约束
- 严格遵守 `Handler -> Service -> Repository` 分层。
- SQL 只在 repository / store 层写，不要在 handler/service 里直接查库。
- PostgreSQL 是权威存储；Redis 只承担热点读缓存与失效，不承载最终真相。
- 分页只用 cursor / keyset，不用 OFFSET。
- 地理相关搜索、发现、附近能力统一复用 `internal/platform/geo` 的 S2 planner，不要在业务模块里重复写 geo 逻辑。
- WebSocket 与 HTTP 必须共享同一套 auth authority；不要单独 new 一份独立的 auth service。
- MUST 写接口优先复用平台层 idempotency replay，不要为单模块另起一套幂等实现。
- 不要自行执行 `deploy/` 目录脚本、数据库重置或 destructive 运维动作，除非用户明确要求。
- 修改 API、字段、错误码、事件 payload 前，先对照 `spec/`。

## 工作流说明
- 通用的多 agent / 多 worktree / 多 feature 分支 / 本地集成分支汇总流程，以仓库根 `CLAUDE.md` 为准；这里不重复展开。

## Docker 测试 / 运行验证（重要）
- 子项目开发时继续遵循根 `CLAUDE.md` 中定义的 Docker 测试 / runtime 验证流程：
  - 测试优先：`./deploy/scripts/test-in-docker.sh --env-name <name>`
  - runtime 变更：`./deploy/scripts/up-runtime.sh ...` + `./deploy/scripts/inspect-runtime.sh ...`

## 运行配置
- 当前后端主要使用环境变量：
  - `APP_HOST`
  - `APP_PORT`
  - `POSTGRES_DSN`
  - `REDIS_DSN`
  - `AUTH_TOKEN_SECRET`
  - 以及 app config 相关的公开站点/默认资源配置

## 常用命令
- `go build ./cmd/server`
- `go test ./...`
- `go vet ./...`
- 单测：`go test ./... -run 'TestName$'`
