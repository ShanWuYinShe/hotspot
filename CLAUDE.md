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
- `backend-go/`：`cmd/server/main.go` 启动；当前采用 `internal/bootstrap + internal/platform + internal/modules` 的 modular-monolith 结构，地理能力统一走 S2。
- `frontend_svelte/`：`src/routes` 是页面与路由分组，`src/lib/api` 集中端点、客户端和业务服务，`src/lib/stores` 放全局状态，`src/lib/websocket` 管事件协议和连接状态。
- `harmonyapp/`：`entry/src/main/ets/features` 按业务域分页面，`core/api`、`core/network`、`core/state`、`core/router`、`core/types` 和 `core/utils` 组成公共基础层，`components` 放可复用 UI。

## 全局约束
- `spec/` 是唯一权威契约来源；除非用户明确要求，不要修改其中文件。
- 临时文件统一放在仓库根目录 `tmp/`。
- 涉及破坏性删除、批量改动、数据库迁移/重置、`deploy/` 脚本、`git push` 之类操作时，先确认用户意图。

## 标准开发流程（重要）
1. **开始前必须确保当前分支是主分支，并且工作区是 clean。** 这样可以确保基线完整，没有遗漏未提交功能。
2. **把需求拆成多个互不干涉的任务。** 每个任务分配给独立 agent，在独立 worktree 和独立 feature 分支中开发。
3. **每个 feature 分支必须自己补足测试并验证通过。** 默认优先使用 `deploy/scripts/test-in-docker.sh` 做干净测试；若改动影响运行行为，再补跑 runtime 验证。
4. **所有 feature 分支完成后，不直接开 PR。** 应当从最新主分支再新建一个专门的“集成分支”，把这些任务分支合并进去，统一解决冲突、补齐遗漏，并确保功能完整、测试全部通过。
5. **多个 feature 分支合并进集成分支后，必须再做一次完整 review。** 要重新检查实现质量、接口契约、冲突处理结果、测试覆盖和整体功能完整性，确认没有因为分支间交互而引入问题。
6. **只有集成分支通过验证和 review 后，才 push 集成分支并创建 PR。** PR 合并完成后，删除所有任务分支和这个集成分支。

## Docker 测试 / 运行验证流程（重要）
- `deploy/` 是统一入口：
  - `deploy/scripts/test-in-docker.sh`：干净的 Docker 测试入口
  - `deploy/scripts/up-runtime.sh`：构建并启动隔离 runtime 环境
  - `deploy/scripts/inspect-runtime.sh`：统一巡检 runtime 状态、入口与日志
- 每个 feature 分支开发完成后，优先运行：
  - `./deploy/scripts/test-in-docker.sh --env-name <name>`
- 对影响 runtime 行为的改动，还应运行：
  - `./deploy/scripts/up-runtime.sh --env-name <name> --recreate backend,frontend,nginx --http-port <http> --https-port <https>`
  - 然后运行：`./deploy/scripts/inspect-runtime.sh --env-name <name> --https-port <https>`
- 集成分支在创建 PR 之前，必须重复执行上述验证。
- 运行与测试继续遵循：**同一套配置 key，不同 datasource value**。不要引入额外环境模式抽象或 Redis key-prefix 机制，除非用户明确要求。
