# HotSpot API - 通知中心

> 本文档定义 HotSpot 通知中心 (`notify`) 相关的 RPC 接口。
> 通知中心提供通知历史查询、未读计数和已读标记等功能，遵循"HTTP API 负责冷启动数据获取，WebSocket 负责实时增量推送"的原则。

## 模块信息

- **命名空间**: `notify`
- **基础路径**: `/api/v1/notify/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 守卫与权限：`api_spec_authz.md`
  - 错误码：`api_spec_errors.md`
  - WebSocket 事件：`api_spec_websocket.md`（通知与事件的关系见下文映射表）

## 端点总览

| 端点 | 说明 | 节点 |
|-----|------|------|
| `POST /api/v1/notify/fetch` | 通知列表 | /api/v1 |
| `POST /api/v1/notify/unread_count` | 未读通知数 | /api/v1 |
| `POST /api/v1/notify/mark_read` | 标记已读 | /api/v1 |

## 类型定义

以下类型定义于 `api_spec_types.md`：`UserInfo`、`MomentSummary`、`FollowSource`、`AnnounceSeverity`、`NotifyType`、`NotifyCategory`、`PaginationParams`、`PaginatedData`。

### 通知与 WebSocket 事件的关系（契约级）

**原则**：事件走 WS（实时），通知走 HTTP（冷启动/补拉）。

- **WS 事件**：实时事件流。payload 携带 `notify_id` 的事件 = 该业务事件同时落库了一条通知记录（`notify_id` 与 `notify/fetch` 返回的 `NotifyItem.notify_id` 为同一主键）；**不携带 `notify_id` 的事件为纯实时事件**（消息、群成员变动、群公告等），与通知中心无关
- **通知记录**：`notify/fetch` 是通知的唯一权威来源，客户端冷启动/补拉时调用一次即可拉取全部通知（含未读）
- **已读同步**：客户端收到带 `notify_id` 的 WS 事件并即时展示后，若用户已在对应界面阅读该内容，调用 `notify/mark_read`（`scope='ids'`，`notify_ids: [notify_id]`）标记已读，避免下次冷启动时重复显示未读
- **去重**：WS 事件与 `notify/fetch` 记录共用 `notify_id` 主键，客户端按 `notify_id` 去重即可

| `NotifyType` | 对应 WebSocket 事件 | 触发场景 |
|-------------------|--------------------|---------|
| `new_follower` | `NewFollower` | 被关注 |
| `mutual_follow` | `MutualFollow` | 形成互相关注 |
| `moment_commented` | `MomentCommented` | 瞬间被评论 |
| `comment_reply` | `CommentReply` | 评论被回复 |
| `mentioned` | `ContentMentioned` | 瞬间/评论中被 @ |
| `group_mentioned` | `GroupMessageMentioned` | 群聊消息中被 @（对被 @ 的每个用户分别落库推送） |
| `reaction_added` | `ReactionAdded` | 内容被添加表情反应（携带 `notify_id`；唯一例外：自反应不落库，此时事件不携带 `notify_id`） |
| `group_invite_received` | `GroupInviteReceived` | 收到入群邀请 |
| `group_application_received` | `GroupApplicationReceived` | 群主/管理员收到入群申请 |
| `group_application_accepted` | `GroupApplicationResponse` | 申请被接受（申请方收到） |
| `group_application_rejected` | `GroupApplicationRejected` | 申请被拒绝（申请人收到，含 reason） |
| `report_resolved` | `ReportResolved` | 举报被处理（举报人收到） |
| `report_dismissed` | `ReportDismissed` | 举报被驳回（举报人收到） |
| `system_announcement` | `SystemAnnouncement` | 系统公告 |
| `nearby_new_moment` | `NearbyNewMoment` | 附近有人发布新瞬间（用户可配置是否接收；服务端节流，同区域短时间内仅推送一次） |

**客户端处理约定**：
- **实时路径**：在线时收到 WS 事件 → 直接更新本地通知列表/未读角标（无需立即拉 HTTP）
- **补拉路径**：冷启动或 WS 断线超阈值时，调用 `notify/fetch` + `notify/unread_count` 校准
- **WS 负载 ≠ 通知负载**：WS 事件字段以 `api_spec_websocket.md` 为准（含完整 content 等）；通知 `payload` 字段以本文件为准（含截断 preview）。两者结构不同，客户端不应混用

### 通知负载（按 type 区分的联合类型）

```typescript
// 注意：NotifyPayload.type 必须与 NotifyItem.type 一致
type NotifyPayload =
  | { type: 'new_follower'; actor: UserInfo; source: FollowSource | null; source_moment_id: string | null }
  | { type: 'mutual_follow'; actor: UserInfo }
  | { type: 'moment_commented'; actor: UserInfo; moment_id: string; moment_summary: MomentSummary; comment_id: string; comment_preview: string }
  | { type: 'comment_reply'; actor: UserInfo; comment_id: string; replied_comment_preview: string; reply_id: string; reply_content_preview: string; moment_id: string; moment_summary: MomentSummary }
  | { type: 'mentioned'; actor: UserInfo; target_type: 'moment' | 'comment'; moment_id: string; moment_summary: MomentSummary; comment_id: string | null; comment_preview: string | null }
  | { type: 'group_mentioned'; actor: UserInfo; group_id: string; group_name: string; message_id: string; content_preview: string | null }
  | { type: 'reaction_added'; actor: UserInfo; target_type: 'message' | 'moment' | 'comment'; target_id: string; emoji: string }  // 自我反应不产生通知
  | { type: 'group_invite_received'; actor: UserInfo; group_id: string; group_name: string; group_avatar_url: string | null; invite_id: string; message: string | null }
  | { type: 'group_application_received'; applicant: UserInfo; group_id: string; group_name: string; application_id: string; message: string | null }  // 推送给群主/群管理员
  | { type: 'group_application_accepted'; group_id: string; group_name: string; application_id: string }  // 不暴露审批者身份（与 rejected 对称，隐私统一）
  | { type: 'group_application_rejected'; group_id: string; group_name: string; application_id: string; reason: string | null }  // 不暴露审批者身份（F7 对称化）
  | { type: 'report_resolved'; report_id: string; target_type: ReportTargetType; target_id: string; resolved_at: string; note: string | null }      // note 为脱敏后的处理说明
  | { type: 'report_dismissed'; report_id: string; target_type: ReportTargetType; target_id: string; dismissed_at: string; note: string | null }
  | { type: 'system_announcement'; announcement_id: string; title: string; severity: AnnounceSeverity }  // title 为公告标题（客户端据此直接渲染通知行）
  | { type: 'nearby_new_moment'; actor: UserInfo; moment_id: string; moment_summary: MomentSummary; distance: number };  // distance 为距离（米）
```

### 通知项

```typescript
interface NotifyItem {
  notify_id: string;
  type: NotifyType;          // 顶层 type 用于快速筛选
  is_read: boolean;
  created_at: string;
  payload: NotifyPayload;    // 按 type 区分的负载，payload.type == type
}
```

---

## 端点详情

### POST /api/v1/notify/fetch

获取当前用户的通知列表。

- **节点**: /api/v1
- **排序**: 按 `created_at DESC` 排序（最新在前），`notify_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FetchNotifyRequest`
```typescript
interface FetchNotifyRequest {
  type: NotifyType | null;          // 按通知类型筛选（不传则返回所有类型）
  is_read: boolean | null;          // 按已读状态筛选（不传则返回全部）
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<NotifyItem>>`

---

### POST /api/v1/notify/unread_count

获取当前用户的未读通知数。冷启动/重连时调用，用于显示 badge 角标。

- **节点**: /api/v1

**Request**: `UnreadCountRequest`
```typescript
interface UnreadCountRequest {}
```

**Response**: `ApiResponse<UnreadCountResult>`
```typescript
interface UnreadCountResult {
  total: number;               // 总未读通知数
  by_type: Record<NotifyCategory, number>;  // 按类型分类的未读数；NotifyCategory 与映射见 api_spec_types.md
}
```

---

### POST /api/v1/notify/mark_read

标记通知为已读。

**范围语义（参数必显式，无缺省翻转）**：
- `scope='ids'`：精确标记指定通知，`notify_ids` 必填（1-50 个；超限返回参数错误）
- `scope='all'`：全部标记为已读（"全部已读"按钮）

- **节点**: /api/v1

**Request**: `MarkReadRequest`
```typescript
interface MarkReadRequest {
  scope: 'ids' | 'all';     // 必填：标记范围（参数必显式通则，见 api_spec_base.md）
  notify_ids: string[] | null;  // scope='ids' 时必填（1-50 个）
}
```

**非法 ID 处理**：不存在或不属于当前用户的 `notify_ids` 将被服务端静默忽略，不返回错误（不泄漏存在性）。

**Response**: `ApiResponse`