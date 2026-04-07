---
description: 启动新的 feature 开发流，并在单 worktree 场景下默认完成本地闭环
argument-hint: <task description>
allowed-tools: Read, Glob, Grep, AskUserQuestion, TaskCreate, TaskUpdate, TaskList, EnterWorktree, ExitWorktree, Bash(python3 .claude/scripts/rust_dev.py *), Bash(python3 .claude/scripts/go_dev.py *), Bash(python3 .claude/scripts/frontend_dev.py *), Bash(python3 .claude/scripts/harmony_worktree.py *), Bash(./deploy/scripts/verify.sh *), Bash(./deploy/scripts/verify-inspect.sh *)
---

围绕以下任务启动新的 feature 开发流：

$ARGUMENTS

该命令负责 feature worktree 的完整闭环编排：从主目录本地 `main` 当前 HEAD 创建 feature worktree，在该 worktree 中完成实现、检查、测试、simplify、strict review、commit，并根据任务形态决定下一步进入 `/sync_back_main` 还是 integration。仓库级 workflow、质量门、风险边界和 commit 规范以根 `CLAUDE.md` 为准。

执行要求：

1. 先检查主目录是否保持在本地 `main` 且无未提交改动；如果不是，先停下来说明原因。
2. 先按任务范围读取根 `CLAUDE.md`、对应子项目 `CLAUDE.md` 与必要的 `spec/`。
3. 分析任务：根据复杂度和文件集决定是否拆分为多个子任务；子任务边界清晰且文件集相对独立时才并行，否则保持单任务。若发现新增需求已明显偏离当前 feature worktree 建立时的原始目标，默认不要继续混在当前 worktree 中实现，而应停止并建议新开 feature worktree。
4. 对非极小任务，优先进入 plan mode；然后立即建立 task list。
5. **必须使用 `EnterWorktree` 工具**从主目录创建 feature worktree；`name` 参数必须遵循 `{子项目名}.{模块名}.{功能描述}` 格式（如 `backend-go.auth.add-oauth`），允许的子项目名见根 `CLAUDE.md`。该三段名表示 feature 的逻辑名：若不存在遗留同 lineage 实例，则首次创建基于当前最新本地 `main HEAD`；若已存在遗留同 lineage feature worktree / branch，则必须创建新的 `-rN` 派生实例，并基于该 lineage 最新 branch 继续之前进度，而不是重新进入旧目录。任何 implement 行为开始前都必须已经在实际创建出的 worktree 中。
6. 在该 feature worktree 中完成 `实现 → check(语法/编译/警告) → test → simplify → strict review → commit` 闭环；任一环节发现问题必须修复后重新循环。check / build 环节通过受控脚本执行：Rust 后端用 `python3 .claude/scripts/rust_dev.py --task <fmt|check|clippy|build|all>`，Go 后端用 `python3 .claude/scripts/go_dev.py --task <check|fmt|vet|build|all>`，前端用 `python3 .claude/scripts/frontend_dev.py --task <install|check|lint|build|all>`，Harmony 客户端用 `python3 .claude/scripts/harmony_worktree.py --task <install|build|check|test|all>`。最终有效结论以 `deploy/scripts/verify.sh --env-name <name>` 为准。
7. 若改动触达前端展示层，追加 `verify-inspect` 环节，由 Claude Code 主动完成页面级确认。
8. 若本任务将进入多子任务 integration 汇总，完成 simplify 和严格 review 后，必须调用 `feature-ready-check` subagent 做最终复核，并在输出中复述其 `## 结论`、`## 阻塞项`、`## 建议下一步`。若本任务为单 worktree 直 sync-back，则该 subagent 不是必选前置 gate。
9. 若本任务为单 worktree，进入 `/sync_back_main`；若为多子任务汇总，进入 integration。远程同步策略以根 `CLAUDE.md` 与 `/sync_back_main` 为准。

结束时请输出：

- 任务摘要
- feature 逻辑名、实际 worktree 路径与实际分支（若为续做派生，说明继承自哪个旧 branch）
- 关键变更范围
- verify 结果
- review 结果
- 本地 commit SHA
- 若为单 worktree：sync-back 结果、ExitWorktree 结果、cleanup 结果
- 最终状态：`synced to local main`、`ready for next step` 或 `blocked`
