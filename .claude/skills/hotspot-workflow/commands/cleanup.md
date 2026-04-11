---
description: 清理无用的 git worktree 和分支
argument-hint: [可选过滤条件]
---

请盘点当前仓库的所有 `git worktree`，识别并清理已合并或废弃的临时工作区。

## 执行步骤
1. **列出所有 worktree**：`git worktree list`
2. **确认主目录状态**：位于 `main` 分支且无未提交改动。
3. **识别可安全删除的分支**：`git branch --merged main`，筛选出你创建的临时 feature 分支（如 `backend-go/`、`frontend/`、`harmony/` 前缀）。
4. **关联 worktree**：为每个可删除分支检查是否存在对应 worktree。
5. **汇报清理清单**：列出建议删除的 worktree 路径及分支，说明原因（已合并、无未提交改动）。状态不明的标记为“需人工确认”。
6. **等待确认后执行**：
   - 删除 worktree：`git worktree remove <路径>`
   - 删除分支：`git branch -D <分支名>`

## 安全边界
- 主目录 worktree 和当前正在使用的 worktree 不在清理范围。
- 非你创建或名称不符合临时分支规范的需特别提示。

开始盘点。
