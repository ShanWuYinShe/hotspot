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

| 端点 | 说明 | 认证 | 幂等 |
|-----|------|------|------|
| `POST /api/v1/private_chats/me/list` | 私聊列表 | 必须 | 无 |
| `POST /api/v1/private_chats/open_the_door` | 私聊详情 | 必须 | 幂等语义 |
| `POST /api/v1/private_chats/me/update_settings` | 更新会话设置 | 必须 | 幂等语义 |
| `POST /api/v1/private_chats/me/delete` | 删除私聊会话 | 必须 | 幂等语义 |

### 私聊消息（`private_messages`）

| 端点 | 说明 | 认证 | 幂等 |
|-----|------|------|------|
| `POST /api/v1/private_messages/send` | 发送私聊消息 | 必须 | MUST |
| `POST /api/v1/private_messages/get_history` | 私聊消息历史 | 必须 | 无 |
| `POST /api/v1/private_messages/update` | 编辑私聊消息 | 必须 | 幂等语义 |
| `POST /api/v1/private_messages/recall` | 撤回私聊消息 | 必须 | 幂等语义 |
| `POST /api/v1/private_messages/me/mark_as_read` | 标记指定私聊会话已读 | 必须 | 幂等语义 |
| `POST /api/v1/private_messages/me/mark_all_read` | 标记所有私聊消息已读 | 必须 | 幂等语义 |

### 群聊消息（`group_messages`）

| 端点 | 说明 | 认证 | 幂等 |
|-----|------|------|------|
| `POST /api/v1/group_messages/send` | 发送群聊消息 | 必须 | MUST |
| `POST /api/v1/group_messages/get_history` | 群聊消息历史 | 必须 | 无 |
| `POST /api/v1/group_messages/update` | 编辑群聊消息 | 必须 | 幂等语义 |
| `POST /api/v1/group_messages/recall` | 撤回群聊消息 | 必须 | 幂等语义 |
| `POST /api/v1/group_messages/me/mark_as_read` | 标记指定群聊已读 | 必须 | 幂等语义 |
| `POST /api/v1/group_messages/me/mark_all_read` | 标记所有群聊消息已读 | 必须 | 幂等语义 |

## 已读状态设计

| 场景 | 字段 | 说明 |
|------|------|------|
| 私聊（1v1） | `is_read` | 发送方看到 `is_read=true` 表示对方已读；实时更新依赖 WebSocket `PrivateMessageRead` |
| 群聊（多人） | `read_count` | 表示已读人数；不推送逐人已读事件，`read_count` 通过历史接口获取 |

`mark_as_read` 接口统一采用 up_to 语义：标记截至某条消息的所有消息为已读。

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `UserInfo`
- `MessageType`
- `SendMessageType`
- `MessageMedia`
- `MessageReplyTo`
- `MessageSent`
- `UserPresenceStatus`
- `PaginationParams`
- `PaginatedData`
- `MessageBlockedReason`

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
}
```

### 消息发送统一规则

**消息类型与字段要求**：
- `text` 需要 `content`
- `image`/`video`/`audio`/`file` 需要 `file_id`
- `share` 需要 `share_code`

**发送语义**：API 调用仅提交发送意图和获取 `message_id`，消息最终态通过 WebSocket 推送获取。

---

## 私聊管理 API

私聊权限由用户设置 `allow_private_chat`（`PrivateChatPermission`，定义于 `api_spec_types.md`）控制；不可发消息原因见 `MessageBlockedReason`（同上）。

### POST /api/v1/private_chats/me/list

获取私聊列表。

- **认证**: 必须
- **幂等**: 无
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
  message_blocked_reason?: MessageBlockedReason; // 仅 can_message=false 时返回
}

// MessageBlockedReason 定义于 api_spec_types.md
```

**说明**：`can_message` 仅反映调用当下的隐私校验结果；若对方后续修改隐私设置或拉黑，`private_messages/send` 仍可能失败，客户端需处理权限错误。

**错误码**: `10000`

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

- **认证**: 必须
- **幂等**: 幂等语义（纯查询：同一 `target_user_id` 的重复调用返回一致结果，永不产生写入）

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

**错误码**: `10000` `10200` `10300`

### POST /api/v1/private_chats/me/update_settings

更新私聊设置。

- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `UpdatePrivateChatSettingsRequest`
```typescript
interface UpdatePrivateChatSettingsRequest {
  target_user_id: string;
  is_muted?: boolean;
  is_pinned?: boolean;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10300`

### POST /api/v1/private_chats/me/delete

删除私聊。

**删除语义**：
- 隐藏/移除该私聊会话入口
- 清空当前用户视角下的该会话历史消息
- 不影响对方视角与对方历史数据
- 若再次发起私聊，将创建一个新的空白会话

- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `DeletePrivateChatRequest`
```typescript
interface DeletePrivateChatRequest {
  target_user_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10300`

---

## 私聊消息 API

消息类型与字段要求见上方「消息发送统一规则」。

### POST /api/v1/private_messages/send

发送私聊消息。

**简化设计**：支持**即发即创**，客户端无需预先调用 create 接口。

**安全性与权限**：
- 必须校验接收方的隐私设置（`allow_private_chat`）
- 必须校验黑名单关系
- 陌生人消息限制

- **认证**: 必须
- **幂等**: MUST（客户端必须携带 `Idempotency-Key`）

**Request**: `SendPrivateMessageRequest`
```typescript
interface SendPrivateMessageRequest {
  target_user_id: string;
  type: SendMessageType;
  content?: string;
  file_id?: string;
  share_code?: string;
  reply_to_message_id?: string;
}
```

**Response**: `ApiResponse<MessageSent>`

**错误码**: `10000` `10200` `10300`

### POST /api/v1/private_messages/get_history

获取私聊消息历史。

- **认证**: 必须
- **幂等**: 无
- **排序**: 按 `sent_at` 时间倒序（最新在前），`message_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `GetPrivateMessagesHistoryRequest`
```typescript
interface GetPrivateMessagesHistoryRequest {
  target_user_id: string;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<PrivateMessageItem>>`

**错误码**: `10000` `10300`

### POST /api/v1/private_messages/update

编辑私聊消息。

**编辑语义**：仅允许编辑当前用户发送的消息，最终展示以 WebSocket 推送的更新事件为准。

**编辑限制**：
- 仅 `type='text'` 的消息支持编辑，其他类型（image/video/audio/file/share/system）返回 `10400`
- 已撤回的消息不可编辑，返回 `10400`

- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `UpdatePrivateMessageRequest`
```typescript
interface UpdatePrivateMessageRequest {
  message_id: string;
  content: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300` `10400`

### POST /api/v1/private_messages/recall

撤回私聊消息。发送者只能撤回 2 分钟内的消息。

- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `RecallPrivateMessageRequest`
```typescript
interface RecallPrivateMessageRequest {
  message_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300` `10400`

### POST /api/v1/private_messages/me/mark_as_read

标记指定私聊会话的消息为已读。

**语义**：标记与指定用户的私聊消息为已读。当不传 `up_to_message_id` 时，标记该会话全部消息为已读。
**up_to_message_id 校验**：当 `up_to_message_id` 不属于指定私聊会话时，返回 `10300`。

- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `MarkPrivateMessagesReadRequest`
```typescript
interface MarkPrivateMessagesReadRequest {
  target_user_id: string;        // 指定会话（必选）
  up_to_message_id?: string;     // 标记截至此消息（含）及之前的所有消息；不传时标记该会话全部
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10300`

### POST /api/v1/private_messages/me/mark_all_read

标记所有私聊消息为已读。

- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `{}`

**Response**: `ApiResponse`

**错误码**: `10000`

---

## 群聊消息 API

消息类型与字段要求见上方「消息发送统一规则」。

### POST /api/v1/group_messages/send

发送群聊消息。

- **认证**: 必须
- **幂等**: MUST（客户端必须携带 `Idempotency-Key`）

**Request**: `SendGroupMessageRequest`
```typescript
interface SendGroupMessageRequest {
  group_id: string;
  type: SendMessageType;
  content?: string;
  file_id?: string;
  share_code?: string;
  reply_to_message_id?: string;
  mentions?: string[];
}
```

**Response**: `ApiResponse<MessageSent>`

**错误码**: `10000` `10300` `10400`

### POST /api/v1/group_messages/get_history

获取群聊消息历史。

- **认证**: 必须
- **幂等**: 无
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

**错误码**: `10000` `10300` `10400`

### POST /api/v1/group_messages/update

编辑群聊消息。

**编辑限制**：
- 仅 `type='text'` 的消息支持编辑，其他类型（image/video/audio/file/share/system）返回 `10400`
- 已撤回的消息不可编辑，返回 `10400`

- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `UpdateGroupMessageRequest`
```typescript
interface UpdateGroupMessageRequest {
  message_id: string;
  content: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300` `10400`

### POST /api/v1/group_messages/recall

撤回群聊消息。发送者可撤回 2 分钟内消息，群组管理员/群主可撤回任何消息。

- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `RecallGroupMessageRequest`
```typescript
interface RecallGroupMessageRequest {
  message_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300` `10400`

### POST /api/v1/group_messages/me/mark_as_read

标记指定群聊的消息为已读。

**语义**：标记指定群组的群聊消息为已读。当不传 `up_to_message_id` 时，标记该群全部消息为已读。
**up_to_message_id 校验**：当 `up_to_message_id` 不属于指定群组时，返回 `10300`。

- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `MarkGroupMessagesReadRequest`
```typescript
interface MarkGroupMessagesReadRequest {
  group_id: string;              // 指定群组（必选）
  up_to_message_id?: string;     // 标记截至此消息（含）及之前的所有消息；不传时标记该群全部
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10300` `10400`

### POST /api/v1/group_messages/me/mark_all_read

标记所有群聊消息为已读。

- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `{}`

**Response**: `ApiResponse`

**错误码**: `10000`
