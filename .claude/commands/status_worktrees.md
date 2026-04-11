---
description: 查看当前主目录、feature worktree 与 integration worktree 的整体状态
argument-hint: [optional filters]
allowed-tools: Bash(python3 .claude/scripts/status_worktrees.py *), Read, Glob, Grep
---

请汇总当前仓库的多 worktree 状态。当前命令的参数为：

$ARGUMENTS

目标不是修改任何代码，而是给出一份清晰的本地工作区总览，帮助判断当前应该继续 `/start_feature`、进入 `/start_integrate`、执行 `/refresh_integrate`，还是先 `/cleanup_worktrees`。优先使用仓库已提供的标准入口 `python3 .claude/scripts/status_worktrees.py` 收集状态；若只是补充只读分析，不必把它理解成唯一入口。

执行要求：
1. 汇总主目录状态：路径、分支、是否在本地 `main` 且无未提交改动。
2. 列出当前 linked worktree，并尽量推断其角色：feature、integration 或 other。
3. 对 feature lineage 输出：当前分支、dirty 状态、`active-latest` / `superseded`、metadata 是否存在、ownership 是否为 `owned` / `foreign` / `takeover_required`。
4. 对 integration 输出：基线 SHA、当前 `main SHA`、`active` / `stale`、`gate_status`、ownership、是否可能 `safe_to_cleanup`。
5. 若 metadata 与 git reality 存在偏差，要明确标记为 drift / status unclear，而不是默认自动接管或自动 cleanup。
6. 推荐下一步命令只基于当前机器状态给出，不要在此命令里修改对象归属或推进 cleanup。
7. 若需要生成补充分析文件，默认写到 `tmp/`。

结束时请输出：
- 主目录路径、分支、状态
- linked worktree 列表
- 每个 worktree 的角色、分支、dirty 状态、ownership、必要时的 lineage 标记
- active feature 候选列表
- integration 状态：`none`、`active` 或 `stale`
- 推荐下一步命令
