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

## 与根流程的关系
- 本地分支、可选 worktree、本地集成分支、本地验证入口和风险边界，以仓库根 `CLAUDE.md` 为准；默认验证入口是 `verify.sh`，默认验证的是当前工作树快照，并且测试/构建检查必须在 Docker 内完成；runtime 行为确认走单实例 `preview`。

## 关键约束
- 严格遵守 `Handler -> Service -> Repository` 分层。
- SQL 只在 repository / store 层写，不要在 handler/service 里直接查库。
- PostgreSQL 是权威存储；Redis 只承担热点读缓存与失效，不承载最终真相。
- 分页只用 cursor / keyset，不用 OFFSET；cursor 对上层保持不透明。
- 地理相关搜索、发现、附近能力统一复用 `internal/platform/geo` 的 S2 planner，不要在业务模块里重复写 geo 逻辑。
- backend-go 不使用 PostGIS、geometry/geography 类型或 `ST_*` 查询。
- 动态 SQL 片段必须走白名单映射，不直接拼接客户端输入。
- WebSocket 与 HTTP 必须共享同一套 auth authority；不要单独 new 一份独立的 auth service。
- MUST 写接口优先复用平台层 idempotency replay，不要为单模块另起一套幂等实现。
- 修改 API、字段、错误码、事件 payload 前，先对照 `spec/`。

## 运行与实现提示
- 事务由 service 控制；repository 负责在已有事务或连接上执行 SQL，不自行开启业务事务。
- 新增 geo 逻辑时，优先看 `internal/platform/geo`、`internal/modules/map` 和已有附近搜索实现，而不是在模块里重写一套 S2 计算。
- 错误码只能来自 spec，HTTP 统一返回 `HTTP 200 + ApiResponse<T>`，业务成功唯一依据是 `code == 0`。

## 运行配置
- 当前后端主要使用环境变量：
  - `APP_HOST`
  - `APP_PORT`
  - `POSTGRES_DSN`
  - `REDIS_DSN`
  - `AUTH_TOKEN_SECRET`
  - 以及 app config 相关的公开站点/默认资源配置

## 常用命令
- 默认验证入口：`./deploy/scripts/verify.sh --env-name <name>`
- 需要 backend-go 构建产物时：`./deploy/scripts/preview-build.sh`
- 需要确定性发布产物时：`./deploy/scripts/release-build.sh --ref <git-ref>`
- 如无用户明确要求，不要直接在宿主环境运行 `go build ./cmd/server`、`go test ./...`、`go vet ./...`
