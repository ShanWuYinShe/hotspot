---
description: 在 integration worktree 中运行标准集成验证并给出下一步建议
argument-hint: [env-name]
allowed-tools: Bash(git status:*), Bash(git branch:*), Bash(git rev-parse:*), Bash(*deploy/scripts/verify.sh*), Read, Glob, Grep
---

请对当前 integration worktree 运行一轮标准集成验证。当前命令的参数为：

$ARGUMENTS

通过仓库标准入口确认当前 integration 结果在统一验证环境中是否通过，并据此决定下一步。仓库级 workflow 以根 `CLAUDE.md` 为准。

执行要求：

1. 确认当前位于 integration worktree；如果不是，停止并说明原因。
2. 读取 integration 元信息，检查基线是否仍然有效；若主目录 `main` 已前进，提示改走 `/refresh_integrate`。
3. 使用仓库标准入口运行验证；最终有效结论以 `deploy/scripts/verify.sh --env-name <name>` 为准。
4. 通过 `.claude/scripts/integration_meta.py` 将 verify 结果回写到元信息。
5. 汇总结果时，区分编译错误、测试失败、构建失败、环境问题及基线问题。
6. 若 verify 通过，下一步进入 simplify / strict review / `/finish_integrate`。
7. 若 verify 失败，明确列出失败范围和建议修复方向。
8. 不要 push。

结束时请输出：

- integration worktree 路径与分支
- 当前 integration 基线 `main HEAD`
- 使用的 env-name
- verify 入口
- verify 结果摘要
- 下一步建议：`proceed to review`、`fix and re-verify` 或 `must refresh`
