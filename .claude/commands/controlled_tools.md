---
description: 说明当前仓库登记的受控脚本及其提供的功能边界
argument-hint: [optional filter]
allowed-tools: Read, Glob, Grep
---

请说明当前仓库登记的受控脚本清单与功能边界。当前命令的参数为：

$ARGUMENTS

该命令的目标不是执行 workflow，而是让 Claude 在动手前先确认：
- 当前有哪些受控脚本
- 每个脚本提供什么功能
- 哪些脚本会改状态、哪些是只读/辅助
- 哪些属于 `.claude/scripts/*.py`，哪些属于 `deploy/scripts/*.sh`
- 如果没有对应的受控脚本，应该先停下来说明缺口，而不是自行拼装命令

执行要求：

1. 读取当前仓库登记的受控脚本清单与对应功能分组。
2. 按用途分组输出：workflow orchestration、dev/check/build、metadata/support、deploy workflow。
3. 对每个脚本说明：功能、适用上下文、是否会改状态。
4. 明确提醒：git 状态变更默认分步执行，不允许使用 `&&`、`;` 或 `||` 把多个 git 状态变更命令串成单条 Bash。
5. 若用户询问某个动作该用什么入口，优先从该清单中选择；若清单里没有对应入口，停止并说明还缺少受控脚本。

结束时请输出：

- 当前受控脚本分组与功能
- 哪些脚本会改状态
- 哪些脚本属于 deploy 受控入口
- 若无对应脚本时的处理原则
