# HotSpot API - 平台管理节点

> 本文档定义 HotSpot 平台管理节点（**/admin/api/v1**）的 RPC 接口，包括权限管理（RBAC）、审计日志、全局配置与内容审核（用户/群组/举报/标签/公告）。
> **节点性质（契约级）**：/admin/api/v1 是独立于客户端 /api/v1 的全局节点，仅供平台管理员使用；平台管理员在 /api/v1 上与普通用户完全同权，平台权限不外溢（见 `api_spec_authz.md`「交叉规则」）。

## 模块信息

- **节点**: `/admin/api/v1`（平台管理员）
- **命名空间**: `rbac` / `audit` / `app_config` / `users` / `groups` / `reports` / `tags` / `announcements` / `stats`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 守卫与权限：`api_spec_authz.md`（平台权限键权威清单）
  - 错误码：`api_spec_errors.md`
  - 客户端配置投影：`api_spec_types.md`（`ClientConfig` 为 `AppConfig` 的客户端可见投影）

## 统一访问控制

- 未登录/登录失效：返回 `10100`（认证失败）
- 已登录但不是平台管理员：返回 `10200` + `forbidden_reason='not_admin'`（message: "非平台管理员"）
- 是平台管理员但缺少接口要求的权限键：返回 `10200` + `forbidden_reason='insufficient_admin_permission'`（message: "平台管理员权限不足: xxx"）
- 权限键清单、superadmin（`'*'`）规则与 PII 键（`read_ip` ×2）例外见 `api_spec_authz.md`「平台权限键」

## 幂等性（平台管理写操作）

平台管理写操作是否需要实现幂等，以 `api_spec_base.md` 的「需要实现幂等的接口」列表为唯一权威。平台管理侧仅 `announcements/create` 需要实现幂等（发布系统公告会产生用户可见的新内容）；其余平台管理写操作（创建角色、授权、封禁、解散、处理举报、更新/删除等）均**不需要实现幂等**——重复请求自然返回一致结果（重名返回 `10400`、已达目标状态视为 no-op 成功等）。

## 端点总览

| 端点 | 说明 | 权限键 |
|-----|------|------|
| `POST /admin/api/v1/rbac/me` | 我的平台管理员权限 | 是平台管理员即可 |
| `POST /admin/api/v1/rbac/permissions/fetch` | 权限字典 | admin.rbac.manage |
| `POST /admin/api/v1/rbac/roles/fetch` | 角色列表 | admin.rbac.manage |
| `POST /admin/api/v1/rbac/roles/create` | 创建角色 | admin.rbac.manage |
| `POST /admin/api/v1/rbac/roles/update` | 更新角色 | admin.rbac.manage |
| `POST /admin/api/v1/rbac/roles/delete` | 删除角色 | admin.rbac.manage |
| `POST /admin/api/v1/rbac/roles/set_permissions` | 设置角色权限 | admin.rbac.manage |
| `POST /admin/api/v1/rbac/admins/fetch` | 平台管理员列表 | admin.rbac.manage |
| `POST /admin/api/v1/rbac/admins/award` | 授权/更新平台管理员 | admin.rbac.manage |
| `POST /admin/api/v1/rbac/admins/revoke` | 撤销平台管理员 | admin.rbac.manage |
| `POST /admin/api/v1/audit/fetch` | 审计日志列表 | admin.audit.read |
| `POST /admin/api/v1/app_config` | 获取完整应用配置 | admin.app_config.read |
| `POST /admin/api/v1/app_config/update` | 更新应用配置（按 section 部分更新） | admin.app_config.write |
| `POST /admin/api/v1/users/search` | 搜索用户 | admin.users.read |
| `POST /admin/api/v1/users/open_the_door` | 用户详情 | admin.users.read |
| `POST /admin/api/v1/users/set_ban` | 设置用户封禁状态（两态统一） | admin.users.write |
| `POST /admin/api/v1/groups/search` | 搜索群组 | admin.groups.read |
| `POST /admin/api/v1/groups/open_the_door` | 群组详情 | admin.groups.read |
| `POST /admin/api/v1/groups/dismiss` | 解散群组 | admin.groups.write |
| `POST /admin/api/v1/reports/fetch` | 举报列表（按状态/类型过滤） | admin.reports.read |
| `POST /admin/api/v1/reports/judge` | 裁决举报（resolve/dismiss） | admin.reports.write |
| `POST /admin/api/v1/tags/search` | 搜索标签 | admin.tags.read |
| `POST /admin/api/v1/tags/set_ban` | 设置标签屏蔽状态（两态统一：屏蔽/解屏蔽，可选物理删除） | admin.tags.write |
| `POST /admin/api/v1/announcements/fetch` | 系统公告列表（按 severity 过滤） | admin.announcements.read |
| `POST /admin/api/v1/announcements/create` | 发布系统公告 | admin.announcements.write |
| `POST /admin/api/v1/announcements/update` | 更新系统公告 | admin.announcements.write |
| `POST /admin/api/v1/announcements/delete` | 删除系统公告 | admin.announcements.write |
| `POST /admin/api/v1/stats/geo` | 地理统计（全站存量 + 区域/热门地点排名 + 每日新增趋势） | admin.stats.read |
| `POST /admin/api/v1/stats/geo/drilldown` | 区域下钻（网格桶内实体明细，keyset 分页） | admin.stats.read |

## 类型定义

以下类型定义于 `api_spec_types.md`：`UserInfo`、`Location`、`GroupSettings`、`FeatureSwitches`、`AppLimits`、`ContactInfo`、`PaginationParams`、`PaginatedData`、`UserItem`、`GroupItem`、`SystemAnnouncementItem`、`UserType`、`PrivacyLevel`、`ReportStatus`、`ReportTargetType`、`ReportAction`、`TagItem`、`AnnounceSeverity`、`ReportItem`。

以下 Admin 扩展类型在通用类型基础上扩展平台管理视角字段（`AdminXxxItem` 仅存在于本节点）。

### 本节点类型

```typescript
// 账号状态（平台管理视角）
type UserAccountStatus = 'normal' | 'banned';
```

## RBAC 管理 API

**设计目标**：
- **superadmin**：`permissions` 包含 `'*'`，表示拥有全部平台权限（除 PII 键 read_ip ×2 需显式持有）
- **普通平台管理员**：通过角色/权限表获得细粒度权限，字段可见性由具体权限决定

### POST /admin/api/v1/rbac/me

获取我的平台管理员权限信息。

- **权限**: 是平台管理员即可（不要求 `admin.rbac.manage`）

**Request**: `RbacMeRequest`
```typescript
interface RbacMeRequest {}
```

**Response**: `ApiResponse<AdminPermissions>`

```typescript
interface AdminRoleBasic {
  role_id: string;
  name: string;
}

interface AdminPermissions {
  permissions: string[];  // '*' 表示拥有全部权限
  roles: AdminRoleBasic[];
}
```

**错误码**: `10200`（not_admin）

---

### POST /admin/api/v1/rbac/permissions/fetch

获取权限字典。

- **权限**: `admin.rbac.manage`
- **排序**: 按 `key ASC` 作为确定性顺序（权限字典为有界数据，无分页）

**Request**: `FetchPermissionsRequest`
```typescript
interface FetchPermissionsRequest {}
```

**Response**: `ApiResponse<AdminPermissionDictionary>`

```typescript
interface AdminPermissionDictionary {
  items: AdminPermissionItem[];
}

interface AdminPermissionItem {
  key: string;
  description: string;
}
```

**错误码**: `10200`（not_admin / insufficient_admin_permission）

---

### POST /admin/api/v1/rbac/roles/fetch

获取角色列表。

- **权限**: `admin.rbac.manage`
- **排序**: 按 `name ASC` 作为确定性顺序（角色为有界数据，无分页）

**Request**: `FetchAdminRolesRequest`
```typescript
interface FetchAdminRolesRequest {} // 直接返回全部角色，无需分页
```

**Response**: `ApiResponse<AdminRoleList>`

```typescript
interface AdminRoleList {
  items: AdminRoleItem[];
}

interface AdminRoleItem {
  role_id: string;
  name: string;
  description: string | null;
  permissions: string[];
}
```

**说明**：RBAC 角色为系统配置数据，数量有限，无需分页；与 `permissions/fetch` 返回格式保持一致。

**错误码**: `10200`（not_admin / insufficient_admin_permission）

---

### POST /admin/api/v1/rbac/roles/create

创建角色。

- **权限**: `admin.rbac.manage`

**Request**: `CreateAdminRoleRequest`
```typescript
interface CreateAdminRoleRequest {
  name: string;
  description: string | null;
}
```

**Response**: `ApiResponse<AdminRoleItem>`

**错误码**: `10200`（not_admin / insufficient_admin_permission） `10400`（重名）

---

### POST /admin/api/v1/rbac/roles/update

更新角色。

- **权限**: `admin.rbac.manage`

**Request**: `UpdateAdminRoleRequest`
```typescript
interface UpdateAdminRoleRequest {
  role_id: string;
  name: string | null;
  description: string | null;
  clear_description: boolean | null;
}
```

**Response**: `ApiResponse`

**错误码**: `10200` `10300` `10400`

---

### POST /admin/api/v1/rbac/roles/delete

删除角色。

- **权限**: `admin.rbac.manage`

**Request**: `DeleteAdminRoleRequest`
```typescript
interface DeleteAdminRoleRequest {
  role_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10200` `10300` `10400`

---

### POST /admin/api/v1/rbac/roles/set_permissions

设置角色权限。

- **权限**: `admin.rbac.manage`

**Request**: `SetAdminRolePermissionsRequest`
```typescript
interface SetAdminRolePermissionsRequest {
  role_id: string;
  permission_keys: string[];
}
```

**说明**：
- `permission_keys` 必须来自权限字典（`POST /admin/api/v1/rbac/permissions/fetch` 返回值）
- 仅 superadmin 可以设置 `'*'`（全量权限）；普通平台管理员设置 `'*'` 时返回 `10200`

**Response**: `ApiResponse`

**错误码**: `10200` `10300`

---

### POST /admin/api/v1/rbac/admins/fetch

获取平台管理员列表。

- **权限**: `admin.rbac.manage`
- **排序**: 按 `granted_at DESC` 排序（最近授权在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FetchAdminsRequest`
```typescript
interface FetchAdminsRequest {
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<AdminOperatorItem>>`

```typescript
interface AdminOperatorItem {
  user: UserInfo;
  granted_at: string;     // 最近一次授权时间（排序主序）
  permissions: string[];  // '*' 表示拥有全部权限
  roles: AdminRoleBasic[];
}
```

**错误码**: `10200`（not_admin / insufficient_admin_permission）

---

### POST /admin/api/v1/rbac/admins/award

授予/更新平台管理员。

- **权限**: `admin.rbac.manage`

**额外约束**：仅 superadmin 可将 `award_all_permissions=true`

**Request**: `AwardAdminRequest`
```typescript
interface AwardAdminRequest {
  user_id: string;
  award_all_permissions: boolean | null;
  role_ids: string[];
}
```

**参数互斥说明**：
- 当 `award_all_permissions === true` 时，`role_ids` 必须为空数组或省略，服务端将忽略传入的角色列表。
- 当 `award_all_permissions !== true` 时，`role_ids` 为必填，至少包含一个角色。
- 若同时传入 `award_all_permissions=true` 且 `role_ids` 非空，视为参数冲突（10000）。

**Response**: `ApiResponse`

**错误码**: `10200` `10300`

---

### POST /admin/api/v1/rbac/admins/revoke

撤销平台管理员。

- **权限**: `admin.rbac.manage`

**额外约束**：撤销 superadmin 仅 superadmin 可执行

**Request**: `RevokeAdminRequest`
```typescript
interface RevokeAdminRequest {
  user_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10200` `10300`

---

## 审计日志 API

**字段可见性**（恒定 nullable：权限不足时字段为 null）：

| 字段 | 所需权限 | 说明 |
|------|----------|------|
| `ip_location` | `admin.audit.read` | 可见 |
| `ip_address` | `admin.audit.read_ip` | 敏感 PII；权限不足时为 null |
| `device_summary` | `admin.audit.read` | User-Agent 脱敏后的设备摘要 |

### POST /admin/api/v1/audit/fetch

查询审计日志列表。

- **权限**: `admin.audit.read`
- **排序**: 按 `created_at DESC` 排序（最新在前），`log_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FetchAuditLogsRequest`
```typescript
interface FetchAuditLogsRequest {
  action: AuditAction | null;
  operator_id: string | null;
  target_type: AuditTargetType | null;
  target_id: string | null;
  start_time: string | null;  // UTC ISO 8601
  end_time: string | null;    // UTC ISO 8601
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<AuditLogItem>>`

> 审计详情不再提供单独端点：`AuditLogItem` 列表项已含全量 `details` 字段。

```typescript
type AuditAction = 
  | 'user.ban'           // 封禁用户
  | 'user.unban'         // 解封用户
  | 'group.dismiss'      // 解散群组（群主经 groups/admin/dismiss，或平台侧经 /admin/api/v1/groups/dismiss）
  | 'group.member_mute'  // 禁言群成员（由群管理员通过 groups/admin/members/set_mute 触发）
  | 'group.member_ban'   // 封禁群成员（由群管理员通过 groups/admin/members/set_ban active=true 触发）
  | 'group.member_remove'// 移除群成员（由群管理员通过 groups/admin/members/remove 触发）
  | 'announcement.create'// 创建公告
  | 'announcement.update'// 更新公告
  | 'announcement.delete'// 删除公告
  | 'role.create'        // 创建角色
  | 'role.update'        // 更新角色
  | 'role.delete'        // 删除角色
  | 'role.set_permissions' // 设置角色权限
  | 'admin.award'        // 授权平台管理员
  | 'admin.revoke'       // 撤销平台管理员
  | 'report.judge'       // 裁决举报
  | 'tag.set_ban'        // 设置标签屏蔽状态（屏蔽/解屏蔽）
  | 'config.update';     // 更新系统配置

type AuditTargetType = 'user' | 'group' | 'announcement' | 'role' | 'admin' | 'report' | 'tag' | 'config';

// 注意：AuditDetails.type 必须与 AuditLogItem.action 一致（即每个 action 对应唯一的 details 变体）
type ConfigValue = string | number | boolean | null | ConfigValue[] | { [key: string]: ConfigValue };

type AuditDetails = 
  | { type: 'user.ban'; reason: string | null; duration_days: number | null }
  | { type: 'user.unban'; reason: string | null }
  | { type: 'group.dismiss'; reason: string | null }
  | { type: 'group.member_mute'; reason: string | null; duration_minutes: number | null }
  | { type: 'group.member_ban'; reason: string | null }
  | { type: 'group.member_remove'; reason: string | null }
  | { type: 'announcement.create'; title: string }
  | { type: 'announcement.update'; title: string; changed_fields: string[] }
  | { type: 'announcement.delete'; title: string }
  | { type: 'role.create'; role_name: string; permissions: string[] }
  | { type: 'role.update'; role_name: string; added_permissions: string[]; removed_permissions: string[] }
  | { type: 'role.delete'; role_name: string }
  | { type: 'role.set_permissions'; role_name: string; permissions: string[] }
  | { type: 'admin.award'; target_user_id: string; role_ids: string[]; award_all_permissions: boolean | null }
  | { type: 'admin.revoke'; target_user_id: string }
  | { type: 'report.judge'; report_id: string; action: ReportAction; note: string | null }
  | { type: 'tag.set_ban'; name: string; active: boolean }
  | { type: 'config.update'; config_key: string; old_value: ConfigValue; new_value: ConfigValue };

interface AuditLogItem {
  log_id: string;
  action: AuditAction;
  operator: UserInfo;
  target_type: AuditTargetType;
  target_id: string;
  target_name: string | null;
  details: AuditDetails | null;
  ip_location: string | null;           // 需 admin.audit.read；权限不足为 null
  ip_address: string | null;            // 需 admin.audit.read_ip；权限不足为 null
  device_summary: string | null;        // 脱敏后的 UA 设备摘要
  created_at: string;
}
```

**错误码**: `10200`（not_admin / insufficient_admin_permission）

---

## 应用配置 API（AppConfig）

**设计说明**：`AppConfig` 是完整的应用配置，由平台管理员统一管理。其中 `features`/`limits`/`contact` 部分构成 `ClientConfig`（投影给客户端），其余为仅服务端使用的运行参数。

### POST /admin/api/v1/app_config

获取完整应用配置（AppConfig）。

- **权限**: `admin.app_config.read`

**Request**: `AppConfigRequest`
```typescript
interface AppConfigRequest {}
```

**Response**: `ApiResponse<AppConfig>`

```typescript
// AppConfig: 完整的应用配置，由平台管理员统一管理
// ClientConfig（见 api_spec_types.md）是 AppConfig 中客户端可见部分的投影
interface AppConfig {
  // === 客户端可见配置（构成 ClientConfig） ===
  features: FeatureSwitches;       // 定义于 api_spec_types.md
  limits: AppLimits;               // 定义于 api_spec_types.md
  contact: ContactInfo;            // 定义于 api_spec_types.md
  // === 仅服务端使用 ===
  upload: UploadConfig;
  defaults: ContentDefaults;
  rate_limits: RateLimitConfig;
  response_defaults: ResponseDefaults;
  // === 元数据 ===
  updated_at: string | null;
  updated_by_user_id: string | null;
}
```

**错误码**: `10200`（not_admin / insufficient_admin_permission）

---

### POST /admin/api/v1/app_config/update

更新应用配置（按 section 部分更新，Patch 语义）。

**设计说明**：`update` 按 section 部分更新（Patch 语义），一次请求可更新单个或多个 section，未传的 section 保持不变。各 section 权限统一为 `admin.app_config.write`。

- **权限**: `admin.app_config.write`

**Request**: `UpdateAppConfigRequest`
```typescript
interface UpdateAppConfigRequest {
  // === 仅服务端使用 ===
  upload: Partial<UploadConfig> | null;              // 上传策略
  defaults: AppConfigDefaultsPatch | null;           // 内容默认值（可单独更新子字段）
  rate_limits: Partial<RateLimitConfig> | null;      // 频率限制（动态生效，无需重启；所有值须为正整数）
  response_defaults: Partial<ResponseDefaults> | null;  // 响应默认值（所有值须为正整数）
  // === 客户端可见（构成 ClientConfig，通过 /public/v1/system/client_config 暴露）===
  features: Partial<FeatureSwitches> | null;         // 功能开关
  limits: Partial<AppLimits> | null;                 // 应用限制
  contact: Partial<ContactInfo> | null;              // 联系/合规信息
}

interface AppConfigDefaultsPatch {
  default_location: Partial<Location> | null;
  default_group_avatar_url: string | null;
  default_user_avatar_url: string | null;
  default_group_settings: Partial<GroupSettings> | null;
}
```
// 至少传一个 section，否则请求失败；未传的 section 保持不变

**Response**: `ApiResponse`

**验证规则**：
- `rate_limits` / `response_defaults` 的数值字段必须为正整数
- `contact` 的 `support_email` 须为有效 Email、`*_url` 须为有效 URL

**错误码**: `10200`（not_admin / insufficient_admin_permission）

---

### 应用配置子类型定义

> 以下子类型构成 `AppConfig`。`FeatureSwitches`/`AppLimits`/`ContactInfo` 定义于 `api_spec_types.md`。

```typescript
interface UploadConfig {
  // 注：本配置为 image/video/audio/file 四档的全局限制；各分类（avatar/moment_image/chat_video 等）的
  // 具体 MIME 白名单与大小上限以 api_spec_media.md「分类限制」表为契约权威。
  // 优先级（契约级）：实际生效大小上限 = min(本配置对应四档全局值, media.md 该分类上限)——平台配置只能收紧、不能放宽；
  // MIME 白名单以 media.md 分类表为准，不因本配置放宽
  image_max_size: number;
  video_max_size: number;
  audio_max_size: number;
  file_max_size: number;
  allowed_image_types: string[];
  allowed_video_types: string[];
}

interface ContentDefaults {
  default_location: Location;              // 默认位置（含坐标和名称）
  default_group_avatar_url: string;
  default_user_avatar_url: string;
  default_group_settings: GroupSettings;
}

interface RateLimitConfig {
  // 全局
  global_per_hour: number;
  user_per_hour: number;
  // 认证类（IP 维度）
  auth_register_per_10minutes: number;
  auth_signin_per_10minutes: number;
  auth_guest_per_10minutes: number;
  auth_refresh_per_10minutes: number;
  auth_reset_per_10minutes: number;
  // 搜索类（用户维度，令牌桶）
  search_per_minute: number;
  // 通用写操作（用户维度，滑动窗口）
  write_per_minute: number;
  // 业务写操作
  moments_create_per_minute: number;
  comments_create_per_minute: number;
  reports_create_per_hour: number;
  // 文件上传 presign（用户维度，滑动窗口）
  upload_presign_per_hour: number;
  // WebSocket 订阅（连接维度，固定窗口）
  ws_subscribe_per_second: number;
  // 平台管理（用户维度）
  admin_read_per_minute: number;
  admin_write_per_minute: number;
}

interface ResponseDefaults {
  search_suggest_count: number;       // 搜索建议返回条数，默认 5
  chat_recent_messages_count: number; // open_the_door 返回的最近消息条数，默认 20
  group_announcements_count: number;  // 群公告列表返回条数，默认 10
  system_announcements_count: number; // 系统公告列表返回条数，默认 10
  comment_top_replies_count: number;  // 顶层评论内嵌的前 N 条回复，默认 3
  user_profile_moments_count: number; // users/open_the_door（include_recent_moments=true）返回的近期瞬间条数，默认 6
}
```

---

## 内容审核 API

### POST /admin/api/v1/users/search

搜索用户（平台管理）。

- **权限**: `admin.users.read`
- **排序**: 默认按照 `created_at DESC` 排序，时间相同以 `user.user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致。

**Request**: `SearchAdminUsersRequest`
```typescript
interface SearchAdminUsersRequest {
  user_id: string | null;
  nickname: string | null;
  login_id: string | null;
  status: UserAccountStatus | null;
  user_type: UserType | null;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<AdminUserItem>>`
```typescript
// AdminUserItem 继承 UserItem；继承的查看者字段（relation/social_context 等）在平台管理视角为 null（无业务意义）；
// 隐私受控字段（stats/location/presence_status 等）在平台管理视角豁免客户端隐私过滤、恒返回真值（见 api_spec_authz.md「交叉规则」第 4 条）
interface AdminUserItem extends UserItem {
  status: UserAccountStatus;
  banned_until: string | null;   // 封禁到期时间；null = 未封禁/永久封禁
  privacy_level: PrivacyLevel;   // 用户的隐私等级（平台管理视角可见）
  created_at: string;
  last_active_at: string | null;
  login_id: string;
  register_ip: string | null;       // 敏感 PII：需 admin.users.read_ip，权限不足为 null
  register_location: string | null;
  last_login_ip: string | null;     // 敏感 PII：需 admin.users.read_ip，权限不足为 null
  last_login_location: string | null;
  email: string | null;
  phone: string | null;
}
```

**错误码**: `10200`（not_admin / insufficient_admin_permission）

---

### POST /admin/api/v1/users/open_the_door

获取用户详情（平台管理）。

- **权限**: `admin.users.read`

**Request**: `OpenAdminUserDoorRequest`
```typescript
interface OpenAdminUserDoorRequest {
  user_id: string;
}
```

**Response**: `ApiResponse<AdminUserItem>`

**错误码**: `10200` `10300`

---

### POST /admin/api/v1/users/set_ban

设置用户封禁状态（两态统一端点，平台管理）。

`active=true`：封禁用户。`active=false`：解封用户——对未处于封禁状态的用户调用视为 no-op，返回成功。

- **权限**: `admin.users.write`

**安全约束**：
- 禁止封禁自身（返回 `10400`）
- 普通平台管理员禁止封禁其他平台管理员（返回 `10200` + `admin_protected_target`）
- 封禁 superadmin 仅限 superadmin 执行（返回 `10200` + `admin_protected_target`）
- 封禁/解封操作均记录审计日志（含 no-op 解封尝试，便于审计追溯）

**Request**: `SetAdminUserBanRequest`
```typescript
interface SetAdminUserBanRequest {
  user_id: string;
  active: boolean;        // true=封禁，false=解封
  reason: string | null;  // 封禁/解封原因（0-500 字符）
  duration_days: number | null; // 仅 active=true：封禁天数，不提供或 0 = 永久封禁；active=false 时忽略
}
```

**推送**：`active=true` 时向被封禁用户的当前 WebSocket 连接推送 `AccountBanned` 事件后断开连接（客户端收到后应立即登出并展示封禁页，见 `api_spec_websocket.md`）。

**Response**: `ApiResponse`

**错误码**: `10200`（not_admin / insufficient_admin_permission / admin_protected_target） `10300` `10400`（封禁自身）

---

### POST /admin/api/v1/groups/search

搜索群组（平台管理）。

- **权限**: `admin.groups.read`
- **排序**: 默认按照 `created_at DESC` 排序，时间相同以 `group.group_id ASC` 作为稳定 tie-breaker；一致性级别为强一致。

**Request**: `SearchAdminGroupsRequest`
```typescript
interface SearchAdminGroupsRequest {
  group_id: string | null;
  name: string | null;
  owner_id: string | null;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<AdminGroupItem>>`
```typescript
// AdminGroupItem 继承 GroupItem；继承的 viewer_state 在平台管理视角为 null
interface AdminGroupItem extends GroupItem {
  owner: UserInfo;
  created_at: string;   // 平台管理视角恒返回（覆盖继承的可空语义）
}
```

**错误码**: `10200`（not_admin / insufficient_admin_permission）

---

### POST /admin/api/v1/groups/open_the_door

获取群组详情（平台管理）。

- **权限**: `admin.groups.read`

**Request**: `OpenAdminGroupDoorRequest`
```typescript
interface OpenAdminGroupDoorRequest {
  group_id: string;
}
```

**Response**: `ApiResponse<AdminGroupItem>`

**错误码**: `10200` `10300`

---

### POST /admin/api/v1/groups/dismiss

解散群组（平台管理）。

- **权限**: `admin.groups.write`

**安全约束**：
- 解散后群内所有成员收到 `GroupDismissed` WebSocket 事件（`reason='policy_violation'`，不暴露平台管理员身份）
- 解散操作会记录审计日志（含完整 `reason` 自由文本）
- **reason 可见性**：`reason` 自由文本仅进审计日志，不推送给群成员
- **级联处理（契约级）**：解散时，该群所有 `pending` 状态的申请与邀请由服务端自动置为 `revoked`（与 `groups/admin/dismiss` 的级联约定一致）

**Request**: `DismissAdminGroupRequest`
```typescript
interface DismissAdminGroupRequest {
  group_id: string;
  reason: string | null;  // 解散原因（0-500 字符）
}
```

**Response**: `ApiResponse`

**错误码**: `10200` `10300`

---

### POST /admin/api/v1/reports/fetch

举报列表（平台管理）。按状态/目标类型过滤（无关键字搜索语义，故命名 `fetch` 而非 `search`）。

- **权限**: `admin.reports.read`
- **排序**: 默认按照 `created_at DESC` 排序，时间相同以 `report_id DESC` 作为稳定 tie-breaker；一致性级别为强一致。

**Request**: `FetchAdminReportsRequest`
```typescript
interface FetchAdminReportsRequest {
  status: ReportStatus | null;
  target_type: ReportTargetType | null;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<AdminReportItem>>`
```typescript
interface AdminReportItem extends ReportItem {
  reporter: UserInfo;
  handled_by: UserInfo | null;
  handle_note: string | null;
}
```

**错误码**: `10200`（not_admin / insufficient_admin_permission）

---

### POST /admin/api/v1/reports/judge

裁决举报（平台管理）：`resolve` 确认有效 / `dismiss` 驳回。

- **权限**: `admin.reports.write`

**安全约束**：
- 仅 `pending` 状态的举报可处理，已终态（`resolved`/`dismissed`）的举报返回 `10400`
- 处理操作会记录审计日志

**动作语义**：
- `resolve`：确认举报有效，执行处理（如封禁用户、删除内容），状态变更为 `resolved`。处理成功后向举报人推送 `ReportResolved` WebSocket 事件并落库通知（`NotifyType='report_resolved'`）
- `dismiss`：驳回举报，不执行任何处理，状态变更为 `dismissed`。驳回后向举报人推送 `ReportDismissed` WebSocket 事件并落库通知（`NotifyType='report_dismissed'`）

**Request**: `JudgeAdminReportRequest`
```typescript
interface JudgeAdminReportRequest {
  report_id: string;
  action: ReportAction;           // 'resolve' | 'dismiss'
  note: string | null;            // 处理备注（0-500 字符）
  ban_duration_days: number | null; // action=resolve 时可选：封禁天数（0=永久），不提供则仅标记处理不执行封禁。
                                //   target_type='user': 直接封禁该用户
                                //   target_type='moment'/'comment'/'message': 封禁内容发布者/发送者
                                //   target_type='group': 不封禁群主（避免连坐无辜），群违规统一走 groups/dismiss 解散；
                                //   若群主本人确为违规者，由平台管理员单独对其发起 users/set_ban
}
```

**权限蕴含（契约级）**：`action='resolve'` 且携带 `ban_duration_days` 时，实际执行封禁操作——调用方除 `admin.reports.write` 外**必须同时具备 `admin.users.write`**（权限不足返回 `10200` + `insufficient_admin_permission`），防止低权限平台管理员经举报通道绕过用户封禁权限边界。仅 `dismiss` 或 resolve 不携带 `ban_duration_days` 时只要求 `admin.reports.write`。

**字段映射说明**：本端点的 `note` 会写入举报详情中的 `handle_note` 字段，并记录到审计日志 `AuditDetails.report.judge.note`。

**Response**: `ApiResponse`

**错误码**: `10200` `10300` `10400`（已终态）

---

### POST /admin/api/v1/tags/search

搜索标签（平台管理）。

- **权限**: `admin.tags.read`
- **排序**: 默认按照 `usage_count DESC` 排序，数值相同以 `name ASC` 作为稳定 tie-breaker；一致性级别为弱一致。

**Request**: `SearchAdminTagsRequest`
```typescript
interface SearchAdminTagsRequest {
  keyword: string | null;  // 标签名模糊搜索
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<AdminTagItem>>`
```typescript
// 扩展 TagItem（定义于 api_spec_types.md）添加平台管理视角字段
interface AdminTagItem extends TagItem {
  tag_id: string;        // 标签唯一标识（内部存储标识，仅用于展示/审计关联，不参与删除寻址）
  created_at: string;
}
```

**错误码**: `10200`（not_admin / insufficient_admin_permission）

---

### POST /admin/api/v1/tags/set_ban

设置标签屏蔽状态（两态统一端点，平台管理）。`active=false` 解屏蔽消除误操作不可逆问题。

**屏蔽语义（软屏蔽，保护历史数据完整性）**：
- `active=true`（默认语义显式传参）：**软屏蔽**——标签不再可被新建/搜索命中，但**已引用该标签的历史瞬间/群组/用户数据保留不变**（不回写改写历史记录）
- `active=false`：解除屏蔽，同名标签恢复可创建/可命中（no-op 成功——对未屏蔽标签解屏蔽返回成功）
- 物理删除（从所有历史引用中移除）为**高危操作**，仅在 `active=true` 时可显式传 `hard_delete: true` 确认；物理删除不可逆，且物理删除后同名标签视为新标签可重新创建
- **新内容约束**：被屏蔽标签出现在新建内容（瞬间/群组/用户资料的 `tags`）中时，该请求失败（`10400`，message 指明被屏蔽的标签名）——对应端点（moments/create、groups/create、users/update_account）的错误码清单已同步列出 `10400`
- **寻址说明**：标签的业务唯一键为 `name`；`AdminTagItem.tag_id` 为内部存储标识，不参与寻址

- **权限**: `admin.tags.write`

**Request**: `SetAdminTagBanRequest`
```typescript
interface SetAdminTagBanRequest {
  name: string;                  // 目标标签名
  active: boolean;               // true=屏蔽，false=解屏蔽
  hard_delete: boolean | null;   // 仅 active=true 时有效：默认 false（软屏蔽）；true 时物理删除（不可逆，慎用）
}
```

**Response**: `ApiResponse`

**错误码**: `10200` `10300`

---

### POST /admin/api/v1/announcements/fetch

系统公告列表（平台管理视角），返回包含平台管理字段的公告列表。

- **权限**: `admin.announcements.read`
- **排序**: 默认按照 `created_at DESC` 排序，时间相同以 `announcement_id DESC` 作为稳定 tie-breaker；一致性级别为强一致。

**说明**：
- 普通用户通过 `POST /public/v1/system/announcements/fetch`（定义于 `api_spec_public.md`）获取公开公告
- 本端点返回 `AdminAnnouncementItem`（含 `created_by`、`view_count` 等平台管理字段）

**Request**: `FetchAdminAnnouncementsRequest`
```typescript
interface FetchAdminAnnouncementsRequest {
  severity: AnnounceSeverity | null;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<AdminAnnouncementItem>>`
```typescript
interface AdminAnnouncementItem extends SystemAnnouncementItem {
  created_by: UserInfo;
  send_push: boolean;
  push_sent_at: string | null;
  view_count: number;
}
```

**错误码**: `10200`（not_admin / insufficient_admin_permission）

---

### POST /admin/api/v1/announcements/create

发布系统公告（平台管理）。

- **权限**: `admin.announcements.write`
- **幂等**: 是（客户端必须携带 `Idempotency-Key`）

**Request**: `CreateAnnouncementRequest`
```typescript
interface CreateAnnouncementRequest {
  title: string;
  content: string;
  severity: AnnounceSeverity;
  action_url: string | null;
  send_push: boolean | null;
}
```

**Response**: `ApiResponse<AdminAnnouncementItem>`（返回完整公告对象，客户端直接渲染）

**错误码**: `10200`（not_admin / insufficient_admin_permission）

---

### POST /admin/api/v1/announcements/update

更新系统公告（平台管理）。

- **权限**: `admin.announcements.write`

**Request**: `UpdateAnnouncementRequest`
```typescript
interface UpdateAnnouncementRequest {
  announcement_id: string;
  title: string | null;
  content: string | null;
  severity: AnnounceSeverity | null;
  action_url: string | null;  // 传空字符串 "" 表示清除链接（与 avatar_file_id 清空约定一致）；null = 不变
}
```

**Response**: `ApiResponse<AdminAnnouncementItem>`（内容类更新返回更新后完整对象）

**错误码**: `10200` `10300`

---

### POST /admin/api/v1/announcements/delete

删除公告（平台管理）。

- **权限**: `admin.announcements.write`

**Request**: `DeleteAnnouncementRequest`
```typescript
interface DeleteAnnouncementRequest {
  announcement_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10200` `10300`

---

## 统计分析（stats）API

### POST /admin/api/v1/stats/geo

地理统计聚合快照：全站存量总数 + 区域排名（S2 网格）+ 热门地点排名（location_name 聚合）。供平台管理端从全局掌握站点数据的地理分布。

- **权限**: `admin.stats.read`

**统计口径（契约级）**：

- 平台管理视角豁免客户端隐私过滤（见 `api_spec_authz.md`「交叉规则」）：用户未注销（`is_deleted = false`）、群组未解散（`deleted_at` 为空）、动态未删除（`is_deleted = false`）即计入，不限可见性/可搜性，封禁账号照常计入
- 地理排名仅统计有有效位置的实体（无效位置不计入 `regions` / `places`，但计入 `totals`）
- 区域维度基于 S2 网格祖先桶聚合（与公开地图 heatmap 同一 token 体系），**不引入行政区概念**（与 `api_spec_public.md`「地图渲染」设计理念一致）
- 热门地点按 `location_name` 精确文本聚合（自由文本同名合并，不做行政区归一化；空地名不计入 `places`）
- `time_range` 只作用于 `regions` / `places`（按各实体 `created_at` 过滤）；`totals` 恒为全站存量现状，不受时间窗影响
- 一致性级别为弱一致（实时聚合快照，非预聚合表）

**Request**: `AdminGeoStatsRequest`
```typescript
interface AdminGeoStatsRequest {
  level: number | null;        // S2 聚合层级，[4,12]，数字越大网格越细；null = 8（约 20km，城市尺度）
  time_range: TimeRange | null; // 参与排名实体的创建时间窗（UTC ISO 8601）；null = 不限
  limit: number | null;        // 排名条数；0/null = 20，服务端裁剪上限 100
  trend_days: number | null;   // 每日新增分布天数，[1,90]；null/0 = 不返回趋势
}

interface TimeRange {
  start_time: string | null;   // 含边界（created_at >= start_time）
  end_time: string | null;     // 含边界（created_at <= end_time）；同时提供时必须 start_time <= end_time
}

interface StatsEntityBreakdown {
  user: number;                // 参与统计的用户数（恒返回真值，平台管理视角无隐私 null 化）
  group: number;
  moment: number;
}

interface AdminRegionRankItem {
  cell_token: string;          // S2 cell token（与 api_spec_public.md「地图渲染」heatmap 的 S2CellToken 同体系）
  level: number;               // 本次请求生效的 S2 层级
  center: GeoPoint;            // 网格中心点（代表坐标，供地图落点渲染）
  total_count: number;
  breakdown: StatsEntityBreakdown;
}

interface AdminPlaceRankItem {
  location_name: string;
  total_count: number;
  breakdown: StatsEntityBreakdown;
}

interface AdminGeoStatsTotals {
  user: number;                // 全站未注销用户数
  group: number;               // 全站未解散群组数
  moment: number;              // 全站未删除动态数
  total: number;               // 三者之和
}

interface AdminGeoStatsTrendPoint {
  date: string;                // YYYY-MM-DD，按 UTC+8 日界分桶（固定偏移，无夏令时）
  user: number;                // 当日新增用户数
  group: number;
  moment: number;
}

interface AdminGeoStatsResponse {
  generated_at: string;        // 统计生成时间（UTC ISO 8601）
  totals: AdminGeoStatsTotals;
  regions: AdminRegionRankItem[]; // 按 total_count 降序，key 升序为稳定 tie-breaker；Top-N，无分页
  places: AdminPlaceRankItem[];   // 排序规则同上
  trends: AdminGeoStatsTrendPoint[]; // 请求 trend_days 时为逐日补零的连续序列（含今天），否则空数组
}
```

**Response**: `ApiResponse<AdminGeoStatsResponse>`

**错误码**: `10000`（level 越界 / 时间窗倒置 / 时间格式非法 / limit 为负 / trend_days 越界）`10200`

### POST /admin/api/v1/stats/geo/drilldown

区域下钻：按 S2 网格桶返回其中指定实体类型的明细列表。`cell_token` 来自 `stats/geo` 的 `regions`；服务端以 token 实际层级为准，请求携带的 `level` 必须与之相同时才接受。

- **权限**: `admin.stats.read`

**Request**: `AdminGeoDrilldownRequest`
```typescript
interface AdminGeoDrilldownRequest {
  cell_token: string;             // 必填，合法 S2 token
  level: number | null;           // 可选；携带时必须等于 token 实际层级
  entity_type: 'user' | 'group' | 'moment'; // 必填
  time_range: TimeRange | null;   // 可选，与 stats/geo 同口径
  pagination: PaginationParams | null; // 可选；limit 默认 20、上限 100
}

interface AdminGeoDrilldownItem {
  object_id: string;              // user_id / group_id / moment_id
  title: string;                  // 昵称 / 群名 / 动态描述（截 60 字）
  location_name: string | null;   // 空地名为 null
  latitude: number;
  longitude: number;
  created_at: string;             // UTC ISO 8601
}

interface AdminGeoDrilldownPage {
  items: AdminGeoDrilldownItem[];
  next_cursor: string | null;     // 无更多数据时为 null
}
```

**Response**: `ApiResponse<AdminGeoDrilldownPage>`

**行为约束（契约级）**：

- 明细按 `created_at` 降序、业务 ID 降序稳定排序；地理口径与 `stats/geo` 一致
- `next_cursor` 恒定 nullable，客户端只透传

**错误码**: `10000`（cell_token 非法 / level 与 token 不一致 / entity_type 非法 / 时间窗倒置 / 时间格式非法 / limit 为负 / cursor 非法）`10200`