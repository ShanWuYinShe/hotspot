---
description: 启动一轮本地 integration 流程并汇总 ready feature
argument-hint: [feature refs or notes]
allowed-tools: Bash(git status:*), Bash(git branch:*), Bash(git rev-parse:*), Bash(git worktree list:*), Bash(git worktree add:*), Bash(git cherry-pick:*), Bash(git log:*), Read, Glob, Grep, AskUserQuestion
---

围绕以下范围启动一轮新的本地 integration 流程：

$ARGUMENTS

该命令负责 integration 准备态：在"多子任务并行开发"的前提下，从主目录最新本地 `main HEAD` 新建临时 integration worktree，把待集成的 feature 成果 replay 进去，并初始化本轮 integration metadata。后续验证交给 `/verify_integrate`，最终 gate 判定交给 `/finish_integrate`。仓库级 workflow 以根 `CLAUDE.md` 为准。

执行要求：

1. 检查主目录是否保持本地 `main + clean`；如果不是，先停下来说明原因。
2. 确认本次任务是否确实属于”多子任务并行开发后的汇总”；若不是，应直接在线性历史上完成，不要进入 integration。
3. 自动确定本次待集成的 feature 列表；若存在多个候选或范围歧义，再请求确认。
4. 在列出本轮纳入项与进入 replay 前，先调用 `local-integration-playbook` subagent 校准当前阶段与风险点，并在输出中复述其 `## 当前阶段`、`## 风险点`、`## 推荐命令`。
5. 从主目录当前最新本地 `main HEAD` 创建新的临时 integration worktree；必须使用 `git worktree add -b <name> <path> main`（`main` 是本地分支引用，**不是 `origin/main`**）。
6. 通过 `.claude/scripts/integration_meta.py` 初始化本轮 integration 元信息。
7. 将各 ready feature 的成果按顺序 cherry-pick / replay 到 integration worktree。
8. 完成 replay 与冲突处理后停止在 integration 准备态；不要 push。

结束时请输出：

- integration 基于的 `main HEAD`
- integration worktree 路径与分支
- 纳入的 feature 列表
- replay / 冲突处理结果
- metadata 初始化结果
- 初始 `gate_status`：`pending`
- 下一步建议：`/verify_integrate` 或继续 replay
