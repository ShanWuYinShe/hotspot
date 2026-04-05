---
description: 查看当前主目录、feature worktree 与 integration worktree 的整体状态
argument-hint: [optional filters]
allowed-tools: Bash(git status:*), Bash(git branch:*), Bash(git rev-parse:*), Bash(git worktree list:*), Bash(git log:*), Read, Glob, Grep
---

请汇总当前仓库的多 worktree 状态。当前命令的参数为：

$ARGUMENTS

目标不是修改任何代码，而是给出一份清晰的本地工作区总览，帮助判断当前应该继续 `/start_feature`、进入 `/start_integrate`、执行 `/refresh_integrate`，还是先 `/cleanup_worktrees`。

执行要求：

1. 汇总主目录状态：路径、分支、是否在本地 `main` 且无未提交改动。
2. 列出当前 linked worktree，并尽量推断其角色：feature、integration 或其他。
3. 输出每个 worktree 的当前分支和 dirty 状态，并尽量标记哪些更像当前会话对象、哪些属于其他会话或状态不明。若发现同一 feature 逻辑名存在 `-rN` 派生实例，需按 lineage 视角额外标记 `active-latest`、`superseded` 或 `status-unclear`。
4. 结合上下文判断哪些 feature worktree 已达到 `ready for integration`；如无法完全自动判断，要明确列出候选项和不确定项。对存在遗留 lineage 的 feature，优先指出当前应继续最新派生实例，还是回主目录重新 `/start_feature` 让系统创建新的派生实例，而不是建议继续进入旧实例。
5. 通过共享脚本 `.claude/scripts/integration_meta.py` 扫描仓库根 `tmp/claude/integrations/*.json`，判断是否存在 integration 状态锚点；若存在，输出其 baseline main SHA、当前 main SHA 以及 `active` / `stale` 判定。
6. 判断是否存在 integration worktree；若存在，输出其路径、分支以及基线是否可能过期。
7. 根据当前状态给出下一步建议：`/start_feature`、`/start_integrate`、`/refresh_integrate`、`/verify_integrate`、`/finish_integrate` 或 `/cleanup_worktrees`。
8. 不要默认修改、接管或 cleanup 任何对象；仓库级 workflow 与风险边界以根 `CLAUDE.md` 为准。

结束时请输出：

- 主目录路径、分支、状态
- linked worktree 列表
- 每个 worktree 的角色、分支、dirty 状态，以及必要时的 lineage 标记（`active-latest` / `superseded` / `status-unclear`）
- ready feature 列表
- integration 状态：`none`、`active` 或 `stale`
- 推荐下一步命令
