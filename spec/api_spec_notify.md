# HotSpot API - 通知中心

> 本文档定义 HotSpot 通知中心 (`notify`) 相关的 RPC 接口。
> 通知中心提供通知历史查询、未读计数和已读标记等功能，遵循"HTTP API 负责冷启动数据获取，WebSocket 负责实时增量推送"的原则。

## 模块信息

- **命名空间**: `notify`
- **基础路径**: `/api/v1/notify/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 错误码：`api_spec_errors.md`

## 端点总览

| 端点 | 说明 | 认证 | 幂等 |
|-----|------|------|------|
| `POST /api/v1/notify/list` | 通知列表 | 必须 | 无 |
| `POST /api/v1/notify/get_unread_count` | 未读通知数 | 必须 | 无 |
| `POST /api/v1/notify/mark_read` | 标记已读（scope 控制范围） | 必须 | 幂等语义 |

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `UserInfo`
- `MomentSummary`
- `FollowSource`
- `AnnounceLevel`
- `NotifyType`
- `NotifyCategory`
- `PaginationParams`
- `PaginatedData`

### 通知类型枚举

`NotifyType` 定义于 `api_spec_types.md`。

### 通知类型 ↔ WebSocket 事件 ↔ 补拉策略（契约级）

通知中心（HTTP）与 WebSocket 实时事件存在大量语义重叠。客户端收到 WS 事件后，应据此更新通知列表/角标；冷启动或补拉则通过 HTTP 通知 API。两者**互相独立但语义对齐**。

| `NotifyType` | 对应 WebSocket 事件 | 触发场景 |
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
| `report_resolved` | `ReportResolved` | 举报被处理（举报人收到） |
| `report_dismissed` | `ReportDismissed` | 举报被驳回（举报人收到） |
| `system_announcement` | `SystemAnnouncement` | 系统公告 |
| `nearby_new_moment` | `NearbyNewMoment` | 附近有人发布新瞬间（用户可配置是否接收；服务端节流，同区域短时间内仅推送一次） |

**客户端处理约定**：
- **实时路径**：在线时收到 WS 事件 → 直接更新本地通知列表/未读角标（无需立即拉 HTTP）
- **补拉路径**：冷启动或 WS 断线超阈值时，调用 `notify/list` + `notify/get_unread_count` 校准
- **去重**：WS 通知类事件 payload 与 HTTP 通知记录**共用同一主键 `notification_id`**（WS payload 首字段即 `notification_id`）。客户端维护 `Set<notification_id>` 即可精确去重，无需依赖业务组合键。详见 `api_spec_websocket.md`「消息可靠性与去重」。
- **WS 负载 ≠ 通知负载**：WS 事件字段以 `api_spec_websocket.md` 为准（含完整 content 等）；通知 `payload` 字段以本文件为准（含截断 preview）。两者结构不同，客户端不应混用

### 通知负载（按 type 区分的联合类型）

```typescript
// 注意：NotifyPayload.type 必须与 NotifyItem.type 一致
// 设计模式与 AuditDetails（见 api_spec_admin_core.md）保持一致
type NotifyPayload =
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
  | { type: 'report_resolved'; report_id: string; target_type: ReportTargetType; target_id: string; resolved_at: string; note?: string }      // 举报被处理，note 为管理员脱敏后的处理说明
  | { type: 'report_dismissed'; report_id: string; target_type: ReportTargetType; target_id: string; dismissed_at: string; note?: string }    // 举报被驳回
  | { type: 'system_announcement'; announcement_id: string; severity: AnnounceLevel }
  | { type: 'nearby_new_moment'; actor: UserInfo; moment_id: string; moment_summary: MomentSummary; distance: number };  // 附近新瞬间，distance 为距离（米）
```

### 通知项

```typescript
interface NotifyItem {
  notification_id: string;
  type: NotifyType;          // 顶层 type 用于快速筛选
  is_read: boolean;
  created_at: string;
  payload: NotifyPayload;    // 按 type 区分的负载，payload.type == type
}
```

---

## 端点详情

### POST /api/v1/notify/list

获取当前用户的通知列表。

- **认证**: 必须
- **幂等**: 无
- **排序**: 按 `created_at DESC` 排序（最新在前），`notification_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `NotificationListRequest`
```typescript
interface NotificationListRequest {
  type?: NotifyType;            // 按通知类型筛选（不传则返回所有类型）
  is_read?: boolean;                  // 按已读状态筛选（不传则返回全部）
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<NotifyItem>>`

**错误码**: `10000`

---

### POST /api/v1/notify/get_unread_count

获取当前用户的未读通知数。冷启动/重连时调用，用于显示 badge 角标。

- **认证**: 必须
- **幂等**: 无

**Request**: `{}`

**Response**: `ApiResponse<UnreadCountResult>`
```typescript
interface UnreadCountResult {
  total: number;               // 总未读通知数
  by_type: Record<NotifyCategory, number>;  // 按类型分类的未读数；NotifyCategory 与映射见 api_spec_types.md
}
// by_type 的 key 为 NotifyCategory（定义于 api_spec_types.md），其与 NotifyType 的契约级映射亦在 types.md 中声明。
// 新增 NotifyType 时必须在该映射中归类，否则 unread_count 分组计数会遗漏。
```

**错误码**: `10000`

---

### POST /api/v1/notify/mark_read

标记通知为已读（统一端点，通过字段组合控制范围，替代原 mark_as_read + mark_all_read）。

**范围语义**：
- 传 `notification_ids`：精确标记指定通知（0-50 个；空数组视为 no-op 成功）
- 传 `type`：按类型批量标记（将该类型所有未读标记为已读）
- 都不传（`scope: 'all'`）：清空全部未读（"全部已读"按钮）

- **认证**: 必须
- **幂等**: 幂等语义（重复标记已读通知不产生错误）

**Request**: `MarkReadRequest`
```typescript
interface MarkReadRequest {
  scope?: 'selected' | 'all';        // 默认 'selected'；'all' 时清空全部未读（忽略 notification_ids/type）
  notification_ids?: string[];       // scope='selected' 时：精确标记（0-50 个；空数组 no-op 成功；超 50 返回 10000）
  type?: NotifyType;           // scope='selected' 时：按类型批量标记
}
// scope='selected' 时，notification_ids 与 type 二选一（同时传或都不传返回 10000，除非 scope='all'）
```

**非法 ID 处理**：不存在或不属于当前用户的 `notification_ids` 将被服务端静默忽略，不返回错误（不泄漏存在性）。

**Response**: `ApiResponse`

**错误码**: `10000`
