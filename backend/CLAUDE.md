# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 这个后端做什么
这是 HotSpot 的 Rust / Axum 主后端，承载大部分对外 API、WebSocket、事件处理、缓存与 PostGIS 地理查询。入口在 `src/main.rs`，路由统一挂到 `/api/v1`，并在启动时装配 tracing、AppState、事件总线和 WebSocket 相关任务。

## 代码组织
- `src/main.rs`：启动日志、加载配置、创建应用状态、挂载路由、启动 HTTP 服务。
- `src/routes/v1/`：按业务域拆分路由，是接口入口的组合层。
- `src/handlers/`：HTTP 边界，只做参数提取、调用 service、返回 `ApiResponse`。
- `src/services/`：业务规则、权限、事务、缓存和事件发布。
- `src/repositories/`：SQLx 仓库层，只负责 SQL 和结果映射。
- `src/dto/`：请求/响应类型。
- `src/events/` 和 `src/websocket/`：事件分发、订阅和实时推送。
- `src/middleware/`：request id 等 HTTP 中间件。
- `src/utils/`：缓存、token、web cookie、客户端 IP 等通用工具。

## 与根流程的关系
- 主目录保持 `main` + clean，所有实际改动都在 worktree 中完成；本地分支、本地集成 worktree、本地验证入口和风险边界，以仓库根 `CLAUDE.md` 为准。
- 后端改动完成后，优先运行根文档定义的本地 Docker 验证流程；影响 runtime 行为时，再补跑 runtime 验证。

## 关键约束
- 只用 `POST` + JSON，业务响应统一为 `HTTP 200 + ApiResponse<T>`。
- 严格遵守 `Handler -> Service -> Repository` 分层，不跨层访问。
- SQL 只在 repository 层写，使用 `sqlx` 宏，不要写非宏查询。
- 分页只用 cursor / keyset，不用 OFFSET；cursor 编解码只在 repository 层完成。
- 距离、范围和排序计算在 PostGIS 完成，不在 Rust 代码里手算。
- 修改接口前先读对应 `spec/api_spec_*.md`。
- 日志要带 request id，不记录密码、Token、隐私内容。

## 运行时提示
- 认证与会话逻辑、黑名单和 WebSocket 行为都已经在现有结构里分层，改动时优先沿用现有模式。
- 新增路由时优先放在 `src/routes/v1/`，再由 `src/main.rs` 的总路由挂载。
- 如果需要自查旧式 SQL 写法，重点检查是否出现非宏 `sqlx::query(`、`sqlx::query_as::<`、`sqlx::query_scalar::<`。

## 常用命令
- 默认验证入口遵循仓库根 `CLAUDE.md` 与 `deploy/` Docker 流程。
- worktree 内的格式化、语法检查和编译通过受控脚本执行：
  - `python3 .claude/scripts/rust_dev.py --task fmt` — 格式检查
  - `python3 .claude/scripts/rust_dev.py --task check` — 语法与类型检查
  - `python3 .claude/scripts/rust_dev.py --task clippy` — lint
  - `python3 .claude/scripts/rust_dev.py --task build` — 编译
  - `python3 .claude/scripts/rust_dev.py --task all` — 以上全部（默认）
- 不允许直接调用 `cargo fmt`/`cargo check`/`cargo clippy`/`cargo build`；`cargo test` 必须通过 `deploy/` 隔离执行。
