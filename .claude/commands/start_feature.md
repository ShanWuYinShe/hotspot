---
description: 启动新的 feature 开发流，并在单 worktree 场景下默认完成本地闭环
argument-hint: <task description>
allowed-tools: Read, Glob, Grep, AskUserQuestion, TaskCreate, TaskUpdate, TaskList, EnterWorktree, ExitWorktree, Bash, Edit, Write
---

围绕以下任务启动新的 feature 开发流：

$ARGUMENTS

该命令负责 feature worktree 的完整闭环：从主目录本地 `main` 当前 HEAD 创建 feature worktree，在该 worktree 中完成实现、检查、测试、review、commit。单任务直接进入 `/sync_back_main`；多子任务在 commit 后转入 integration。仓库级 workflow、质量门、风险边界和 commit 规范以根 `CLAUDE.md` 为准。

执行要求：

1. 先检查主目录是否保持在本地 `main` 且无未提交改动；如果不是，先停下来说明原因。
2. 先按任务范围读取根 `CLAUDE.md`、对应子项目 `CLAUDE.md` 与必要的 `spec/`。
3. 分析任务：根据复杂度和文件集决定是否拆分为多个子任务；子任务边界清晰且文件集相对独立时才并行，否则保持单任务。
4. 对非极小任务，优先进入 plan mode；然后立即建立 task list。
5. **必须使用 `EnterWorktree` 工具**从主目录当前最新本地 `main HEAD` 创建新的 feature worktree；`name` 参数必须遵循 `{子项目名}.{模块名}.{功能描述}` 格式（如 `backend-go.auth.add-oauth`），允许的子项目名见根 `CLAUDE.md`；任何 implement 行为开始前都必须已经在该 worktree 中。
6. 在该 feature worktree 中完成 `实现 → check(语法/编译/警告) → test → simplify → strict review → commit` 闭环；任一环节发现问题必须修复后重新循环。最终有效结论以 `deploy/scripts/verify.sh --env-name <name>` 为准；若改动涉及 `harmonyapp/`，还必须额外在非主目录 worktree 中运行 `.claude/scripts/harmony_worktree.py`，并在失败时按错误修复后重跑同一入口，直到通过或明确阻塞。该脚本直接基于当前 worktree 执行，不需要额外 `cd`，统一完成 Harmony 安装、测试与编译闭环。
7. 若改动触达前端展示层，追加 `verify-inspect` 环节，由 Claude Code 主动完成页面级确认。
8. 完成 simplify 和严格 review 后，必须调用 `feature-ready-check` subagent 做最终复核，并在输出中复述其 `## 结论`、`## 阻塞项`、`## 建议下一步`。
9. 若本任务为单 worktree，进入 `/sync_back_main`；若为多子任务汇总，进入 integration。远程同步策略以根 `CLAUDE.md` 与 `/sync_back_main` 为准。

结束时请输出：

- 任务摘要
- feature worktree 路径与分支
- 关键变更范围
- verify 结果
- review 结果
- 本地 commit SHA
- 若为单 worktree：sync-back 结果、ExitWorktree 结果、cleanup 结果
- 最终状态：`synced to local main`、`ready for integration` 或 `blocked`
