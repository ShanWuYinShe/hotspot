---
description: 检查当前 integration 是否已满足线性同步回主目录 main 的条件
argument-hint: [optional notes]
allowed-tools: Bash(git status:*), Bash(git branch:*), Bash(git rev-parse:*), Bash(git log:*), Bash(git diff:*), Bash(git worktree list:*), Bash(git worktree remove:*), Bash(git branch -d:*), ExitWorktree, Read, Glob, Grep
---

请对当前 integration worktree 做最后的完成检查。当前命令的参数为：

$ARGUMENTS

确认这轮 integration 是否已经满足"可线性同步回主目录本地 `main`"的全部条件。仓库级 workflow 以根 `CLAUDE.md` 为准。面向人类的结论使用 `ready to sync`、`blocked`、`must refresh`；回写 metadata 时映射为 `ready_to_sync`、`blocked`、`must_refresh`。

执行要求：

1. 确认当前位于 integration worktree；如果不是，停止并说明原因。
2. 检查基线是否仍然有效；若主目录 `main` 已前进，输出 `must refresh`，提示改走 `/refresh_integrate`。
3. 检查 integration verify 是否已完成并通过。
4. 检查 simplify、严格 review 与 review 问题修复是否已完成。
5. 检查提交历史是否已通过 `rebase + squash` 收敛为单个 commit。
6. 必须调用 `integration-review-gate` subagent 复核当前 gate 结论，并在输出中复述其 `## 结论`、`## Baseline`、`## 阻塞项`、`## 建议下一步`。
7. 通过 `.claude/scripts/integration_meta.py` 将最终 gate 结果回写到元信息。
8. 若结论为 `ready to sync`，下一步进入 `/sync_back_main`。远程同步策略以根 `CLAUDE.md` 与 `/sync_back_main` 为准。

结束时请输出：

- integration worktree 路径与分支
- 当前 integration 基线 `main HEAD`
- 主目录当前最新本地 `main HEAD`
- integration verify 状态
- simplify 状态
- strict review 状态
- 历史线性化状态
- 当前结论：`ready to sync`、`blocked` 或 `must refresh`
- 若为 `ready to sync`，说明下一步应进入 `/sync_back_main`
- 若不是 `ready to sync`，明确列出阻塞项
