---
description: 启动新的 feature 开发流，并在单 worktree 场景下默认完成本地闭环
argument-hint: <task description>
allowed-tools: Read, Glob, Grep, AskUserQuestion, TaskCreate, TaskUpdate, TaskList, EnterWorktree, ExitWorktree, Bash, Edit, Write
---

围绕以下任务启动新的 feature 开发流：

$ARGUMENTS

该命令负责单 worktree happy path：从主目录本地 `main` 当前 HEAD 创建 feature worktree，在该 worktree 中完成实现、验证、review、commit；若任务未拆成多 worktree，则后续直接进入 `/sync_back_main` 完成本地闭环。若任务确实属于多 worktree 汇总，则在完成本地 commit 后转入 integration。

执行要求：

1. 先自动检查最近一轮已完成成果是否已经回到本地 `main`；只有仓库状态歧义时才请求确认。
2. 检查主目录是否保持本地 `main + clean`；如果不是，先停下来说明原因，不要继续实现。
3. 先按任务范围读取根 `CLAUDE.md`、对应子项目 `CLAUDE.md` 与必要的 `spec/`；仓库级 workflow 与风险边界以根 `CLAUDE.md` 为准。
4. 对非极小任务，优先进入 plan mode；然后立即建立 task list。
5. **必须使用 `EnterWorktree` 工具**从主目录当前最新本地 `main HEAD` 创建新的 feature worktree（不要使用 `git worktree add`，不要使用 `origin/main` 作为基准点）；任何 implement 行为开始前都必须已经在该 worktree 中，主目录 `main` 只负责控制面。
6. 在该 feature worktree 中完成实现、标准 verify、simplify、严格 review 与本地 commit；最终有效结论以 `deploy/scripts/verify.sh --env-name <name>` 为准。
7. 若改动触达前端页面、组件、路由、layout、store、hook 或样式等展示层，则在开发内环追加 `deploy/scripts/verify-inspect.sh --env-name <name>`，由 `deploy/` 提供隔离环境与稳定入口，再由 Claude Code 主动访问页面、查看 console、核对关键 UI 与少量关键交互流；需要候选级入口预演时再进入 preview。
8. 完成 simplify 和严格 review 后，必须调用 `feature-ready-check` subagent 做最终复核，并在输出中复述其 `## 结论`、`## 阻塞项`、`## 建议下一步`。
9. 若本任务并未拆成多 agent + 多 worktree 并行开发，则下一步进入 `/sync_back_main`；若属于多 worktree 汇总任务，则进入 integration；不要 push。

结束时请输出：

- 任务摘要
- feature worktree 路径与分支
- 关键变更范围
- verify 结果
- review 结果
- 本地 commit SHA
- 若为单 worktree：sync-back 结果、ExitWorktree 结果、cleanup 结果
- 最终状态：`synced to local main`、`ready for integration` 或 `blocked`
