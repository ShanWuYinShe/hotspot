# HotSpot API - 聊天

> 本文档定义 HotSpot 私聊管理 (`private_chats`)、私聊消息 (`private_messages`) 和群聊消息 (`group_messages`) 相关的 RPC 接口。

## 模块信息

- **命名空间**: `private_chats` / `private_messages` / `group_messages`
- **基础路径**: `/api/v1/private_chats/`、`/api/v1/private_messages/`、`/api/v1/group_messages/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 守卫与权限：`api_spec_authz.md`
  - 错误码：`api_spec_errors.md`
  - WebSocket：`api_spec_websocket.md`
  - 群组管理：`api_spec_group.md`

## 端点总览

### 私聊管理（`private_chats`）

| 端点 | 说明 | 节点 |
|-----|------|------|
| `POST /api/v1/private_chats/fetch` | 私聊列表 | /api/v1 |
| `POST /api/v1/private_chats/open_the_door` | 私聊详情 | /api/v1 |
| `POST /api/v1/private_chats/update_preferences` | 更新会话偏好 | /api/v1 |
| `POST /api/v1/private_chats/delete` | 删除私聊会话 | /api/v1 |

### 私聊消息（`private_messages`）

| 端点 | 说明 | 节点 |
|-----|------|------|
| `POST /api/v1/private_messages/send` | 发送私聊消息 | /api/v1 |
| `POST /api/v1/private_messages/fetch` | 私聊消息历史 | /api/v1 |
| `POST /api/v1/private_messages/update` | 编辑私聊消息 | /api/v1 |
| `POST /api/v1/private_messages/recall` | 撤回私聊消息 | /api/v1 |
| `POST /api/v1/private_messages/mark_read` | 标记私聊消息已读（scope 控制范围） | /api/v1 |

### 群聊消息（`group_messages`）

| 端点 | 说明 | 节点 |
|-----|------|------|
| `POST /api/v1/group_messages/send` | 发送群聊消息 | /api/v1 |
| `POST /api/v1/group_messages/fetch` | 群聊消息历史 | /api/v1 |
| `POST /api/v1/group_messages/update` | 编辑群聊消息 | /api/v1 |
| `POST /api/v1/group_messages/recall` | 撤回自己发送的群聊消息 | /api/v1 |
| `POST /api/v1/group_messages/mark_read` | 标记群聊消息已读（scope 控制范围） | /api/v1 |

> **群管理员撤回任意成员消息**走群管理守卫子树：`POST /api/v1/groups/admin/messages/recall`（见 `api_spec_group.md`）。两个端点各司其职，无混用。

## 已读状态设计

| 场景 | 字段 | 说明 |
|------|------|------|
| 私聊（1v1） | `is_read` | 发送方看到 `is_read=true` 表示对方已读；实时更新依赖 WebSocket `PrivateMessageRead` |
| 群聊（多人） | `read_count` | 表示已读人数；服务端**节流聚合**后通过 WebSocket `GroupMessageReadCountUpdated` 推送变更（3-5 秒窗口聚合，非逐人广播）。客户端无需轮询——订阅群频道后会收到该事件。`read_count` 为最终一致值 |

`mark_read` 接口统一采用 up_to 语义：标记截至某条消息的所有消息为已读。

**跨设备已读同步（已知取舍）**：本协议**不提供专门的 self-sync 事件**（即不推送"你在另一端读了消息"给自己其他设备）。单设备单会话模型下，同一设备的已读状态实时准确；但**跨设备**场景（用户在手机+Web 同时登录），在一端标记已读后，另一端的 `is_read`/`read_count` 不会实时更新——需等该端下次拉取 `private_messages/fetch`/`group_messages/fetch` 或冷启动 `/api/v1/system/bootstrap` 时刷新。客户端应在切前台/切会话时主动拉取一次历史以收敛已读状态。

> **多端推送 vs self-sync 的边界**：用户多设备各自独立鉴权、各自订阅 WebSocket，因此**消息内容类事件**（`PrivateMessageNew`/`Edited`/`Recalled` 等"推送给发送者其他设备"的事件）天然到达所有在线设备——这属于推送目标设计，**不是** self-sync。self-sync 特指已读状态等具体状态的跨设备实时同步，当前通过冷启动 + HTTP 补拉收敛。

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `UserInfo`
- `MessageType`
- `MessageMedia`、`MessageBrief`、`Reaction`、`ReactionItem`
- `PaginatedData`、`PaginationParams`
- （`last_message` 亦为 `MessageBrief`）
- `UserPresenceStatus`

> 不可发消息原因统一为 `10200` + `data.forbidden_reason`（`blocked_by_user` / `privacy_setting` / `stranger_quota_exceeded`），权威定义见 `api_spec_errors.md`。

### 本模块类型

```typescript
// 可发送的消息类型（system 由服务端生成，客户端不可发送）
type SendMessageType = Exclude<MessageType, 'system'>;

// 群聊禁言拒绝信息（group_messages/send 被禁言拦截时，随 10400 响应 data 返回）
// 结构化携带截止时间，避免客户端从 message 文本解析 ISO 时间戳（errors.md 禁止客户端解析 message 决定行为）
interface GroupMuteStatus {
  muted_until: string;   // 禁言截止时间（UTC ISO 8601）；与 GroupViewerState.muted_until 同源
}

// 消息发送成功响应（私聊/群聊通用）
interface MessageSentResult {
  message_id: string;
  sent_at: string;
}
```

### PrivateMessageItem

私聊消息项，用于 `private_messages/fetch` 响应。

```typescript
interface PrivateMessageItem {
  message_id: string;
  type: MessageType;

  content: string | null;
  media: MessageMedia | null;
  share_code: string | null;

  sender: UserInfo;

  reply_to: MessageBrief | null;

  sent_at: string;
  edited_at: string | null;
  recalled_at: string | null;  // 非 null 即表示已撤回
  is_read: boolean;      // 消息是否已被对方读取（默认 false）
  reactions: Reaction[];  // 表情反应聚合（空数组 = 无反应）
}
```

### GroupMessageItem

群聊消息项，用于 `group_messages/fetch` 响应；与私聊统一使用 `sender: UserInfo`。

```typescript
interface GroupMessageItem {
  message_id: string;
  type: MessageType;

  content: string | null;
  media: MessageMedia | null;
  share_code: string | null;

  sender: UserInfo;

  reply_to: MessageBrief | null;
  mentions: string[];  // @UserIDs
  recalled_by: UserInfo | null;  // 撤回操作者（群管理员撤回时非作者本人）

  sent_at: string;
  edited_at: string | null;
  recalled_at: string | null;  // 非 null 即表示已撤回

  read_count: number | null;  // 已读人数（最终一致）；null = 不适用（system 消息/已撤回消息）
  reactions: Reaction[];  // 表情反应聚合（空数组 = 无反应）
}
```

### 消息发送统一规则

**消息类型与字段要求**：
- `text` 需要 `content`
- `image`/`video`/`audio`/`file` 需要 `file_id`
- `share` 需要 `share_code`，服务端必须校验 `share_code` 存在且有效（未撤销/未过期），无效返回 `10400`（防发送失效链接）

**群聊 @ 通知**：`group_messages/send` 携带 `mentions` 时，服务端对每个被 @ 的用户分别推送 `GroupMessageMentioned` 事件并落库通知（`NotifyType='group_mentioned'`，见 `api_spec_notify.md`）；被 @ 用户需为该群成员，非成员 ID 忽略。

**发送语义**：API 调用仅提交发送意图和获取 `message_id`，消息最终态通过 WebSocket 推送获取。

---

## 私聊管理 API

私聊权限由用户设置 `allow_private_chat`（`ChatPerm`）控制。

### POST /api/v1/private_chats/fetch

获取私聊列表。

- **节点**: /api/v1
- **排序**: 按 `is_pinned DESC, last_activity_at DESC` 排序（置顶优先，最新会话在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FetchPrivateChatsRequest`
```typescript
interface FetchPrivateChatsRequest {
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<PrivateChatItem>>`

```typescript
// PrivateChatItem: 私聊统一数据类型（用于列表、详情等所有场景，形状恒定）
interface PrivateChatItem {
  other_user: UserInfo;
  unread_count: number;
  is_muted: boolean;
  is_pinned: boolean;
  last_activity_at: string;
  
  // 列表场景返回（详情场景可为 null）
  last_message: MessageBrief | null;
  
  // 隐私受控（恒定 nullable：null = 对方 presence_visible=false）
  presence_status: UserPresenceStatus | null;
  
  // 详情场景返回（列表为 null）
  created_at: string | null;
  can_message: boolean;          // 反映调用当下的隐私校验结果（列表场景由服务端批量校验缓存）
  forbidden_reason: ForbiddenReason | null; // 仅 can_message=false 时非 null：blocked_by_user / privacy_setting / stranger_quota_exceeded
}
```

**说明**：`can_message` 仅反映调用当下的隐私校验结果；若对方后续修改隐私设置或拉黑，`private_messages/send` 仍可能失败，客户端需处理权限错误。

---

### POST /api/v1/private_chats/open_the_door

获取私聊详情（纯查询，无创建副作用）。

**设计原则**（lazy creation，与 Mattermost/WhatsApp/Telegram 等业界做法一致）：
- **纯只读**：不创建会话，读操作无写副作用
- **会话不存在时返回空态**：不落库，客户端据此渲染"发起聊天"入口
- 会话的实际创建由 `private_messages/send` 即发即创承担（首次发消息时物化会话）

**首次聊天流程**：
1. 调用 `open_the_door` 查询会话状态
2. 会话存在 → 返回详情 + 最近消息；会话不存在 → 返回空态（`chat` 为空态对象，`recent_messages.items` 为空数组）
3. 客户端展示"发起聊天"入口，用户发送首条消息时调用 `send`（send 即发即创）

响应始终包含最近消息列表（`PaginatedData<PrivateMessageItem>`，会话不存在时为空），条数由 `AppConfig.response_defaults.chat_recent_messages_count` 控制。

- **节点**: /api/v1

**Request**: `OpenPrivateChatDoorRequest`
```typescript
interface OpenPrivateChatDoorRequest {
  target_user_id: string;
}
```

**Response**: `ApiResponse<PrivateChatOpenResult>`

```typescript
interface PrivateChatOpenResult {
  chat: PrivateChatItem;              // 会话不存在时返回空态对象（unread_count=0、is_muted/is_pinned=false、can_message 反映当下隐私校验）
  recent_messages: PaginatedData<PrivateMessageItem>;  // 始终返回，会话不存在时 items 为空数组
}
```

**10200 边界**：仅当对方 `allow_private_chat='nobody'`（硬禁止）时本端点返回 `10200`；软限制（对方为 `'friends'` 但双方未互相关注、陌生人配额耗尽）不拦截本端点，通过 `chat.can_message=false` + `chat.forbidden_reason` 表达。发送时的最终判定以 `private_messages/send` 为准。

**错误码**: `10200`（privacy_setting 对方不允许私聊） `10300`

---

### POST /api/v1/private_chats/update_preferences

更新私聊偏好（静音/置顶）。

- **节点**: /api/v1

**Request**: `UpdatePrivateChatPreferencesRequest`
```typescript
interface UpdatePrivateChatPreferencesRequest {
  target_user_id: string;
  is_muted: boolean | null;
  is_pinned: boolean | null;
}
```

**Response**: `ApiResponse`

**错误码**: `10300`

---

### POST /api/v1/private_chats/delete

删除私聊。

**删除语义**：
- 隐藏/移除该私聊会话入口
- 清空当前用户视角下的该会话历史消息
- 不影响对方视角与对方历史数据
- 若再次发起私聊，将创建一个新的空白会话

- **节点**: /api/v1

**Request**: `DeletePrivateChatRequest`
```typescript
interface DeletePrivateChatRequest {
  target_user_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10300`

---

## 私聊消息 API

### POST /api/v1/private_messages/send

发送私聊消息。

**简化设计**：支持**即发即创**（首次发消息自动创建会话）。

**安全性与权限**：
- 必须校验接收方的隐私设置（`allow_private_chat`）
- 必须校验黑名单关系
- 陌生人消息限制：每日向非互相关注用户（陌生人）发送私聊消息的上限由 `AppConfig.limits.stranger_message_daily_quota` 控制（客户端通过 `ClientConfig.limits` 读取），超限返回 `10200` + `stranger_quota_exceeded`

**错误响应 data（本端点显式声明）**：当发送被拒绝（返回 `10200`）时，`data` 携带结构化原因 `{ forbidden_reason: ForbiddenReason }`：
- `blocked_by_user`：已被对方拉黑
- `privacy_setting`：对方隐私设置不允许（`allow_private_chat` 限制）
- `stranger_quota_exceeded`：陌生人消息限额已用完
其余错误（`10300`）不返回 data。

- **节点**: /api/v1
- **幂等**: 是（客户端必须携带 `Idempotency-Key`）

**Request**: `SendPrivateMessageRequest`
```typescript
interface SendPrivateMessageRequest {
  target_user_id: string;
  type: SendMessageType;
  content: string | null;
  file_id: string | null;          // 来自 media/complete_upload 返回的 file_id（image/video/audio/file 类型）
  share_code: string | null;
  reply_to_message_id: string | null;
}
```

**Response**: `ApiResponse<MessageSentResult>`

**错误码**: `10200`（blocked_by_user / privacy_setting / stranger_quota_exceeded） `10300` `10400`（`share_code` 无效/已撤销/已过期、`reply_to_message_id` 无效）

---

### POST /api/v1/private_messages/fetch

获取私聊消息历史。

- **节点**: /api/v1
- **排序**: 按 `sent_at` 时间倒序（最新在前），`message_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FetchPrivateMessagesRequest`
```typescript
interface FetchPrivateMessagesRequest {
  target_user_id: string;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<PrivateMessageItem>>`

**错误码**: `10300`

---

### POST /api/v1/private_messages/update

编辑私聊消息（仅作者）。

**编辑语义**：仅允许编辑当前用户发送的消息，最终展示以 WebSocket 推送的更新事件为准。

**编辑限制**：
- 仅 `type='text'` 的消息支持编辑，其他类型（image/video/audio/file/share/system）返回 `10400`
- 已撤回的消息不可编辑，返回 `10400`

- **节点**: /api/v1

**Request**: `UpdatePrivateMessageRequest`
```typescript
interface UpdatePrivateMessageRequest {
  message_id: string;
  content: string;
}
```

**Response**: `ApiResponse<PrivateMessageItem>`（内容类更新返回更新后完整对象）

**错误码**: `10200`（not_resource_owner） `10300` `10400`

---

### POST /api/v1/private_messages/recall

撤回私聊消息（仅作者）。发送者只能撤回撤回时间窗内的消息，窗口由 `AppConfig.limits.message_recall_window_seconds` 控制，超时返回 `10400`。

- **节点**: /api/v1

**Request**: `RecallPrivateMessageRequest`
```typescript
interface RecallPrivateMessageRequest {
  message_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10200`（not_resource_owner） `10300` `10400`（超时）

---

### POST /api/v1/private_messages/mark_read

标记私聊消息为已读（统一端点，通过 `scope` 控制范围）。

**up_to 语义**：标记截至某条消息（含）及之前的所有消息为已读。`up_to_message_id` 不属于目标会话时返回 `10300`。

- **节点**: /api/v1

**Request**: `MarkPrivateReadRequest`
```typescript
interface MarkPrivateReadRequest {
  scope: 'session' | 'all';
  target_user_id: string | null;    // scope='session' 时必填：指定私聊会话
  up_to_message_id: string | null;  // scope='session' 时可选：标记截至此消息；不传则标记该会话全部
}
// 参数互斥：scope='all' 时不得传 up_to_message_id（同传视为参数冲突 10000）；scope='session' 时 target_user_id 必填
```

**Response**: `ApiResponse`

**错误码**: `10300`（`up_to_message_id` 不属于该会话）

---

## 群聊消息 API

消息类型与字段要求见上方「消息发送统一规则」。

### POST /api/v1/group_messages/send

发送群聊消息（群成员）。

- **节点**: /api/v1
- **幂等**: 是（客户端必须携带 `Idempotency-Key`）

**禁言处理**：被禁言成员发消息返回 `10400`。客户端可据 `GroupViewerState.is_muted` + `muted_until`（见 `api_spec_types.md`）预判，禁用群聊输入框并展示剩余时间，避免必失败的请求。

**错误响应 data（本端点显式声明）**：当发送被禁言拦截（返回 `10400`，禁言原因）时，`data` 携带结构化信息 `{ mute: GroupMuteStatus }`（`GroupMuteStatus` 定义于本模块「类型定义」），含 `muted_until` 截止时间——客户端据此展示剩余时间，**无需从 `message` 文本解析时间戳**。其余 `10400` 场景与 `10300` 不返回 data。

**Request**: `SendGroupMessageRequest`
```typescript
interface SendGroupMessageRequest {
  group_id: string;
  type: SendMessageType;
  content: string | null;
  file_id: string | null;
  share_code: string | null;
  reply_to_message_id: string | null;
  mentions: string[];
}
```

**Response**: `ApiResponse<MessageSentResult>`

**错误码**: `10200`（not_group_member 非群成员） `10300` `10400`（禁言等）

---

### POST /api/v1/group_messages/fetch

获取群聊消息历史（群成员）。

- **节点**: /api/v1
- **排序**: 按 `sent_at` 时间倒序（最新在前），`message_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FetchGroupMessagesRequest`
```typescript
interface FetchGroupMessagesRequest {
  group_id: string;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<GroupMessageItem>>`
**返回说明**：`items` 统一包含完整群聊历史（含 `type='system'` 的群系统消息）；客户端可按展示需求自行过滤。

**错误码**: `10200`（not_group_member） `10300` `10400`

---

### POST /api/v1/group_messages/update

编辑群聊消息（仅作者）。

**编辑限制**：
- 仅 `type='text'` 的消息支持编辑，其他类型返回 `10400`
- 已撤回的消息不可编辑，返回 `10400`

- **节点**: /api/v1

**Request**: `UpdateGroupMessageRequest`
```typescript
interface UpdateGroupMessageRequest {
  message_id: string;
  content: string;
}
```

**Response**: `ApiResponse<GroupMessageItem>`

**错误码**: `10200`（not_resource_owner） `10300` `10400`

---

### POST /api/v1/group_messages/recall

撤回**自己发送的**群聊消息（单角色端点：仅作者）。撤回时间窗由 `AppConfig.limits.message_recall_window_seconds` 控制，超时返回 `10400`。

> 群管理员撤回任意成员消息使用群管理守卫子树端点 `groups/admin/messages/recall`（见 `api_spec_group.md`）。

- **节点**: /api/v1

**Request**: `RecallGroupMessageRequest`
```typescript
interface RecallGroupMessageRequest {
  message_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10200`（not_resource_owner） `10300` `10400`（超时）

---

### POST /api/v1/group_messages/mark_read

标记群聊消息为已读（统一端点，通过 `scope` 控制范围）。

**up_to 语义**：标记截至某条消息（含）及之前的所有消息为已读。`up_to_message_id` 不属于目标群组时返回 `10300`。

- **节点**: /api/v1

**Request**: `MarkGroupReadRequest`
```typescript
interface MarkGroupReadRequest {
  scope: 'group' | 'all';
  group_id: string | null;            // scope='group' 时必填：指定群组
  up_to_message_id: string | null;    // scope='group' 时可选：标记截至此消息；不传则标记该群全部
}
// 参数互斥：scope='all' 时不得传 up_to_message_id（同传视为参数冲突 10000）；scope='group' 时 group_id 必填
```

**Response**: `ApiResponse`

**错误码**: `10200`（not_group_member） `10300` `10400`