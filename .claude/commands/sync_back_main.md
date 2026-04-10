---
description: 执行 rebase + fast-forward 的 sync-back，将 worktree 成果线性同步回主目录本地 main
argument-hint: [optional notes]
allowed-tools: Bash(python3 .claude/scripts/sync_back_main.py *), ExitWorktree, Read, Glob, Grep
---

请将当前已完成的本地结果同步回主目录本地 `main`。当前命令的参数为：

$ARGUMENTS

该命令通过 rebase + fast-forward 路径将 worktree 成果同步到主目录本地 `main`。仓库级 workflow 以根 `CLAUDE.md` 为准。本命令不承担补做 verify / review / gate 的职责。

执行要求：

1. 先确认主目录当前处于本地 `main` 且无未提交改动；如果不是，停止并说明原因。
2. 先确认当前结果属于本会话创建或明确接管的对象；若对象归属不明，停止并说明原因。
3. 若当前任务是单 feature worktree 结果，确认该 worktree 已完成 verify、严格 review 与本地 commit，并已整理为适合同步的线性历史；若状态不满足要求，必须先停止 sync-back，回到来源 worktree 继续整理。
4. 若当前任务是 integration 结果，确认其 gate 已达到 `ready to sync`，且历史已收敛为单个 commit；若 integration 当前仍未收敛或 baseline 已过期，则必须先整理到适合同步的程度。
5. 若发现来源分支历史不满足同步前提，则必须直接停止并输出 `blocked`；不要自动继续。
6. 调用 `sync_back_main.py` 执行：检查线性历史 → rebase main → fast-forward 主目录 main。
7. feature 分支允许在开发期间保留多个 commit；integration 在 sync-back 前必须是单 commit。无论哪种来源，都不得包含 merge commit。
8. sync-back 完成后，若当前会话仍驻留在该 worktree 中，请继续 `ExitWorktree`，并在主目录进入 `/cleanup_worktrees`。
9. 临时文件、验证截图和中间产物默认写入 `tmp/`，不要污染主目录或 worktree 根目录。

结束时请输出：

- 主目录当前 `main` 状态
- 本次同步来源（feature 或 integration）
- 历史检查结果
- rebase 结果
- fast-forward 结果
- sync-back 最终结果
- ExitWorktree 结果
- cleanup 建议
- 最终主目录状态：`main + no-uncommitted-changes` 或 `blocked`
