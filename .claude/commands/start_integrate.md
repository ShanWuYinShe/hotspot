---
description: 启动一轮本地 integration 流程并汇总 ready feature
argument-hint: [feature refs or notes]
allowed-tools: Bash(git status:*), Bash(git branch:*), Bash(git rev-parse:*), Bash(git worktree list:*), Bash(git worktree add:*), Bash(git cherry-pick:*), Bash(git log:*), Read, Glob, Grep, AskUserQuestion
---

围绕以下范围启动一轮新的本地 integration 流程：

$ARGUMENTS

该命令只负责 integration 准备态：在“同一任务已拆分成多 agent + 多 worktree 并行开发”的前提下，从主目录最新本地 `main HEAD` 新建一个临时 integration worktree，把待集成的 feature 成果 replay 进去，并初始化本轮 integration metadata。后续验证交给 `/verify_integrate`，最终 gate 判定交给 `/finish_integrate`。

执行要求：

1. 先自动检查最近一轮已完成成果是否已经回到本地 `main`；只有状态歧义时才请求确认。
2. 检查主目录是否保持本地 `main + clean`；如果不是，先停下来说明原因。
3. 先自动确认本次任务是否确实属于“多 agent + 多 worktree 并行开发后的汇总”；若不是，应直接在线性历史上完成，不要进入 integration。
4. 自动确定本次待集成的 feature 列表；若存在多个候选或范围歧义，再请求确认，不要默默扩大范围。
5. 在列出本轮纳入项与进入 replay 前，先调用 `local-integration-playbook` subagent 校准当前阶段、边界、风险点与推荐命令，并在输出中复述其 `## 当前阶段`、`## 风险点`、`## 推荐命令`。
6. 从主目录当前最新本地 `main HEAD` 创建新的临时 integration worktree。
7. 通过共享脚本 `.claude/scripts/integration_meta.py` 初始化本轮 integration 元信息，并写入仓库根 `tmp/claude/integrations/<integration-id>.json`；至少记录：`integration_id`、`baseline_main_sha`、`integration_branch`、`integration_worktree_path`、`planned_features`、`verify_status`、`review_status`、`gate_status`、`created_at`、`updated_at`。
8. 将各 ready feature 的成果按顺序 cherry-pick / replay 到 integration worktree；`git cherry-pick` 仅用于 integration replay。
9. 完成 replay 与冲突处理后停止在 integration 准备态；不要 push。

结束时请输出：

- integration 基于的 `main HEAD`
- integration worktree 路径与分支
- 纳入的 feature 列表
- replay / 冲突处理结果
- metadata 初始化结果
- 初始 `gate_status`：`pending`
- 下一步建议：`/verify_integrate` 或继续 replay
