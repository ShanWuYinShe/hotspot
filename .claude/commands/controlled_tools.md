---
description: 说明当前仓库通过 skill 和严格入口已经提供了哪些 workflow 能力
argument-hint: [optional filter]
allowed-tools: Read, Glob, Grep
---

请说明当前仓库已经提供了哪些可供 Claude 优先发现和使用的 workflow 能力。当前命令的参数为：

$ARGUMENTS

该命令的目标不是执行 workflow，而是帮助 Claude 在动手前快速建立正确心智模型：
- 当前仓库有哪些 **推荐 skill 入口**
- 哪些入口对应 **严格流程 / 隔离运行 / 高确定性控制面**
- 哪些事情可以直接做，不必因为没有脚本就停下
- 哪些临时文件、中间文件应统一放到 `tmp/`

执行要求：

1. 读取当前仓库登记的 skill / 受控脚本 / deploy 入口清单与分组。
2. 按用途分组输出：
   - skill capability
   - strict workflow scripts
   - deploy isolation entrypoints
   - direct actions Claude can usually do safely
3. 对每个入口说明：功能、适用上下文、是否会改状态、是否属于推荐入口还是严格入口。
4. 明确提醒：
   - workflow 规则以根 `CLAUDE.md` 为准；本命令只做能力地图。
   - git 状态变更默认分步执行，不允许使用 `&&`、`;` 或 `||` 把多个 git 状态变更命令串成单条 Bash。
   - 临时文件、验证截图和中间产物默认写入 `tmp/`，避免污染 git 状态。
5. 若用户询问某个动作该用什么入口：
   - 先判断是否已有 skill 可用；
   - 再判断是否属于严格入口场景；
   - 若只是普通分析或低副作用动作，明确说明可以直接做，而不是默认要求补脚本。

结束时请输出：

- 当前 skill 能力地图
- 当前严格入口清单
- deploy 隔离入口清单
- Claude 通常可直接执行的动作范围
- `tmp/` 使用原则
