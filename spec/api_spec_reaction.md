# HotSpot API - 表情反应（Reaction）

> 本文档定义 HotSpot 表情反应 (`reactions`) 相关的 RPC 接口，覆盖 `message` / `moment` / `comment` 三类目标的统一反应操作与列表查询。

## 模块信息

- **命名空间**: `reactions`
- **基础路径**: `/api/v1/reactions/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`（`ReactTarget`、`Reaction`、`ReactionItem`）
  - 错误码：`api_spec_errors.md`
  - WebSocket：`api_spec_websocket.md`（`ReactionAdded` / `ReactionRemoved` 事件）

## 端点总览

| 端点 | 说明 | 认证 |
|-----|------|------|

| `POST /api/v1/reactions/set` | 设置表情反应（两态统一） | 必须 |
| `POST /api/v1/reactions/list` | 反应列表（统一三类目标） | 可选 |
## 类型定义

以下类型定义于 `api_spec_types.md`：
- `ReactTarget`
- `Reaction`
- `ReactionItem`
- `PaginationParams`
- `PaginatedData`

## 端点详情

### POST /api/v1/reactions/set

设置表情反应（两态统一端点）。覆盖消息、瞬间、评论三类目标。

- **认证**: 必须

**Request**: `SetReactionRequest`
```typescript
interface SetReactionRequest {
  target_type: ReactTarget;  // 'message' | 'moment' | 'comment'
  target_id: string;                // message_id / moment_id / comment_id
  emoji: string;                    // Unicode 码点字符串，须在 AppConfig.limits.reaction_emojis 白名单内
  active: boolean;                  // true=添加/切换反应，false=取消反应
}
```

**Response**: `ApiResponse`

**行为细节**：
- `active=true` 且用户此前对该目标用过其他 emoji：视为切换（旧 emoji 计数 -1，新 emoji 计数 +1），不分多次调用
- `active=true` 且 emoji 与当前一致：no-op 成功
- `active=false`：移除当前用户对该目标的反应（若 emoji 与当前不一致仍视为成功，幂等）
- 目标资源不存在返回 `10300`；emoji 不在白名单请求失败
- 私聊/群聊消息场景：若被对方拉黑或对方 `allow_private_chat='nobody'`，仍允许 reaction（reaction 是公开互动信号，不受私聊权限约束）

**目标可见性（隐私拒绝）**：目标资源**存在但调用方不可见**时，统一返回 `10300`（按不存在处理，与 base.md 隐私拒绝策略一致，不暴露资源存在性）。可见性以目标的挂载上下文判定：`comment` → 所属瞬间；`message` → 所属会话（私聊：仅参与者；群聊：仅群成员，含被移除/退群用户对历史消息的反应不可见）；`moment` → 瞬间隐私等级。**列表与设置两端点均适用**——防止匿名用户凭 `message_id`/`comment_id` 枚举"谁对这条不可见消息反应过"。

**WS 推送**：成功后触发 `ReactionAdded` / `ReactionRemoved` 事件（服务端统一推送、不做自反应判断，客户端自行过滤自反应场景，见 `api_spec_websocket.md`）；`ReactionAdded` 落库 `reaction_added` 通知（自反应不落库）。

**错误码**: `10300`

---

### POST /api/v1/reactions/list

获取表情反应列表（统一 moment / comment / message 三类目标）。按反应时间倒序，支持按 emoji 过滤。

- **认证**: 可选
- **排序**: 默认按 `reacted_at` **时间倒序**（最近反应在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `ListReactionsRequest`
```typescript
interface ListReactionsRequest {
  target_type: ReactTarget;  // 'message' | 'moment' | 'comment'
  target_id: string;
  emoji?: string;                   // 可选：按特定 emoji 过滤；不传则返回所有 emoji 的反应者
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<ReactionItem>>`（`ReactionItem` 定义于 `api_spec_types.md`）

**目标可见性**：与 `reactions/set` 一致——目标存在但调用方不可见（私聊非参与者、非群成员、私有瞬间等）统一返回 `10300`（见 `reactions/set`「目标可见性」）。

**错误码**: `10300`
