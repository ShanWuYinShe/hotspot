# Spec 目录 - Agent Guide

> 本目录是 HotSpot 所有 API 契约和编码规范的**唯一权威来源**。

## 目录作用

`spec/` 中的 `.md` 文件定义了 API 接口契约、共享类型、错误码、WebSocket 事件以及各子项目的编码规范。所有实现代码必须严格遵循这些文件。

## 文件结构（精简说明）

`spec/` 目录下的具体文件结构和模块划分，请 Agent 通过文件系统和文件内容自行探索；本 `AGENTS.md` 不再维护完整清单，以避免信息漂移。
常用文件示例（名称仅作导航参考，以实际仓库为准）：`api_spec_index.md`、`api_spec_base.md`、`api_spec_types.md`、`api_spec_errors.md` 等。

## 文件内部优先级

当 spec 文件之间出现冲突时，按以下优先级：

```
api_spec_base > api_spec_types > api_spec_errors > 各模块 spec
```

## Agent 操作规则

### 读取 Spec

- 实现任何 API 相关功能前，**必须先阅读**对应模块的 spec 文件。
- 查找接口：先查 `api_spec_index.md` 定位模块，再查具体模块 spec。
- 不确定类型时查 `api_spec_types.md`，不确定错误码时查 `api_spec_errors.md`。

### 禁止修改 Spec

- 除非用户**明确要求修改规范**，否则禁止修改本目录下任何文件。
- Agent 不得根据实现代码反向修改 spec（spec 优先于实现）。

### 引用方式

- 引用 spec 时使用文件名和小节标题，例如「参见 `api_spec_auth.md` 的注册端点」。
- **禁止使用数字章节编号**（如「第 3 章」「3.2 节」），只使用小节标题。

### 安全规则

文件删除、批量变更、Git 操作规范等全局规则见项目根目录 `AGENTS.md` 的「权限边界」和「Git 操作规范」章节，此处不重复。

> **⚠️ `tmp/` 路径规则**：所有临时文件必须存放在项目根目录的 `tmp/` 下。详见根目录 `AGENTS.md` 的「Agent 行为准则」章节。

**Spec 目录额外约束**：禁止自行删除本目录下任何文件，如需删除须列清单并经用户确认。

### 关键全局约束速查

分页、幂等、错误信封、位置与距离、ID/时间格式、错误码和可见性等通用规则，在 `api_spec_base.md` 与 `api_spec_errors.md` 中已有完整定义；本文件不再重复细节，Agent 应直接查阅对应 spec。
