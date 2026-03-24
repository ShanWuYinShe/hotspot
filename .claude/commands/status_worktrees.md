---
description: 查看当前主目录、feature worktree 与 integration worktree 的整体状态
argument-hint: [optional filters]
allowed-tools: Bash(git status:*), Bash(git branch:*), Bash(git rev-parse:*), Bash(git worktree list:*), Bash(git log:*), Read, Glob, Grep
---

请汇总当前仓库的多 worktree 状态。当前命令的参数为：

$ARGUMENTS

目标不是修改任何代码，而是给出一份清晰的本地工作区总览，帮助判断当前应该继续 `/start_feature`、进入 `/start_integrate`、执行 `/refresh_integrate`，还是先 `/cleanup_worktrees`。

执行要求：

1. 汇总主目录状态：路径、分支、是否 `main + clean`。
2. 列出当前 linked worktree，并尽量推断其角色：feature、integration 或其他。
3. 输出每个 worktree 的当前分支和 dirty 状态。
4. 结合上下文判断哪些 feature worktree 已达到 `ready for integration`；如无法完全自动判断，要明确列出候选项和不确定项。
5. 通过共享脚本 `.claude/scripts/integration_meta.py` 扫描仓库根 `tmp/claude/integrations/*.json`，判断是否存在 integration 状态锚点；若存在，输出其 baseline main SHA、当前 main SHA 以及 `active` / `stale` 判定。
6. 判断是否存在 integration worktree；若存在，输出其路径、分支以及基线是否可能过期。
7. 根据当前状态给出下一步建议：`/start_feature`、`/start_integrate`、`/refresh_integrate`、`/verify_integrate`、`/finish_integrate` 或 `/cleanup_worktrees`。
8. 不要默认修改任何 worktree，不要 cleanup，不要同步回 `main`。

结束时请输出：

- 主目录路径、分支、状态
- linked worktree 列表
- 每个 worktree 的角色、分支、dirty 状态
- ready feature 列表
- integration 状态：`none`、`active` 或 `stale`
- 推荐下一步命令
