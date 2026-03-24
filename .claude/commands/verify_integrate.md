---
description: 在 integration worktree 中运行标准集成验证并给出下一步建议
argument-hint: [env-name]
---

请对当前 integration worktree 运行一轮标准集成验证。当前命令的参数为：

$ARGUMENTS

目标不是做宿主机上的临时检查，而是通过仓库标准入口确认当前 integration 结果在统一验证环境中是否通过，并据此决定下一步是继续修复、进入 review，还是停止并 refresh integration。

执行要求：

1. 先确认当前位于 integration worktree；如果不是，停止并说明原因。
2. 检查当前 integration 基线是否仍然有效；若主目录 `main` 已前进，先提示改走 `/refresh_integrate`。
3. 使用仓库标准入口运行验证；最终有效结论统一以 `deploy/scripts/verify.sh --env-name <name>` 为准。
4. 汇总 verify 结果时，区分编译错误、测试失败、构建失败、环境问题，以及基线问题。
5. 若 verify 通过，说明当前 integration 已具备进入 simplify / strict review / `/finish_integrate` 的基础。
6. 若 verify 失败，明确列出失败范围和建议修复方向。
7. 不要默认同步回 `main`，不要 push。

结束时请输出：

- integration worktree 路径与分支
- 当前 integration 基线 `main HEAD`
- 使用的 env-name
- verify 入口
- verify 结果摘要
- 下一步建议：`proceed to review`、`fix and re-verify` 或 `must refresh`
