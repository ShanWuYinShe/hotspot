---
description: 检查当前 integration 是否已满足线性同步回主目录 main 的条件
argument-hint: [optional notes]
allowed-tools: Bash(git status:*), Bash(git branch:*), Bash(git rev-parse:*), Bash(git log:*), Bash(git diff:*), Bash(git worktree list:*), Bash(git worktree remove:*), Bash(git branch -d:*), ExitWorktree, Read, Glob, Grep
---

请对当前 integration worktree 做最后的完成检查。当前命令的参数为：

$ARGUMENTS

目标不是停在阶段性结论，而是确认这轮 integration 是否已经满足“可线性同步回主目录本地 `main`”的全部条件。面向人类的 reviewer 结论使用 `ready to sync`、`blocked`、`must refresh`；回写 runtime metadata 时必须分别映射为 `ready_to_sync`、`blocked`、`must_refresh`。

执行要求：

1. 先确认当前位于 integration worktree；如果不是，停止并说明原因。
2. 检查当前 integration 基线是否仍然有效；若主目录 `main` 已前进，则输出 `must refresh`，并提示改走 `/refresh_integrate`。
3. 检查 integration verify 是否已经完成并通过；不要把 feature worktree 的局部通过当作 integration verify 通过。
4. 检查 simplify、严格 review 与 review 问题修复是否已经完成。
5. 检查当前提交历史是否已通过 `rebase + squash` 或等价线性整理方式收敛为适合回主目录本地 `main` 的单个 commit，避免 merge commit。
6. 读取仓库根 `tmp/claude/integrations/<integration-id>.json`，同步校验 `baseline_main_sha`、`verify_status`、`review_status`、`gate_status`。
7. 必须调用 `integration-review-gate` subagent 复核当前 gate 结论，并在最终输出中复述其 `## 结论`、`## Baseline`、`## 阻塞项`、`## 建议下一步`。
8. 通过共享脚本 `.claude/scripts/integration_meta.py` 将最终 gate 结果回写到 integration 元信息中的 `gate_status`、`review_status`、`gate_summary`、`review_summary` 与 `updated_at`。
9. 若结论为 `ready to sync`，则下一步应进入 `/sync_back_main` 按线性方式同步回主目录本地 `main`；sync-back 完成后先 `ExitWorktree`，再回到主目录 cleanup。只有在基线过期、对象归属不明或存在未确认丢弃动作时，才停下来说明阻塞项；不要 push。

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
