---
description: 执行 sync-back 前的线性化检查与前置整理，并给出后续收尾建议
argument-hint: [optional notes]
allowed-tools: Bash(python3 .claude/scripts/sync_back_main.py *), ExitWorktree, Read, Glob, Grep
---

请将当前已完成的本地结果线性同步回主目录本地 `main`。当前命令的参数为：

$ARGUMENTS

该命令负责 sync-back 前的线性化检查与前置整理：确认当前结果是否已经满足进入后续主目录前进、ExitWorktree 与 cleanup 的前提。这里的 `main` 始终指主目录的本地 `main` 分支。本命令不承担补做 verify / review / gate 的职责，也不在当前实现中直接完成主目录前进与收尾删除动作。仓库级 workflow 以根 `CLAUDE.md` 为准。

执行要求：

1. 先确认主目录当前处于本地 `main` 且无未提交改动；如果不是，停止并说明原因。
2. 先确认当前结果属于本会话创建或明确接管的对象；若对象归属不明，停止并说明原因。
3. 若当前任务是单 worktree 结果，确认该 worktree 已完成 verify、严格 review 与本地 commit，并已整理为满足当前线性同步规则的结果；若状态不满足要求，必须先停止 sync-back，回到来源 worktree 继续整理历史。
4. 若当前任务是 integration 结果，确认其 gate 已达到 `ready to sync`，且历史已收敛为满足当前线性同步规则的结果；若 integration 当前仍未收敛或 baseline 已过期，则必须先 replay / rebase / squash / refresh-rebuild，把状态整理到适合同步的程度。
5. 若发现来源分支或 integration 当前历史不满足同步前提，则必须直接停止并输出 `blocked`，说明需要先调整状态；不要自动继续。
6. 线性化检查与前置整理虽属于 sync-back 闭环的一部分，但执行时必须分阶段推进：每一步先观察结果，再进入下一步。
7. 若线性化与前置整理通过，后续主目录前进、`ExitWorktree` 与 cleanup 仍应作为后续步骤单独推进。
8. 任何需要丢弃未确认改动、涉及其他会话对象或需要历史改写的情况，都不要自动继续。

结束时请输出：

- 主目录当前 `main` 状态
- 本次同步来源（single-worktree 或 integration）
- rebase-only 校验结果
- 线性同步方式
- sync-back 结果
- ExitWorktree 结果
- cleanup 结果
- 最终主目录状态：`main + no-uncommitted-changes` 或 `blocked`
