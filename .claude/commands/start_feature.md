---
description: 启动新的 feature 开发流并推进到 ready for integration
argument-hint: <task description>
---

围绕以下任务启动新的 feature 开发流：

$ARGUMENTS

目标不是直接合并回主目录 `main`，而是把当前任务推进到 **ready for integration**。

执行要求：

1. 先检查最近一轮已完成成果是否已经回到本地 `main`；如果不确定，先问用户。
2. 检查主目录是否保持本地 `main + clean`；如果不是，先停下来说明原因，不要继续实现。
3. 先读根 `CLAUDE.md`，再按任务范围读对应子项目 `CLAUDE.md`；如涉及接口、字段、错误码、分页、幂等或 WebSocket，先读 `spec/`，但不要默认修改 `spec/`。
4. 对非极小任务，优先进入 plan mode；然后立即建立 task list。
5. 创建新的 feature worktree；后续所有实现、验证、review、commit 都在该 worktree 中完成，不要在主目录写代码。
6. 在 feature worktree 中完成实现，避免无关重构、额外环境模式抽象和与任务无关的复杂化。
7. 运行标准验证；最终有效结论以 `deploy/scripts/verify.sh --env-name <name>` 为准，不要把宿主机临时命令结果当成最终结论。
8. 完成 simplify 和严格 review；必要时可使用 `feature-ready-check` subagent 复核当前 feature 是否达到进入 integration 的质量门。
9. 当任务真正完成且 review 通过后，创建本地 commit。
10. 不要默认进入 integration，不要同步回 `main`，不要 push。

结束时请输出：

- 任务摘要
- feature worktree 路径与分支
- 关键变更范围
- verify 结果
- review 结果
- 本地 commit SHA
- 最终状态：`ready for integration` 或 `blocked`
