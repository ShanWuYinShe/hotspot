# HotSpot API - 管理后台审核

> 本文档定义 HotSpot 管理后台内容审核相关的 RPC 接口，包括用户管理、群组管理、举报审核、标签管理和系统公告管理。
> 核心管理功能（RBAC、审计、配置）定义在 `api_spec_admin_core.md`。

## 模块信息

- **命名空间**: `admin.users`、`admin.groups`、`admin.reports`、`admin.tags`、`admin.announcements`
- **基础路径**: `/api/v1/admin/users/`、`/api/v1/admin/groups/`、`/api/v1/admin/reports/`、`/api/v1/admin/tags/`、`/api/v1/admin/announcements/`
- **引用**: `api_spec_base.md`、`api_spec_types.md`（含 `UserItem`/`GroupItem`/`ReportItem` 等）、`api_spec_errors.md`、`api_spec_admin_core.md`、`api_spec_system.md`

## 端点总览

| 端点 | 说明 | 认证 |
|-----|------|------|

| `POST /api/v1/admin/users/search` | 搜索用户 | 必须 + `admin.users.read` |
| `POST /api/v1/admin/users/open_the_door` | 用户详情 | 必须 + `admin.users.read` |
| `POST /api/v1/admin/users/set_ban` | 设置用户封禁状态（两态统一） | 必须 + `admin.users.write` |
| `POST /api/v1/admin/groups/search` | 搜索群组 | 必须 + `admin.groups.read` |
| `POST /api/v1/admin/groups/open_the_door` | 群组详情 | 必须 + `admin.groups.read` |
| `POST /api/v1/admin/groups/dismiss` | 解散群组 | 必须 + `admin.groups.write` |
| `POST /api/v1/admin/reports/list` | 举报列表（按状态/类型过滤） | 必须 + `admin.reports.read` |
| `POST /api/v1/admin/reports/handle` | 处理举报 | 必须 + `admin.reports.write` |
| `POST /api/v1/admin/tags/search` | 搜索标签 | 必须 + `admin.tags.read` |
| `POST /api/v1/admin/tags/delete` | 删除/屏蔽标签 | 必须 + `admin.tags.write` |
| `POST /api/v1/admin/announcements/search` | 搜索系统公告 | 必须 + `admin.announcements.read` |
| `POST /api/v1/admin/announcements/create` | 发布系统公告 | 必须 + `admin.announcements.write` |
| `POST /api/v1/admin/announcements/update` | 更新系统公告 | 必须 + `admin.announcements.write` |
| `POST /api/v1/admin/announcements/delete` | 删除系统公告 | 必须 + `admin.announcements.write` |
## 类型定义

以下类型定义于 `api_spec_types.md`：
- `UserItem`
- `GroupItem`
- `SystemAnnouncementItem`
- `PaginationParams`
- `PaginatedData`
- `UserType`
- `PrivacyLevel`
- `ReportStatus`
- `ReportTargetType`
- `ReportAction`
- `TagTrend`
- `AnnounceLevel`
- `UserInfo`
- `ReportItem`

以下 Admin 扩展类型在通用类型基础上扩展管理员视角字段（部分使用 `extends`，部分为结构化扩展）。

### 本模块类型

```typescript
// 账号状态（系统管理员视角）
type UserAccountStatus = 'normal' | 'banned';
```

## 端点详情

### POST /api/v1/admin/users/search

搜索用户（管理员）。

- **认证**: 必须 + `admin.users.read`
- **排序**: 默认按照 `created_at DESC` 排序，时间相同以 `user.user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致。

**Request**: `AdminUserSearchRequest`
```typescript
interface AdminUserSearchRequest {
  user_id?: string;
  nickname?: string;
  login_id?: string;
  status?: UserAccountStatus;
  user_type?: UserType;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<AdminUserItem>>`
```typescript
interface AdminUserItem extends UserItem {
  status: UserAccountStatus;
  banned_until?: string;   // 封禁到期时间（UTC ISO 8601）；status='banned' 时返回，不传/不存在表示永久封禁
  privacy_level: PrivacyLevel;  // 用户的隐私等级（base.md：管理员可查看用户的 privacy_level）
  created_at: string;
  last_active_at?: string;
  login_id: string;
  register_ip?: string;
  register_location?: string;
  last_login_ip?: string;
  last_login_location?: string;
  email?: string;
  phone?: string;
}
```

**错误码**: `10200`

---

### POST /api/v1/admin/users/open_the_door

获取用户详情（管理员）。

- **认证**: 必须 + `admin.users.read`

**Request**: `AdminUserOpenTheDoorRequest`
```typescript
interface AdminUserOpenTheDoorRequest {
  user_id: string;
}
```

**Response**: `ApiResponse<AdminUserItem>`

**错误码**: `10200` `10300`

---

### POST /api/v1/admin/users/set_ban

设置用户封禁状态（两态统一端点，管理员）。

`active=true`：封禁用户。`active=false`：解封用户——对未处于封禁状态的用户调用视为 no-op，返回成功（与 group `members/set_ban` 一致，遵循设状态操作重复达同状态应成功的业界共识）。

- **认证**: 必须 + `admin.users.write`

**安全约束**：
- 禁止封禁自身（返回 `10400`）
- 普通管理员禁止封禁其他管理员（返回 `10200`）
- 封禁超级管理员仅限超级管理员执行（返回 `10200`）
- 封禁/解封操作均记录审计日志（含 no-op 解封尝试，便于审计追溯）

**Request**: `AdminSetUserBanRequest`
```typescript
interface AdminSetUserBanRequest {
  user_id: string;
  active: boolean;        // true=封禁，false=解封
  reason?: string;        // 封禁/解封原因（0-500 字符）
  duration_days?: number; // 仅 active=true：封禁天数，不提供或 0 = 永久封禁；active=false 时忽略
}
```

**Response**: `ApiResponse`

**错误码**: `10200` `10300` `10400`

---

### POST /api/v1/admin/groups/search

搜索群组（管理员）。

- **认证**: 必须 + `admin.groups.read`
- **排序**: 默认按照 `created_at DESC` 排序，时间相同以 `group.group_id ASC` 作为稳定 tie-breaker；一致性级别为强一致。

**Request**: `AdminGroupSearchRequest`
```typescript
interface AdminGroupSearchRequest {
  group_id?: string;
  name?: string;
  owner_id?: string;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<AdminGroupItem>>`
```typescript
interface AdminGroupItem extends GroupItem {
  created_at: string;
  owner: UserInfo;
}
```

**错误码**: `10200`

---

### POST /api/v1/admin/groups/open_the_door

获取群组详情（管理员）。

- **认证**: 必须 + `admin.groups.read`

**Request**: `AdminGroupOpenTheDoorRequest`
```typescript
interface AdminGroupOpenTheDoorRequest {
  group_id: string;
}
```

**Response**: `ApiResponse<AdminGroupItem>`

**错误码**: `10200` `10300`

---

### POST /api/v1/admin/groups/dismiss

解散群组（管理员）。

- **认证**: 必须 + `admin.groups.write`

**安全约束**：
- 解散后群内所有成员收到 `GroupDismissed` WebSocket 事件（`reason='policy_violation'`，不暴露管理员身份）
- 解散操作会记录审计日志（含完整 `reason` 自由文本）
- **reason 可见性**：`reason` 自由文本仅进审计日志（管理员可见），不推送给群成员（成员只收到 `policy_violation` 分类，避免泄露具体处置细节与管理员存在）
- **级联处理（契约级）**：解散时，该群所有 `pending` 状态的申请与邀请由服务端自动置为 `revoked`（与 `groups/dismiss` 的级联约定一致，见 `api_spec_group.md`）

**Request**: `AdminDismissGroupRequest`
```typescript
interface AdminDismissGroupRequest {
  group_id: string;
  reason?: string;  // 解散原因（0-500 字符）
}
```

**Response**: `ApiResponse`

**错误码**: `10200` `10300`

---

### POST /api/v1/admin/reports/list

举报列表（管理员）。按状态/目标类型过滤（无关键字搜索语义，故命名为 list 而非 search）。

- **认证**: 必须 + `admin.reports.read`
- **排序**: 默认按照 `created_at DESC` 排序，时间相同以 `report_id DESC` 作为稳定 tie-breaker；一致性级别为强一致。

**Request**: `AdminReportListRequest`
```typescript
interface AdminReportListRequest {
  status?: ReportStatus;
  target_type?: ReportTargetType;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<AdminReportItem>>`
```typescript
interface AdminReportItem extends ReportItem {
  reporter: UserInfo;
  handled_by?: UserInfo;
  handle_note?: string;
}
```

**错误码**: `10200`

---

### POST /api/v1/admin/reports/handle

处理举报（管理员）。

- **认证**: 必须 + `admin.reports.write`

**安全约束**：
- 仅 `pending` 状态的举报可处理，已终态（`resolved`/`dismissed`）的举报返回 `10400`
- 处理操作会记录审计日志

**动作语义**：
- `resolve`：确认举报有效，执行处理（如封禁用户、删除内容），状态变更为 `resolved`。处理成功后向举报人推送 `ReportResolved` WebSocket 事件并落库通知（`NotifyType='report_resolved'`）
- `dismiss`：驳回举报，不执行任何处理，状态变更为 `dismissed`。驳回后向举报人推送 `ReportDismissed` WebSocket 事件并落库通知（`NotifyType='report_dismissed'`）

**Request**: `AdminReportHandleRequest`
```typescript
interface AdminReportHandleRequest {
  report_id: string;
  action: 'resolve' | 'dismiss';
  note?: string;                // 处理备注（0-500 字符）
  ban_duration_days?: number;   // action=resolve 时可选：封禁天数（0=永久），不提供则仅标记处理不执行封禁。
                                // target_type='user': 直接封禁该用户
                                // target_type='moment'/'comment'/'message': 封禁内容发布者/发送者
                                //   （封禁对象始终为人，不是内容本身）
                                // target_type='group': 不封禁群主（避免连坐无辜），群违规统一走解散群处理；
                                //   管理员对群组违规应调用 admin/groups/dismiss 解散，而非通过本字段封禁群主。
                                //   若群主本人确为违规者，由管理员单独对其发起 users/ban。
}
```

**字段映射说明**：本端点的 `note` 会写入举报详情中的 `handle_note` 字段，并记录到审计日志 `AuditDetails.report.handle.note`。

**Response**: `ApiResponse`

**错误码**: `10200` `10300` `10400`

---

### POST /api/v1/admin/tags/search

搜索标签（管理员）。

- **认证**: 必须 + `admin.tags.read`
- **排序**: 默认按照 `usage_count DESC` 排序，数值相同以 `name ASC` 作为稳定 tie-breaker；一致性级别为弱一致。

**Request**: `AdminTagSearchRequest`
```typescript
interface AdminTagSearchRequest {
  keyword?: string;  // 标签名模糊搜索
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<AdminTagItem>>`
```typescript
// 扩展 TagItem（定义于 api_spec_types.md）添加管理员视角字段
interface AdminTagItem extends TagItem {
  tag_id: string;        // 标签唯一标识（内部存储标识，仅用于展示/审计关联，不参与删除寻址）
  created_at: string;
}
```

**错误码**: `10200`

---

### POST /api/v1/admin/tags/delete

屏蔽标签（管理员）。

**删除语义（软屏蔽，保护历史数据完整性）**：
- 默认**软屏蔽**：标签不再可被新建/搜索命中，但**已引用该标签的历史瞬间/群组/用户数据保留不变**（不回写改写历史记录），避免大范围不可逆的数据修改
- 物理删除（从所有历史引用中移除）为**高危操作**，需在 Request 中显式传 `hard_delete: true` 确认；物理删除不可逆，会改写所有引用该标签的资源
- 屏蔽后同名标签不可再次创建（当前不提供取消屏蔽端点；误屏蔽需服务端维护处理）
- **新内容约束**：被屏蔽标签出现在新建内容（瞬间/群组/用户资料的 `tags`）中时，该请求失败（`10400`，message 指明被屏蔽的标签名）——防止绕过屏蔽重新引入
- **寻址说明**：标签的业务唯一键为 `name`（与普通 API `TagItem` 一致，`TagItem` 定义于 `api_spec_types.md`）；`AdminTagItem.tag_id`（定义于本文件「类型定义」小节）为内部存储标识，仅用于展示/审计关联，不参与删除寻址

- **认证**: 必须 + `admin.tags.write`

**Request**: `AdminDeleteTagRequest`
```typescript
interface AdminDeleteTagRequest {
  name: string;            // 要屏蔽/删除的标签名
  hard_delete?: boolean;   // 默认 false（软屏蔽）；true 时物理删除（从所有历史引用移除，不可逆，慎用）
}
```

**Response**: `ApiResponse`

**错误码**: `10200` `10300`

---

### POST /api/v1/admin/announcements/search

搜索系统公告（管理员视角），返回包含管理员字段的公告列表。

- **认证**: 必须 + `admin.announcements.read`
- **排序**: 默认按照 `created_at DESC` 排序，时间相同以 `announcement_id DESC` 作为稳定 tie-breaker；一致性级别为弱一致。

**说明**：
- 普通用户通过 `POST /api/v1/system/list_announcements`（定义于 `api_spec_system.md`）获取公开公告
- 此端点返回 `AdminAnnouncementItem`（含 `created_by`、`view_count` 等管理员字段）

**Request**: `AdminAnnouncementSearchRequest`
```typescript
interface AdminAnnouncementSearchRequest {
  severity?: AnnounceLevel;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<AdminAnnouncementItem>>`
```typescript
interface AdminAnnouncementItem extends SystemAnnouncementItem {
  created_by: UserInfo;
  send_push: boolean;
  push_sent_at?: string;
  view_count: number;
}
```

**错误码**: `10200`

---

### POST /api/v1/admin/announcements/create

发布系统公告（管理员）。

- **认证**: 必须 + `admin.announcements.write`
- **幂等**: 是（客户端必须携带 `Idempotency-Key`）

**Request**: `CreateAnnouncementRequest`
```typescript
interface CreateAnnouncementRequest {
  title: string;
  content: string;
  severity: AnnounceLevel;
  action_url?: string;
  send_push?: boolean;
}
```

**Response**: `ApiResponse<AdminAnnouncementItem>`（返回完整公告对象，客户端直接渲染；`AdminAnnouncementItem` 定义见本文件「类型定义」小节）

**错误码**: `10200`

---

### POST /api/v1/admin/announcements/update

更新系统公告（管理员）。

- **认证**: 必须 + `admin.announcements.write`

**Request**: `UpdateAnnouncementRequest`
```typescript
interface UpdateAnnouncementRequest {
  announcement_id: string;
  title?: string;
  content?: string;
  severity?: AnnounceLevel;
  action_url?: string;
}
```

**Response**: `ApiResponse<AdminAnnouncementItem>`（内容类更新返回更新后完整对象，见 `api_spec_base.md`「设计规范」）

**错误码**: `10200` `10300`

---

### POST /api/v1/admin/announcements/delete

删除公告（管理员）。

- **认证**: 必须 + `admin.announcements.write`

**Request**: `DeleteAnnouncementRequest`
```typescript
interface DeleteAnnouncementRequest {
  announcement_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10200` `10300`
