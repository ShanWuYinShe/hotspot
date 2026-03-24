---
description: 启动一轮本地 integration 流程并汇总 ready feature
argument-hint: [feature refs or notes]
---

围绕以下范围启动一轮新的本地 integration 流程：

$ARGUMENTS

目标不是直接 merge，而是从主目录最新本地 `main HEAD` 新建一个临时 integration worktree，把本次待集成的 feature 成果集中进去，完成冲突解决、验证、review 和线性化准备，最终达到 **integration ready**。

执行要求：

1. 先检查最近一轮已完成成果是否已经回到本地 `main`；如果不确定，先问用户。
2. 检查主目录是否保持本地 `main + clean`；如果不是，先停下来说明原因。
3. 确认本次待集成的 feature 列表；如用户未明确指定，先列出候选项并请求确认，不要默默扩大范围。
4. 优先参考 `local-integration-playbook` subagent 的流程边界；需要判断当前 integration 是否可继续时，可使用 `integration-review-gate` subagent 做复核。
5. 从主目录当前最新本地 `main HEAD` 创建新的临时 integration worktree；不要复用旧 integration worktree，不要在主目录做集成修改。
6. 将各 ready feature 的成果按顺序 cherry-pick / replay 到 integration worktree；不要使用 merge。冲突只在 integration worktree 中解决。
7. 在 integration worktree 中运行集成验证，最终有效结论以 `deploy/scripts/verify.sh --env-name <name>` 为准。
8. 完成 simplify、严格 review 和必要的修复；如主目录 `main` 在集成过程中前进，则当前 integration 基线视为过期，应改走 `/refresh_integrate`，不要继续在旧 integration 上推进。
9. 通过 `rebase + squash` 或等价线性整理方式收敛提交历史，避免 merge commit。
10. 不要默认同步回 `main`，不要 push，完成后用 `/finish_integrate` 给出最终 gate 结论。

结束时请输出：

- integration 基于的 `main HEAD`
- integration worktree 路径与分支
- 纳入的 feature 列表
- replay / 冲突处理结果
- verify 结果
- review 结果
- 线性化后的提交状态
- 最终状态：`integration ready`、`blocked` 或 `must refresh`
