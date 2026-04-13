# feature / testing

## 何时读取
在实现完成后、进入本地校验阶段时读取本文件。

## 环境约束
- 所有校验命令**必须在 worktree 目录内执行**，不得切回宿主主目录。
- 若 worktree 内缺少必要工具链且无法安装，视为阻塞并报告给用户。

## 校验范围
本阶段只跑**不依赖外部服务**的单元测试与静态检查。需要数据库或 Redis 的集成测试由 `integration.md` 阶段的 `verify.sh` 在 Docker 中执行。

各子项目的单元测试命令：

- `backend/`：`cargo test`
- `backend-go/`：`go test -short ./...`（`-short` 跳过需要外部服务的集成测试）
- `frontend_svelte/`：`pnpm check && pnpm build`
- `harmonyapp/`：运行当前环境可用的 Harmony 校验命令；若工具链缺失且无法完成必要校验，则视为阻塞

## 规则
- 补齐与本次改动直接相关的必要测试。
- 只要某个受影响子项目存在本地可执行校验，就不能跳过。
- 若测试失败，优先修复实现或测试本身，而不是绕过校验。
- 若失败原因是缺少数据库 / Redis 连接，说明这些是集成测试，跳过并转由 `verify.sh` 验证。
