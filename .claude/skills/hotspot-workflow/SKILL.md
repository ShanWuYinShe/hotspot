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
- 若用户请求是盘点或清理 worktree / 临时分支，按 `cleanup` 模式处理。
- 若无法明确判断模式，不要猜测，直接向用户追问。

## 按需加载的参考文件
- `feature` 模式的准备与实现：[references/feature/implementation.md](references/feature/implementation.md)
- `feature` 模式的本地测试与子项目校验：[references/feature/testing.md](references/feature/testing.md)
- 仅在涉及数据库、迁移、跨服务联调或真实环境验证时读取：[references/feature/integration.md](references/feature/integration.md)
- `feature` 模式的整理提交、线性合并、清理与完成标准：[references/feature/finish.md](references/feature/finish.md)
- `cleanup` 模式的完整执行步骤与安全边界：[references/cleanup.md](references/cleanup.md)
- 两种模式共用的停止条件：[references/stop-conditions.md](references/stop-conditions.md)

## 使用规则
- 只在进入对应阶段时读取对应的 reference 文件，不要一开始预读全部内容。
- 在 `feature` 模式下，按顺序读取：
  1. [references/feature/implementation.md](references/feature/implementation.md)
  2. [references/feature/testing.md](references/feature/testing.md)
  3. 仅在需要真实环境验证时读取 [references/feature/integration.md](references/feature/integration.md)
  4. [references/feature/finish.md](references/feature/finish.md)
  5. 需要判断是否应中止时读取 [references/stop-conditions.md](references/stop-conditions.md)
- 在 `cleanup` 模式下，先读取 [references/cleanup.md](references/cleanup.md)，并在遇到风险判断时读取 [references/stop-conditions.md](references/stop-conditions.md)。
- 除非遇到真正阻塞的问题，否则不要在中途停下来征求确认。
