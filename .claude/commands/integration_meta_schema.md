---
description: 说明 integration 运行时元信息文件的路径约定和最小字段集
argument-hint: [integration-id]
allowed-tools: Read, Glob, Grep
---

请说明当前仓库 integration 运行时元信息文件的约定。当前命令参数为：

$ARGUMENTS

目标是统一说明 `tmp/claude/integrations/<integration-id>.json` 的路径、字段和读写原则，供 `/start_integrate`、`/refresh_integrate`、`/verify_integrate`、`/finish_integrate`、`/status_worktrees` 使用。

约定如下：

1. 运行时元信息文件统一放在仓库根：`tmp/claude/integrations/`。
2. 每轮 integration 对应一个 JSON 文件：`tmp/claude/integrations/<integration-id>.json`。
3. 最小字段集：
   - `integration_id`
   - `baseline_main_sha`
   - `integration_branch`
   - `integration_worktree_path`
   - `planned_features`
   - `verify_status`
   - `review_status`
   - `gate_status`
   - `updated_at`
4. `start_integrate` 负责初始化文件；`refresh_integrate` 负责刷新基线；`verify_integrate` / `finish_integrate` 负责更新状态；`status_worktrees` 和 `pre_tool_use.py` 只读取，不在 hook 里写状态。
5. 该文件是运行时状态锚点，不是业务配置，也不是审计日志；不要把无关字段塞进去。

结束时请输出：
- 当前约定路径
- 最小字段集
- 各命令的读写职责
- 如何据此判断 `active` / `stale`
