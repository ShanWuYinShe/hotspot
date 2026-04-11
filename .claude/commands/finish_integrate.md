---
description: 检查当前 integration 是否已满足线性同步回主目录 main 的条件
argument-hint: [optional notes]
allowed-tools: Bash(python3 .claude/scripts/finish_integrate.py *), Read, Glob, Grep
---

请对当前 integration worktree 做最后的完成检查。该命令只负责最终 gate 判定与 metadata 回写，不负责新增 replay、大规模修复或临时重写流程。当前命令的参数为：

$ARGUMENTS

确认这轮 integration 是否已经满足“可线性同步回主目录本地 `main`”的全部条件。仓库级 workflow 以根 `CLAUDE.md` 为准。面向人类的结论使用 `ready to sync`、`blocked`、`must refresh`；回写 metadata 时映射为 `ready_to_sync`、`blocked`、`must_refresh`。

执行要求：
1. 确认当前位于 integration worktree；如果不是，停止并说明原因。
2. 读取 integration metadata，检查 baseline 是否仍然有效；若主目录 `main` 已前进，输出 `must refresh`。
3. 检查 `planned_features` 的 replay 是否全部完成。
4. 检查 integration verify、simplify、strict review 是否都已完成并通过。
5. 检查提交历史是否已通过 `rebase + squash` 收敛为单个 commit，且当前 integration 不含 merge commit。
6. 通过 `.claude/scripts/integration_meta.py` 将最终 gate 结果回写到元信息。
7. 只有当 gate 明确回写为 `ready_to_sync` 时，下一步才允许进入 `/sync_back_main`。
8. 若有调试输出、截图或中间分析文件，默认写入 `tmp/`。

结束时请输出：
- integration worktree 路径与分支
- 当前 integration 基线 `main HEAD`
- 主目录当前最新本地 `main HEAD`
- replay 完整性状态
- integration verify 状态
- simplify 状态
- strict review 状态
- 历史线性化状态
- 当前结论：`ready to sync`、`blocked` 或 `must refresh`
- 若不是 `ready to sync`，明确列出阻塞项
