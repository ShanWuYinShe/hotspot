---
description: 在 integration worktree 中运行标准集成验证并给出下一步建议
argument-hint: [env-name]
allowed-tools: Bash(git status:*), Bash(git branch:*), Bash(git rev-parse:*), Bash(*deploy/scripts/verify.sh*), Read, Glob, Grep
---

请对当前 integration worktree 运行一轮标准集成验证。当前命令的参数为：

$ARGUMENTS

目标不是做宿主机上的临时检查，而是通过仓库标准入口确认当前 integration 结果在统一验证环境中是否通过，并据此决定下一步是继续修复、进入 review，还是停止并 refresh integration。

执行要求：

1. 先确认当前位于 integration worktree；如果不是，停止并说明原因。
2. 读取仓库根 `tmp/claude/integrations/<integration-id>.json`，检查当前 integration 基线是否仍然有效；若主目录 `main` 已前进，先提示改走 `/refresh_integrate`。
3. 使用仓库标准入口运行验证；最终有效结论统一以 `deploy/scripts/verify.sh --env-name <name>` 为准。
4. 通过共享脚本 `.claude/scripts/integration_meta.py` 将 verify 结果回写到 integration 元信息中的 `verify_status`、`verify_env_name`、`verify_summary` 与 `updated_at`。
5. 汇总 verify 结果时，区分编译错误、测试失败、构建失败、环境问题，以及基线问题。
6. 若 verify 通过，说明当前 integration 已具备进入 simplify / strict review / `/finish_integrate` 的基础。
7. 若 verify 失败，明确列出失败范围和建议修复方向。
8. verify 通过后，若后续 gate 与对象归属均满足规则，应继续进入 review / finish，并在满足条件时直接线性同步回主目录本地 `main`；若需要单独入口，可复用 `/sync_back_main` 的同名规则；不要 push。

结束时请输出：

- integration worktree 路径与分支
- 当前 integration 基线 `main HEAD`
- 使用的 env-name
- verify 入口
- verify 结果摘要
- 下一步建议：`proceed to review`、`fix and re-verify` 或 `must refresh`
