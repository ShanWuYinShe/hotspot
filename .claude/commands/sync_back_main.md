---
description: 将当前已完成的本地结果线性同步回主目录本地 main，并在退出 worktree 后于主目录完成 cleanup
argument-hint: [optional notes]
allowed-tools: Bash(git status:*), Bash(git branch:*), Bash(git rev-parse:*), Bash(git log:*), Bash(git diff:*), Bash(git rebase:*), Bash(git merge --ff-only:*), Bash(git worktree list:*), Bash(git worktree remove:*), Bash(git branch -d:*), ExitWorktree, Read, Glob, Grep
---

请将当前已完成的本地结果线性同步回主目录本地 `main`。当前命令的参数为：

$ARGUMENTS

目标不是停在阶段性 readiness 结论，而是在满足仓库规则时，自动完成以下闭环：确认当前结果已可交付、按线性历史同步回主目录本地 `main`、退出当前 worktree，再回到主目录清理本会话中已完成的 worktree / branch。这里的 `main` 始终指主目录的本地 `main` 分支，而不是远程分支。

执行要求：

1. 先确认主目录当前处于本地 `main + clean`；如果不是，停止并说明原因。
2. 先确认当前结果属于本会话创建或明确接管的对象；若对象归属不明，停止并说明原因。
3. 若当前任务是单 worktree 结果，确认该 worktree 已完成 verify、严格 review 与本地 commit，然后优先使用 `rebase` 整理来源分支，再以 fast-forward 方式把主目录本地 `main` 前进到该结果；不要默认使用 `git cherry-pick` 把单 worktree 结果合回主目录。
4. 若当前任务是 integration 结果，确认其 gate 已达到 `ready to sync`，且历史已通过 `rebase + squash` 或等价方式收敛为单个 commit，再按线性历史同步回主目录本地 `main`。
5. 同步过程中只允许使用线性方式；不要引入 merge commit，不要 push。
6. sync-back 完成后，若当前会话仍驻留在 Claude worktree session 创建的 worktree 中，必须先执行 `ExitWorktree` 返回主目录。
7. cleanup 只能在主目录执行；返回主目录后，对本会话中已完成、已同步、无未确认改动的 worktree / branch 执行 cleanup。
8. 任何需要丢弃未确认改动、涉及其他会话对象或需要历史改写的情况，都不要自动继续。

结束时请输出：

- 主目录当前 `main` 状态
- 本次同步来源（single-worktree 或 integration）
- 线性同步方式
- sync-back 结果
- ExitWorktree 结果
- cleanup 结果
- 最终主目录状态：`main + clean` 或 `blocked`
