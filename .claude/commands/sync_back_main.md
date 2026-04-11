---
description: 执行 rebase + fast-forward 的 sync-back，将 worktree 成果线性同步回主目录本地 main
argument-hint: [optional notes]
allowed-tools: Bash(python3 .claude/scripts/sync_back_main.py *), ExitWorktree, Read, Glob, Grep
---

请将当前已完成的本地结果同步回主目录本地 `main`。当前命令的参数为：

$ARGUMENTS

该命令通过 rebase + fast-forward 路径将 worktree 成果同步到主目录本地 `main`。仓库级 workflow 以根 `CLAUDE.md` 为准。本命令不承担补做 verify / review / gate 的职责；它只消费已存在的 runtime metadata 和当前线性历史。

执行要求：
1. 先确认主目录当前处于本地 `main` 且无未提交改动；如果不是，停止并说明原因。
2. 先确认当前结果属于 `owned` 对象；若对象归属不明，停止并说明原因。
3. 若当前任务是单 feature worktree 结果，确认 feature metadata 已表达 `check/test/simplify/review/verify/commit` 全部满足，且当前历史适合同步。
4. 若当前任务是 integration 结果，确认其 metadata 已表达 `gate_status == ready_to_sync`，且历史已收敛为单个 commit。
5. 若发现来源分支历史不满足同步前提，则必须直接停止并输出 `blocked`；不要自动继续。
6. 调用 `sync_back_main.py` 执行：检查线性历史 → rebase main → fast-forward 主目录 main → 回写 `sync_status` / `synced_main_sha`。
7. sync-back 完成后，若当前会话仍驻留在该 worktree 中，请继续 `ExitWorktree`，并在主目录进入 `/cleanup_worktrees`。
8. 临时文件、验证截图和中间产物默认写入 `tmp/`，不要污染主目录或 worktree 根目录。

结束时请输出：
- 主目录当前 `main` 状态
- 本次同步来源（feature 或 integration）
- ownership 检查结果
- 历史检查结果
- rebase 结果
- fast-forward 结果
- metadata 回写结果
- sync-back 最终结果
- ExitWorktree 结果
- cleanup 建议
- 最终主目录状态：`main + no-uncommitted-changes` 或 `blocked`
