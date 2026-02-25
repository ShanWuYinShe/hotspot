# Spec 目录 - Agent Guide

> 本目录是 HotSpot 所有 API 契约和编码规范的**唯一权威来源**。

## 目录作用

`spec/` 中的 `.md` 文件定义了 API 接口契约、共享类型、错误码、WebSocket 事件以及各子项目的编码规范。所有实现代码必须严格遵循这些文件。

## 文件结构

| 文件 | 作用 | 引用频率 |
|------|------|----------|
| `api_spec_index.md` | API 契约总索引，查接口从这里开始 | 高 |
| `api_spec_base.md` | 全局规则：分页、幂等、Token、错误信封 | 高 |
| `api_spec_types.md` | 跨模块共享类型定义 | 高 |
| `api_spec_errors.md` | 错误码定义（唯一来源） | 高 |
| `api_spec_websocket.md` | WebSocket 事件契约 | 中 |
| `api_spec_auth.md` | 认证模块 API | 按需 |
| `api_spec_user.md` | 用户模块 API | 按需 |
| `api_spec_moment.md` | 瞬间模块 API | 按需 |
| `api_spec_chat.md` | 聊天模块 API | 按需 |
| `api_spec_group.md` | 群组模块 API | 按需 |
| `api_spec_feed.md` | 信息流模块 API | 按需 |
| `api_spec_search.md` | 搜索模块 API | 按需 |
| `api_spec_map.md` | 地图模块 API | 按需 |
| `api_spec_share.md` | 分享模块 API | 按需 |
| `api_spec_media.md` | 媒体模块 API | 按需 |
| `api_spec_report.md` | 举报模块 API | 按需 |
| `api_spec_notification.md` | 通知模块 API | 按需 |
| `api_spec_system.md` | 系统模块 API | 按需 |
| `api_spec_admin_core.md` | 管理端核心 API | 按需 |
| `api_spec_admin_moderation.md` | 管理端审核 API | 按需 |
| `backend_rule.md` | 后端编码规范 | 后端开发时 |
| `frontend_rule.md` | 前端编码规范 | 前端开发时 |
| `harmonyapp_rule.md` | 鸿蒙客户端编码规范 | 鸿蒙开发时 |
| `backend_doc.md` | 后端功能边界文档 | 后端开发时 |
| `frontend_doc.md` | 前端功能边界文档 | 前端开发时 |
| `harmonyapp_doc.md` | 鸿蒙客户端功能边界文档 | 鸿蒙开发时 |

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

文件删除、批量变更、Git 操作规范等全局规则见根目录 `AGENTS.md` 的「权限边界」和「Git 操作规范」章节，此处不重复。

**Spec 目录额外约束**：禁止自行删除本目录下任何文件，如需删除须列清单并经用户确认。

### 关键全局约束速查

以下内容在 `api_spec_base.md` 中有完整定义，这里仅列出 Agent 最容易遗忘的：

- **仅 POST**，不使用 GET/PUT/DELETE/PATCH。
- **游标分页**，禁止 OFFSET，`cursor` 不透明。
- **幂等键**：写操作须检查是否在 MUST 幂等列表中。
- **位置**：请求用 `Location`，响应用 `RelativeLocation`，距离由后端算。
- **ID**：统一 `xxx_id`，类型为 `string`。
- **时间**：UTC ISO 8601。
- **错误码**：只能使用 `api_spec_errors.md` 中已有的码，不得自行编号。
- **可见性标注**：`@auth-only`、`@owner-only`、`@group-admin-only` 等控制字段返回。
