---
alwaysApply: false
---
# HotSpot API - 管理后台核心

> 本文档定义 HotSpot 管理后台核心功能的 RPC 接口，包括权限管理（RBAC）、审计日志和全局配置。
> 内容审核相关接口（用户/群组/举报/标签/公告管理）定义在 `@spec/api_spec_admin_moderation.md`。

## 模块信息

- **命名空间**: 
  - `admin.rbac`
  - `admin.audit`
  - `admin.app_config`
- **基础路径**: 
  - `/api/v1/admin/rbac/`
  - `/api/v1/admin/audit/`
  - `/api/v1/admin/app_config/`
- **引用**: 
  - 基础规范：`@spec/api_spec_base.md`
  - 通用类型：`@spec/api_spec_types.md`
  - 错误码：`@spec/api_spec_errors.md`
  - 客户端配置类型：`@spec/api_spec_system.md`（`ClientConfig`、`FeatureSwitches`、`AppLimits`、`ContactInfo`）

## 统一访问控制

仅对 `/api/v1/admin/` 路径下接口生效：
- 未登录/登录失效：返回 `10100`（认证失败）
- 已登录但不是管理员：返回 `10200`（权限不足，message: "非系统管理员"）
- 是管理员但缺少接口要求的权限：返回 `10200`（权限不足，message: "系统管理员权限不足: xxx"）

## 幂等性（管理员写操作）

管理员写操作的幂等性遵循 `@spec/api_spec_base.md` 的分级规则：
- **MUST**：具有"创建/授权/封禁/解散/处理"语义的接口必须携带 `Idempotency-Key`（见各端点的幂等标记）
- **幂等语义**：具有"更新/删除/撤销"语义的接口为天然幂等，重复请求返回一致结果，无需 `Idempotency-Key`

## 权限 key 约定

所有管理员权限键必须严格遵循以下约定：

| 权限 key | 说明 |
|----------|------|
| `admin.rbac.manage` | 管理员权限管理（授权/角色/权限） |
| `admin.reports.read` | 查看举报列表 |
| `admin.reports.write` | 处理举报 |
| `admin.app_config.read` | 查看应用配置（AppConfig） |
| `admin.app_config.write` | 修改应用配置（AppConfig） |
| `admin.tags.read` | 查看标签列表 |
| `admin.tags.write` | 删除/屏蔽标签 |
| `admin.announcements.read` | 查看公告列表 |
| `admin.announcements.write` | 发布/删除公告 |
| `admin.users.read` | 查看用户列表/详情 |
| `admin.users.write` | 封禁/解封用户 |
| `admin.groups.read` | 查看群组列表/详情 |
| `admin.groups.write` | 解散群组 |
| `admin.audit.read` | 查看审计日志（不含 `ip_address` 字段） |
| `admin.audit.read_ip` | 查看审计日志 IP 地址（敏感 PII，独立权限控制，需同时具备 admin.audit.read） |

## 端点总览

| 端点 | 说明 | 认证 | 幂等 |
|------|------|------|------|
| `POST /api/v1/admin/rbac/get_me` | 我的管理员权限 | 必须 | 无 |
| `POST /api/v1/admin/rbac/list_permissions` | 权限字典 | 必须 + `admin.rbac.manage` | 无 |
| `POST /api/v1/admin/rbac/roles/list` | 角色列表 | 必须 + `admin.rbac.manage` | 无 |
| `POST /api/v1/admin/rbac/roles/create` | 创建角色 | 必须 + `admin.rbac.manage` | MUST |
| `POST /api/v1/admin/rbac/roles/update` | 更新角色 | 必须 + `admin.rbac.manage` | 幂等语义 |
| `POST /api/v1/admin/rbac/roles/delete` | 删除角色 | 必须 + `admin.rbac.manage` | 幂等语义 |
| `POST /api/v1/admin/rbac/roles/set_permissions` | 设置角色权限 | 必须 + `admin.rbac.manage` | 幂等语义 |
| `POST /api/v1/admin/rbac/admins/list` | 管理员列表 | 必须 + `admin.rbac.manage` | 无 |
| `POST /api/v1/admin/rbac/admins/grant` | 授权/更新管理员 | 必须 + `admin.rbac.manage` | MUST |
| `POST /api/v1/admin/rbac/admins/revoke` | 撤销管理员 | 必须 + `admin.rbac.manage` | 幂等语义 |
| `POST /api/v1/admin/audit/list` | 审计日志列表 | 必须 + `admin.audit.read` | 无 |
| `POST /api/v1/admin/audit/open_the_door` | 审计日志详情 | 必须 + `admin.audit.read` | 无 |
| `POST /api/v1/admin/app_config/get` | 获取完整应用配置 | 必须 + `admin.app_config.read` | 无 |
| `POST /api/v1/admin/app_config/update_upload` | 更新上传策略 | 必须 + `admin.app_config.write` | 幂等语义 |
| `POST /api/v1/admin/app_config/update_defaults` | 更新内容默认值 | 必须 + `admin.app_config.write` | 幂等语义 |
| `POST /api/v1/admin/app_config/update_rate_limits` | 更新频率限制 | 必须 + `admin.app_config.write` | 幂等语义 |
| `POST /api/v1/admin/app_config/update_response_defaults` | 更新响应默认值 | 必须 + `admin.app_config.write` | 幂等语义 |
| `POST /api/v1/admin/app_config/update_features` | 更新功能开关 | 必须 + `admin.app_config.write` | 幂等语义 |
| `POST /api/v1/admin/app_config/update_limits` | 更新应用限制 | 必须 + `admin.app_config.write` | 幂等语义 |
| `POST /api/v1/admin/app_config/update_contact` | 更新联系信息 | 必须 + `admin.app_config.write` | 幂等语义 |

## 类型定义

以下类型定义于 `@spec/api_spec_types.md`：
- `UserInfo`
- `GroupSettings`
- `PaginationParams`
- `PaginatedData<T>`

`ApiResponse<T>` 定义于 `@spec/api_spec_base.md`。

## 端点详情

### RBAC 管理 API

**设计目标**：
- **超级管理员**：`permissions` 包含 `'*'`，表示拥有全部管理员权限（包括所有细粒度权限）
- **普通管理员**：通过角色/权限表获得细粒度权限，字段可见性由具体权限决定

### POST /api/v1/admin/rbac/get_me

获取我的管理员权限信息。

- **认证**: 必须（仅要求"是管理员"，不要求 `admin.rbac.manage`）
- **幂等**: 无

**Request**: `{}`

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

**错误码**: `10200`

---

### POST /api/v1/admin/rbac/list_permissions

获取权限字典。

- **认证**: 必须 + `admin.rbac.manage`
- **幂等**: 无

**Request**: `{}`

**Response**: `ApiResponse<{ items: AdminPermissionItem[] }>`

```typescript
interface AdminPermissionItem {
  key: string;
  description: string;
}
```

**错误码**: `10200`

---

### POST /api/v1/admin/rbac/roles/list

获取角色列表。

- **认证**: 必须 + `admin.rbac.manage`
- **幂等**: 无

**Request**: `AdminRbacRolesListRequest`
```typescript
interface AdminRbacRolesListRequest {} // 直接返回全部角色，无需分页
```

**Response**: `ApiResponse<{ items: AdminRoleItem[] }>`

```typescript
interface AdminRoleItem {
  role_id: string;
  name: string;
  description?: string;
  permissions: string[];
}
```

**说明**：
- RBAC 角色为系统配置数据，数量有限，无需分页
- 与 `list_permissions` 返回格式保持一致

**错误码**: `10000` `10200`

---

### POST /api/v1/admin/rbac/roles/create

创建角色。

- **认证**: 必须 + `admin.rbac.manage`
- **幂等**: MUST（客户端必须携带 `Idempotency-Key`）

**Request**: `AdminRbacRoleCreateRequest`
```typescript
interface AdminRbacRoleCreateRequest {
  name: string;
  description?: string;
}
```

**Response**: `ApiResponse<AdminRoleItem>`

**错误码**: `10000` `10200` `10400`

---

### POST /api/v1/admin/rbac/roles/update

更新角色。

- **认证**: 必须 + `admin.rbac.manage`
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `AdminRbacRoleUpdateRequest`
```typescript
interface AdminRbacRoleUpdateRequest {
  role_id: string;
  name?: string;
  description?: string;
  clear_description?: boolean;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300` `10400`

---

### POST /api/v1/admin/rbac/roles/delete

删除角色。

- **认证**: 必须 + `admin.rbac.manage`
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `AdminRbacRoleDeleteRequest`
```typescript
interface AdminRbacRoleDeleteRequest {
  role_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10200` `10300` `10400`

---

### POST /api/v1/admin/rbac/roles/set_permissions

设置角色权限。

- **认证**: 必须 + `admin.rbac.manage`
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `AdminRbacRoleSetPermissionsRequest`
```typescript
interface AdminRbacRoleSetPermissionsRequest {
  role_id: string;
  permission_keys: string[];
}
```

**说明**：
- `permission_keys` 必须来自权限字典（`POST /api/v1/admin/rbac/list_permissions` 返回值）
- 仅超级管理员可以设置 `'*'`（全量权限）；普通管理员设置 `'*'` 时返回 `10200`

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300`

---

### POST /api/v1/admin/rbac/admins/list

获取管理员列表。

- **认证**: 必须 + `admin.rbac.manage`
- **幂等**: 无
- **排序**: 按 `granted_at DESC` 排序（最近授权在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `AdminRbacAdminsListRequest`
```typescript
interface AdminRbacAdminsListRequest {
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<AdminOperatorItem>>`

```typescript
interface AdminOperatorItem {
  user: UserInfo;
  permissions: string[];  // '*' 表示拥有全部权限
  roles: AdminRoleBasic[];
}
```

**错误码**: `10000` `10200`

---

### POST /api/v1/admin/rbac/admins/grant

授予/更新管理员。

- **认证**: 必须 + `admin.rbac.manage`
- **幂等**: MUST（客户端必须携带 `Idempotency-Key`）

**额外约束**：仅超级管理员可将 `grant_all_permissions=true`

**Request**: `AdminRbacGrantAdminRequest`
```typescript
interface AdminRbacGrantAdminRequest {
  user_id: string;
  grant_all_permissions?: boolean;
  role_ids: string[];
}
```

**参数互斥说明**：
- 当 `grant_all_permissions === true` 时，`role_ids` 必须为空数组或省略，服务端将忽略传入的角色列表。
- 当 `grant_all_permissions !== true` 时，`role_ids` 为必填，至少包含一个角色。
- 若同时传入 `grant_all_permissions=true` 且 `role_ids` 非空，视为参数冲突，返回 `10000`（message: "grant_all_permissions 与 role_ids 互斥"）。

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300`

---

### POST /api/v1/admin/rbac/admins/revoke

撤销管理员。

- **认证**: 必须 + `admin.rbac.manage`
- **幂等**: 幂等语义（重复请求返回一致结果）

**额外约束**：撤销超级管理员仅超级管理员可执行

**Request**: `AdminRbacRevokeAdminRequest`
```typescript
interface AdminRbacRevokeAdminRequest {
  user_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10200` `10300`

---

## 审计日志 API

**字段可见性**：

| 字段 | 所需权限 | 说明 |
|------|----------|------|
| `ip_location` | `admin.audit.read` | 可见 |
| `ip_address` | `admin.audit.read_ip` | 敏感 PII，独立权限控制；权限不足时字段缺省（key 不出现，不返回 `null`） |
| `device_summary` | `admin.audit.read` | User-Agent 脱敏后的设备摘要 |

### POST /api/v1/admin/audit/list

查询审计日志列表。

- **认证**: 必须 + `admin.audit.read`
- **幂等**: 无
- **排序**: 按 `created_at DESC` 排序（最新在前），`log_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `AuditLogListRequest`
```typescript
interface AuditLogListRequest {
  action?: AuditAction;
  operator_id?: string;
  target_type?: AuditTargetType;
  target_id?: string;
  start_time?: string;  // UTC ISO 8601
  end_time?: string;    // UTC ISO 8601
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<AuditLogItem>>`

```typescript
type AuditAction = 
  | 'user.ban'           // 封禁用户
  | 'user.unban'         // 解封用户
  | 'group.dismiss'      // 解散群组
  | 'group.member_mute'  // 禁言群成员（由群管理员通过 groups/members/mute 触发）
  | 'group.member_ban'   // 封禁群成员（由群管理员通过 groups/members/ban 触发）
  | 'group.member_remove'// 移除群成员（由群管理员通过 groups/members/remove 触发）
  | 'announcement.create'// 创建公告
  | 'announcement.update'// 更新公告
  | 'announcement.delete'// 删除公告
  | 'role.create'        // 创建角色
  | 'role.update'        // 更新角色
  | 'role.delete'        // 删除角色
  | 'role.set_permissions' // 设置角色权限
  | 'admin.grant'        // 授权管理员
  | 'admin.revoke'       // 撤销管理员
  | 'report.handle'      // 处理举报
  | 'tag.delete'         // 删除/屏蔽标签
  | 'permission.grant'   // 授予权限
  | 'permission.revoke'  // 撤销权限
  | 'config.update';     // 更新系统配置

type AuditTargetType = 'user' | 'group' | 'announcement' | 'role' | 'admin' | 'report' | 'tag' | 'config';

// 注意：AuditDetails.type 必须与 AuditLogItem.action 一致（即每个 action 对应唯一的 details 变体）
// ReportHandleAction 定义于 @spec/api_spec_types.md
type AuditDetails = 
  | { type: 'user.ban'; reason?: string; duration_days?: number }
  | { type: 'user.unban'; reason?: string }
  | { type: 'group.dismiss'; reason?: string }
  | { type: 'group.member_mute'; reason?: string; duration_minutes?: number }
  | { type: 'group.member_ban'; reason?: string }
  | { type: 'group.member_remove'; reason?: string }
  | { type: 'announcement.create'; title: string }
  | { type: 'announcement.update'; title: string; changed_fields?: string[] }
  | { type: 'announcement.delete'; title: string }
  | { type: 'role.create'; role_name: string; permissions: string[] }
  | { type: 'role.update'; role_name: string; added_permissions?: string[]; removed_permissions?: string[] }
  | { type: 'role.delete'; role_name: string }
  | { type: 'role.set_permissions'; role_name: string; permissions: string[] }
  | { type: 'admin.grant'; target_user_id: string; role_ids: string[]; grant_all_permissions?: boolean }
  | { type: 'admin.revoke'; target_user_id: string }
  | { type: 'report.handle'; report_id: string; action: ReportHandleAction; note?: string }
  | { type: 'tag.delete'; name: string }
  | { type: 'permission.grant'; permission: string }
  | { type: 'permission.revoke'; permission: string }
  | { type: 'config.update'; config_key: string; old_value: unknown; new_value: unknown };

interface AuditLogItem {
  log_id: string;
  action: AuditAction;
  operator: UserInfo;
  target_type: AuditTargetType;
  target_id: string;
  target_name?: string;
  details?: AuditDetails;
  ip_location?: string;           // IP 地理位置（需 admin.audit.read）
  ip_address?: string;            // 需 admin.audit.read_ip；权限不足时字段缺省
  device_summary?: string;        // 脱敏后的 UA 设备摘要（需 admin.audit.read）
  created_at: string;
}
```

**错误码**: `10000` `10200`

---

### POST /api/v1/admin/audit/open_the_door

获取审计日志详情。

- **认证**: 必须 + `admin.audit.read`
- **幂等**: 无

**Request**: `AuditLogOpenTheDoorRequest`
```typescript
interface AuditLogOpenTheDoorRequest {
  log_id: string;
}
```

**Response**: `ApiResponse<AuditLogItem>`

**错误码**: `10000` `10200` `10300`

---

## 应用配置 API（AppConfig）

**设计说明**：`AppConfig` 是完整的应用配置，由管理员统一管理。其中 `features`/`limits`/`contact` 部分构成 `ClientConfig`（投影给客户端），其余为仅服务端使用的运行参数。

### POST /api/v1/admin/app_config/get

获取完整应用配置（AppConfig）。

- **认证**: 必须 + `admin.app_config.read`
- **幂等**: 无

**Request**: `{}`

**Response**: `ApiResponse<AppConfig>`

```typescript
// AppConfig: 完整的应用配置，由管理员统一管理
// ClientConfig（见 @spec/api_spec_system.md）是 AppConfig 中客户端可见部分的投影
interface AppConfig {
  // === 客户端可见配置（构成 ClientConfig） ===
  features: FeatureSwitches;       // 定义于 @spec/api_spec_system.md
  limits: AppLimits;               // 定义于 @spec/api_spec_system.md
  contact: ContactInfo;            // 定义于 @spec/api_spec_system.md
  // === 仅服务端使用 ===
  upload: UploadConfig;
  defaults: ContentDefaults;
  rate_limits: RateLimitConfig;
  response_defaults: ResponseDefaults;
  // === 元数据 ===
  updated_at?: string;
  updated_by_user_id?: string;
}

```

**错误码**: `10200`

---

### POST /api/v1/admin/app_config/update_upload

更新上传策略。

- **认证**: 必须 + `admin.app_config.write`
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `UpdateUploadConfigRequest`
```typescript
interface UploadConfig {
  image_max_size: number;
  video_max_size: number;
  audio_max_size: number;
  file_max_size: number;
  allowed_image_types: string[];
  allowed_video_types: string[];
}

interface UpdateUploadConfigRequest {
  upload: Partial<UploadConfig>;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200`

---

### POST /api/v1/admin/app_config/update_defaults

更新内容默认值。

- **认证**: 必须 + `admin.app_config.write`
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `UpdateDefaultsRequest`
```typescript
interface ContentDefaults {
  default_location: Location;              // 默认位置（含坐标和名称），Location 定义于 @spec/api_spec_types.md
  default_group_avatar: string;
  default_user_avatar: string;
  default_group_settings: GroupSettings;    // GroupSettings 定义于 @spec/api_spec_types.md
}

interface UpdateDefaultsRequest {
  default_location?: Partial<Location>;    // 可单独更新 point 或 location_name
  default_group_avatar?: string;
  default_user_avatar?: string;
  default_group_settings?: Partial<GroupSettings>;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200`

---

### POST /api/v1/admin/app_config/update_rate_limits

更新频率限制配置。

- **认证**: 必须 + `admin.app_config.write`
- **幂等**: 幂等语义（重复请求返回一致结果）

**说明**：动态调整 Rate Limit 策略，无需重启服务即可生效。

**Request**: `UpdateRateLimitsRequest`
```typescript
interface RateLimitConfig {
  // 全局
  global_per_hour: number;
  user_per_hour: number;
  // 认证类（IP 维度）
  auth_register_per_10minutes: number;
  auth_login_per_10minutes: number;
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
  // 管理员（用户维度）
  admin_read_per_minute: number;
  admin_write_per_minute: number;
}

interface UpdateRateLimitsRequest {
  rate_limits: Partial<RateLimitConfig>;
}
```

**Response**: `ApiResponse`

**验证规则**：
- 所有值必须为正整数

**错误码**: `10000` `10200`

---

### POST /api/v1/admin/app_config/update_response_defaults

更新响应默认值配置。

控制非分页接口的服务端返回数量（如搜索建议条数、热门标签条数、聊天开门时的最近消息条数）。

- **认证**: 必须 + `admin.app_config.write`
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `UpdateResponseDefaultsRequest`
```typescript
interface ResponseDefaults {
  search_suggest_count: number;       // 搜索建议返回条数，默认 5
  trending_tags_count: number;        // 热门标签返回条数（feed/list_trending），默认 10
  hot_moments_count: number;          // feed/list_trending 热门瞬间返回条数，默认 20
  chat_recent_messages_count: number; // open_the_door 返回的最近消息条数，默认 20
  group_announcements_count: number;  // 群公告列表返回条数，默认 10
  system_announcements_count: number; // 系统公告列表返回条数（system/list_announcements），默认 10
}

interface UpdateResponseDefaultsRequest {
  response_defaults: Partial<ResponseDefaults>;
}
```

**Response**: `ApiResponse`

**验证规则**：
- 所有值必须为正整数

**错误码**: `10000` `10200`

---

## 客户端可见配置管理

**设计说明**：以下端点管理 `AppConfig` 中客户端可见的部分（`features`/`limits`/`contact`），这些字段构成 `ClientConfig`，通过公开端点 `system/get_client_config`（见 `@spec/api_spec_system.md`）暴露给客户端。

> 统一使用 `admin.app_config.read` / `admin.app_config.write` 权限。
> 客户端可通过公开端点 `system/get_client_config`（见 `@spec/api_spec_system.md`）获取 ClientConfig 投影，无需认证。

### POST /api/v1/admin/app_config/update_features

更新功能开关。

- **认证**: 必须 + `admin.app_config.write`
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `UpdateFeaturesRequest`
```typescript
interface UpdateFeaturesRequest {
  enable_public_registration?: boolean;
  enable_guest_login?: boolean;
  enable_moments?: boolean;
  enable_nearby_people?: boolean;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200`

---

### POST /api/v1/admin/app_config/update_limits

更新应用限制。

- **认证**: 必须 + `admin.app_config.write`
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `UpdateLimitsRequest`
```typescript
interface UpdateLimitsRequest {
  limits: Partial<AppLimits>;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200`

---

### POST /api/v1/admin/app_config/update_contact

更新联系信息。

- **认证**: 必须 + `admin.app_config.write`
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `UpdateContactRequest`
```typescript
interface UpdateContactRequest {
  support_email?: string;
  privacy_policy_url?: string;
  terms_of_service_url?: string;
  official_website?: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200`
