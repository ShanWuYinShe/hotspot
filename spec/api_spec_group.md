# HotSpot API - 群组管理

> 本文档定义 HotSpot 群组管理 (`groups`) 相关的 RPC 接口，包括群组 CRUD、成员管理、申请/邀请管理和群公告。
> 群聊消息 (`group_messages`) 接口定义在 `api_spec_chat.md`。

## 模块信息

- **命名空间**: `groups`
- **基础路径**: `/api/v1/groups/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 错误码：`api_spec_errors.md`
  - WebSocket：`api_spec_websocket.md`

## 端点总览

| 端点 | 说明 | 认证 | 幂等 |
|-----|------|------|------|
| `POST /api/v1/groups/search` | 搜索群组（支持关键字/标签/位置组合） | 可选 | 无 |
| `POST /api/v1/groups/get_detail` | 群组详情 | 可选 | 无 |
| `POST /api/v1/groups/create` | 创建群组 | 必须 | MUST |
| `POST /api/v1/groups/update` | 更新群组 | 必须 | 幂等语义 |
| `POST /api/v1/groups/dismiss` | 解散群组 | 必须 | 幂等语义 |
| `POST /api/v1/groups/members/list` | 成员列表 | 可选 | 无 |
| `POST /api/v1/groups/members/search` | 搜索成员 | 可选 | 无 |
| `POST /api/v1/groups/members/invite` | 邀请成员 | 必须 | 幂等语义 |
| `POST /api/v1/groups/members/invites/list` | 我发出的邀请列表 | 必须 | 无 |
| `POST /api/v1/groups/members/invites/revoke` | 撤销已发邀请 | 必须 | 幂等语义 |
| `POST /api/v1/groups/members/set_role` | 设置角色 | 必须 | 幂等语义 |
| `POST /api/v1/groups/members/set_mute` | 设置禁言状态（两态统一） | 必须 | 幂等语义 |
| `POST /api/v1/groups/members/ban` | 封禁成员 | 必须 | 幂等语义 |
| `POST /api/v1/groups/members/unban` | 解除封禁 | 必须 | 幂等语义 |
| `POST /api/v1/groups/members/list_banned` | 封禁列表 | 必须 | 无 |
| `POST /api/v1/groups/members/remove` | 移除成员 | 必须 | 幂等语义 |
| `POST /api/v1/groups/applications/list` | 申请列表 | 必须 | 无 |
| `POST /api/v1/groups/applications/approve` | 同意申请 | 必须 | 幂等语义 |
| `POST /api/v1/groups/applications/reject` | 拒绝申请 | 必须 | 幂等语义 |
| `POST /api/v1/groups/update_preferences` | 更新我的群组偏好 | 必须 | 幂等语义 |
| `POST /api/v1/groups/join` | 加入群组（根据 join_method 自动处理） | 必须 | MUST |
| `POST /api/v1/groups/transfer_ownership` | 转让群主 | 必须 | SHOULD |
| `POST /api/v1/groups/announcements/create` | 发布公告 | 必须 | MUST |
| `POST /api/v1/groups/announcements/list` | 公告列表 | 可选 | 无 |
| `POST /api/v1/groups/announcements/update` | 更新公告 | 必须 | 幂等语义 |
| `POST /api/v1/groups/announcements/delete` | 删除公告 | 必须 | 幂等语义 |
| `POST /api/v1/groups/list_my` | 我的群组列表 | 必须 | 无 |
| `POST /api/v1/groups/list_applications` | 我的申请列表 | 必须 | 无 |
| `POST /api/v1/groups/cancel_application` | 取消申请 | 必须 | 幂等语义 |
| `POST /api/v1/groups/list_invites` | 我的邀请列表 | 必须 | 无 |
| `POST /api/v1/groups/accept_invite` | 接受邀请 | 必须 | 幂等语义 |
| `POST /api/v1/groups/reject_invite` | 拒绝邀请 | 必须 | 幂等语义 |
| `POST /api/v1/groups/leave` | 退出群组 | 必须 | 幂等语义 |

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `GroupInfo`
- `GroupItem`
- `GroupSettings`
- `MyGroupState`
- `GroupAnnouncementItem`
- `GroupRole`
- `GroupJoinMethod`
- `Location`
- `PaginationParams`
- `GeoPoint`
- `PaginatedData`
- `UserInfo`
- `UserPresenceStatus`

### 本模块类型

```typescript
// 可设置的角色（不含 owner，仅在本模块使用）
type SettableGroupRole = 'admin' | 'member';

// 申请/邀请状态（群申请与群邀请共用同一套状态枚举）
// 'revoked': 邀请/申请发起者主动撤销，或过期自动撤销（群已不存在/被 ban/超时）
type ApplicationStatus = 'pending' | 'accepted' | 'rejected' | 'revoked';
```

## 端点详情

### POST /api/v1/groups/search

统一搜索群组。支持关键字、标签、位置范围三种条件的自由组合。

- **认证**: 可选
- **幂等**: 无
- **排序**: 
  - 默认 `sort_by='relevance'`：按相关性优先排序
  - `sort_by='distance'`：按距离由近到远排序（未提供 `radius` 时按默认 `5_000` 米计算）
  - `sort_by='newest'`：按创建时间倒序
  - `sort_by='popular'`：按成员数/活跃度排序
  - **稳定 tie-breaker**：任意 `sort_by` 模式下，当主序字段相同，以 `group_id ASC` 作为稳定排序，保证游标分页一致性
  - 一致性级别为弱一致（搜索场景允许短暂延迟）

**Request**: `SearchGroupsRequest`
```typescript
interface SearchGroupsRequest {
  keyword?: string;    // 关键字搜索（匹配名称、描述等）
  tag?: string;        // 标签筛选
  radius?: number;     // 位置范围（米），100-50000，配合 reference_point 使用；未提供时默认 5_000 米，并受 AppConfig.limits.search_radius_max 上限约束
  sort_by?: 'relevance' | 'distance' | 'newest' | 'popular';  // 默认: relevance
  reference_point?: GeoPoint; // 浏览锚点位置（可选；不传时服务端按 api_spec_types.md「reference_point 解析规则」回退链解析）
  pagination?: PaginationParams;
}
// 至少提供 keyword、tag、radius 中的一个；不满足时返回 10000（参数错误）。
// reference_point 不算搜索条件，仅用于距离计算和排序，不能单独作为搜索条件。
// 当仅缺省 radius 且提供了 keyword/tag 时，服务端按默认 5_000 米计算，并受 AppConfig.limits.search_radius_max 上限约束。
```

**Response**: `ApiResponse<PaginatedData<GroupItem>>`

**错误码**: `10000`

---

### POST /api/v1/groups/get_detail

获取群组详情。认证可选，字段可见性规则见 `api_spec_base.md`。

- **认证**: 可选
- **幂等**: 无

**Request**: `GroupOpenTheDoorRequest`
```typescript
interface GroupOpenTheDoorRequest {
  group_id: string;
}
```

**Response**: `ApiResponse<GroupItem>`

**错误码**: `10000` `10300`

---

### POST /api/v1/groups/create

创建群组。

- **认证**: 必须
- **幂等**: MUST（客户端必须携带 `Idempotency-Key`）

**Request**: `CreateGroupRequest`
```typescript
interface CreateGroupRequest {
  name: string;
  description?: string;
  avatar_file_id?: string;
  tags?: string[];
  password?: string;
  join_method?: GroupJoinMethod;  // 默认: 'open'
  is_searchable?: boolean;         // 默认: true
  max_members?: number;            // 默认: 200
  location: Location;
}
```

**Response**: `ApiResponse<GroupItem>`
```typescript
// 群组创建响应：直接返回完整 GroupItem（与 moments/create 返回 MomentItem 保持一致）
// 不再使用独立的 GroupCreated 类型，客户端无需额外请求即可获得完整群组数据
```

**错误码**: `10000` `10300`

---

### POST /api/v1/groups/update

更新群组信息。

- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `UpdateGroupRequest`
```typescript
interface UpdateGroupRequest {
  group_id: string;
  name?: string;
  description?: string;
  avatar_file_id?: string;
  tags?: string[];
  password?: string;
  clear_password?: boolean;  // true 时移除群密码（与 password 互斥，同时传入视为参数冲突返回 10000）
  join_method?: GroupJoinMethod;
  is_searchable?: boolean;
  max_members?: number;
  location?: Location;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300`

---

### POST /api/v1/groups/dismiss

解散群组（仅群主）。

**解散行为**：解散后向所有在线群成员推送 WebSocket `GroupDismissed` 事件（`reason='owner_dismissed'`）。不暴露解散者身份（隐私保护，操作记录仅保留在审计日志）。

**级联处理（契约级）**：解散时，该群所有 `pending` 状态的申请与邀请由服务端自动置为 `revoked`（群已不存在，无需推送单独事件；客户端下次拉取列表时自然不可见）。

- **认证**: 必须
- **幂等**: 幂等语义（重复解散已不存在的群视为 no-op 成功）

**Request**: `DismissGroupRequest`
```typescript
interface DismissGroupRequest {
  group_id: string;
  reason?: string;  // 解散说明（0-500 字符，仅进审计日志；成员收到的是 owner_dismissed 分类，不接收自由文本）
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300`

---

### POST /api/v1/groups/members/list

获取成员列表。

- **认证**: 可选（未登录或未加入群组的用户仅能看到公开字段，`@group-admin-only` 字段仍按可见性控制）
- **幂等**: 无
- **排序**: 按 `joined_at` 时间正序（最早加入在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `GroupMembersRequest`
```typescript
interface GroupMembersRequest {
  group_id: string;
  role?: GroupRole;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<GroupMemberItem>>`
```typescript
// 群组成员列表项
interface GroupMemberItem {
  user: UserInfo;
  role: GroupRole;
  joined_at: string;
  muted_until?: string;           // @group-admin-only（或被禁言者自身可见）
  presence_status?: UserPresenceStatus;  // 隐私受控：成员隐私设置时不返回
  last_online_at?: string;
}
```

**错误码**: `10000` `10300`

---

### POST /api/v1/groups/members/search

搜索群组成员。用于大群场景中快速定位特定成员。

- **认证**: 可选（未登录或非成员与未加入用户看到相同字段；`@group-admin-only` 字段仅群管理员可见）
- **幂等**: 无
- **排序**: 按匹配度优先，同匹配度按 `joined_at` 正序，`user_id ASC` 作为稳定 tie-breaker；一致性级别为弱一致（搜索场景允许短暂延迟）

**Request**: `SearchGroupMembersRequest`
```typescript
interface SearchGroupMembersRequest {
  group_id: string;
  keyword: string;    // 按昵称搜索
  role?: GroupRole;   // 可选：按角色过滤
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<GroupMemberItem>>`

**错误码**: `10000` `10300` `10400`

`10400` 典型场景：调用方不具备当前群组成员搜索资格（如未加入群组且群设置不允许检索）。

---

### POST /api/v1/groups/members/invite

邀请成员。

**重复/已达目标状态处理**（幂等语义，与业界共识一致）：
- 若对同一用户存在 pending 状态的邀请，返回已有邀请信息（不重复创建），避免重复通知骚扰。
- 被邀请用户**已是群成员**时，视为目标已达，返回成功（no-op，不创建邀请、不报错）。

- **认证**: 必须
- **幂等**: 幂等语义（对同一用户重复邀请返回已有邀请信息，已是群成员视为 no-op 成功）

**Request**: `InviteMemberRequest`
```typescript
interface InviteMemberRequest {
  group_id: string;
  user_id: string;
  message?: string;
}
```

**Response**: `ApiResponse<Invitation>`
```typescript
// 邀请响应（被邀请者已是成员时，返回成功但不创建邀请，此时邀请信息可省略/为占位）
interface Invitation {
  invite_id: string;
  expires_at: string;
}
```

**错误码**: `10000` `10200` `10300` `10400`（`10400` 仅用于真正无法完成的冲突，如邀请已被封禁的用户）

---

### POST /api/v1/groups/members/invites/list

我发出的邀请列表。权限：群主、群管理员或系统管理员。

- **认证**: 必须
- **幂等**: 无
- **排序**: 默认按照 `created_at DESC` 排序，时间相同以 `invite_id DESC` 作为稳定 tie-breaker；一致性级别为强一致。

**Request**: `GroupSentInvitesRequest`
```typescript
interface GroupSentInvitesRequest {
  group_id: string;
  status?: ApplicationStatus | 'all';  // 默认 'pending'；查历史需显式传 'all'
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<GroupSentInviteItem>>`
```typescript
interface GroupSentInviteItem {
  invite_id: string;
  invitee: UserInfo;
  inviter: UserInfo;           // 邀请发起者（多管理员场景可见是谁发的）
  status: ApplicationStatus;
  message?: string;
  created_at: string;
  expires_at?: string;         // UTC ISO 8601，由 AppConfig.limits.group_invite_ttl_hours 控制
  responded_at?: string;       // UTC ISO 8601，进入终态时间
}
```

`ApplicationStatus='revoked'` 表示邀请由发起者主动撤销（含过期自动撤销），客户端需与 `rejected`（受邀者拒绝）区分展示。

**过期语义（契约级）**：超过 `expires_at` 的 pending 邀请由服务端自动置为 `revoked`（与发起者主动撤销同状态，便于客户端统一展示为"已失效"），并向受邀人推送 `GroupInviteExpired` 事件（见 `api_spec_websocket.md`）。`members/invites/list` 默认仅返回 `pending`，查历史需显式传 `status`。

**错误码**: `10000` `10200` `10300` `10400`

---

### POST /api/v1/groups/members/invites/revoke

撤销已发出的待处理邀请。权限：群主、群管理员或系统管理员。

- **认证**: 必须
- **幂等**: 幂等语义（重复撤销返回成功）

**Request**: `RevokeSentInviteRequest`
```typescript
interface RevokeSentInviteRequest {
  group_id: string;
  invite_id: string;
  reason?: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300` `10400`

---

### 成员管理

以下成员管理操作统一权限要求：群主或群组管理员可操作，系统管理员按 RBAC 权限可操作。

### POST /api/v1/groups/members/set_role

设置成员角色。幂等语义：设置为目标角色与当前一致时视为 no-op，返回成功。

**降级与 pending 邀请**：邀请是群行为（非个人），inviter 降级为普通成员后，其历史发出的 pending 邀请由其他管理员/owner 继续可见与管理（不自动失效）。**owner 降级**（转交所有权）前若有 pending 邀请，需先 `members/invites/revoke` 或在 `transfer_ownership` 中一并转交管理权，否则返回 `10400`。

- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果；目标角色与当前一致时返回成功）

**Request**: `SetMemberRoleRequest`
```typescript
interface SetMemberRoleRequest {
  group_id: string;
  user_id: string;
  role: SettableGroupRole;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300` `10400`（`10400` 用于真正无法完成的状态冲突，如降级最后一个管理员等）

---

### POST /api/v1/groups/members/set_mute

设置成员禁言状态（两态统一端点，替代原 mute/unmute 成对动词）。

- **认证**: 必须
- **幂等**: 幂等语义（`active=false` 对未禁言成员视为 no-op 成功；`active=true` 重复禁言刷新 duration）

**Request**: `SetMuteRequest`
```typescript
interface SetMuteRequest {
  group_id: string;
  user_id: string;
  active: boolean;
  duration_minutes?: number;  // active=true 时必填（1-43200，最多 30 天）；active=false 时忽略
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300` `10400`

---

### POST /api/v1/groups/members/ban

封禁成员。封禁后成员将被移出群组且无法再次加入。

**级联处理（契约级）**：封禁时，被 ban 用户**发出的**该群 pending 申请置 `revoked`；**发给**被 ban 用户的该群 pending 邀请置 `revoked`（避免被封用户仍收到邀请再加入）。

- **认证**: 必须
- **幂等**: 幂等语义（对已封禁成员重复封禁视为 no-op 成功）

**Request**: `BanMemberRequest`
```typescript
interface BanMemberRequest {
  group_id: string;
  user_id: string;
  reason?: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300` `10400`

---

### POST /api/v1/groups/members/unban

解除封禁成员。幂等语义：对未处于封禁状态的成员调用视为 no-op，返回成功（不报错），与 Discord/Slack 等"幂等设状态操作重复达同状态应成功"的业界共识一致。

- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果；未封禁成员视为已达目标状态，返回成功）

**Request**: `UnbanMemberRequest`
```typescript
interface UnbanMemberRequest {
  group_id: string;
  user_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300`

---

### POST /api/v1/groups/members/list_banned

获取封禁列表。

- **认证**: 必须
- **幂等**: 无
- **排序**: 按 `banned_at` 时间倒序（最近封禁在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `BannedMembersRequest`
```typescript
interface BannedMembersRequest {
  group_id: string;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<BannedMemberItem>>`
```typescript
// 封禁成员列表项
interface BannedMemberItem {
  user: UserInfo;
  banned_at: string;
  // === @group-admin-only ===
  reason?: string;
  banned_by?: UserInfo;
}
```

**错误码**: `10000` `10200` `10300`

---

### POST /api/v1/groups/members/remove

移除成员。

- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `RemoveMemberRequest`
```typescript
interface RemoveMemberRequest {
  group_id: string;
  user_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300` `10400`

---

### POST /api/v1/groups/applications/list

获取群组申请列表。

- **认证**: 必须
- **幂等**: 无
- **排序**: 按 `created_at` 时间倒序（最新申请在前），`application_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `GroupApplicationsRequest`
```typescript
interface GroupApplicationsRequest {
  group_id: string;
  status?: ApplicationStatus | 'all';  // 默认 'pending'（仅待处理，管理员工作台语义）；查历史需显式传 'all'
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<GroupApplicationItem>>`
```typescript
// ApplicationStatus 定义于本模块「类型定义」小节

// 群组申请列表项（群管理员视角与申请人视角共用此类型）
interface GroupApplicationItem {
  application_id: string;
  group: GroupInfo;              // 目标群组（申请人视角的 me/applications/list 跨群，需群上下文）
  applicant: UserInfo;
  message?: string;
  status: ApplicationStatus;
  created_at: string;
  expires_at?: string;           // UTC ISO 8601，申请有效期截止（由 AppConfig.limits.group_application_ttl_hours 控制）；0 TTL 时省略
  handled_at?: string;           // UTC ISO 8601，进入终态（accepted/rejected/revoked）的时间
  // === @group-admin-only（群管理员视角返回，申请人视角的 handled_by 不返回）===
  handled_by?: UserInfo;
}
```

**错误码**: `10000` `10200` `10300`

---

### POST /api/v1/groups/applications/approve

审批群组申请。

**边界情况**：对已处于 `accepted` 状态的申请重复调用，按幂等语义返回成功。

- **认证**: 必须
- **幂等**: 幂等语义（对已处于 accepted 状态的申请重复调用返回成功）

**Request**: `ApproveGroupApplicationRequest`
```typescript
interface ApproveGroupApplicationRequest {
  group_id: string;        // 群级鉴权（与 revoke 对齐）；须与 application 所属群一致，否则 10000
  application_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300` `10400`

---

### POST /api/v1/groups/applications/reject

拒绝群组申请。拒绝后向申请人推送 `GroupApplicationRejected` 通知（含 `reason`，对应 `NotifyType='group_application_rejected'`）。

- **认证**: 必须
- **幂等**: 幂等语义（对已处于 rejected 状态的申请重复调用返回成功）

**Request**: `RejectGroupApplicationRequest`
```typescript
interface RejectGroupApplicationRequest {
  group_id: string;        // 群级鉴权
  application_id: string;
  reason?: string;         // 0-200 字符，拒绝原因，会随通知送达申请人
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300` `10400`

---

### POST /api/v1/groups/announcements/create

发布群公告。

- **认证**: 必须
- **幂等**: MUST（客户端必须携带 `Idempotency-Key`）

**Request**: `CreateGroupAnnouncementRequest`
```typescript
interface CreateGroupAnnouncementRequest {
  group_id: string;
  title: string;
  content: string;
  is_pinned?: boolean;
}
```

**Response**: `ApiResponse<GroupAnnouncementItem>`

**错误码**: `10000` `10200` `10300` `10400`

---

### POST /api/v1/groups/announcements/list

获取群公告列表。

- **认证**: 可选
- **幂等**: 无
- **排序**: 按 `is_pinned DESC, created_at DESC` 排序，`announcement_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `GroupAnnouncementsRequest`
```typescript
interface GroupAnnouncementsRequest {
  group_id: string;
}
// 返回条数由 AppConfig.response_defaults.group_announcements_count 控制
```

**Response**: `ApiResponse<{ items: GroupAnnouncementItem[] }>`

**说明**：
- 普通用户只需获取最新的 N 条群公告，无需分页翻阅历史

**错误码**: `10000` `10300`

---

### POST /api/v1/groups/announcements/update

更新群公告。

- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `UpdateGroupAnnouncementRequest`
```typescript
interface UpdateGroupAnnouncementRequest {
  group_id: string;
  announcement_id: string;
  title?: string;
  content?: string;
  is_pinned?: boolean;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300`

---

### POST /api/v1/groups/announcements/delete

删除群公告。

- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `DeleteGroupAnnouncementRequest`
```typescript
interface DeleteGroupAnnouncementRequest {
  group_id: string;
  announcement_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300`

---

### POST /api/v1/groups/update_preferences

更新我的群组偏好（静音/置顶，区别于 `groups/update` 群信息更新）。


- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `UpdateGroupMySettingsRequest`
```typescript
interface UpdateGroupMySettingsRequest {
  group_id: string;
  is_muted?: boolean;
  is_pinned?: boolean;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10300` `10400`

---

### POST /api/v1/groups/join

加入群组（智能路由）。根据群组 `join_method` 自动处理：
- `open`：直接加入，返回 `status: 'joined'`
- `open` + 有密码：需提供 `password`，验证通过后直接加入
- `approval`：创建加入申请，返回 `status: 'pending_approval'`
- `invite_only`：返回错误 `10400`（该群组仅限邀请加入）


- **认证**: 必须
- **幂等**: MUST（客户端必须携带 `Idempotency-Key`）

**Request**: `JoinGroupRequest`
```typescript
interface JoinGroupRequest {
  group_id: string;
  password?: string;   // open+password 群组需要
  message?: string;    // approval 群组可选附言
}
```

**Response**: `ApiResponse<JoinGroupResult>`
```typescript
// 申请响应
interface Application {
  application_id: string;
}

// 加入群组响应
interface JoinGroupResult {
  status: 'joined' | 'pending_approval';
  application?: Application;  // 仅 pending_approval 时返回
}
```

**错误码**: `10000` `10200` `10300` `10400`

---

### POST /api/v1/groups/transfer_ownership

转让群主。

**边界情况**：`new_owner_id` 不是该群成员时，返回 `10400`（"目标用户不是群成员"）。

- **认证**: 必须
- **幂等**: SHOULD（建议客户端携带 `Idempotency-Key`）

**Request**: `TransferOwnershipRequest`
```typescript
interface TransferOwnershipRequest {
  group_id: string;
  new_owner_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300` `10400`

---

### POST /api/v1/groups/list_my

获取我的群组。


**说明**：登录态必定返回 `my_state` 字段，包含 `role`、`is_joined`、`unread_count`、`is_muted`、`is_pinned`、`joined_at` 等群组内状态。本端点无 `reference_point` 参数，返回的 `GroupItem.location` 中 `distance` 和 `bearing_degrees` 为 `undefined`。

- **认证**: 必须
- **幂等**: 无
- **排序**: 默认按 `is_pinned DESC, last_activity_at DESC` 排序（置顶优先，最近活跃在前），`group_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `MyGroupsRequest`
```typescript
interface MyGroupsRequest {
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<GroupItem>>`

**错误码**: `10000`

---

### POST /api/v1/groups/list_applications

获取我的群组申请列表。查看当前用户提交的所有群组申请及其审核状态。

- **认证**: 必须
- **幂等**: 无
- **排序**: 按 `created_at` 时间倒序（最新申请在前），`application_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `MyApplicationsRequest`
```typescript
interface MyApplicationsRequest {
  status?: ApplicationStatus | 'all';  // 默认 'pending'（我待处理的申请）；查历史需显式传 'all'
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<GroupApplicationItem>>`

**错误码**: `10000`

---

### POST /api/v1/groups/cancel_application

取消群组申请。取消后向群管理员推送 `GroupApplicationCancelled` 通知（对应 `NotifyType='group_application_cancelled'`）。


- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `CancelApplicationRequest`
```typescript
interface CancelApplicationRequest {
  group_id: string;         // 群级鉴权
  application_id: string;
  reason?: string;          // 0-200 字符，取消原因，对齐 admin 侧 reject 的 reason
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300` `10400`

---

### POST /api/v1/groups/list_invites

获取我的群组邀请列表。


- **认证**: 必须
- **幂等**: 无
- **排序**: 按 `created_at` 时间倒序（最新邀请在前），`invite_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `MyInvitesRequest`
```typescript
interface MyInvitesRequest {
  status?: ApplicationStatus | 'all';  // 默认 'pending'（待我处理的邀请）；查历史需显式传 'all'
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<GroupInviteItem>>`
```typescript
// 群组邀请列表项（受邀人视角，来自 me/invites/list）
interface GroupInviteItem {
  invite_id: string;
  group: GroupInfo;
  inviter: UserInfo;
  message?: string;
  status: ApplicationStatus;
  created_at: string;
  expires_at?: string;       // UTC ISO 8601，由 AppConfig.limits.group_invite_ttl_hours 控制
  responded_at?: string;     // UTC ISO 8601，进入终态（accepted/rejected/revoked）时间
}
```

**错误码**: `10000`

---

### POST /api/v1/groups/accept_invite

接受群组邀请。


- **认证**: 必须
- **幂等**: 幂等语义（同一邀请不可重复接受，天然幂等）

**Request**: `AcceptInviteRequest`
```typescript
interface AcceptInviteRequest {
  invite_id: string;
}
```

**Response**: `ApiResponse<InviteAcceptedResult>`（客户端可立即跳转群组；完整成员数据通过 WebSocket `GroupMemberJoined` 推送）
```typescript
interface InviteAcceptedResult {
  group_id: string;   // 接受的群组 ID，便于客户端直接跳转/拉取群详情
}
```

**错误码**: `10000` `10300` `10400`

---

### POST /api/v1/groups/reject_invite

拒绝群组邀请。

**推送**：拒绝成功后向邀请发起方推送 `GroupInviteRejected` 事件并落库通知（`NotifyType='group_invite_rejected'`，含 `reason`，见 `api_spec_websocket.md` 与 `api_spec_notify.md`）。


- **认证**: 必须
- **幂等**: 幂等语义（同一邀请不可重复拒绝，天然幂等）

**Request**: `RejectInviteRequest`
```typescript
interface RejectInviteRequest {
  invite_id: string;
  reason?: string;          // 0-200 字符，拒绝原因（对齐其他负向转移端点）
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10300` `10400`

---

### POST /api/v1/groups/leave

退出群组。

**限制**：群主不能直接退出群组，必须先通过 `groups/transfer_ownership` 转让群主身份，否则返回 `10400`（message: "群主需先转让群主身份"）。

**级联处理（契约级）**：退群用户作为**申请人**的该群 pending 申请自动置 `revoked`；作为 inviter 发出的 pending 邀请**保留**（邀请是群行为，由其他管理员管理），但如果退群的是该群最后一个管理员/owner，则已由 `transfer_ownership` 限制提前拦截（不会发生"无人管理 pending 邀请"的状态）。

- **认证**: 必须
- **幂等**: 幂等语义（重复退出已不在的群视为 no-op 成功）

**Request**: `LeaveGroupRequest`
```typescript
interface LeaveGroupRequest {
  group_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10300` `10400`
