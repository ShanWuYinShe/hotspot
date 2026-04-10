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
- worktree / integration / sync-back / quality gate / 风险边界等仓库级流程，以仓库根 `CLAUDE.md` 为唯一权威来源；本文件只补充 backend-go 的局部架构、实现约束与验证提示。
- 标准 workflow、skill 入口、受控脚本边界、`tmp/` 使用原则都以根 `CLAUDE.md` 为准；这里不重复复制仓库级规则。
- backend-go 的最终验证入口默认是 `verify.sh`；若涉及 runtime 行为，再结合候选预演或 inspect 入口确认。

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

## 常用能力
- 普通分析、代码阅读、局部低副作用开发动作可直接进行。
- 当仓库已经提供 skill 时，优先通过 skill 发现并使用标准 workflow 入口。
- 统一验证与隔离运行仍优先走 `deploy/` 提供的入口。
- 若需要落调试输出、日志片段或临时分析文件，默认写到当前 worktree 的 `tmp/`。
