# HotSpot API - 聊天

> 本文档定义 HotSpot 私聊管理 (`private_chats`)、私聊消息 (`private_messages`) 和群聊消息 (`group_messages`) 相关的 RPC 接口。

## 模块信息

- **命名空间**:
  - `private_chats`
  - `private_messages`
  - `group_messages`
- **基础路径**:
  - `/api/v1/private_chats/`
  - `/api/v1/private_messages/`
  - `/api/v1/group_messages/`
- **引用**:
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 错误码：`api_spec_errors.md`
  - WebSocket：`api_spec_websocket.md`
  - 群组管理：`api_spec_group.md`

## 端点总览

### 私聊管理（`private_chats`）

| 端点 | 说明 | 认证 |
|-----|------|------|

| `POST /api/v1/private_chats/list` | 私聊列表 | 必须 |
| `POST /api/v1/private_chats/get_detail` | 私聊详情 | 必须 |
| `POST /api/v1/private_chats/update_preferences` | 更新会话偏好 | 必须 |
| `POST /api/v1/private_chats/delete` | 删除私聊会话 | 必须 |
### 私聊消息（`private_messages`）

| 端点 | 说明 | 认证 |
|-----|------|------|

| `POST /api/v1/private_messages/send` | 发送私聊消息 | 必须 |
| `POST /api/v1/private_messages/get_history` | 私聊消息历史 | 必须 |
| `POST /api/v1/private_messages/update` | 编辑私聊消息 | 必须 |
| `POST /api/v1/private_messages/recall` | 撤回私聊消息 | 必须 |
| `POST /api/v1/private_messages/mark_read` | 标记私聊消息已读（scope 控制范围） | 必须 |
### 群聊消息（`group_messages`）

| 端点 | 说明 | 认证 |
|-----|------|------|

| `POST /api/v1/group_messages/send` | 发送群聊消息 | 必须 |
| `POST /api/v1/group_messages/get_history` | 群聊消息历史 | 必须 |
| `POST /api/v1/group_messages/update` | 编辑群聊消息 | 必须 |
| `POST /api/v1/group_messages/recall` | 撤回群聊消息 | 必须 |
| `POST /api/v1/group_messages/mark_read` | 标记群聊消息已读（scope 控制范围） | 必须 |
## 已读状态设计

| 场景 | 字段 | 说明 |
|------|------|------|
| 私聊（1v1） | `is_read` | 发送方看到 `is_read=true` 表示对方已读；实时更新依赖 WebSocket `PrivateMessageRead` |
| 群聊（多人） | `read_count` | 表示已读人数；服务端**节流聚合**后通过 WebSocket `GroupMessageReadCountUpdated` 推送变更（3-5 秒窗口聚合，非逐人广播，避免大群广播风暴）。客户端无需轮询 `get_history` 获取 read_count——订阅群频道后会收到该事件。`read_count` 为最终一致值（见 `api_spec_base.md`「计数器一致性」）。 |

`mark_read` 接口统一采用 up_to 语义：标记截至某条消息的所有消息为已读。

**跨设备已读同步（已知取舍）**：当前协议不提供多端实时同步事件（self-sync 已从契约移除，见 `api_spec_index.md`「预留」）。单设备单会话模型下，同一设备的已读状态实时准确；但**跨设备**场景（用户在手机+Web 同时登录），在一端标记已读后，另一端的 `is_read`/`read_count` 不会实时更新——需等该端下次拉取 `get_history` 或冷启动 `system/bootstrap` 时刷新。这是有意的取舍（避免多端同步协议的复杂度）；未来若引入实时多端同步，将作为独立 WS 协议版本设计。客户端应在切前台/切会话时主动拉取一次历史以收敛已读状态。

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `UserInfo`
- `MessageType`
- `MessageMedia`、`MessageReplyTo`、`Reaction`、`ReactionItem`
- `PaginatedData`、`PaginationParams`

### 本模块类型

```typescript
// 可发送的消息类型（system 由服务端生成，客户端不可发送）
type SendMessageType = Exclude<MessageType, 'system'>;

// 不可发消息原因（应用层使用，非列类型；随 send 失败响应 data 返回）
type BlockReason =
  | 'blocked'                  // 已被对方拉黑（明确提示，不掩饰拉黑状态）
  | 'privacy'                  // 对方隐私设置不允许（allow_private_chat 限制）
  | 'stranger_quota_exceeded'; // 陌生人消息限额已用完

// 消息发送成功响应（私聊/群聊通用）
interface MessageSent {
  message_id: string;
  sent_at: string;
}
```

### PrivateMessageItem

私聊消息项，用于 `private_messages/get_history` 响应。

```typescript
interface PrivateMessageItem {
  message_id: string;
  type: MessageType;

  content?: string;
  media?: MessageMedia;
  share_code?: string;

  sender: UserInfo;

  reply_to?: MessageReplyTo;

  sent_at: string;
  edited_at?: string;
  recalled_at?: string;  // 存在即表示已撤回
  is_read: boolean;      // 消息是否已被对方读取（默认 false）
  reactions?: Reaction[];  // 表情反应聚合（无反应时省略）
}
```

### GroupMessageItem

群聊消息项，用于 `group_messages/get_history` 响应；与私聊统一使用 `sender: UserInfo`。

```typescript
interface GroupMessageItem {
  message_id: string;
  type: MessageType;

  content?: string;
  media?: MessageMedia;
  share_code?: string;

  sender: UserInfo;

  reply_to?: MessageReplyTo;
  mentions?: string[];  // @UserIDs
  recalled_by?: UserInfo;  // 撤回操作者（群管理员/群主/发送者）

  sent_at: string;
  edited_at?: string;
  recalled_at?: string;  // 存在即表示已撤回

  read_count?: number;
  reactions?: Reaction[];  // 表情反应聚合（无反应时省略）
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

私聊权限由用户设置 `allow_private_chat`（`ChatPerm`，定义于 `api_spec_types.md`）控制；不可发消息原因见 `BlockReason`（同上）。

### POST /api/v1/private_chats/list

获取私聊列表。

- **认证**: 必须
- **排序**: 按 `is_pinned DESC, last_activity_at DESC` 排序（置顶优先，最新会话在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `MyPrivateChatsRequest`
```typescript
interface MyPrivateChatsRequest {
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<PrivateChatItem>>`

```typescript
// PrivateChatItem: 私聊统一数据类型（用于列表、详情等所有场景）
interface PrivateChatItem {
  other_user: UserInfo;
  unread_count: number;
  is_muted: boolean;
  is_pinned: boolean;
  last_activity_at: string;
  
  // 列表场景返回
  last_message?: {
    message_id: string;
    content?: string;
    type: MessageType;
    share_code?: string;
    sender: UserInfo;
    sent_at: string;
  };
  
  // 隐私受控（对方隐私设置时不返回）
  presence_status?: UserPresenceStatus;
  
  // 详情场景返回
  created_at?: string;
  can_message: boolean;
  message_blocked_reason?: BlockReason; // 仅 can_message=false 时返回
}
```

**说明**：`can_message` 仅反映调用当下的隐私校验结果；若对方后续修改隐私设置或拉黑，`private_messages/send` 仍可能失败，客户端需处理权限错误。

---

### POST /api/v1/private_chats/get_detail

获取私聊详情（纯查询，无创建副作用）。

**设计原则**（lazy creation，与 Mattermost/WhatsApp/Telegram 等业界做法一致）：
- **纯只读**：不创建会话，读操作无写副作用
- **会话不存在时返回空态**：不落库，客户端据此渲染"发起聊天"入口
- 会话的实际创建由 `private_messages/send` 即发即创承担（首次发消息时物化会话）

**首次聊天流程**：
1. 调用 `get_detail` 查询会话状态
2. 会话存在 → 返回详情 + 最近消息；会话不存在 → 返回空态（`chat` 为空态对象，`recent_messages.items` 为空数组）
3. 客户端展示"发起聊天"入口，用户发送首条消息时调用 `send`（send 即发即创）

响应始终包含最近消息列表（`PaginatedData<PrivateMessageItem>`，会话不存在时为空），条数由 `AppConfig.response_defaults.chat_recent_messages_count` 控制。

- **认证**: 必须

**Request**: `PrivateChatOpenTheDoorRequest`
```typescript
interface PrivateChatOpenTheDoorRequest {
  target_user_id: string;
}
```

**Response**: `ApiResponse<PrivateChatOpenResult>`

```typescript
interface PrivateChatOpenResult {
  chat: PrivateChatItem;              // 会话不存在时返回空态对象（unread_count=0、is_muted/is_pinned=false、can_message 反映当下隐私校验）
  recent_messages: PaginatedData<PrivateMessageItem>;  // 始终返回，会话不存在时 items 为空数组；条数由 AppConfig.response_defaults.chat_recent_messages_count 控制
}
```

**错误码**: `10200` `10300`

### POST /api/v1/private_chats/update_preferences

更新私聊偏好（静音/置顶）。

- **认证**: 必须

**Request**: `UpdatePrivateChatPreferencesRequest`
```typescript
interface UpdatePrivateChatPreferencesRequest {
  target_user_id: string;
  is_muted?: boolean;
  is_pinned?: boolean;
}
```

**Response**: `ApiResponse`

**错误码**: `10300`

### POST /api/v1/private_chats/delete

删除私聊。

**删除语义**：
- 隐藏/移除该私聊会话入口
- 清空当前用户视角下的该会话历史消息
- 不影响对方视角与对方历史数据
- 若再次发起私聊，将创建一个新的空白会话

- **认证**: 必须

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

消息类型与字段要求见上方「消息发送统一规则」。

### POST /api/v1/private_messages/send

发送私聊消息。

**简化设计**：支持**即发即创**，客户端无需预先调用 create 接口。

**安全性与权限**：
- 必须校验接收方的隐私设置（`allow_private_chat`）
- 必须校验黑名单关系
- 陌生人消息限制：每日向非互相关注用户（陌生人）发送私聊消息的上限由 `AppConfig.limits.stranger_message_daily_quota` 控制（客户端通过 `ClientConfig.limits` 读取），超限返回 `10200`

**错误响应 data（本端点显式声明，见 `api_spec_base.md` envelope 约定）**：当发送被拒绝（返回 `10200`）时，`data` 携带结构化原因 `{ reason: BlockReason }`（`BlockReason` 定义于本模块「类型定义」）：
- `'blocked'`：已被对方拉黑
- `'privacy'`：对方隐私设置不允许（`allow_private_chat` 限制）
- `'stranger_quota_exceeded'`：陌生人消息限额已用完
其余错误（`10300`）不返回 `data`。

- **认证**: 必须
- **幂等**: 是（客户端必须携带 `Idempotency-Key`）

**Request**: `SendPrivateMessageRequest`
```typescript
interface SendPrivateMessageRequest {
  target_user_id: string;
  type: SendMessageType;
  content?: string;
  file_id?: string;          // 来自 media/complete_upload 返回的 file_id（image/video/audio/file 类型）
  share_code?: string;
  reply_to_message_id?: string;
}
```

**Response**: `ApiResponse<MessageSent>`

**错误码**: `10200` `10300`

### POST /api/v1/private_messages/get_history

获取私聊消息历史。

- **认证**: 必须
- **排序**: 按 `sent_at` 时间倒序（最新在前），`message_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `GetPrivateMessagesHistoryRequest`
```typescript
interface GetPrivateMessagesHistoryRequest {
  target_user_id: string;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<PrivateMessageItem>>`

**错误码**: `10300`

### POST /api/v1/private_messages/update

编辑私聊消息。

**编辑语义**：仅允许编辑当前用户发送的消息，最终展示以 WebSocket 推送的更新事件为准。

**编辑限制**：
- 仅 `type='text'` 的消息支持编辑，其他类型（image/video/audio/file/share/system）返回 `10400`
- 已撤回的消息不可编辑，返回 `10400`

- **认证**: 必须

**Request**: `UpdatePrivateMessageRequest`
```typescript
interface UpdatePrivateMessageRequest {
  message_id: string;
  content: string;
}
```

**Response**: `ApiResponse<PrivateMessageItem>`（内容类更新返回更新后完整对象；见 `api_spec_base.md`「设计规范」）

**错误码**: `10200` `10300` `10400`

### POST /api/v1/private_messages/recall

撤回私聊消息。发送者只能撤回撤回时间窗内的消息，窗口由 `AppConfig.limits.message_recall_window_seconds` 控制（客户端通过 `ClientConfig.limits` 读取），超时返回 `10400`。

- **认证**: 必须

**Request**: `RecallPrivateMessageRequest`
```typescript
interface RecallPrivateMessageRequest {
  message_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10200` `10300` `10400`

### POST /api/v1/private_messages/mark_read

标记私聊消息为已读（统一端点，通过 `scope` 控制范围）。

**up_to 语义**：标记截至某条消息（含）及之前的所有消息为已读。`up_to_message_id` 不属于目标会话时返回 `10300`。

- **认证**: 必须

**Request**: `MarkPrivateReadRequest`
```typescript
interface MarkPrivateReadRequest {
  scope: 'session' | 'all';
  target_user_id?: string;       // scope='session' 时必填：指定私聊会话
  up_to_message_id?: string;     // scope='session' 时可选：标记截至此消息；不传则标记该会话全部
}
```

**Response**: `ApiResponse`

**错误码**: `10300`（`up_to_message_id` 不属于该会话）

---

## 群聊消息 API

消息类型与字段要求见上方「消息发送统一规则」。

### POST /api/v1/group_messages/send

发送群聊消息。

- **认证**: 必须
- **幂等**: 是（客户端必须携带 `Idempotency-Key`）

**禁言处理**：被禁言成员发消息返回 `10400`，`message` 含禁言截止时间（如"您已被禁言至 2024-01-15T08:30:00.000Z"）。客户端可据 `MyGroupState.is_muted` + `muted_until`（见 `api_spec_types.md`）预判，禁用群聊输入框并展示剩余时间，避免必失败的请求。

**Request**: `SendGroupMessageRequest`
```typescript
interface SendGroupMessageRequest {
  group_id: string;
  type: SendMessageType;
  content?: string;
  file_id?: string;          // 来自 media/complete_upload 返回的 file_id（image/video/audio/file 类型）
  share_code?: string;
  reply_to_message_id?: string;
  mentions?: string[];
}
```

**Response**: `ApiResponse<MessageSent>`

**错误码**: `10300` `10400`

### POST /api/v1/group_messages/get_history

获取群聊消息历史。

- **认证**: 必须
- **排序**: 按 `sent_at` 时间倒序（最新在前），`message_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `GetGroupMessagesHistoryRequest`
```typescript
interface GetGroupMessagesHistoryRequest {
  group_id: string;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<GroupMessageItem>>`
**返回说明**：`items` 统一包含完整群聊历史（含 `type='system'` 的群系统消息）；客户端可按展示需求自行过滤。

**错误码**: `10300` `10400`

### POST /api/v1/group_messages/update

编辑群聊消息。

**编辑限制**：
- 仅 `type='text'` 的消息支持编辑，其他类型（image/video/audio/file/share/system）返回 `10400`
- 已撤回的消息不可编辑，返回 `10400`

- **认证**: 必须

**Request**: `UpdateGroupMessageRequest`
```typescript
interface UpdateGroupMessageRequest {
  message_id: string;
  content: string;
}
```

**Response**: `ApiResponse<GroupMessageItem>`（内容类更新返回更新后完整对象；见 `api_spec_base.md`「设计规范」）

**错误码**: `10200` `10300` `10400`

### POST /api/v1/group_messages/recall

撤回群聊消息。发送者可撤回撤回时间窗内的消息（窗口由 `AppConfig.limits.message_recall_window_seconds` 控制），群组管理员/群主可撤回任何消息，超时返回 `10400`。

- **认证**: 必须

**Request**: `RecallGroupMessageRequest`
```typescript
interface RecallGroupMessageRequest {
  message_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10200` `10300` `10400`

### POST /api/v1/group_messages/mark_read

标记群聊消息为已读（统一端点，通过 `scope` 控制范围）。

**up_to 语义**：标记截至某条消息（含）及之前的所有消息为已读。`up_to_message_id` 不属于目标群组时返回 `10300`。

- **认证**: 必须

**Request**: `MarkGroupReadRequest`
```typescript
interface MarkGroupReadRequest {
  scope: 'group' | 'all';
  group_id?: string;              // scope='group' 时必填：指定群组
  up_to_message_id?: string;      // scope='group' 时可选：标记截至此消息；不传则标记该群全部
}
```

**Response**: `ApiResponse`

**错误码**: `10300` `10400`
