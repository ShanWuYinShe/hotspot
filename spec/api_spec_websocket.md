# HotSpot API - WebSocket 实时事件定义

> 本文档定义 HotSpot WebSocket 实时事件、连接与鉴权规范、订阅机制等。

---

## 快速概览

### 我能订阅什么？

| 订阅类型 | 订阅参数 | 数量限制 | 说明 |
|---------|---------|---------|------|
| **用户状态** | `{type: 'user', ids: [user_id...]}` | 最多 500 个 | 按场景订阅用户上线/离线/输入状态，离开场景立即取消 |
| **群聊消息** | `{type: 'group', ids: [group_id...]}` | 最多 100 个 | 按场景订阅群消息/成员变动/群管理事件，离开场景立即取消 |

### 服务端会推送什么？

#### 自动推送（无需订阅）

| 分类 | 事件 | 说明 |
|-----|------|------|
| **连接管理** | `ConnectionEstablished` | 推送给当前连接，告知连接 ID、服务端时间与心跳配置 |
| | `Pong` | 推送给当前连接的心跳响应 |
| | `Error` | 推送给当前连接的错误通知（severity 指导是否需要重连） |
| | `SubscribeAck` | 推送给当前连接的订阅操作响应 |
| **私聊消息** | `PrivateMessageNew` | 推送给消息接收者 + 发送者的其他在线设备 |
| | `PrivateMessageEdited` | 推送给私聊对端 + 发送者的其他在线设备 |
| | `PrivateMessageRecalled` | 推送给私聊对端 + 发送者的其他在线设备 |
| | `PrivateMessageRead` | 推送给消息另一方 + 已读方的其他在线设备 |
| **社交通知** | `NewFollower` | 推送给被关注者，告知是谁关注了 Ta |
| | `MutualFollow` | 推送给刚形成互相关注关系的用户本身 |
| **瞬间互动** | `MomentLiked` | 推送给瞬间作者，告知谁点赞了 |
| | `MomentCommented` | 推送给瞬间作者，附评论内容摘要 |
| | `CommentLiked` | 推送给被点赞的评论作者 |
| | `CommentReply` | 推送给被回复的评论作者 |
| | `MessageMentioned` | 推送给被 @ 的用户（瞬间或评论中被提及） |
| **群组邀请** | `GroupInviteReceived` | 推送给被邀请加入群组的用户 |
| | `GroupInviteResponse` | 推送给邀请发起者，告知邀请被接受 |
| | `GroupApplicationReceived` | 推送给群主/管理员，告知有新的入群申请 |
| | `GroupApplicationResponse` | 推送给申请加入群的用户，告知申请被接受 |
| **用户** | `UserProfileUpdated` | 推送给正在与其私聊的对端以及其当前所在群聊的订阅者 |
| **系统** | `SystemAnnouncement` | 推送给所有在线连接的系统公告 |
| | `AccountBanned` | 推送给被封禁账号的当前连接，客户端收到后应立即登出 |

#### 订阅用户状态后推送

| 事件 | 说明 |
|-----|------|
| `UserOnline` | 用户上线 |
| `UserOffline` | 用户离线 |
| `PrivateTyping` | 用户正在输入（仅私聊对端可见） |

#### 订阅群聊后推送

| 分类 | 事件 | 说明 |
|-----|------|------|
| **消息** | `GroupMessageNew` | 新消息 |
| | `GroupMessageEdited` | 消息被编辑 |
| | `GroupMessageRecalled` | 消息被撤回 |
| | `GroupTyping` | 成员正在输入 |
| **成员变动** | `GroupMemberJoined` | 新成员加入 |
| | `GroupMemberLeft` | 成员主动离开 |
| | `GroupMemberRemoved` | 成员被移除（踢出/封禁） |
| | `GroupMemberMuted` | 成员被禁言 |
| | `GroupMemberUnmuted` | 成员被解除禁言 |
| **群管理** | `GroupAnnouncementCreated` | 新群公告发布 |
| | `GroupUpdated` | 群组信息更新（名称/头像/描述等所有可修改字段） |
| | `GroupDismissed` | 群组解散 |

> **多端同步（self-sync）**：当前不提供专用的多设备状态同步事件。单设备单会话模型下（见 `api_spec_auth.md`），同一用户在同一设备上只有一个活跃 session，跨设备状态同步通过冷启动 `system/bootstrap` + 各域 HTTP API 补拉实现。未来如需实时多端同步，将作为独立协议版本重新设计（不在此版本预留具体事件形状，避免预留形状与未来需求不匹配）。

### 客户端能发送什么？

| 事件 | 说明 |
|-----|------|
| `Auth` | 鉴权（连接后 5 秒内必须发送） |
| `Ping` | 心跳 |
| `SetSubscriptions` | 设置订阅（覆盖式） |
| `PrivateTyping` | 发送私聊输入状态 |
| `GroupTyping` | 发送群聊输入状态 |

---

## 连接与鉴权

### 连接端点

| 端点 | 协议 | 说明 |
|------|------|------|
| `/ws/v1` | `ws://` / `wss://` | WebSocket v1 连接端点（所有客户端统一使用） |

- WebSocket 端点独立于 HTTP API 路径（不在 `/api/v1/` 前缀下），但使用相同的版本号策略
- 版本号与 HTTP API 版本对齐：`/api/v1/` 对应 `/ws/v1`，未来 API v2 对应 `/ws/v2`
- Web 端：浏览器通过同源策略自动携带 Cookie 完成鉴权
- App 端：连接建立后通过 Auth 事件携带 access_token

### 设计理念

- **定位**：WebSocket 只负责"实时推送增量"（新消息/状态变更/提醒等）。
- **协作模式**：冷启动/重连后的历史与补拉一律通过 HTTP API；WebSocket 仅推送实时事件。

### 依赖说明

以下类型定义于 `api_spec_types.md`，本模块事件负载直接引用：
- `UserInfo`、`ClientType`、`MessageType`
- `MessageMedia`、`MessageReplyTo`
- `GroupInfo`、`GroupRole`、`GroupJoinMethod`、`GroupAnnouncementItem`
- `SystemAnnouncementItem`、`MomentSummary`
- `FollowSource`、`UserSettings`、`Location`

- 错误码定义于 `api_spec_errors.md`
- 所有事件类型和负载结构必须严格遵循本规范定义

### 鉴权机制

- Web 端：使用 HttpOnly Cookie 中的 `access_token` 完成鉴权，`Auth` 事件不携带 token。
- App 端：`Auth` 事件必须携带 `access_token`。
- 连接建立后，客户端必须在 **5 秒内**发送 `Auth`；鉴权通过后服务端推送 `ConnectionEstablished`（含 `resume_token` 与 `server_seq`），并允许后续订阅/业务事件。
- 鉴权失败或超时：服务端发送 `Error` 事件后断开连接。**断开使用标准 WS close code**（鉴权失败用 `4001`，详见下方「Close Code 规范」）——`Error` 事件 payload 内的 `code` 字段（如 `20000`）是**应用层错误码**，不作为 WS close code 使用。
- Token 过期：服务端断开（close `4001`）；客户端刷新 token 后重连并重新鉴权。

### Close Code 规范（契约级）

WebSocket close code 受 RFC 6455 约束（范围 1000-4999），应用层错误码（定义于 `api_spec_errors.md`，如 `20000`/`20100` 等）**不得**作为 close code 使用。本协议使用的 close code：

| Close Code | 含义 | 客户端行为 |
|------------|------|-----------|
| `1000` | 正常关闭（服务端主动优雅关闭） | 不自动重连 |
| `1008` | Policy Violation（协议违规，如发送未鉴权事件） | 记录日志，不自动重连 |
| `4001` | 认证失败/Token 失效/会话被驱逐（`SessionRevoked`） | 刷新 token 或重新登录后重连 |
| `4002` | 限流（连接级或订阅级超限） | 按 `Retry-After` 等待后重连 |
| `1011` | 服务端内部错误 | 指数退避重连 |

> 说明：`Error` 事件（见下方）的 payload `code` 字段仍使用 `api_spec_errors.md` 的应用层错误码（2xxxx 段），用于客户端精确的错误处理与提示；它与 close code 是两个独立维度。

### 重连恢复策略（契约级要求）

#### 1. 重连流程（resume token 优先）

- 重连成功后必须重新发送 `Auth`，可携带上一次连接的 `resume_token`（来自 `ConnectionEstablished`）。
- **resume_token 有效**（服务端短时状态未过期，TTL ≈ 2 × ping_timeout）：服务端推送断线期间的增量事件（按 `server_seq` 续传），客户端无需全量重放订阅、无需 HTTP 补拉。订阅状态由服务端从 resume 状态恢复。
- **resume_token 失效/未携带**（超过 TTL，或服务端不可恢复）：回退到全量恢复模式——客户端使用 `SetSubscriptions`（`operation: 'set'`）做全量重放，客户端本地维护订阅状态为权威；并按下方「数据补拉」策略补拉历史。

> **实现约束（生产部署必需）**：resume token 要求服务端为每个连接缓存 TTL 内按 `server_seq` 排序的出站事件队列。WS 服务多实例部署时，用户重连可能落到与断线时不同的实例——**必须使用跨实例共享的事件存储**（Redis Streams / Kafka 等），单实例内存方案会导致 resume 跨实例失效。具体存储选型非契约（实现中立），但"跨实例可恢复"是 resume 语义的硬前提，不满足则等同于未实现 resume（客户端将总是退化为全量重放）。
>
> **降级保证**：resume 是"补漏"优化，不是状态依赖。即使 resume 机制故障或存储不可用，客户端回退到全量重放仍能正常工作（graceful degradation）——功能不丢，仅重连开销略增。

#### 2. 数据补拉（resume 失效时的回退）

当 resume_token 失效时，客户端按端侧策略设置补拉阈值（如 10 秒）：超阈值时通过 HTTP `get_history` 按游标分页补拉，未超阈值时直接等待后续 WebSocket 推送。

#### 3. 去重规则

HTTP API 返回的数据可能与 WebSocket 推送的事件重复（resume 续传或补拉都可能产生）。去重方式按资源主键执行（参见 `api_spec_base.md`）：

- **持久化事件**：去重规则分两类：
  - **通知类事件**（关注/瞬间互动/群邀请申请/系统公告/reaction）：payload 携带 `notification_id`，客户端按 `notification_id` 去重——它与 HTTP `notifications/list` 返回的 `NotificationItem.notification_id` 为同一主键（见 api_spec_notification.md）。
  - **其它持久化事件**（消息等）：按业务主键或稳定组合键去重（如 `message_id`、`comment_id`）。resume 续传的事件携带单调递增的 `server_seq`，客户端也可按 seq 去重。
- **状态快照事件**：直接覆盖（保留最新状态），无需去重。

---

## 心跳机制

### 参数配置

心跳参数由服务端通过 `ConnectionEstablished.config` 动态下发，客户端不得硬编码。

| 参数 | 字段名 | 说明 |
|-----|--------|------|
| 心跳间隔 | `ping_interval` | 客户端按此间隔发送 `Ping`（秒） |
| Pong 超时 | `ping_timeout` | 客户端发送 `Ping` 后等待 `Pong` 的超时时间（秒） |
| 服务端超时 | - | 服务端内部配置，3 倍心跳间隔未收到任何消息则断开连接 |

**参考值**（实际以服务端配置为准）：
- `ping_interval`: 30 秒
- `ping_timeout`: 10 秒
- 服务端超时：90 秒

### Ping/Pong 事件格式

```typescript
// 客户端发送 Ping
type PingPayload = Record<string, never>;  // 空对象

// 服务端响应 Pong
type PongPayload = Record<string, never>;  // 空对象
```

**说明**：心跳机制只需保活连接，无需传递数据

### 客户端实现参考

- 连接建立并鉴权成功后，从 `ConnectionEstablished.config` 读取心跳配置
- 按 `ping_interval` 间隔发送 `Ping` 事件
- 发送 `Ping` 后启动 `ping_timeout` 超时监听
- 收到 `Pong` 后重置超时计时器
- 如果 `ping_timeout` 时间内未收到 `Pong`，视为连接异常，触发重连

### 重连参考策略（可按端侧调整）

- 可使用指数退避（如 1s、2s、4s...，最大 60s）
- 可加入随机抖动（jitter）降低重连风暴风险
- 可设置最大重连次数，超过后转为手动重连模式
- 重连成功后应重新发送 `Auth`，等待 `ConnectionEstablished` 并恢复订阅

### 重连触发条件

- Ping 超时（`ping_timeout` 时间内无 Pong）
- WebSocket 连接断开（onerror/onclose）
- 网络状态变化（online 事件）
- 前台切换（从后台回到前台）

### 服务端主动断开场景

以下场景服务端会主动断开 WebSocket 连接，客户端**不应自动重连**：

- **Token 失效**（被登出或撤销会话）：服务端发送 Error（severity=fatal），然后断开
- **账号被封禁**：服务端推送 AccountBanned 事件，然后断开
- **密码被修改**：非当前会话的 WebSocket 连接被断开（Error severity=fatal）

---

## 事件类型定义

### 服务端推送事件

**通用事件格式（强类型，契约级）**：
```typescript
// 统一事件 Envelope：服务端推送与客户端发送均使用该结构
// - type：事件类型（强约束为枚举）
// - payload：事件负载（按 type 决定结构；未在契约中声明的字段/事件禁止出现）
interface WebSocketEvent<TType = string, TPayload = unknown> {
  type: TType;
  payload: TPayload;
}

// 服务端推送事件：type 必须来自 ServerEventType；payload 为 unknown，具体由各事件接口约束
type ServerWebSocketEvent = WebSocketEvent<ServerEventType, unknown>;

// 客户端发送事件：type 必须来自 ClientEventType；payload 为 unknown，具体由各事件接口约束
type ClientWebSocketEvent = WebSocketEvent<ClientEventType, unknown>;
```

**消息可靠性与去重**：
- 基于 TCP：协议层保证消息不丢失、不乱序、不重复，应用层无需维护 seq。
- **持久化事件**：去重按事件性质——通知类事件按 `notification_id`（与 HTTP 通知 API 同主键），消息类事件按业务主键（`message_id` 等）。
- **状态快照事件**：表示最新状态（如在线状态、输入状态、成员变更），按 `user_id + 事件类型` 或 `group_id + user_id + 事件类型` 覆盖即可。

> 重连后的数据补拉与去重策略详见「连接与鉴权」中的「重连恢复策略」。

**WebSocket 通用类型定义**：
```typescript
// 服务端推送事件类型枚举（强制：WebSocketEvent.type 必须来自该枚举）
type ServerEventType =
  // 连接管理
  | "ConnectionEstablished" | "Pong"
  // 会话
  | "SessionRevoked"              // 当前连接的会话被驱逐/撤销（设备被踢），客户端须立即清本地态并断开
  // 用户状态
  | "UserOnline" | "UserOffline"
  // 关注关系
  | "NewFollower" | "MutualFollow"
  // 互动（含表情反应）
  | "ReactionAdded" | "ReactionRemoved"   // 表情反应增减（覆盖 message/moment/comment 三类目标）
  | "MomentLiked" | "MomentCommented" | "CommentLiked" | "CommentReply" | "MessageMentioned"
  // 聊天消息 - 私聊
  | "PrivateMessageNew" | "PrivateMessageEdited" | "PrivateMessageRecalled"
  | "PrivateMessageRead"
  | "PrivateTyping"               // 原 PrivateUserTyping，与客户端事件 PrivateTyping 对齐
  // 聊天消息 - 群聊
  | "GroupMessageNew" | "GroupMessageEdited" | "GroupMessageRecalled"
  | "GroupMessageReadCountUpdated"  // 群消息已读数变更（节流聚合推送，替代逐人已读广播）
  | "GroupTyping"                 // 原 GroupUserTyping，对齐命名
  // 群组管理
  | "GroupMemberJoined" | "GroupMemberLeft" | "GroupMemberRemoved"
  | "GroupMemberRoleChanged"      // 成员角色变更（升/降级 admin），补原缺失的实时角色刷新事件
  | "GroupMemberMuted" | "GroupMemberUnmuted"
  | "GroupInviteReceived" | "GroupInviteRevoked" | "GroupInviteExpired" | "GroupInviteResponse"
  | "GroupApplicationReceived" | "GroupApplicationCancelled" | "GroupApplicationExpired" | "GroupApplicationRejected" | "GroupApplicationResponse"
  | "GroupAnnouncementCreated"
  | "GroupUpdated" | "GroupDismissed"
  // 用户
  | "UserProfileUpdated"
  // 系统
  | "SystemAnnouncement" | "AccountBanned"
  // 协议控制
  | "Error" | "SubscribeAck";

// 客户端发送事件类型枚举（强制：WebSocketEvent.type 必须来自该枚举）
type ClientEventType =
  | "Auth"
  | "Ping"
  | "SetSubscriptions"
  | "PrivateTyping" | "GroupTyping";

```

**连接管理事件**：
```typescript
interface ConnectionEstablishedPayload {
  connection_id: string;
  user_id: string;
  server_time: string;             // 服务端当前时间（UTC ISO 8601），客户端用于判断断开时长
  resume_token: string;            // 本次连接的恢复令牌；重连时透传给 Auth 以尝试增量恢复（TTL ≈ 2×ping_timeout）
  server_seq: number;              // 当前连接的单调递增序号基线；后续每个推送事件携带 server_seq 供续传/去重

  // 服务端配置（动态配置，客户端不得硬编码）
  config: {
    ping_interval: number;         // 心跳间隔（秒）
    ping_timeout: number;          // Pong 超时（秒）
    max_subscriptions: {
      user: number;                // 用户订阅上限
      group: number;               // 群组订阅上限
    };
  };
}
type ConnectionEstablishedEvent = { type: "ConnectionEstablished"; payload: ConnectionEstablishedPayload; };

// PongPayload 定义见上文「心跳机制」
type PongEvent = { type: "Pong"; payload: PongPayload; };

// 会话被驱逐/撤销（单设备单会话模型下，同设备新登录或 admin 撤销触发）
// 客户端收到后必须：清本地用户态 + 断开连接 + 不自动重连（提示"账号在其他设备登录"或"会话已失效"）
interface SessionRevokedPayload {
  reason: 'device_evicted' | 'revoked_by_user' | 'password_changed';
  revoked_at: string;   // UTC ISO 8601
}
type SessionRevokedEvent = { type: "SessionRevoked"; payload: SessionRevokedPayload; };
// 注：服务端推送此事件后以 close code 4001 断开连接（见「Close Code 规范」）

// 订阅操作类型（SetSubscriptions 与 SubscribeAck 共用）
// - subscribe: 增量添加（进入场景时订阅新的 ID）
// - unsubscribe: 增量移除（退出场景时取消订阅）
// - set: 全量覆盖（重连时同步完整订阅列表，ids 为空则清空该类型订阅）
type SubscriptionOperation = 'subscribe' | 'unsubscribe' | 'set';

// 订阅确认事件（服务端回复 SetSubscriptions）
// 订阅失败原因
type SubscriptionFailureReason =
  | 'invalid'         // 订阅项无效（格式错误、目标不存在等）
  | 'denied'          // 无权限（无权限订阅、已被拉黑等）
  | 'limit_exceeded'  // 超出订阅数量限制
  | 'error';          // 服务端错误（可重试）

interface FailedSubscriptionId {
  type: 'user' | 'group';        // 失败项的订阅类型（批量请求需明确类型）
  subscription_id: string;       // 失败的订阅目标 ID（user 类型为 user_id，group 类型为 group_id）
  reason: SubscriptionFailureReason;
}

interface SubscribeAckPayload {
  client_request_id: string;                          // 回传客户端请求ID
  operation: SubscriptionOperation;                  // 回传操作类型
  accepted: { type: 'user' | 'group'; ids: string[] }[];  // 已接受的订阅项（按 type 分组）
  code: number;                                      // 0表示成功，非0表示失败
  message?: string;                                  // 错误消息（失败时）
  failed_ids?: FailedSubscriptionId[];               // 失败的订阅项
}
type SubscribeAckEvent = { type: "SubscribeAck"; payload: SubscribeAckPayload; };

// code=0 表示完全成功（所有 items 均已处理且无 failed_ids）；
// 只要 failed_ids 非空（无论原因是 invalid、denied、limit_exceeded 还是 error），code 必须非 0，使用 20100（订阅错误）。
// 非 0 时，已成功处理的订阅项仍然生效，客户端需根据 failed_ids 回滚本地状态中失败的部分。
```

**在线状态事件**：
```typescript
// 说明：客户端已通过 UserInfo 缓存了用户的 nickname/avatar_url 等信息，
// 在线/离线事件仅需 user_id 标识用户 + 时间戳，客户端据此更新状态指示器。
interface UserOnlinePayload {
  user_id: string;
  online_at: string;
}
type UserOnlineEvent = { type: "UserOnline"; payload: UserOnlinePayload; };

interface UserOfflinePayload {
  user_id: string;
  offline_at: string;           // 离线时间（即最后在线时间）
}
type UserOfflineEvent = { type: "UserOffline"; payload: UserOfflinePayload; };
```

**在线/输入状态事件触发约定**：

- **UserOnline**
  - 触发：用户任意设备建立 WebSocket 连接并完成 `Auth` 鉴权后，服务端将该用户标记为在线
  - 推送：推送给所有已订阅该用户状态（`type: 'user'`）的订阅者
  - 多设备：同一用户多设备在线只会维持"在线"状态；重复连接不应反复产生"离线/上线抖动"

- **UserOffline**
  - 触发：用户所有 WebSocket 连接都断开后（连接计数归零），服务端将其标记为离线
  - 推送：推送给所有已订阅该用户状态（`type: 'user'`）的订阅者
  - 离线时间：`offline_at` 为服务端判定离线的时间（UTC ISO 8601）

- **PrivateTyping / GroupTyping**
  - 触发（客户端侧）：
    - 私聊：发送 `PrivateTyping { target_user_id, is_typing }`
    - 群聊：发送 `GroupTyping { group_id, is_typing }`
  - 触发（服务端侧）：
    - `PrivateTyping`：推送给已订阅该用户状态（`type: 'user'`）且为私聊对端的订阅者
    - `GroupTyping`：推送给订阅该群聊的成员
  - Typing 状态属于"瞬时状态"，不落库、不补发历史
  - **隐私保护**：通过订阅机制和 target_user_id 限定保护隐私，无需额外权限检查
  - **节流**：服务端权威节流（默认 3 秒内同一用户重复 typing 只推送一次）；客户端建议"不超过 3 秒发一次"，以服务端为准
  
  - **节流策略（性能优化，防止广播风暴）**：
    
    **客户端发送限制**：
    - 最多每 3 秒发送一次 `PrivateTyping`/`GroupTyping`
    - 使用防抖（debounce）：用户停止输入 3 秒后自动发送 `is_typing: false`
    - 切换聊天对象时立即发送 `is_typing: false` 到上一个对象
    
    **服务端推送限制**：
    - 3 秒内的重复输入状态自动过滤（相同用户+相同状态）
    - 群聊输入状态合并推送：同一群组多人同时输入时，合并为批量通知

- **PrivateMessageRead**
  - 触发：已读状态发生变化时触发
  - 推送目标：私聊对端 + 发送方的其他设备（多端同步时）
  - 该事件用于即时交互反馈，不作为离线补偿依据

**关注关系事件**：
```typescript
// FollowSource 定义于 api_spec_types.md

// 命名约定：
// - actor: 执行操作的人（如关注者、点赞者、评论者），与 NotificationPayload.actor 语义一致
// - user: 事件主体（如加入群的人、被操作的人），不一定是操作发起者
//
// 持久化通知类事件统一携带 notification_id：
// - 下列"关注/瞬间互动/群邀请申请/系统公告"事件均会落库为通知记录（见 api_spec_notification.md）
// - payload 首字段 notification_id 与 HTTP notifications/list 返回的 NotificationItem.notification_id 为同一主键
// - 客户端据此在 WS 实时路径与 HTTP 补拉路径间精确去重（按 notification_id），无需依赖业务组合键
// - 与瞬时状态事件（UserOnline/Typing 等，无 notification_id）形成明确区分

interface NewFollowerPayload {
  notification_id: string;
  actor: UserInfo;
  followed_at: string;
  
  // 关注来源（可选，让用户了解被关注的原因）
  source?: FollowSource;
  source_moment_id?: string;  // 如果是通过瞬间关注，提供瞬间 ID
}
type NewFollowerEvent = { type: "NewFollower"; payload: NewFollowerPayload; };

interface MutualFollowPayload {
  notification_id: string;
  actor: UserInfo;
  mutual_at: string;
}
type MutualFollowEvent = { type: "MutualFollow"; payload: MutualFollowPayload; };
```

**生活瞬间互动事件**：
```typescript
// MomentSummary 定义于 api_spec_types.md

// 表情反应（统一 message/moment/comment 三类目标）
// ReactionTargetType / Reaction 定义于 api_spec_types.md
interface ReactionAddedPayload {
  notification_id: string;          // 仅当 actor ≠ 资源作者时携带（自我反应不通知）；自反应时省略
  target_type: ReactionTargetType;  // 'message' | 'moment' | 'comment'
  target_id: string;                // message_id / moment_id / comment_id
  emoji: string;
  actor: UserInfo;
  reacted_at: string;
}
type ReactionAddedEvent = { type: "ReactionAdded"; payload: ReactionAddedPayload; };

interface ReactionRemovedPayload {
  // 反应移除不产生通知（非持久化通知），无 notification_id
  target_type: ReactionTargetType;
  target_id: string;
  emoji: string;
  actor: UserInfo;
  removed_at: string;
}
type ReactionRemovedEvent = { type: "ReactionRemoved"; payload: ReactionRemovedPayload; };

interface MomentLikedPayload {
  notification_id: string;
  moment_id: string;
  moment_summary: MomentSummary;
  actor: UserInfo;
  liked_at: string;
}
type MomentLikedEvent = { type: "MomentLiked"; payload: MomentLikedPayload; };

interface MomentCommentedPayload {
  notification_id: string;
  moment_id: string;
  moment_summary: MomentSummary;
  comment_id: string;
  actor: UserInfo;
  content: string;              // 评论完整内容（对应 Notification 的 comment_preview 为截断版本）
  commented_at: string;
}
type MomentCommentedEvent = { type: "MomentCommented"; payload: MomentCommentedPayload; };

interface CommentLikedPayload {
  notification_id: string;
  comment_id: string;
  moment_id: string;
  moment_summary: MomentSummary;
  comment_preview: string;
  actor: UserInfo;
  liked_at: string;
}
type CommentLikedEvent = { type: "CommentLiked"; payload: CommentLikedPayload; };

interface CommentReplyPayload {
  notification_id: string;
  moment_id: string;
  moment_summary: MomentSummary;
  comment_id: string;
  replied_comment_preview: string;
  reply_id: string;
  actor: UserInfo;
  content: string;
  replied_at: string;
}
type CommentReplyEvent = { type: "CommentReply"; payload: CommentReplyPayload; };

interface MessageMentionedPayload {
  notification_id: string;
  target_type: 'moment' | 'comment';
  moment_id: string;
  comment_id?: string;
  moment_summary: MomentSummary;
  comment_preview?: string;
  actor: UserInfo;
  mentioned_at: string;
}
type MessageMentionedEvent = { type: "MessageMentioned"; payload: MessageMentionedPayload; };
```

**私聊消息事件**:

```typescript
interface PrivateMessageNewPayload {
  message_id: string;
  target_user_id: string; // 接收者ID (对于发送者多端同步=接收方ID)
  sender: UserInfo;       // 统一使用 UserInfo
  type: MessageType;
  content?: string;       // 文本内容
  share_code?: string;    // share 类型消息
  reply_to?: MessageReplyTo;
  // 媒体信息
  media?: MessageMedia;
  sent_at: string;
}
type PrivateMessageNewEvent = { type: "PrivateMessageNew"; payload: PrivateMessageNewPayload; };

interface PrivateMessageEditedPayload {
  message_id: string;
  target_user_id: string;
  content: string;
  edited_at: string;
}
type PrivateMessageEditedEvent = { type: "PrivateMessageEdited"; payload: PrivateMessageEditedPayload; };

interface PrivateMessageRecalledPayload {
  message_id: string;
  target_user_id: string;
  recalled_by: UserInfo;  // 统一使用 UserInfo
  recalled_at: string;
}
type PrivateMessageRecalledEvent = { type: "PrivateMessageRecalled"; payload: PrivateMessageRecalledPayload; };

interface PrivateMessageReadPayload {
  // 接收方天然知道是私聊对端读了消息，无需冗余 target_user_id/reader_id（原三 ID 设计冗余，已合并）
  up_to_message_id: string;  // 标记已读至此消息（含）
  read_at: string;           // UTC ISO 8601，读取时间
}
type PrivateMessageReadEvent = { type: "PrivateMessageRead"; payload: PrivateMessageReadPayload; };

interface PrivateTypingPayload {
  actor: UserInfo;     // 正在打字的用户（统一用 actor: UserInfo 表达"动作发起者"，与通知类事件一致）
  is_typing: boolean;
}
type PrivateTypingEvent = { type: "PrivateTyping"; payload: PrivateTypingPayload; };
```

**群聊消息事件**:

```typescript
interface GroupMessageNewPayload {
  message_id: string;
  group_id: string;
  sender: UserInfo;
  type: MessageType;
  content?: string;
  share_code?: string;    // share 类型消息
  reply_to?: MessageReplyTo;
  mentions?: string[];     // user_id
  // 媒体信息
  media?: MessageMedia;
  sent_at: string;
}
type GroupMessageNewEvent = { type: "GroupMessageNew"; payload: GroupMessageNewPayload; };

interface GroupMessageEditedPayload {
  message_id: string;
  group_id: string;
  content: string;
  edited_at: string;
}
type GroupMessageEditedEvent = { type: "GroupMessageEdited"; payload: GroupMessageEditedPayload; };

interface GroupMessageRecalledPayload {
  message_id: string;
  group_id: string;
  recalled_by: UserInfo;  // 撤回操作者（群管理员/群主/发送者）
  recalled_at: string;
}
type GroupMessageRecalledEvent = { type: "GroupMessageRecalled"; payload: GroupMessageRecalledPayload; };

// 群消息已读数变更（节流聚合推送）
// 解决"群 read_count 不逐人广播"导致的客户端显示滞后：服务端对同一消息的多次已读事件做时间窗口聚合
// （建议窗口 3-5 秒），窗口结束时推送一次最新 read_count，而非每个成员读取都广播
interface GroupMessageReadCountUpdatedPayload {
  group_id: string;
  message_id: string;
  read_count: number;        // 当前已读人数（最终一致，见 api_spec_base.md「计数器一致性」）
  updated_at: string;        // UTC ISO 8601，本次聚合窗口的结束时间
}
type GroupMessageReadCountUpdatedEvent = { type: "GroupMessageReadCountUpdated"; payload: GroupMessageReadCountUpdatedPayload; };
// 推送目标：订阅了该 group 的成员（消息发送者尤其关心自己消息的已读数）
// 注：客户端不应将此事件用于精确计数，仅用于 UI 的"已有 N 人已读"展示刷新

interface GroupTypingPayload {
  group_id: string;
  actor: UserInfo;     // 正在打字的成员（统一 actor: UserInfo）
  is_typing: boolean;
}
type GroupTypingEvent = { type: "GroupTyping"; payload: GroupTypingPayload; };
```

**群组事件**：

```typescript
interface GroupMemberJoinedPayload {
  group_id: string;
  user: UserInfo;
  role: GroupRole;  // 引用 api_spec_types.md
  member_count: number; // 当前成员总数
  joined_at: string;
}
type GroupMemberJoinedEvent = { type: "GroupMemberJoined"; payload: GroupMemberJoinedPayload; };

interface GroupInviteReceivedPayload {
  notification_id: string;
  invite_id: string;
  group: GroupInfo;  // 引用 api_spec_types.md
  inviter: UserInfo;
  message?: string;
  created_at: string;
}
type GroupInviteReceivedEvent = { type: "GroupInviteReceived"; payload: GroupInviteReceivedPayload; };

interface GroupInviteResponsePayload {
  // 仅邀请被接受时推送（拒绝不推送）。原 status:'accepted' 单值字段已移除——事件名即语义，无需冗余枚举
  notification_id: string;
  invite_id: string;
  group_id: string;
  user: UserInfo;       // 接受邀请的用户
  responded_at: string;
}
type GroupInviteResponseEvent = { type: "GroupInviteResponse"; payload: GroupInviteResponsePayload; };

// 邀请被发起方（管理员）主动撤销 → 推送给受邀人
interface GroupInviteRevokedPayload {
  notification_id: string;
  invite_id: string;
  group_id: string;
  revoked_by: UserInfo;   // 撤销邀请的管理员
  revoked_at: string;
  reason?: string;
}
type GroupInviteRevokedEvent = { type: "GroupInviteRevoked"; payload: GroupInviteRevokedPayload; };

// 邀请过期（超 expires_at 自动失效）→ 推送给受邀人
interface GroupInviteExpiredPayload {
  notification_id: string;
  invite_id: string;
  group_id: string;
  expired_at: string;
}
type GroupInviteExpiredEvent = { type: "GroupInviteExpired"; payload: GroupInviteExpiredPayload; };

interface GroupApplicationReceivedPayload {
  notification_id: string;
  application_id: string;
  group_id: string;
  applicant: UserInfo;
  message?: string;
  created_at: string;
}
type GroupApplicationReceivedEvent = { type: "GroupApplicationReceived"; payload: GroupApplicationReceivedPayload; };

interface GroupApplicationResponsePayload {
  // 仅申请被接受时推送。原 status:'accepted' 单值字段已移除
  notification_id: string;
  application_id: string;
  group: GroupInfo;
  responded_at: string;
}
type GroupApplicationResponseEvent = { type: "GroupApplicationResponse"; payload: GroupApplicationResponsePayload; };

// 申请被发起方（申请人）主动取消 → 推送给群管理员
interface GroupApplicationCancelledPayload {
  notification_id: string;
  application_id: string;
  group_id: string;
  applicant: UserInfo;
  cancelled_at: string;
  reason?: string;
}
type GroupApplicationCancelledEvent = { type: "GroupApplicationCancelled"; payload: GroupApplicationCancelledPayload; };

// 申请过期（超 expires_at 自动失效）→ 推送给群管理员
interface GroupApplicationExpiredPayload {
  notification_id: string;
  application_id: string;
  group_id: string;
  applicant: UserInfo;
  expired_at: string;
}
type GroupApplicationExpiredEvent = { type: "GroupApplicationExpired"; payload: GroupApplicationExpiredPayload; };

// 申请被群管理员拒绝 → 推送给申请人（对应 NotificationType='group_application_rejected'，含 reason）
interface GroupApplicationRejectedPayload {
  notification_id: string;
  application_id: string;
  group: GroupInfo;
  rejected_by: UserInfo;   // 处理管理员（申请人可见管理员身份，与举报场景不同——群申请是双向契约）
  rejected_at: string;
  reason?: string;
}
type GroupApplicationRejectedEvent = { type: "GroupApplicationRejected"; payload: GroupApplicationRejectedPayload; };

// 群公告发布：本质是持久化通知，补 notification_id 以支持与 notifications/list 去重
interface GroupAnnouncementCreatedPayload {
  notification_id: string;      // 与 notifications/list 的 NotificationItem.notification_id 同主键
  announcement: GroupAnnouncementItem;
}
type GroupAnnouncementCreatedEvent = { type: "GroupAnnouncementCreated"; payload: GroupAnnouncementCreatedPayload; };

// 成员离开（自愿离开）
interface GroupMemberLeftPayload {
  group_id: string;
  user: UserInfo;
  left_at: string;
  member_count: number;        // 剩余成员总数
}
type GroupMemberLeftEvent = { type: "GroupMemberLeft"; payload: GroupMemberLeftPayload; };

// 成员被移除（被踢出/被封禁）
interface GroupMemberRemovedPayload {
  group_id: string;
  user: UserInfo;
  removed_by: UserInfo;         // 操作者
  reason?: 'kicked' | 'banned'; // 移除原因
  removed_at: string;
  member_count: number;
}
type GroupMemberRemovedEvent = { type: "GroupMemberRemoved"; payload: GroupMemberRemovedPayload; };

// 成员角色变更（升/降级 admin）——补原缺失的实时角色刷新事件
// GroupRole 定义于 api_spec_types.md（'owner' | 'admin' | 'member'）
interface GroupMemberRoleChangedPayload {
  group_id: string;
  user: UserInfo;
  old_role: GroupRole;
  new_role: GroupRole;
  changed_by: UserInfo;         // 操作者（owner 或 admin）
  changed_at: string;
}
type GroupMemberRoleChangedEvent = { type: "GroupMemberRoleChanged"; payload: GroupMemberRoleChangedPayload; };

// 群组信息更新
// 字段可见性：服务端按订阅者的群角色决定实际推送哪些字段，与 HTTP GroupItem 可见性规则一致（见 api_spec_types.md）：
//   - 所有群成员可见：name/avatar_url/description/tags/location/has_password
//   - 仅群管理员（owner/admin）可见：join_method/is_searchable/max_members
//   非管理员成员不会收到 join_method/is_searchable/max_members 的变更（服务端过滤）
interface GroupUpdatedPayload {
  group_id: string;
  updated_fields: {            // 只包含变更的字段；按上方可见性规则过滤后推送
    name?: string;
    avatar_url?: string;
    description?: string;
    tags?: string[];
    join_method?: GroupJoinMethod;   // @group-admin-only：仅推送给群管理员订阅者
    is_searchable?: boolean;         // @group-admin-only：仅推送给群管理员订阅者
    max_members?: number;            // @group-admin-only：仅推送给群管理员订阅者
    has_password?: boolean;          // @member-only：推送给群成员订阅者
    location?: Location;
  };
  updated_by: UserInfo;        // 操作者（统一用嵌套 UserInfo，与 types.md「对用户引用」约定一致，原裸 user_id 已修正）
  updated_at: string;
}
type GroupUpdatedEvent = { type: "GroupUpdated"; payload: GroupUpdatedPayload; };

// 群组解散（群主主动解散或因违规被解散）
// 注意：系统管理员解散群组时，服务端必须向所有在线群成员推送此事件（无论客户端是否主动订阅该群组频道）。
// 设计说明：不暴露操作者身份（deleted_by），不暴露系统管理员的存在。
// 'policy_violation' 表示因违反平台规则被解散，不透露具体操作方式。
// 详细操作记录保留在审计日志中（仅管理员可见）。
interface GroupDismissedPayload {
  group_id: string;
  dismissed_at: string;
  reason?: 'owner_dismissed' | 'policy_violation';  // 解散原因
}
type GroupDismissedEvent = { type: "GroupDismissed"; payload: GroupDismissedPayload; };

// 群组成员禁言（推送给被操作用户 + 群组订阅者）
interface GroupMemberMutedPayload {
  group_id: string;
  user: UserInfo;               // 被禁言的用户
  muted_by: UserInfo;            // 操作者
  muted_until: string;          // 禁言截止时间
  duration_minutes: number;
  muted_at: string;
}
type GroupMemberMutedEvent = { type: "GroupMemberMuted"; payload: GroupMemberMutedPayload; };

// 群组成员解除禁言（推送给被操作用户 + 群组订阅者）
interface GroupMemberUnmutedPayload {
  group_id: string;
  user: UserInfo;               // 被解除禁言的用户
  unmuted_by: UserInfo;          // 操作者
  unmuted_at: string;
}
type GroupMemberUnmutedEvent = { type: "GroupMemberUnmuted"; payload: GroupMemberUnmutedPayload; };

```

**用户事件**：
```typescript
// 用户资料变更（推送给：正在与其私聊的用户 + 其当前所在群聊的其他订阅者）
// 属于自动推送（无需订阅），推送范围天然受限于活跃会话，维护成本低
interface UserProfileUpdatedPayload {
  user: UserInfo;               // 更新后的用户基础信息
  updated_at: string;
}
type UserProfileUpdatedEvent = { type: "UserProfileUpdated"; payload: UserProfileUpdatedPayload; };
```

**系统事件**：
```typescript
// SystemAnnouncementItem 定义于 api_spec_types.md
interface SystemAnnouncementPayload {
  notification_id: string;
  announcement_id: string;
  title: string;
  content: string;
  severity: AnnouncementSeverity;
  action_url?: string;
  created_at: string;
}
type SystemAnnouncementEvent = { type: "SystemAnnouncement"; payload: SystemAnnouncementPayload; };

// 错误严重程度
type ErrorSeverity = 
  | 'fatal'        // 致命错误：必须断开连接（token 过期、账号被封）
  | 'recoverable'  // 可恢复错误：可以重试（临时错误、限流）
  | 'warning';     // 警告：仅提示（订阅失败、权限不足）

interface ErrorPayload {
  code: number;          // 错误码：WebSocket 业务期错误优先使用 20000-20999，幂等相关错误使用 22000-22099（详见 api_spec_errors.md）
                         // 禁止使用 HTTP API 的 10000-19999 作为 WebSocket Error 事件 code
  message: string;       // 错误消息（用户友好，不泄露内部细节）
  
  // 错误分级和处理指导
  severity: ErrorSeverity;       // 错误严重程度
  retryable: boolean;            // 是否可重试
  retry_after?: number;          // 建议重试延迟（秒），仅当 retryable=true 时有效
  
  // 上下文信息（帮助调试）
  context?: {
    operation?: string;          // 失败的操作（如 "subscribe", "send_message"）
    resource_type?: string;      // 资源类型（如 "user", "group"）
    resource_id?: string;        // 资源ID
  };
}
type ErrorEvent = { type: "Error"; payload: ErrorPayload; };

// 账号被封禁（自动推送给被封禁用户，客户端收到后应立即登出）
interface AccountBannedPayload {
  notification_id: string;     // 持久化通知主键（补：原缺失，账号封禁属持久化事件，应支持去重）
  reason: string;              // 封禁原因
  duration_days: number;       // 封禁时长（天），0=永久
  banned_at: string;
  banned_until?: string;       // 封禁到期时间（UTC ISO 8601）；不返回/省略表示永久封禁。客户端据此提示"何时可重新登录"，无需自行计算 banned_at + duration_days
}
type AccountBannedEvent = { type: "AccountBanned"; payload: AccountBannedPayload; };
```

**多端同步事件（⏳ 延期实现）**：

> payload 定义见文末「[延期实现事件（Roadmap）](#延期实现事件roadmap)」小节。这些事件当前不实现，客户端通过冷启动 HTTP API 补拉实现多端同步。

**错误码定义**：

错误码定义以 `api_spec_errors.md` 为准（精简后每个类型仅一个错误码，具体场景通过 `message` 描述）。

| 错误码 | 类型 | 严重性 | message 示例 |
|-------|------|--------|-------------|
| `20000` | 连接/协议错误 | Fatal | "鉴权失败"、"心跳超时"、"消息格式错误"、"协议不兼容" |
| `20100` | 订阅错误 | Recoverable/Warning | "订阅频道不存在"、"订阅权限不足"、"订阅数量超出限制" |
| `20200` | 消息错误 | Warning | "消息内容过长"、"消息格式错误"、"消息发送频率过高" |
| `20300` | 限流 | Recoverable | "触发限流"、"请求过于频繁" |
| `20400` | 权限错误 | Warning | "无权限"、"已被拉黑"、"群组已解散" |
| `20500` | 服务端错误 | Recoverable | "内部错误"、"服务暂时不可用" |

**错误处理策略**：

| 严重性 | 客户端处理 | 重连行为 |
|-------|----------|---------|
| Fatal | 断开连接，清除会话，提示用户重新登录 | 不重连 |
| Recoverable | 显示错误提示，按 retry_after 延迟后重试 | 保持连接 |
| Warning | 显示警告提示，不影响正常使用 | 保持连接 |

### 客户端发送事件

**连接管理**：
```typescript
interface AuthPayload {
  client_type: ClientType;
  access_token?: string;   // App 端必填，Web 端不传（使用 Cookie）
  device_id?: DeviceId;    // 设备标识（支撑单设备单会话模型；App 端必填，Web 端由 Cookie 自动携带）
  resume_token?: string;   // 可选：上次连接的 resume_token，用于尝试增量恢复（见「重连恢复策略」）
}
type AuthEvent = { type: "Auth"; payload: AuthPayload; };

// PingPayload 定义见上文「心跳机制」
type PingEvent = { type: "Ping"; payload: PingPayload; };

interface PrivateTypingPayload {
  target_user_id: string;
  is_typing: boolean;
}
type PrivateTypingEvent = { type: "PrivateTyping"; payload: PrivateTypingPayload; };

interface GroupTypingPayload {
  group_id: string;
  is_typing: boolean;
}
type GroupTypingEvent = { type: "GroupTyping"; payload: GroupTypingPayload; };
```

**订阅管理**：

WebSocket订阅机制提供用户状态订阅和群组订阅两种类型：

```typescript
// SubscriptionOperation 定义见上文「连接管理事件」

// SetSubscriptions - 订阅管理（批量结构，单请求可同时操作 user 和 group 两类订阅）
interface SetSubscriptionsPayload {
  client_request_id: string;     // 客户端生成，用于与 SubscribeAck 对齐
  operation: SubscriptionOperation;
  items: SubscriptionItem[];     // 订阅项列表，可混合 user/group 类型，单次请求覆盖多类订阅
}

interface SubscriptionItem {
  type: 'user' | 'group';
  ids: string[];                 // 订阅目标 ID 列表（user_id 或 group_id）
}
type SetSubscriptionsEvent = { type: "SetSubscriptions"; payload: SetSubscriptionsPayload; };
```

---

## 订阅机制详解

### 推送模型

WebSocket 推送分为**自动推送**和**订阅推送**两种模式（详见「快速概览」）。

**设计理念**：
- **自动推送**：基于社交关系的事件（私聊消息、关注通知等），无需额外订阅
- **订阅推送**：基于用户主动关注的实时状态（在线状态、输入状态等），需要订阅后推送
- **订阅粒度**：订阅"类别"（用户/群组），自动收到该类别下的所有相关事件
- **场景驱动**：订阅由客户端按当前使用场景显式设置，场景退出必须立即取消订阅
- **负载控制**：服务端不会默认订阅全部资源，未订阅不推送

### 订阅操作说明

**订阅方式**：

客户端使用 `SetSubscriptions` 管理订阅列表，支持三种操作：

- **`subscribe`（增量添加）**：进入场景时，发送新增的 ID 列表
- **`unsubscribe`（增量移除）**：退出场景时，发送要移除的 ID 列表
- **`set`（全量覆盖）**：重连后同步完整订阅列表，`ids` 为空则清空该类型订阅

**场景管理**：
- 进入场景：使用 `subscribe` 操作添加需要订阅的 ID
- 退出场景：使用 `unsubscribe` 操作移除不再需要的 ID
- 重连恢复：使用 `set` 操作发送完整的本地订阅列表，确保与服务端同步

客户端本地维护订阅状态，所有 `ids` 为 string 类型。

**隐私约束**：
- `type: 'user'` 的订阅必须校验目标用户的 `presence_visible` 设置。若目标用户将在线状态对外隐藏（`presence_visible=false`），服务端必须拒绝该订阅项，并在 `SubscribeAck` 的 `failed_ids` 中返回 `reason='denied'`，同时不推送任何 UserOnline/UserOffline 事件。
- `type: 'group'` 的订阅**必须校验成员资格**：仅当前已加入该群组的成员可订阅。非成员订阅返回 `reason='denied'`，不推送任何群消息/成员变动/`GroupUpdated`（含 `has_password`/`join_method` 等可见性受控字段）。这是群组事件可见性的前置门槛，确保 `has_password`（@member-only）与 `join_method`/`is_searchable`/`max_members`（@group-admin-only）不被非成员获取。

**类型约束**：
- 单次订阅请求仅允许一个 `type`，不同类型需分开请求

**订阅数量限制说明**：

| 订阅类型 | 数量限制 | 超出处理 |
|---------|---------|----------|
| 用户状态（`user`） | 最多 500 个 | 返回错误码 `20100` |
| 群聊（`group`） | 最多 100 个 | 返回错误码 `20100` |

**限制说明**：
- 服务端按 `ids` 的顺序逐个处理订阅，直到达到该类型限制为止；超过上限的 `ids` 会被收集到 `failed_ids` 并返回 `reason='limit_exceeded'`，但之前已成功订阅的部分继续生效。
- 客户端需根据 `failed_ids` 自行回滚本地状态，确保与服务端保持一致。
- 超出限制时：服务端在 `SubscribeAck` 事件中返回 `code !== 0`，错误码为 `20100`（订阅错误），message 描述具体原因
- `failed_ids` 字段会包含失败的订阅 id 及其原因
- 限制数量由服务端配置决定，客户端不得硬编码假设具体数值

**订阅失败处理**：

| 失败原因 | `reason` 值 | 处理策略 |
|---------|------------|---------|
| 订阅项无效（格式错误/不存在） | `invalid` | 检查订阅项，移除无效项 |
| 无权限（已被拉黑/群组已解散） | `denied` | 提示用户无权限 |
| 超出订阅数量限制 | `limit_exceeded` | 提示用户取消部分订阅后再试 |
| 服务端错误 | `error` | 延迟重试（Exponential Backoff） |
