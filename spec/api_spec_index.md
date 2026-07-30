# HotSpot API 规范文档（契约唯一权威）

> 本文档是跨端契约的唯一权威来源：HTTP 端点、请求/响应字段与类型、错误码、WebSocket 事件与负载、统一 envelope。
>
> 内容按逻辑拆分为多个文件，便于维护和查阅。

## 文档结构

### 基础规范

| 文件 | 说明 |
|-----|------|
| [api_spec_base.md](api_spec_base.md) | API 设计原则、全局约束、命名规范、Token 语义、分页排序、幂等性、安全性 |
| [api_spec_types.md](api_spec_types.md) | 跨模块复用的通用类型定义（Domain Primitives + Domain Entities） |
| [api_spec_errors.md](api_spec_errors.md) | 所有 HTTP API 和 WebSocket 错误码定义 |
| [api_spec_websocket.md](api_spec_websocket.md) | WebSocket 事件、连接与鉴权规范、订阅机制 |

### 核心业务 API

| 文件 | 说明 |
|-----|------|
| [api_spec_auth.md](api_spec_auth.md) | 认证 API（登录、注册、游客、Token 刷新、密码重置、会话管理） |
| [api_spec_user.md](api_spec_user.md) | 用户 API（搜索/详情/主页聚合、账户资料/设置、关注/拉黑/备注） |
| [api_spec_moment.md](api_spec_moment.md) | 生活瞬间与评论 API |
| [api_spec_chat.md](api_spec_chat.md) | 聊天 API（私聊管理、私聊消息、群聊消息） |
| [api_spec_group.md](api_spec_group.md) | 群组管理 API（群组 CRUD、成员管理、申请/邀请、群公告） |

### 内容与发现 API

| 文件 | 说明 |
|-----|------|
| [api_spec_feed.md](api_spec_feed.md) | 发现域 Feed API（关注动态、附近瞬间/用户/群组、热门内容） |
| [api_spec_search.md](api_spec_search.md) | 搜索与建议 API（综合建议） |
| [api_spec_map.md](api_spec_map.md) | 地图渲染 API |
| [api_spec_share.md](api_spec_share.md) | 通用分享系统 API（位置、动态、群组等） |

### 辅助功能 API

| 文件 | 说明 |
|-----|------|
| [api_spec_reaction.md](api_spec_reaction.md) | 表情反应 API（message / moment / comment 统一操作与列表） |
| [api_spec_media.md](api_spec_media.md) | 文件与媒体上传 API（Presigned 模式） |
| [api_spec_report.md](api_spec_report.md) | 举报 API |
| [api_spec_notify.md](api_spec_notify.md) | 通知中心 API（通知历史、未读计数、已读标记） |
| [api_spec_system.md](api_spec_system.md) | 系统健康检查、版本信息、客户端配置、公告、冷启动聚合 |

### 管理员 API

| 文件 | 说明 |
|-----|------|
| [api_spec_admin_core.md](api_spec_admin_core.md) | 管理后台核心（RBAC 权限管理、审计日志、服务端配置、客户端配置） |
| [api_spec_admin_moderation.md](api_spec_admin_moderation.md) | 管理后台审核（用户/群组/举报/标签/公告管理） |

## 权威边界与冲突处理

- 本规范是 HotSpot 前后端交互契约的唯一权威（端点、字段、类型、错误码、事件、响应 envelope）
- 若本规范与工程实现或各子项目 `AGENTS.md` 中的实现规范冲突，以本规范为准
- 引用本规范时仅使用小节标题或锚点标题，禁止使用数字章节号（如"第X章/第X节"）

**内部优先级**（本规范内部冲突时）：
1. `api_spec_base.md`
2. `api_spec_types.md`
3. `api_spec_errors.md`
4. 各模块 `api_spec_*.md`

## 文档说明

- 本规范为 **Contract Only**：仅定义对外可观测的端点/字段/类型/错误码/事件；实现细节、自检与工程化约束不在本规范范围
- 通用约定（时间格式、分页/幂等、Token 语义、`@auth-only` 等）统一见 `api_spec_base.md`，通用类型统一见 `api_spec_types.md`

## 非目标（Out of scope）

以下能力**不在本契约范围**内，本规范不为其定义端点：

- 支付/交易能力
- 端侧离线同步算法
- 消息全文搜索

## 预留（Reserved / 未实现）

以下能力**当前不实现**，但契约为其预留扩展点，待未来版本落地。客户端不应依赖这些端点/字段的存在：

- **消息送达回执（delivered）**：当前只到"已读"（`is_read`/`read_count`）。WS 消息相关 payload 预留 `delivered_to?: string[]` 字段位（默认不返回），未来实现送达回执时启用。
- **第三方登录（OAuth）**：当前仅支持账号密码 + 游客。`api_spec_auth.md` 预留 `auth/oauth/{provider}/callback` 端点签名（标记 `@planned`），provider 枚举 `'wechat' | 'qq' | 'apple'`，待集成第三方 IdP 时实现。
- **多端实时同步（self-sync）**：单设备单会话模型下，跨设备状态同步通过冷启动 `system/bootstrap` + HTTP 补拉实现。未来如需实时多端同步，将作为独立 WS 协议版本重新设计（不在此版本预留具体事件形状）。
- **邮箱/手机号变更**：`users/email/request_change`、`users/email/confirm_change`、`users/phone/request_change`、`users/phone/confirm_change`——依赖邮件/短信验证码发送服务，待服务接入后实现。
- **Feed 负反馈**：`feed/dismiss`——标记内容为"不感兴趣"，用于改善推荐质量。推荐算法优化为后续迭代重点，当前阶段优先完成核心 Feed 流。
- **综合搜索聚合**：`search/all` 已移除（原设计为一次请求返回三类结果首屏）。客户端可分别调用 `users/search`、`groups/search`、`moments/search` 实现同等效果，各自独立分页更灵活。

## 快速索引（按路径前缀）

| 路径前缀 | 对应文档 |
|---------|---------|
| `/api/v1/auth/` | `api_spec_auth.md` |
| `/api/v1/users/` | `api_spec_user.md` |
| `/api/v1/moments/` | `api_spec_moment.md` |
| `/api/v1/comments/` | `api_spec_moment.md` |
| `/api/v1/reactions/` | `api_spec_reaction.md` |
| `/api/v1/private_chats/` | `api_spec_chat.md` |
| `/api/v1/private_messages/` | `api_spec_chat.md` |
| `/api/v1/group_messages/` | `api_spec_chat.md` |
| `/api/v1/groups/` | `api_spec_group.md` |
| `/api/v1/feed/` | `api_spec_feed.md` |
| `/api/v1/search/` | `api_spec_search.md` |
| `/api/v1/map/` | `api_spec_map.md` |
| `/api/v1/share/` | `api_spec_share.md` |
| `/api/v1/media/` | `api_spec_media.md` |
| `/api/v1/reports/` | `api_spec_report.md` |
| `/api/v1/notify/` | `api_spec_notify.md` |
| `/api/v1/system/` | `api_spec_system.md` |
| `/api/v1/admin/` | `api_spec_admin_core.md` + `api_spec_admin_moderation.md` |
