---
description: 当 integration 基线过期时，基于最新 main HEAD 重建并重新集成
argument-hint: [feature refs or notes]
allowed-tools: Bash(git status:*), Bash(git branch:*), Bash(git rev-parse:*), Bash(git worktree list:*), Bash(git worktree add:*), Bash(git cherry-pick:*), Bash(git log:*), Read, Glob, Grep, AskUserQuestion
---

围绕以下范围刷新当前 integration：

$ARGUMENTS

目标不是在旧 integration worktree 上继续修补，也不是 merge 最新 `main`，而是仅在当前任务确实属于多 agent + 多 worktree 并行开发汇总时，基于主目录当前最新本地 `main HEAD` 重新建立 integration 基线，并重新 replay 本次待集成的 feature 成果。

执行要求：

1. 先确认当前或最近的 integration worktree 是否已经基线过期；如果没有过期，明确说明无需 refresh。
2. 先自动确认本次任务仍然属于多 agent + 多 worktree 并行开发后的汇总；只有范围或归属歧义时才请求确认。
3. 检查主目录是否保持本地 `main + clean`；如果不是，先停下来说明原因。
4. 自动确定本轮 refresh 后要重新纳入的 feature 列表；如存在多个候选或范围不清晰，再请求确认。
5. 停止继续在旧 integration worktree 上 edit / verify / review / commit，不要 merge 最新 `main`。
6. 从主目录当前最新本地 `main HEAD` 创建新的临时 integration worktree。
7. 通过共享脚本 `.claude/scripts/integration_meta.py` 重建或更新仓库根 `tmp/claude/integrations/<integration-id>.json`，刷新 `baseline_main_sha`、`integration_branch`、`integration_worktree_path`、`planned_features`、`source_integration_id` 与 `updated_at`。
8. 重新将本轮 feature 成果按顺序 cherry-pick / replay 到新的 integration worktree；冲突只在新的 integration worktree 中解决。
9. 重新运行 integration verify，完成 simplify、严格 review 和必要修复，并通过 `rebase + squash` 或等价线性整理方式把结果收敛为一个 commit。
10. 当结果满足规则时，应继续同步回主目录本地 `main` 并 cleanup；只有基线过期、对象归属不明或存在未确认丢弃动作时才停止；不要 push。

结束时请输出：

- 旧 integration 基线 `main HEAD`
- 新 integration 基线 `main HEAD`
- 新 integration worktree 路径与分支
- 本轮重新纳入的 feature 列表
- replay / 冲突处理结果
- verify 结果
- review 结果
- metadata 刷新结果
- 刷新后的 `gate_status`：`pending`
- 下一步建议：重新 replay、`/verify_integrate`、strict review
