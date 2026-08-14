# HotSpot API 规范文档（契约唯一权威）

> 本文档是跨端契约的唯一权威来源：HTTP 端点、请求/响应字段与类型、错误码、WebSocket 事件与负载、统一 envelope。
>
> 内容按逻辑拆分为多个文件，便于维护和查阅。

## 文档结构

### 基础规范

| 文件 | 说明 |
|-----|------|
| [api_spec_base.md](api_spec_base.md) | API 设计原则、节点与路径架构、命名规范、参数必显式、类型稳定性、Token 语义、分页排序、幂等性、安全性 |
| [api_spec_authz.md](api_spec_authz.md) | 守卫与权限模型：认证档位、资源约束、群管理守卫子树、平台权限键、ForbiddenReason、权限矩阵（148 端点全表） |
| [api_spec_types.md](api_spec_types.md) | 跨模块复用的通用类型定义（Domain Primitives + Domain Entities，恒定 nullable 通则） |
| [api_spec_errors.md](api_spec_errors.md) | 所有 HTTP API 和 WebSocket 错误码定义 + ForbiddenReason 权威定义 |
| [api_spec_websocket.md](api_spec_websocket.md) | WebSocket 事件、连接与鉴权规范、订阅机制 |

### 核心业务 API

| 文件 | 说明 |
|-----|------|
| [api_spec_auth.md](api_spec_auth.md) | 认证 API（登录、注册、游客、Token 刷新、密码重置、会话管理） |
| [api_spec_user.md](api_spec_user.md) | 用户 API（搜索/详情/主页聚合、账户资料/设置、关注/拉黑/备注） |
| [api_spec_moment.md](api_spec_moment.md) | 生活瞬间 API（搜索/CRUD/收藏/我创建的瞬间） |
| [api_spec_comment.md](api_spec_comment.md) | 评论 API（两级评论模型、软删除语义） |
| [api_spec_chat.md](api_spec_chat.md) | 聊天 API（私聊管理、私聊消息、群聊消息） |
| [api_spec_group.md](api_spec_group.md) | 群组管理 API（公开/成员路径 + groups/admin 守卫子树） |

### 内容与发现 API

| 文件 | 说明 |
|-----|------|
| [api_spec_discovery.md](api_spec_discovery.md) | 发现与聚合 API（feed 信息流、search 建议、map 地图渲染） |
| [api_spec_share.md](api_spec_share.md) | 通用分享系统 API（位置、动态、群组等） |

### 辅助功能 API

| 文件 | 说明 |
|-----|------|
| [api_spec_reaction.md](api_spec_reaction.md) | 表情反应 API（message / moment / comment 统一操作与列表） |
| [api_spec_media.md](api_spec_media.md) | 文件与媒体上传 API（Presigned 模式） |
| [api_spec_report.md](api_spec_report.md) | 举报 API |
| [api_spec_notify.md](api_spec_notify.md) | 通知中心 API（通知历史、未读计数、已读标记） |
| [api_spec_system.md](api_spec_system.md) | 系统健康检查、版本信息、客户端配置、公告、冷启动聚合 |

### 平台管理 API

| 文件 | 说明 |
|-----|------|
| [api_spec_admin.md](api_spec_admin.md) | 平台管理节点 /admin/v1（RBAC 权限管理、审计日志、服务端配置、内容审核） |

## 权威边界与冲突处理

- 本规范是 HotSpot 前后端交互契约的唯一权威（端点、字段、类型、错误码、事件、响应 envelope）
- 若本规范与工程实现或各子项目 AGENTS.md 中的实现规范冲突，以本规范为准
- 引用本规范时仅使用小节标题或锚点标题，禁止使用数字章节号

**内部优先级**（本规范内部冲突时）：
1. api_spec_base.md
2. api_spec_authz.md
3. api_spec_types.md
4. api_spec_errors.md
5. 各模块 api_spec_*.md

## 端点存废审查记录（契约级）

本版（v1 重设计）对端点集做过一次完整存废审查，结论如下，后续端点增删沿用同一标准：

**存废判定标准**：① 两端点返回的是否**同一份数据**；② 是否已有跨用户端点天然支持「自己视角」——同时满足才合并，否则保留。

**删除/合并**：
- `moments/list_my`：拥有者视角（private + 软删占位）与公开视角是不同数据，按「视角分离」保留为独立端点并改名 `moments/list_created`；`users/list_moments` 收紧为纯公开视角（user_id 必传）
- `admin/audit/open_the_door`：删除——审计列表项已含全量 `details`，纯冗余

**争议保留（记录理由）**：
- `users/list_friends`/`list_following`/`list_followers` 三个不合并：排序键不同、UI 独立分页，合并成判别参数反而增加心智负担
- `groups/list_joined` 与 `users/list_groups` 并存：成员视角（viewer_state/私密群）vs 公开主页视角（仅 searchable 群），数据不同
- `admin/users|groups/open_the_door` 保留：单资源全量详情 ≠ 分页搜索
- `system/get_meta` 保留：版本状态巡检（sunset 机制依赖）
- `media` 4 端点保留：单/批量 presign+complete 支撑多图与聊天批量场景

## 设计要点速览（v1 重设计）

- **三位置划分**：/api/v1（普通用户）、/api/v1/groups/admin（群管理守卫子树）、/admin/v1（平台管理员节点）；平台管理员在 /api/v1 与普通用户完全同权
- **一个端点 = 一种角色 + 一个固定形状**：不同视角不同端点（OwnerXxxItem / AdminXxxItem / XxxViewerState），@xxx-only 注解全量废除
- **恒定 nullable**：权限/隐私条件字段恒在，null = 隐藏/不适用；`next_cursor: string | null`
- **命名**：动词表 + 状态限定词（created/joined/submitted/sent/received/favorites/blocked/banned/reacted）；`_my` 后缀与 `My` 类型前缀全量废除；`open_the_door` 为详情动词（玩梗保护条款保留）；禁用 `notification`（统一 `notify`）
- **参数必显式**：目标 ID 一律必传；行为范围参数显式声明（如 `scope: 'ids' | 'all'`）；禁止缺省翻转语义
- **失败语义**：`10200` 必附 `data.forbidden_reason`（12 项枚举，见 api_spec_errors.md）

## 快速索引（按路径前缀）

| 路径前缀 | 对应文档 |
|---------|---------|
| /api/v1/auth/ | api_spec_auth.md |
| /api/v1/users/ | api_spec_user.md |
| /api/v1/moments/ | api_spec_moment.md |
| /api/v1/comments/ | api_spec_comment.md |
| /api/v1/reactions/ | api_spec_reaction.md |
| /api/v1/private_chats/ | api_spec_chat.md |
| /api/v1/private_messages/ | api_spec_chat.md |
| /api/v1/group_messages/ | api_spec_chat.md |
| /api/v1/groups/ | api_spec_group.md |
| /api/v1/groups/admin/ | api_spec_group.md（守卫子树）+ api_spec_authz.md |
| /api/v1/feed/ | api_spec_discovery.md |
| /api/v1/search/ | api_spec_discovery.md |
| /api/v1/map/ | api_spec_discovery.md |
| /api/v1/share/ | api_spec_share.md |
| /api/v1/media/ | api_spec_media.md |
| /api/v1/reports/ | api_spec_report.md |
| /api/v1/notify/ | api_spec_notify.md |
| /api/v1/system/ | api_spec_system.md |
| /admin/v1/ | api_spec_admin.md + api_spec_authz.md |
| /ws/v1 | api_spec_websocket.md |