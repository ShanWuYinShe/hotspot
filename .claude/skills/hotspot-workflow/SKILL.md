---
name: hotspot-workflow
description: >
  实现功能、修复 bug、重构、补测试、联调、验证变更或清理 worktree /
  分支时使用。支持 `feature` 模式（worktree 内端到端实现）和 `cleanup`
  模式（清理已合并或废弃的 worktree 与分支）。
argument-hint: <feature|cleanup> [...]
---

输入：`$ARGUMENTS`

这是 HotSpot 项目的单一 workflow skill 入口。保持本文件精简；详细流程按需读取引用文件。

## 模式判定
- 若第一个参数 `$1` 是 `feature`，将其后的内容视为功能或代码任务描述。
- 若第一个参数 `$1` 是 `cleanup`，将其后的内容视为可选过滤条件。
- 若用户没有显式提供模式，但请求是清晰的本地代码任务，按 `feature` 模式处理。
- 若请求是 bug、失败日志或异常行为排查，优先按 `feature` 模式进入并转入调试路径，而不是直接进入重型实现流程。
- 若用户请求是盘点或清理 worktree / 临时分支，按 `cleanup` 模式处理。
- 若无法明确判断模式，不要猜测，直接向用户追问。

## 风险分流
- 先根据根 `CLAUDE.md` 判断任务级别。
- `L0/L1`：读取实现参考后可直接执行，不必先进入设计或正式计划。
- `L2`：先写简短计划，再继续实现与验证。
- `L3`：先设计，再计划，再进入实现。
- 若任务核心是定位 bug、测试失败或异常行为，优先使用调试流程，而不是把问题当成普通功能开发。

## 默认终点
- `feature` 模式的默认目标是达到“开发完成”或“准备合并”状态，而不是默认自动完成合并回主线、删除 worktree 或删除临时分支。
- 只有在用户明确要求进入最终整理、同步主线或清理阶段时，才继续执行对应收尾动作。

## 按需加载的参考文件
- `feature` 模式的准备与实现：[references/feature/implementation.md](references/feature/implementation.md)
- `feature` 模式的本地测试与子项目校验：[references/feature/testing.md](references/feature/testing.md)
- 仅在涉及数据库、迁移、跨服务联调或真实环境验证时读取：[references/feature/integration.md](references/feature/integration.md)
- `feature` 模式的整理提交、线性合并、清理与完成标准：[references/feature/finish.md](references/feature/finish.md)
- `cleanup` 模式的完整执行步骤与安全边界：[references/cleanup.md](references/cleanup.md)
- 两种模式共用的停止条件：[references/stop-conditions.md](references/stop-conditions.md)

## 最小执行清单
- 先读根 `CLAUDE.md`，再读相关子项目 `CLAUDE.md`
- 判断任务级别：`L0/L1/L2/L3`
- 在 `.worktrees/` 下的 worktree 内完成所有修改、测试与提交
- 先完成本地校验；只有命中条件时才升级到隔离验证
- 若动作涉及主线同步、删除 worktree / 分支或其他高副作用收尾动作，先确认用户是否明确要求

## 使用规则
- 只在进入对应阶段时读取对应的 reference 文件，不要一开始预读全部内容。
- 在 `feature` 模式下，按顺序读取：
  1. [references/feature/implementation.md](references/feature/implementation.md)
  2. `L2/L3` 或任务性质要求时，再进入计划 / 设计 / 调试技能
  3. [references/feature/testing.md](references/feature/testing.md)
  4. 仅在准备合并且需要真实环境验证时读取 [references/feature/integration.md](references/feature/integration.md)
  5. [references/feature/finish.md](references/feature/finish.md)
  6. 需要判断是否应中止时读取 [references/stop-conditions.md](references/stop-conditions.md)
- 在 `cleanup` 模式下，先读取 [references/cleanup.md](references/cleanup.md)，并在遇到风险判断时读取 [references/stop-conditions.md](references/stop-conditions.md)。
- 除非遇到真正阻塞的问题，否则不要在中途停下来征求确认。
