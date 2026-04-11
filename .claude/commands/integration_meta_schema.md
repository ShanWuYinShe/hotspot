---
description: 说明 integration 运行时元信息文件的路径约定和状态字段
argument-hint: [integration-id]
allowed-tools: Read, Glob, Grep
---

请说明当前仓库 integration 运行时元信息文件的约定。当前命令参数为：

$ARGUMENTS

目标是统一说明 `tmp/claude/integrations/<integration-id>.json` 的路径、字段和读写原则，供 `/start_integrate`、`/refresh_integrate`、`/verify_integrate`、`/finish_integrate`、`/sync_back_main`、`/cleanup_worktrees`、`/status_worktrees` 使用。

约定如下：

1. integration metadata 统一放在仓库根：`tmp/claude/integrations/`。
2. 每轮 integration 对应一个 JSON 文件：`tmp/claude/integrations/<integration-id>.json`。
3. 它是 runtime state，不是业务配置，也不是长期审计日志；只放当前自动流程推进需要的字段。
4. 共享脚本 `.claude/scripts/integration_meta.py` 负责字段 schema、状态枚举与读写 API；命令、hook、脚本若要读写 integration metadata，应统一复用它，而不是各自手写路径拼接和 JSON 解析。
5. metadata 至少应覆盖：
   - identity：`schema_version`、`integration_id`、`integration_branch`、`integration_worktree_path`
   - baseline / ownership：`baseline_main_sha`、`owner_token`、`ownership_state`
   - replay：`planned_features[]`、`replay_status`、`replay_summary`
   - gates：`verify_status`、`review_status`、`simplify_status`、`gate_status`
   - lifecycle：`sync_status`、`cleanup_status`、`synced_main_sha`
   - evidence：`verify_env_name`、`verify_summary`、`review_summary`、`simplify_summary`、`gate_summary`、`sync_summary`、`cleanup_summary`、`verify_evidence_path`、`review_evidence_path`、`gate_evidence_path`
   - lineage：`source_integration_id`
   - timestamps：`created_at`、`updated_at`
6. `planned_features[]` 是对象数组，至少包含：
   - `source_branch`
   - `source_head_sha`
   - `replayed_head_sha`
   - `replay_status`
   - `replay_summary`
7. 派生状态如 `active` / `stale` / `safe_to_cleanup` 是根据 metadata + 当前本地 `main HEAD` + git 现实共同计算出的结果，不必单独持久化。

结束时请输出：
- 当前 metadata 路径
- 字段分组
- 各命令 / 脚本的读写职责
- 如何据此判断 `active` / `stale` / `ready_to_sync` / `safe_to_cleanup`
