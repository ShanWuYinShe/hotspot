# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概览
HotSpot 是一个多端 monorepo：
- `backend/`：Rust + Axum 主后端，负责大部分业务 API、WebSocket、事件总线和 PostGIS 地理查询。
- `backend-go/`：Go + chi 辅助后端，现阶段聚焦认证、用户和 S2 地理查询。
- `frontend_svelte/`：SvelteKit Web 前端。
- `harmonyapp/`：HarmonyOS ArkTS 客户端。
- `spec/`：API 契约和行为规范的权威来源。

## 先读什么
- 任何接口、字段、错误码或 WebSocket 变更前，先读 `spec/api_spec_index.md`、对应模块 spec，以及 `spec/api_spec_base.md`、`spec/api_spec_types.md`、`spec/api_spec_errors.md`。
- 进入子项目工作时，先读该目录下的 `CLAUDE.md`，再按需看源码。

## 常用命令
### Rust 后端
- `cd backend && cargo check`
- `cd backend && cargo clippy`
- `cd backend && cargo test`
- 单测：`cd backend && cargo test <测试名> -- --exact`

### Go 后端
- `cd backend-go && go build ./cmd/server`
- `cd backend-go && go test ./...`
- `cd backend-go && go vet ./...`
- 单测：`cd backend-go && go test ./... -run 'TestName$'`

### Web 前端
- `cd frontend_svelte && pnpm check`
- `cd frontend_svelte && pnpm build`
- `cd frontend_svelte && pnpm dev`
- 当前仓库未配置独立前端单测脚本，主要用 `pnpm check` / `pnpm build` 验证。

### HarmonyOS 客户端
- `cd harmonyapp && OHPM="/Applications/DevEco-Studio.app/Contents/tools/ohpm/bin/ohpm" "$OHPM" install --all --registry https://ohpm.openharmony.cn/ohpm/ --strict_ssl true`
- `cd harmonyapp && DEVECO_SDK_HOME="/Applications/DevEco-Studio.app/Contents/sdk" DEVECO_SDK_NODE="/Applications/DevEco-Studio.app/Contents/tools/node/bin/node" CODELINTER="/Applications/DevEco-Studio.app/Contents/plugins/codelinter/run/index.js" "$DEVECO_SDK_NODE" "$CODELINTER" -c code-linter.json5 entry/src/main/ets/`
- `hvigorw` 编译命令由用户手动运行，不要代跑。

## 代码组织
- `backend/`：`routes/v1` 组合路由，`handlers` 只做 HTTP 边界，`services` 放业务，`repositories` 放 SQL，`dto` 放请求/响应类型，`events` / `websocket` 处理实时链路。
- `backend-go/`：`cmd/server/main.go` 启动，`internal/app` 负责装配，`handler -> service -> repository` 严格分层，`geo/s2util` 负责 S2 相关计算，`middlewares` 管 request id / 认证 / 超时 / 日志。
- `frontend_svelte/`：`src/routes` 是页面与路由分组，`src/lib/api` 集中端点、客户端和业务服务，`src/lib/stores` 放全局状态，`src/lib/websocket` 管事件协议和连接状态。
- `harmonyapp/`：`entry/src/main/ets/features` 按业务域分页面，`core/api`、`core/network`、`core/state`、`core/router`、`core/types` 和 `core/utils` 组成公共基础层，`components` 放可复用 UI。

## 全局约束
- `spec/` 是唯一权威契约来源；除非用户明确要求，不要修改其中文件。
- 临时文件统一放在仓库根目录 `tmp/`。
- 涉及破坏性删除、批量改动、数据库迁移/重置、`deploy/` 脚本、`git push` 之类操作时，先确认用户意图。

## 推荐工作流（重要）
- 当前仓库默认采用 **多 agent + worktree + 多分支 + PR** 的协作方式推进复杂任务。
- 当主工作区存在未提交改动时，**不要**直接在主工作区上继续开发；优先创建独立 worktree，在干净分支中实现变更。
- 复杂任务应优先拆成多个独立主题分支，每个主题在自己的 worktree 中开发、测试、修复。
- 每个分支在本地验证通过后，再创建 PR 合并回主分支；不要把多个无关主题堆到同一个分支。
- 若使用多个 agent 并行开发，优先让它们在各自 worktree 中工作，避免互相污染未提交改动。

## Docker 测试 / 运行验证流程（重要）
- `deploy/` 目录现在是标准入口：
  - `deploy/scripts/test-in-docker.sh`：干净的 Docker 测试入口
  - `deploy/scripts/up-runtime.sh`：构建并启动隔离 runtime 环境
  - `deploy/scripts/inspect-runtime.sh`：统一巡检 runtime 状态、入口与日志
- 对 **测试导向** 变更，优先运行：
  - `./deploy/scripts/test-in-docker.sh --env-name <name>`
- 对 **影响 runtime 行为** 的变更，优先运行：
  - `./deploy/scripts/up-runtime.sh --env-name <name> --recreate backend,frontend,nginx --http-port <http> --https-port <https>`
  - 然后运行：`./deploy/scripts/inspect-runtime.sh --env-name <name> --https-port <https>`
- 后端运行与测试都继续遵循“同一套配置 key，不同 datasource value”的原则；不要引入额外环境模式抽象或 Redis key-prefix 机制，除非用户明确要求。

## 推荐工作流（重要）
- 当前仓库默认采用 **多 agent + worktree + 多分支 + PR** 的协作方式推进复杂任务。
- 当主工作区存在未提交改动时，**不要**直接在主工作区上继续开发；优先创建独立 worktree，在干净分支中实现变更。
- 复杂任务应优先拆成多个独立主题分支，每个主题在自己的 worktree 中开发、测试、修复。
- 每个分支在本地验证通过后，再创建 PR 合并回主分支；不要把多个无关主题堆到同一个分支。
- 若使用多个 agent 并行开发，优先让它们在各自 worktree 中工作，避免互相污染未提交改动。

## Docker 测试 / 运行验证流程（重要）
- `deploy/` 目录现在是标准入口：
  - `deploy/scripts/test-in-docker.sh`：干净的 Docker 测试入口
  - `deploy/scripts/up-runtime.sh`：构建并启动隔离 runtime 环境
  - `deploy/scripts/inspect-runtime.sh`：统一巡检 runtime 状态、入口与日志
- 对 **测试导向** 变更，优先运行：
  - `./deploy/scripts/test-in-docker.sh --env-name <name>`
- 对 **影响 runtime 行为** 的变更，优先运行：
  - `./deploy/scripts/up-runtime.sh --env-name <name> --recreate backend,frontend,nginx --http-port <http> --https-port <https>`
  - 然后运行：`./deploy/scripts/inspect-runtime.sh --env-name <name> --https-port <https>`
- 后端运行与测试都继续遵循“同一套配置 key，不同 datasource value”的原则；不要引入额外环境模式抽象或 Redis key-prefix 机制，除非用户明确要求。
