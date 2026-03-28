---
description: 当 integration 基线过期时，基于最新 main HEAD 重建并重新集成
argument-hint: [feature refs or notes]
allowed-tools: Bash(git status:*), Bash(git branch:*), Bash(git rev-parse:*), Bash(git worktree list:*), Bash(git worktree add:*), Bash(git cherry-pick:*), Bash(git log:*), Read, Glob, Grep, AskUserQuestion
---

围绕以下范围刷新当前 integration：

$ARGUMENTS

该命令负责 integration 基线刷新：当旧 integration 基线过期时，基于主目录当前最新本地 `main HEAD` 重新建立 integration worktree，并重新 replay feature 成果。后续验证交给 `/verify_integrate`，最终 gate 判定交给 `/finish_integrate`。仓库级 workflow 以根 `CLAUDE.md` 为准。

执行要求：

1. 确认当前或最近的 integration worktree 是否已经基线过期；如果没有过期，明确说明无需 refresh。
2. 检查主目录是否保持在本地 `main` 且无未提交改动；如果不是，先停下来说明原因。
3. 自动确定本轮 refresh 后要重新纳入的 feature 列表；如范围不清晰，再请求确认。
4. 停止在旧 integration worktree 上继续操作，不要 merge 最新 `main`。
5. 从主目录当前最新本地 `main HEAD` 创建新的临时 integration worktree；必须使用 `git worktree add -b <name> <path> main`（`main` 是本地分支引用，**不是 `origin/main`**）。
6. 通过 `.claude/scripts/integration_meta.py` 刷新 integration 元信息。
7. 重新将本轮 feature 成果按顺序 cherry-pick / replay 到新的 integration worktree；冲突只在新的 integration worktree 中解决。
8. refresh 完成后停止在新的 integration 准备态；远程同步策略以根 `CLAUDE.md` 与 `/sync_back_main` 为准。

结束时请输出：

- 旧 integration 基线 `main HEAD`
- 新 integration 基线 `main HEAD`
- 新 integration worktree 路径与分支
- 本轮重新纳入的 feature 列表
- replay / 冲突处理结果
- metadata 刷新结果
- 刷新后的 `gate_status`：`pending`
- 下一步建议：重新 replay 或 `/verify_integrate`
