# HotSpot API - 通知中心

> 本文档定义 HotSpot 通知中心 (`notifications`) 相关的 RPC 接口。
> 通知中心提供通知历史查询、未读计数和已读标记等功能，替代 WebSocket 连接事件中的 `unread_summary`，遵循"HTTP API 负责冷启动数据获取，WebSocket 负责实时增量推送"的原则。

## 模块信息

- **命名空间**: `notifications`
- **基础路径**: `/api/v1/notifications/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 错误码：`api_spec_errors.md`

## 端点总览

| 端点 | 说明 | 认证 | 幂等 |
|-----|------|------|------|
| `POST /api/v1/notifications/list` | 通知列表 | 必须 | 无 |
| `POST /api/v1/notifications/unread_count` | 未读通知数 | 必须 | 无 |
| `POST /api/v1/notifications/mark_as_read` | 标记已读 | 必须 | 幂等语义 |
| `POST /api/v1/notifications/mark_all_read` | 标记全部已读 | 必须 | 幂等语义 |

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `UserInfo`
- `MomentSummary`
- `FollowSource`
- `AnnouncementSeverity`
- `PaginationParams`
- `PaginatedData`

### 通知类型枚举

`NotificationType` 定义于 `api_spec_types.md`。

### 通知类型 ↔ WebSocket 事件 ↔ 补拉策略（契约级）

通知中心（HTTP）与 WebSocket 实时事件存在大量语义重叠。客户端收到 WS 事件后，应据此更新通知列表/角标；冷启动或补拉则通过 HTTP 通知 API。两者**互相独立但语义对齐**。

| `NotificationType` | 对应 WebSocket 事件 | 触发场景 |
|-------------------|--------------------|---------|
| `new_follower` | `NewFollower` | 被关注 |
| `mutual_follow` | `MutualFollow` | 形成互相关注 |
| `moment_liked` | `MomentLiked` | 瞬间被点赞 |
| `moment_commented` | `MomentCommented` | 瞬间被评论 |
| `comment_liked` | `CommentLiked` | 评论被点赞 |
| `comment_reply` | `CommentReply` | 评论被回复 |
| `mentioned` | `MessageMentioned` | 瞬间/评论中被 @ |
| `group_invite_received` | `GroupInviteReceived` | 收到入群邀请 |
| `group_invite_accepted` | `GroupInviteResponse` | 邀请被接受（邀请方收到） |
| `group_application_received` | `GroupApplicationReceived` | 群主/管理员收到入群申请 |
| `group_application_accepted` | `GroupApplicationResponse` | 申请被接受（申请方收到） |
| `system_announcement` | `SystemAnnouncement` | 系统公告 |

**客户端处理约定**：
- **实时路径**：在线时收到 WS 事件 → 直接更新本地通知列表/未读角标（无需立即拉 HTTP）
- **补拉路径**：冷启动或 WS 断线超阈值时，调用 `notifications/list` + `notifications/unread_count` 校准
- **去重**：WS 事件与 HTTP 补拉可能重叠，按 `notification_id` 去重（持久化事件）或按"最新状态覆盖"（状态快照）
- **WS 负载 ≠ 通知负载**：WS 事件字段以 `api_spec_websocket.md` 为准（含完整 content 等）；通知 `payload` 字段以本文件为准（含截断 preview）。两者结构不同，客户端不应混用

### 通知负载（按 type 区分的联合类型）

```typescript
// 注意：NotificationPayload.type 必须与 NotificationItem.type 一致
// 设计模式与 AuditDetails（见 api_spec_admin_core.md）保持一致
type NotificationPayload =
  | { type: 'new_follower'; actor: UserInfo; source?: FollowSource; source_moment_id?: string }
  | { type: 'mutual_follow'; actor: UserInfo }
  | { type: 'moment_liked'; actor: UserInfo; moment_id: string; moment_summary: MomentSummary }
  | { type: 'moment_commented'; actor: UserInfo; moment_id: string; moment_summary: MomentSummary; comment_id: string; comment_preview: string }
  | { type: 'comment_liked'; actor: UserInfo; comment_id: string; comment_preview: string; moment_id: string; moment_summary: MomentSummary }
  | { type: 'comment_reply'; actor: UserInfo; comment_id: string; replied_comment_preview: string; reply_id: string; reply_content_preview: string; moment_id: string; moment_summary: MomentSummary }
  | { type: 'mentioned'; actor: UserInfo; target_type: 'moment' | 'comment'; moment_id: string; moment_summary: MomentSummary; comment_id?: string; comment_preview?: string }
  | { type: 'group_invite_received'; actor: UserInfo; group_id: string; group_name: string; group_avatar_url?: string; invite_id: string; message?: string }
  | { type: 'group_invite_accepted'; group_id: string; group_name: string; invite_id: string; user: UserInfo }  // 推送给邀请发起者，user 为接受邀请的人
  | { type: 'group_application_received'; applicant: UserInfo; group_id: string; group_name: string; application_id: string }  // 推送给群主/管理员
  | { type: 'group_application_accepted'; group_id: string; group_name: string; application_id: string }  // 不暴露审批者身份（隐私考虑：审批者可能是群管理员或群主，不应在通知中暴露其身份给申请者）
  | { type: 'system_announcement'; announcement_id: string; severity: AnnouncementSeverity };
```

### 通知项

```typescript
interface NotificationItem {
  notification_id: string;
  type: NotificationType;          // 顶层 type 用于快速筛选
  is_read: boolean;
  created_at: string;
  payload: NotificationPayload;    // 按 type 区分的负载，payload.type == type
}
```

---

## 端点详情

### POST /api/v1/notifications/list

获取当前用户的通知列表。

- **认证**: 必须
- **幂等**: 无
- **排序**: 按 `created_at DESC` 排序（最新在前），`notification_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `NotificationListRequest`
```typescript
interface NotificationListRequest {
  type?: NotificationType;            // 按通知类型筛选（不传则返回所有类型）
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<NotificationItem>>`

**错误码**: `10000`

---

### POST /api/v1/notifications/unread_count

获取当前用户的未读通知数。冷启动/重连时调用，用于显示 badge 角标。

- **认证**: 必须
- **幂等**: 无

**Request**: `{}`

**Response**: `ApiResponse<UnreadCountResult>`
```typescript
interface UnreadCountResult {
  total: number;               // 总未读通知数
  by_type: {                   // 按类型分类的未读数（映射关系见下表）
    social: number;
    interaction: number;
    group: number;
    system: number;
  };
}
// NotificationType -> by_type 映射（契约级，实现必须遵循）：
// | by_type       | NotificationType                                                      |
// |---------------|-----------------------------------------------------------------------|
// | social        | new_follower, mutual_follow                                           |
// | interaction   | moment_liked, moment_commented, comment_liked, comment_reply, mentioned|
// | group         | group_invite_received, group_invite_accepted, group_application_received, group_application_accepted |
// | system        | system_announcement                                                   |
```

**错误码**: `10000`

---

### POST /api/v1/notifications/mark_as_read

批量标记指定通知为已读。

- **认证**: 必须
- **幂等**: 幂等语义（重复标记同一通知不产生错误）

**Request**: `MarkAsReadRequest`
```typescript
interface MarkAsReadRequest {
  notification_ids: string[];  // 要标记为已读的通知 ID 列表（1-100 个，空数组或超过 100 个返回 10000）
}
```

**非法 ID 处理**：不存在或不属于当前用户的 `notification_ids` 将被服务端静默忽略，不返回错误。

**Response**: `ApiResponse`

**错误码**: `10000`

---

### POST /api/v1/notifications/mark_all_read

标记当前用户的所有通知为已读。

- **认证**: 必须
- **幂等**: 幂等语义（重复调用不产生错误）

**Request**: `{}`

**Response**: `ApiResponse`

**错误码**: `10000`
