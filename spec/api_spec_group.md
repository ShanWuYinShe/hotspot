# HotSpot API - 群组管理

> 本文档定义 HotSpot 群组管理 (`groups`) 相关的 RPC 接口，包括公开/成员路径与群管理守卫子树（`/api/v1/groups/admin/*`）。
> 群聊消息 (`group_messages`) 接口定义在 `api_spec_chat.md`。

## 模块信息

- **命名空间**: `groups`
- **基础路径**: `/api/v1/groups/`（公开/成员）与 `/api/v1/groups/admin/`（守卫子树）
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 守卫与权限：`api_spec_authz.md`
  - 错误码：`api_spec_errors.md`
  - WebSocket：`api_spec_websocket.md`
  - 匿名镜像版（/public/v1 同名端点）：`api_spec_public.md`

## 端点总览

### 公开/成员路径（`/api/v1/groups/`）

| 端点 | 说明 | 节点 |
|-----|------|------|
| `POST /api/v1/groups/search` | 搜索群组（支持关键字/标签/位置组合；双版本，匿名版在 /public/v1） | /api/v1 |
| `POST /api/v1/groups/open_the_door` | 群组详情（双版本） | /api/v1 |
| `POST /public/v1/groups/announcements/fetch` | 群公告列表（仅 public 版，见 `api_spec_public.md`） | /public/v1 |
| `POST /api/v1/groups/create` | 创建群组 | /api/v1 |
| `POST /api/v1/groups/enter` | 加入群组（根据 join_method 自动处理） | /api/v1 |
| `POST /api/v1/groups/exit` | 退出群组 | /api/v1 |
| `POST /api/v1/groups/fetch_joined` | 我加入的群组（成员视角） | /api/v1 |
| `POST /api/v1/groups/update_preferences` | 更新我的群组偏好 | /api/v1 |
| `POST /api/v1/groups/members/fetch` | 成员列表（双版本） | /api/v1 |
| `POST /api/v1/groups/members/search` | 搜索成员（双版本） | /api/v1 |
| `POST /api/v1/groups/applications/fetch_submitted` | 我提交的申请列表 | /api/v1 |
| `POST /api/v1/groups/applications/cancel` | 取消申请 | /api/v1 |
| `POST /api/v1/groups/invites/fetch_received` | 我收到的邀请列表 | /api/v1 |
| `POST /api/v1/groups/invites/accept` | 接受邀请 | /api/v1 |
| `POST /api/v1/groups/invites/reject` | 拒绝邀请 | /api/v1 |

### 群管理守卫子树（`/api/v1/groups/admin/`，调用者必须是目标群 owner/admin）

| 端点 | 说明 | 守卫 |
|-----|------|------|
| `POST /api/v1/groups/admin/update` | 更新群组 | 守卫(owner/admin) |
| `POST /api/v1/groups/admin/settings` | 群组设置 | 守卫(owner/admin) |
| `POST /api/v1/groups/admin/dismiss` | 解散群组 | owner 专属 |
| `POST /api/v1/groups/admin/invites/create` | 邀请成员 | 守卫(owner/admin) |
| `POST /api/v1/groups/admin/invites/fetch_sent` | 我发出的邀请列表 | 守卫(owner/admin) |
| `POST /api/v1/groups/admin/invites/revoke` | 撤销已发邀请 | 守卫(owner/admin) |
| `POST /api/v1/groups/admin/members/fetch` | 成员列表（管理视角） | 守卫(owner/admin) |
| `POST /api/v1/groups/admin/members/set_role` | 设置角色 | 守卫(owner/admin) |
| `POST /api/v1/groups/admin/members/set_mute` | 设置禁言状态（两态统一） | 守卫(owner/admin) |
| `POST /api/v1/groups/admin/members/set_ban` | 设置封禁状态（两态统一） | 守卫(owner/admin) |
| `POST /api/v1/groups/admin/members/fetch_banned` | 封禁列表 | 守卫(owner/admin) |
| `POST /api/v1/groups/admin/members/remove` | 移除成员 | 守卫(owner/admin) |
| `POST /api/v1/groups/admin/applications/fetch` | 申请列表（群收件箱） | 守卫(owner/admin) |
| `POST /api/v1/groups/admin/applications/approve` | 同意申请 | 守卫(owner/admin) |
| `POST /api/v1/groups/admin/applications/reject` | 拒绝申请 | 守卫(owner/admin) |
| `POST /api/v1/groups/admin/announcements/create` | 发布公告 | 守卫(owner/admin) |
| `POST /api/v1/groups/admin/announcements/update` | 更新公告 | 守卫(owner/admin) |
| `POST /api/v1/groups/admin/announcements/delete` | 删除公告 | 守卫(owner/admin) |
| `POST /api/v1/groups/admin/transfer_ownership` | 转让群主 | owner 专属 |
| `POST /api/v1/groups/admin/messages/recall` | 撤回任意成员消息 | 守卫(owner/admin) |

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `GroupInfo`
- `GroupItem`
- `GroupViewerState`
- `GroupJoinedItem`
- `GroupSettings`
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

// 公开成员列表项（members/fetch 与 members/search 共用，形状恒定）
// role/joined_at 对所有可见（用于展示成员规模与角色构成）
// presence 字段为隐私受控恒定 nullable（null = 成员 presence_visible=false 或未登录）
interface GroupMemberItem {
  user: UserInfo;
  role: GroupRole;
  joined_at: string;
  presence_status: UserPresenceStatus | null;
  last_online_at: string | null;
}

// 管理视角成员列表项（仅 groups/admin/members/fetch 返回；muted_until 为真值，null=未禁言）
interface AdminGroupMemberItem {
  user: UserInfo;
  role: GroupRole;
  joined_at: string;
  muted_until: string | null;
}

// 邀请响应（groups/admin/invites/create）
// - 正常创建：返回 invite_id / expires_at
// - 被邀请者已是成员（no-op 成功，不创建邀请）：invite_id / expires_at 为 null
interface Invitation {
  invite_id: string | null;
  expires_at: string | null;
}

// 我发出的邀请列表项
interface GroupSentInviteItem {
  invite_id: string;
  invitee: UserInfo;
  inviter: UserInfo;           // 邀请发起者（多管理员场景可见是谁发的）
  status: ApplicationStatus;
  message: string | null;
  created_at: string;
  expires_at: string | null;
  responded_at: string | null; // 进入终态时间
}

// 群组申请列表项（群管理员视角与申请人视角共用此类型）
interface GroupApplicationItem {
  application_id: string;
  group: GroupInfo;              // 目标群组（申请人视角的 fetch_submitted 跨群，需群上下文）
  applicant: UserInfo;
  message: string | null;
  status: ApplicationStatus;
  created_at: string;
  expires_at: string | null;     // 申请有效期截止
  handled_at: string | null;     // 进入终态时间
}

// 群组邀请列表项（受邀人视角）
interface GroupInviteItem {
  invite_id: string;
  group: GroupInfo;
  inviter: UserInfo;
  message: string | null;
  status: ApplicationStatus;
  created_at: string;
  expires_at: string | null;
  responded_at: string | null;
}

// 封禁成员列表项（仅守卫子树 fetch_banned 返回）
interface BannedMemberItem {
  user: UserInfo;
  banned_at: string;
  reason: string | null;
  banned_by: UserInfo | null;
}

// 申请响应
interface Application {
  application_id: string;
}

// 加入群组响应
interface EnterGroupResult {
  status: 'joined' | 'pending_approval';
  application: Application | null;  // 仅 pending_approval 时返回
}

// 接受邀请响应
interface InviteAcceptedResult {
  group_id: string;   // 接受的群组 ID，便于客户端直接跳转/拉取群详情
}

// 群公告列表（有界数据，不分页）
interface GroupAnnouncementList {
  items: GroupAnnouncementItem[];
}
```

## 公开/成员路径端点详情

### POST /api/v1/groups/search

统一搜索群组。支持关键字、标签、位置范围三种条件的自由组合。

**radius 语义（契约级）**：`radius` 仅**显式提供时**按半径过滤结果；缺省时不做半径过滤。`sort_by='distance'` 的距离计算依赖 `reference_point`（按 `api_spec_types.md`「reference_point 解析规则」回退链解析）。

- **节点**: /api/v1（双版本端点：匿名版 `/public/v1/groups/search` 见 `api_spec_public.md`）
- **排序**: 
  - 默认 `sort_by='relevance'`：按相关性优先排序
  - `sort_by='distance'`：按距离由近到远排序
  - `sort_by='newest'`：按创建时间倒序
  - `sort_by='popular'`：按成员数/活跃度排序
  - **稳定 tie-breaker**：任意 `sort_by` 模式下，当主序字段相同，以 `group_id ASC` 作为稳定排序，保证游标分页一致性
  - 一致性级别为弱一致

**Request**: `SearchGroupsRequest`
```typescript
interface SearchGroupsRequest {
  keyword: string | null;    // 关键字搜索（匹配名称、描述等）
  tag: string | null;        // 标签筛选
  radius: number | null;     // 搜索半径（米）；显式提供时按半径过滤，缺省不过滤
  sort_by: 'relevance' | 'distance' | 'newest' | 'popular' | null;  // 默认: relevance
  reference_point: GeoPoint | null;
  pagination: PaginationParams | null;
}
// 至少提供 keyword、tag、radius 中的一个；不满足时请求失败（10000）
```

**Response**: `ApiResponse<PaginatedData<GroupItem>>`

---

### POST /api/v1/groups/open_the_door

获取群组详情（登录视角 `GroupItem`，含查看者状态 `viewer_state`/`friends_in_group`；匿名版返回 `PublicGroupItem`）。

- **节点**: /api/v1（双版本端点：匿名版 `/public/v1/groups/open_the_door` 见 `api_spec_public.md`）

**Request**: `OpenGroupDoorRequest`
```typescript
interface OpenGroupDoorRequest {
  group_id: string;
}
```

**Response**: `ApiResponse<GroupItem>`

**错误码**: `10300`

---

### POST /api/v1/groups/create

创建群组。

- **节点**: /api/v1
- **幂等**: 是（客户端必须携带 `Idempotency-Key`）

**Request**: `CreateGroupRequest`
```typescript
interface CreateGroupRequest {
  name: string;
  description: string | null;
  avatar_file_id: string | null;   // 来自 media/complete_upload 返回的 file_id
  tags: string[];
  password: string | null;
  join_method: GroupJoinMethod | null;  // 默认: 'direct'
  requires_approval: boolean | null;    // 默认: false
  is_searchable: boolean | null;         // 默认: true
  members_searchable: boolean | null;    // 默认: true
  max_members: number | null;            // 默认: 200
  location: Location;
}
```

**Response**: `ApiResponse<GroupItem>`（创建者即群主，viewer_state 返回 owner 状态）

**错误码**: `10300`（avatar_file_id 无效） `10400`（tags 含被屏蔽标签）

---

### POST /api/v1/groups/enter

加入群组（智能路由）。根据群组 `join_method`（准入方式）与 `requires_approval`（是否需审核）自动处理：
- `direct` + 无需审核：直接加入，返回 `status: 'joined'`
- `direct` + 需审核：创建加入申请，返回 `status: 'pending_approval'`
- `password`：需提供 `password`，验证通过后按 `requires_approval` 直接加入或创建申请
- `invite_only`：**本端点不支持**——仅受邀用户可加入，加入唯一入口为 `groups/invites/accept`（对 invite_only 群调用 join 返回 `10400`（该群组仅限邀请加入）；受邀后按 `requires_approval` 直接加入或创建申请）

- **节点**: /api/v1

**自然去重**（无需 `Idempotency-Key`）：
- approval 路径：若同一用户对该群已有 `pending` 申请，返回已有 `application_id`（不重复创建、不重复通知管理员）
- direct 路径：成员关系 `(group_id, user_id)` 唯一，已加入视为 no-op 成功（返回 `status: 'joined'`）

**Request**: `EnterGroupRequest`
```typescript
interface EnterGroupRequest {
  group_id: string;
  password: string | null;   // password 群组需要
  message: string | null;    // approval 群组可选附言
}
```

**推送**：
- 直接加入（`status='joined'`）时向群内推送 `GroupMemberJoined` 事件（见 `api_spec_websocket.md`）
- 创建申请（`status='pending_approval'`）时向群管理员推送 `GroupApplicationReceived` 事件并落库通知（见 `api_spec_notify.md`）

**Response**: `ApiResponse<EnterGroupResult>`

**错误码**: `10200`（banned_in_group 被封禁无法加入） `10300` `10400`

---

### POST /api/v1/groups/exit

退出群组。

**限制**：群主不能直接退出群组，必须先通过 `groups/admin/transfer_ownership` 转让群主身份，否则返回 `10400`（message: "群主需先转让群主身份"）。

**级联处理（契约级）**：退群用户作为**申请人**的该群 pending 申请自动置 `revoked`；作为 inviter 发出的 pending 邀请**保留**（邀请是群行为，由其他管理员管理）。

**推送**：退群成功后向群内推送 `GroupMemberLeft` 事件。

- **节点**: /api/v1

**Request**: `ExitGroupRequest`
```typescript
interface ExitGroupRequest {
  group_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10200`（not_group_member） `10300` `10400`（群主）

---

### POST /api/v1/groups/fetch_joined

获取我加入的群组（成员视角：含私密群、viewer_state、last_message）。

**说明**：返回 `GroupJoinedItem`，`viewer_state` 必非 null。本端点无 `reference_point` 参数，返回的 `GroupItem.location` 中 distance/bearing 为 null。

- **节点**: /api/v1
- **排序**: 默认按 `is_pinned DESC, last_activity_at DESC` 排序（置顶优先，最近活跃在前），`group_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FetchJoinedGroupsRequest`
```typescript
interface FetchJoinedGroupsRequest {
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<GroupJoinedItem>>`

---

### POST /api/v1/groups/update_preferences

更新我的群组偏好（静音/置顶，区别于守卫子树的 `groups/admin/update` 群信息更新）。

- **节点**: /api/v1

**Request**: `UpdateGroupPreferencesRequest`
```typescript
interface UpdateGroupPreferencesRequest {
  group_id: string;
  is_muted: boolean | null;
  is_pinned: boolean | null;
}
```

**Response**: `ApiResponse`

**错误码**: `10200`（not_group_member） `10300` `10400`

---

### POST /api/v1/groups/members/fetch

获取成员列表（公开成员信息；管理字段见守卫子树 `groups/admin/members/fetch`）。

- **节点**: /api/v1（双版本端点：匿名版 `/public/v1/groups/members/fetch` 见 `api_spec_public.md`）
- **可见性**: `members_searchable=false` 时非成员调用返回 `10200` + `not_group_member`
- **排序**: 按 `joined_at` 时间正序（最早加入在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FetchGroupMembersRequest`
```typescript
interface FetchGroupMembersRequest {
  group_id: string;
  role: GroupRole | null;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<GroupMemberItem>>`

**错误码**: `10200`（not_group_member） `10300`

---

### POST /api/v1/groups/members/search

搜索群组成员。用于大群场景中快速定位特定成员。

- **节点**: /api/v1（双版本端点：匿名版 `/public/v1/groups/members/search` 见 `api_spec_public.md`）
- **可见性**: `members_searchable=false` 时非成员调用返回 `10200` + `not_group_member`
- **排序**: 按匹配度优先，同匹配度按 `joined_at` 正序，`user_id ASC` 作为稳定 tie-breaker；一致性级别为弱一致

**Request**: `SearchGroupMembersRequest`
```typescript
interface SearchGroupMembersRequest {
  group_id: string;
  keyword: string;    // 按昵称搜索
  role: GroupRole | null;   // 可选：按角色过滤
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<GroupMemberItem>>`

**错误码**: `10200`（not_group_member：非成员且 members_searchable=false） `10300`

---

### POST /api/v1/groups/applications/fetch_submitted

获取我提交的群组申请列表（申请人视角，跨群）。

- **节点**: /api/v1
- **排序**: 按 `created_at` 时间倒序（最新申请在前），`application_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FetchSubmittedApplicationsRequest`
```typescript
interface FetchSubmittedApplicationsRequest {
  status: ApplicationStatus | 'all' | null;  // 默认 'pending'；查历史需显式传 'all'
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<GroupApplicationItem>>`

---

### POST /api/v1/groups/applications/cancel

取消群组申请。取消后向群管理员推送 `GroupApplicationCancelled` 事件（纯实时，不落库通知）。

- **节点**: /api/v1

**Request**: `CancelApplicationRequest`
```typescript
interface CancelApplicationRequest {
  group_id: string;         // 群级鉴权
  application_id: string;
  reason: string | null;    // 0-200 字符，取消原因
}
```

**Response**: `ApiResponse`

**错误码**: `10200`（not_resource_owner 非本人申请） `10300` `10400`

---

### POST /api/v1/groups/invites/fetch_received

获取我收到的群组邀请列表。

- **节点**: /api/v1
- **排序**: 按 `created_at` 时间倒序（最新邀请在前），`invite_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FetchReceivedInvitesRequest`
```typescript
interface FetchReceivedInvitesRequest {
  status: ApplicationStatus | 'all' | null;  // 默认 'pending'；查历史需显式传 'all'
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<GroupInviteItem>>`

---

### POST /api/v1/groups/invites/accept

接受群组邀请。

- **节点**: /api/v1

**Request**: `AcceptInviteRequest`
```typescript
interface AcceptInviteRequest {
  invite_id: string;
}
```

**Response**: `ApiResponse<InviteAcceptedResult>`

**推送**：
- 向群内推送 `GroupMemberJoined` 事件
- 向邀请发起方推送 `GroupInviteResponse` 事件并落库通知（见 `api_spec_notify.md`）

**错误码**: `10300` `10400`

---

### POST /api/v1/groups/invites/reject

拒绝群组邀请。

**推送**：拒绝成功后向邀请发起方推送 `GroupInviteRejected` 事件（纯实时，不落库通知，含 `reason`）。

- **节点**: /api/v1

**Request**: `RejectInviteRequest`
```typescript
interface RejectInviteRequest {
  invite_id: string;
  reason: string | null;    // 0-200 字符，拒绝原因
}
```

**Response**: `ApiResponse`

**错误码**: `10300` `10400`

## 群管理守卫子树端点详情（/api/v1/groups/admin/）

> **守卫（子树级统一）**：调用者必须是**请求目标群**的 owner/admin，否则 `10200` + `not_group_admin`（非成员为 `not_group_member`）。以下端点不再逐端点声明权限，仅标注 owner 专属与目标保护规则。守卫细节见 `api_spec_authz.md`。

### POST /api/v1/groups/admin/update

更新群组信息（owner/admin）。

- **守卫**: 守卫(owner/admin)

**Request**: `UpdateGroupRequest`
```typescript
interface UpdateGroupRequest {
  group_id: string;
  name: string | null;
  description: string | null;
  avatar_file_id: string | null;   // 来自 media/complete_upload 返回的 file_id
  tags: string[] | null;
  password: string | null;
  clear_password: boolean | null;  // true 时移除群密码（与 password 互斥，同时传入视为参数冲突）
  join_method: GroupJoinMethod | null;
  requires_approval: boolean | null;
  next_activity: string | null;    // 设置/更新下次活动预告；传空字符串清除
  is_searchable: boolean | null;
  members_searchable: boolean | null;
  max_members: number | null;
  location: Location | null;
}
```

**Response**: `ApiResponse<GroupItem>`（内容类更新返回更新后完整对象）

**推送**：更新成功后向所有在线群成员推送 `GroupUpdated` 事件（管理字段 is_searchable/max_members/members_searchable 的变更不随事件下发，成员下次拉取详情时感知；见 `api_spec_websocket.md`）。

**错误码**: `10200`（not_group_admin） `10300`

---

### POST /api/v1/groups/admin/settings

获取群组设置（管理视角）。

- **守卫**: 守卫(owner/admin)

**Request**: `GroupSettingsRequest`
```typescript
interface GroupSettingsRequest {
  group_id: string;
}
```

**Response**: `ApiResponse<GroupSettings>`（含 has_password/is_searchable/members_searchable/max_members 等管理字段）

**错误码**: `10200`（not_group_admin） `10300`

---

### POST /api/v1/groups/admin/dismiss

解散群组（仅群主）。

**解散行为**：解散后向所有在线群成员推送 WebSocket `GroupDismissed` 事件（`reason='owner_dismissed'`）。不暴露解散者身份（隐私保护，操作记录仅保留在审计日志）。

**级联处理（契约级）**：解散时，该群所有 `pending` 状态的申请与邀请由服务端自动置为 `revoked`（群已不存在，无需推送单独事件）。

- **守卫**: owner 专属

**Request**: `DismissGroupRequest`
```typescript
interface DismissGroupRequest {
  group_id: string;
  reason: string | null;  // 解散说明（0-500 字符，仅进审计日志；成员收到的是 owner_dismissed 分类，不接收自由文本）
}
```

**Response**: `ApiResponse`

**错误码**: `10200`（not_group_owner） `10300`

---

### POST /api/v1/groups/admin/invites/create

邀请成员。

**重复/已达目标状态处理**（与业界共识一致）：
- 若对同一用户存在 pending 状态的邀请，返回已有邀请信息（不重复创建），避免重复通知骚扰。
- 被邀请用户**已是群成员**时，视为目标已达，返回成功（no-op，不创建邀请、不报错）。

- **守卫**: 守卫(owner/admin)

**Request**: `CreateGroupInviteRequest`
```typescript
interface CreateGroupInviteRequest {
  group_id: string;
  user_id: string;
  message: string | null;
}
```

**Response**: `ApiResponse<Invitation>`

**推送**：创建邀请后向受邀人推送 `GroupInviteReceived` 事件并落库通知；no-op 路径（已是成员）不推送。

**错误码**: `10200`（not_group_admin） `10300` `10400`（如邀请已被封禁的用户）

---

### POST /api/v1/groups/admin/invites/fetch_sent

我发出的邀请列表。

- **守卫**: 守卫(owner/admin)
- **排序**: 默认按照 `created_at DESC` 排序，时间相同以 `invite_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FetchSentInvitesRequest`
```typescript
interface FetchSentInvitesRequest {
  group_id: string;
  status: ApplicationStatus | 'all' | null;  // 默认 'pending'；查历史需显式传 'all'
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<GroupSentInviteItem>>`

`ApplicationStatus='revoked'` 表示邀请由发起者主动撤销（含过期自动撤销），客户端需与 `rejected`（受邀者拒绝）区分展示。

**过期语义（契约级）**：超过 `expires_at` 的 pending 邀请由服务端自动置为 `revoked`，并向受邀人推送 `GroupInviteExpired` 事件（见 `api_spec_websocket.md`）。

**错误码**: `10200`（not_group_admin） `10300` `10400`

---

### POST /api/v1/groups/admin/invites/revoke

撤销已发出的待处理邀请。

**推送**：撤销成功后向受邀人推送 `GroupInviteRevoked` 事件。

- **守卫**: 守卫(owner/admin)

**Request**: `RevokeSentInviteRequest`
```typescript
interface RevokeSentInviteRequest {
  group_id: string;
  invite_id: string;
  reason: string | null;
}
```

**Response**: `ApiResponse`

**错误码**: `10200`（not_group_admin） `10300` `10400`

---

### POST /api/v1/groups/admin/members/fetch

获取成员列表（管理视角：含 `muted_until` 真值）。

- **守卫**: 守卫(owner/admin)
- **排序**: 按 `joined_at` 时间正序，`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FetchAdminGroupMembersRequest`
```typescript
interface FetchAdminGroupMembersRequest {
  group_id: string;
  role: GroupRole | null;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<AdminGroupMemberItem>>`

**错误码**: `10200`（not_group_admin） `10300`

---

### POST /api/v1/groups/admin/members/set_role

设置成员角色。设置为目标角色与当前一致时视为 no-op，返回成功。

**降级与 pending 邀请**：邀请是群行为（非个人），inviter 降级为普通成员后，其历史发出的 pending 邀请由其他管理员/owner 继续可见与管理（不自动失效）。**owner 降级**（转交所有权）前若有 pending 邀请，需先 `invites/revoke` 或在 `transfer_ownership` 中一并转交管理权，否则返回 `10400`。

**推送**：成功后向群内推送 `GroupMemberRoleChanged` 事件（含 `transfer_ownership` 触发的 owner 变更）。

- **守卫**: 守卫(owner/admin)

**Request**: `SetMemberRoleRequest`
```typescript
interface SetMemberRoleRequest {
  group_id: string;
  user_id: string;
  role: SettableGroupRole;
}
```

**Response**: `ApiResponse`

**错误码**: `10200`（not_group_admin） `10300` `10400`（如降级最后一个管理员）

---

### POST /api/v1/groups/admin/members/set_mute

设置成员禁言状态（两态统一端点）。

- **守卫**: 守卫(owner/admin)

**Request**: `SetMuteRequest`
```typescript
interface SetMuteRequest {
  group_id: string;
  user_id: string;
  active: boolean;
  duration_minutes: number | null;  // active=true 时必填（1-43200，最多 30 天）；active=false 时忽略
}
```

**目标保护**：对**群主**执行禁言返回 `10400`（群主不受禁言）；管理员可禁言其他管理员（含互相禁言）。

**推送**：成功后向群内推送 `GroupMemberMuted` / `GroupMemberUnmuted` 事件（见 `api_spec_websocket.md`），被禁言者自身通过 `GroupViewerState.muted_until` 感知。

**Response**: `ApiResponse`

**错误码**: `10200`（not_group_admin） `10300` `10400`（群主保护）

---

### POST /api/v1/groups/admin/members/set_ban

设置成员封禁状态（两态统一端点）。

`active=true`：封禁成员，封禁后成员将被移出群组且无法再次加入。

**级联处理（契约级）**：封禁时，被 ban 用户**发出的**该群 pending 申请置 `revoked`；**发给**被 ban 用户的该群 pending 邀请置 `revoked`。

`active=false`：解除封禁。对未处于封禁状态的成员调用视为 no-op，返回成功。

- **守卫**: 守卫(owner/admin)

**Request**: `SetMemberBanRequest`
```typescript
interface SetMemberBanRequest {
  group_id: string;
  user_id: string;
  active: boolean;
  reason: string | null;
}
```

**目标保护**：对**群主**执行封禁返回 `10400`（群主不可被封禁）。

**推送**：`active=true` 时向群内推送 `GroupMemberRemoved` 事件（封禁即移除）；`active=false`（解封）不推送。

**Response**: `ApiResponse`

**错误码**: `10200`（not_group_admin） `10300` `10400`（群主保护）

---

### POST /api/v1/groups/admin/members/fetch_banned

获取封禁列表。封禁名单属群管理信息，仅守卫子树可见（普通成员返回 `10200`）。

- **守卫**: 守卫(owner/admin)
- **排序**: 按 `banned_at` 时间倒序（最近封禁在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FetchBannedMembersRequest`
```typescript
interface FetchBannedMembersRequest {
  group_id: string;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<BannedMemberItem>>`

**错误码**: `10200`（not_group_admin） `10300`

---

### POST /api/v1/groups/admin/members/remove

移除成员。

**目标保护**：对**群主**执行移除返回 `10400`（群主不可被移除，需先 `transfer_ownership`）；管理员可移除其他管理员（含互相移除）。

**推送**：成功后向群内推送 `GroupMemberRemoved` 事件。

- **守卫**: 守卫(owner/admin)

**Request**: `RemoveMemberRequest`
```typescript
interface RemoveMemberRequest {
  group_id: string;
  user_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10200`（not_group_admin） `10300` `10400`（群主保护）

---

### POST /api/v1/groups/admin/applications/fetch

获取群组申请列表（群收件箱，管理员工作台）。

- **守卫**: 守卫(owner/admin)
- **排序**: 按 `created_at` 时间倒序（最新申请在前），`application_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FetchGroupApplicationsRequest`
```typescript
interface FetchGroupApplicationsRequest {
  group_id: string;
  status: ApplicationStatus | 'all' | null;  // 默认 'pending'；查历史需显式传 'all'
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<GroupApplicationItem>>`

**错误码**: `10200`（not_group_admin） `10300`

---

### POST /api/v1/groups/admin/applications/approve

审批群组申请。

**边界情况**：对已处于 `accepted` 状态的申请重复调用，视为 no-op 返回成功。

**推送**：审批通过后向申请人推送 `GroupApplicationResponse` 事件并落库通知（见 `api_spec_notify.md`）。

- **守卫**: 守卫(owner/admin)

**Request**: `ApproveGroupApplicationRequest`
```typescript
interface ApproveGroupApplicationRequest {
  group_id: string;        // 群级鉴权；须与 application 所属群一致，否则请求失败
  application_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10200`（not_group_admin） `10300` `10400`

---

### POST /api/v1/groups/admin/applications/reject

拒绝群组申请。拒绝后向申请人推送 `GroupApplicationRejected` 事件并落库通知（含 `reason`；**不暴露审批者身份**，与 accepted 对称，见 `api_spec_notify.md`）。

- **守卫**: 守卫(owner/admin)

**Request**: `RejectGroupApplicationRequest`
```typescript
interface RejectGroupApplicationRequest {
  group_id: string;        // 群级鉴权
  application_id: string;
  reason: string | null;   // 0-200 字符，拒绝原因，会随通知送达申请人
}
```

**Response**: `ApiResponse`

**错误码**: `10200`（not_group_admin） `10300` `10400`

---

### POST /api/v1/groups/admin/announcements/create

发布群公告。

- **守卫**: 守卫(owner/admin)
- **幂等**: 是（客户端必须携带 `Idempotency-Key`）

**Request**: `CreateGroupAnnouncementRequest`
```typescript
interface CreateGroupAnnouncementRequest {
  group_id: string;
  title: string;
  content: string;
  is_pinned: boolean | null;
}
```

**Response**: `ApiResponse<GroupAnnouncementItem>`

**错误码**: `10200`（not_group_admin） `10300` `10400`

---

### POST /api/v1/groups/admin/announcements/update

更新群公告。

- **守卫**: 守卫(owner/admin)

**Request**: `UpdateGroupAnnouncementRequest`
```typescript
interface UpdateGroupAnnouncementRequest {
  group_id: string;
  announcement_id: string;
  title: string | null;
  content: string | null;
  is_pinned: boolean | null;
}
```

**Response**: `ApiResponse<GroupAnnouncementItem>`

**错误码**: `10200`（not_group_admin） `10300`

---

### POST /api/v1/groups/admin/announcements/delete

删除群公告。

- **守卫**: 守卫(owner/admin)

**Request**: `DeleteGroupAnnouncementRequest`
```typescript
interface DeleteGroupAnnouncementRequest {
  group_id: string;
  announcement_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10200`（not_group_admin） `10300`

---

### POST /api/v1/groups/admin/transfer_ownership

转让群主。

**边界情况**：`new_owner_id` 不是该群成员时，返回 `10400`（"目标用户不是群成员"）。

- **守卫**: owner 专属

**自然去重**：首次成功后原群主失去群主权限，重发同一请求返回 `10200`（not_group_owner），不会重复转让。无需 `Idempotency-Key`。

**Request**: `TransferOwnershipRequest`
```typescript
interface TransferOwnershipRequest {
  group_id: string;
  new_owner_id: string;
}
```

**推送**：转让成功后向群内推送 `GroupMemberRoleChanged` 事件（owner 变更）。

**Response**: `ApiResponse`

**错误码**: `10200`（not_group_owner） `10300` `10400`（目标非成员）

---

### POST /api/v1/groups/admin/messages/recall

撤回任意成员发送的群聊消息（群管理员/群主）。与客户端 `group_messages/recall`（仅作者撤回自己的消息）角色分离，各司其职。

**无时间窗限制**：管理员撤回不受 `AppConfig.limits.message_recall_window_seconds` 约束。

- **守卫**: 守卫(owner/admin)

**Request**: `RecallGroupMessageAdminRequest`
```typescript
interface RecallGroupMessageAdminRequest {
  group_id: string;    // 群级鉴权
  message_id: string;
}
```

**Response**: `ApiResponse`

**推送**：撤回成功后向群订阅者推送 `GroupMessageRecalled` 事件（含 `recalled_by`）。

**错误码**: `10200`（not_group_admin） `10300` `10400`