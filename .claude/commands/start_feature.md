---
description: 启动新的 feature 开发流并推进到 ready for integration
argument-hint: <task description>
allowed-tools: Read, Glob, Grep, AskUserQuestion, TaskCreate, TaskUpdate, TaskList, EnterWorktree, Bash(git status:*), Bash(git branch:*), Bash(git rev-parse:*), Bash(git worktree list:*), Bash(git log:*), Bash(*deploy/scripts/verify.sh*)
---

围绕以下任务启动新的 feature 开发流：

$ARGUMENTS

目标不是直接合并回主目录 `main`，而是先把当前任务推进到 **ready for integration**；若本任务并未拆成多 agent + 多 worktree 并行开发，则后续不需要额外新建 integration worktree。

执行要求：

1. 先自动检查最近一轮已完成成果是否已经回到本地 `main`；只有仓库状态歧义时才请求确认。
2. 检查主目录是否保持本地 `main + clean`；如果不是，先停下来说明原因，不要继续实现。
3. 先按任务范围读取根 `CLAUDE.md`、对应子项目 `CLAUDE.md` 与必要的 `spec/`；仓库级 workflow 与风险边界以根 `CLAUDE.md` 为准。
4. 对非极小任务，优先进入 plan mode；然后立即建立 task list。
5. 创建新的 feature worktree。
6. 在该 feature worktree 中完成实现、验证、review 与 commit。
7. 运行标准验证；最终有效结论以 `deploy/scripts/verify.sh --env-name <name>` 为准。
8. 完成 simplify 和严格 review 后，必须调用 `feature-ready-check` subagent 做最终 readiness 复核，并在输出中复述其 `## 结论`、`## 阻塞项`、`## 建议下一步`。
9. 当任务真正完成且 `feature-ready-check` 结论为 `ready for integration` 时，再创建本地 commit。
10. 若本任务并未拆成多 agent + 多 worktree 并行开发，则后续应默认按线性历史直接同步回主目录本地 `main` 并 cleanup；若需要单独入口，可复用 `/sync_back_main` 的同名规则；若属于多 worktree 汇总任务，则进入 integration；不要 push。

结束时请输出：

- 任务摘要
- feature worktree 路径与分支
- 关键变更范围
- verify 结果
- review 结果
- 本地 commit SHA
- 最终状态：`ready for integration` 或 `blocked`
