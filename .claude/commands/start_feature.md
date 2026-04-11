---
description: 启动新的 feature 开发流，并在需要时使用仓库标准入口完成验证与同步
argument-hint: <task description>
allowed-tools: Read, Glob, Grep, AskUserQuestion, TaskCreate, TaskUpdate, TaskList, EnterWorktree, ExitWorktree, Bash(python3 .claude/scripts/feature_meta.py *), Bash(python3 .claude/scripts/rust_dev.py *), Bash(python3 .claude/scripts/go_dev.py *), Bash(python3 .claude/scripts/frontend_dev.py *), Bash(python3 .claude/scripts/harmony_worktree.py *), Bash(./deploy/scripts/verify.sh *), Bash(./deploy/scripts/verify-inspect.sh *)
---

围绕以下任务启动新的 feature 开发流：

$ARGUMENTS

该命令是仓库推荐的 feature 入口：从主目录本地 `main` 当前 HEAD 创建 feature worktree，在该 worktree 中完成实现、检查、测试、simplify、strict review，并根据任务形态决定下一步进入 `/sync_back_main` 还是 integration。仓库级 workflow、质量门、风险边界和 commit 规范以根 `CLAUDE.md` 为准。

执行要求：
1. 先检查主目录是否保持在本地 `main` 且无未提交改动；如果不是，先停下来说明原因。
2. 先按任务范围读取根 `CLAUDE.md`、对应子项目 `CLAUDE.md` 与必要的 `spec/`。
3. 分析任务：根据复杂度和文件集决定是否拆分为多个子任务；子任务边界清晰且文件集相对独立时才并行，否则保持单任务。若发现新增需求已明显偏离当前 feature worktree 建立时的原始目标，默认不要继续混在当前 worktree 中实现，而应停止并建议新开 feature worktree。
4. 对非极小任务，优先进入 plan mode；然后建立 task list。
5. **使用 `EnterWorktree` 工具**从主目录创建 feature worktree；`name` 参数必须遵循 `{子项目名}.{模块名}.{功能描述}` 格式。若已存在同 lineage 实例，则创建新的 `-rN` 派生实例，并基于该 lineage 最新 branch 继续。
6. feature worktree 创建后，应为当前分支初始化 feature runtime metadata，写入 `owner_token`、`baseline_main_sha`、`task_summary` 与各质量门的初始状态。
7. 在 feature worktree 中完成 `实现 → check(语法/编译/警告) → test → simplify → strict review → commit` 闭环；任一环节发现问题必须修复后重新循环，并把当前质量门状态回写 metadata。
8. 当仓库已经提供标准入口时，优先使用标准入口：
   - 需要统一 check/build 入口时，可使用 `rust_dev.py`、`go_dev.py`、`frontend_dev.py`、`harmony_worktree.py`
   - 最终有效验证结论以 `deploy/scripts/verify.sh --env-name <name>` 为准
   - 若改动触达前端展示层，追加 `deploy/scripts/verify-inspect.sh --env-name <name>` 提供隔离页面入口
9. 若本任务为单 worktree 且 feature metadata 已表达 `ready_to_sync`，则进入 `/sync_back_main`；若为多子任务汇总，则只把该 feature 标为 `ready_for_integration`，后续进入 integration。
10. 中间文件、验证截图和调试输出默认写入当前 worktree 的 `tmp/`，不要污染 worktree 根目录。

结束时请输出：
- 任务摘要
- feature 逻辑名、实际 worktree 路径与实际分支（若为续做派生，说明继承自哪个旧 branch）
- 关键变更范围
- 使用了哪些标准入口 / 直接动作
- verify 结果
- review 结果
- feature metadata 状态摘要
- 本地 commit SHA 或当前线性历史状态
- 若为单 worktree：sync-back 结果、ExitWorktree 结果、cleanup 结果
- 最终状态：`synced to local main`、`ready for integration`、`ready for next step` 或 `blocked`
