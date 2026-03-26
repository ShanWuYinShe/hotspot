---
description: 在主目录中清理已完成使命且可安全回收的 feature 或 integration worktree
argument-hint: [optional targets]
allowed-tools: Read, Glob, Grep, AskUserQuestion, ExitWorktree, Bash(git status:*), Bash(git branch:*), Bash(git rev-parse:*), Bash(git worktree list:*), Bash(git log:*), Bash(git worktree remove:*), Bash(git branch -d:*)
---

请执行一轮 worktree 收尾清理。当前命令的参数为：

$ARGUMENTS

目标不是盲目删除所有 worktree，而是基于当前状态，识别哪些 feature worktree 和 integration worktree 已完成使命、已同步或已废弃，并在遵守安全边界的前提下推进清理，最终让主目录恢复到更干净、可继续下一轮工作的状态。cleanup 只能在主目录侧执行；当前 worktree 不负责回收自己。

执行要求：

1. 盘点当前所有 linked worktree，并标记其角色、dirty 状态、是否已完成、是否状态不明。
2. 对每个 worktree 判断：是否已完成使命、是否已同步回本地 `main`、是否仍有未提交或未合并改动、当前会话是否仍在使用它。
3. 默认只清理当前会话创建或明确接管的 worktree / branch；除非用户明确指定，不要处理其他会话对象。
4. 若当前会话仍驻留在某个通过 Claude worktree session 创建的 worktree 中，必须先执行 `ExitWorktree` 返回主目录，再做 `git worktree remove`；不要在当前 worktree 内直接做最终 cleanup。
5. 仅对本会话中明确满足“已完成、已同步回本地 `main`、无未提交或未合并改动”的对象执行 cleanup；对状态不明或仍有未合并改动的 worktree，明确保留原因。
6. 删除本会话管理的 worktree 时，默认同步删除其对应非主分支 branch；安全回收优先使用 `git worktree remove` 与 `git branch -d`。
7. cleanup 完成后，确认主目录是否仍然是 `main + clean`。
8. 不要 push，不要改写历史；仓库级 workflow 与风险边界以根 `CLAUDE.md` 为准。

结束时请输出：

- 主目录状态
- 全部 worktree 列表
- 每个 worktree 的判断结果：`safe to cleanup`、`keep` 或 `needs confirmation`
- 若已清理，说明清理结果；若未清理，说明原因
- 当前会话是否仍驻留在某个 worktree 中
- 下一步建议：`return to main`、`refresh_integrate`、`start_feature` 或 `nothing to cleanup`
