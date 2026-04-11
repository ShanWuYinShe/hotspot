---
description: 在 integration worktree 中运行标准集成验证并给出下一步建议
argument-hint: [env-name]
allowed-tools: Bash(python3 .claude/scripts/verify_integrate.py *), Read, Glob, Grep
---

请对当前 integration worktree 运行一轮标准集成验证。当前命令的参数为：

$ARGUMENTS

该命令负责：读取当前 integration metadata → 检查 baseline / ownership / replay completeness → 运行标准 verify 入口 → 回写 verify 状态。仓库级 workflow 以根 `CLAUDE.md` 为准。

执行要求：
1. 确认当前位于 integration worktree；如果不是，停止并说明原因。
2. 读取 integration metadata，检查对象是否 `owned`，以及 baseline 是否仍然有效；若 `stale`，提示改走 `/refresh_integrate`。
3. 检查 `planned_features` 是否都已 replay 完整；若 replay 未完成，不要进入 verify。
4. 最终有效结论以 `deploy/scripts/verify.sh --env-name <name>` 为准；局部自检不替代最终质量门。
5. 通过 `.claude/scripts/integration_meta.py` 回写 `verify_status`、`verify_summary`、`verify_env_name` 和 evidence path。
6. 若 verify 通过，下一步先完成 simplify 与 strict review，再进入 `/finish_integrate`。
7. 若 verify 失败，明确列出失败范围和建议修复方向。
8. 若有日志、截图或中间输出，默认写入 `tmp/`。

结束时请输出：
- integration worktree 路径与分支
- 当前 integration 基线 `main HEAD`
- ownership 状态
- 使用的 env-name
- verify 入口
- verify 结果摘要
- 下一步建议：`proceed to review`、`fix and re-verify` 或 `must refresh`
