---
description: 启动一轮本地 integration 流程并汇总 ready feature
argument-hint: [feature refs or notes]
allowed-tools: Bash(git status:*), Bash(git branch:*), Bash(git rev-parse:*), Bash(git worktree list:*), Bash(git worktree add:*), Bash(git cherry-pick:*), Bash(git log:*), Read, Glob, Grep, AskUserQuestion
---

围绕以下范围启动一轮新的本地 integration 流程：

$ARGUMENTS

目标不是直接 merge，而是仅在“同一任务已拆分成多 agent + 多 worktree 并行开发”的前提下，从主目录最新本地 `main HEAD` 新建一个临时 integration worktree，把本次待集成的 feature 成果集中进去，完成冲突解决、验证、review 和线性化准备；初始化完成后，本轮 integration runtime metadata 应处于 `pending`，后续再推进到 `ready_to_sync`、`blocked` 或 `must_refresh`。

执行要求：

1. 先自动检查最近一轮已完成成果是否已经回到本地 `main`；只有状态歧义时才请求确认。
2. 检查主目录是否保持本地 `main + clean`；如果不是，先停下来说明原因。
3. 先自动确认本次任务是否确实属于“多 agent + 多 worktree 并行开发后的汇总”；若不是，应直接在线性历史上完成，不要进入 integration。
4. 自动确定本次待集成的 feature 列表；若存在多个候选或范围歧义，再请求确认，不要默默扩大范围。
5. 在列出本轮纳入项与进入 replay 前，先调用 `local-integration-playbook` subagent 校准当前阶段、边界、风险点与推荐命令，并在输出中复述其 `## 当前阶段`、`## 风险点`、`## 推荐命令`。
6. 从主目录当前最新本地 `main HEAD` 创建新的临时 integration worktree。
7. 通过共享脚本 `.claude/scripts/integration_meta.py` 初始化本轮 integration 元信息，并写入仓库根 `tmp/claude/integrations/<integration-id>.json`；至少记录：`integration_id`、`baseline_main_sha`、`integration_branch`、`integration_worktree_path`、`planned_features`、`verify_status`、`review_status`、`gate_status`、`created_at`、`updated_at`。
8. 将各 ready feature 的成果按顺序 cherry-pick / replay 到 integration worktree。
9. 在 integration worktree 中运行集成验证，最终有效结论以 `deploy/scripts/verify.sh --env-name <name>` 为准。
10. 完成 simplify、严格 review 和必要的修复；若基线过期则改走 `/refresh_integrate`。
11. 通过 `rebase + squash` 或等价线性整理方式把多处结果收敛为一个 commit。
12. 当结果满足规则时，应继续同步回主目录本地 `main` 并 cleanup；只有基线过期、对象归属不明或存在未确认丢弃动作时才停止；不要 push。

结束时请输出：

- integration 基于的 `main HEAD`
- integration worktree 路径与分支
- 纳入的 feature 列表
- replay / 冲突处理结果
- verify 结果
- review 结果
- 线性化后的提交状态
- metadata 初始化结果
- 初始 `gate_status`：`pending`
- 下一步建议：`/verify_integrate` 或继续 replay / simplify / strict review
