---
description: 在主目录中清理已完成使命且可安全回收的 feature 或 integration worktree
argument-hint: [optional targets]
allowed-tools: Read, Glob, Grep, AskUserQuestion, ExitWorktree, Bash(python3 .claude/scripts/cleanup_worktrees.py *)
---

请执行一轮 worktree 收尾清理。当前命令的参数为：

$ARGUMENTS

该命令只负责主目录 cleanup：识别哪些 feature worktree 和 integration worktree 已完成使命、已同步或已废弃，并在遵守安全边界的前提下推进清理，最终让主目录恢复到更干净、可继续下一轮工作的状态。当前 worktree 不负责回收自己。默认按对象逐步处理，不要把多个 git 删除动作用 `&&`、`;` 或 `||` 串在同一条 Bash 中。

执行要求：
1. 盘点当前所有 linked worktree，并标记其角色、dirty 状态、ownership、是否已 sync-back、是否状态不明。
2. 对每个 worktree 判断：是否 `owned`、metadata 是否存在、`sync_status` 是否已 `synced`、branch tip 是否已等于本地 `main`、当前 worktree 是否干净、是否属于 superseded lineage。
3. 默认只清理 `owned` 且 `safe_to_cleanup` 的对象；除非用户明确指定，不要处理其他会话对象。
4. 若当前会话仍驻留在某个通过 Claude worktree session 创建的 worktree 中，必须先执行 `ExitWorktree` 返回主目录，再做 cleanup。
5. 删除本会话管理的 worktree 时，默认同步删除其对应非主分支 branch，但必须按对象分步执行：先单独判断并执行 `git worktree remove`，重新观察结果后，再决定是否单独执行分支删除。
6. cleanup 成功后，要同步清理对应 runtime metadata；blocked / keep 的对象只更新 metadata 状态，不要强删。
7. cleanup 完成后，确认主目录是否仍然在本地 `main` 且无未提交改动。
8. 若需要保存盘点结果或调试输出，默认写入 `tmp/`。

结束时请输出：
- 主目录状态
- 全部 worktree 列表
- 每个 worktree 的判断结果：`safe to cleanup`、`keep` 或 `needs confirmation`
- 若已清理，说明清理结果；若未清理，说明原因
- 当前会话是否仍驻留在某个 worktree 中
- 下一步建议：`return to main`、`refresh_integrate`、`start_feature` 或 `nothing to cleanup`
