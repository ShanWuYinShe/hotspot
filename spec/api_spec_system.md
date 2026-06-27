# HotSpot API - 系统与全局配置

> 本文档定义 HotSpot 系统级公开接口，包括健康检查、版本元信息、客户端配置和系统公告。所有端点为公开资源，无需认证，统一在 `/api/v1/system/` 路径下。

## 模块信息

- **命名空间**: `system`
- **基础路径**: `/api/v1/system/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 错误码：`api_spec_errors.md`

## 端点总览

| 端点 | 说明 | 认证 | 幂等 |
|-----|------|------|------|
| `POST /api/v1/system/get_meta` | API 版本元信息 | 无 | 无 |
| `POST /api/v1/system/health` | 健康检查 | 无 | 无 |
| `POST /api/v1/system/list_announcements` | 系统公告列表 | 无 | 无 |
| `POST /api/v1/system/get_client_config` | 客户端公开配置 | 无 | 无 |

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `SystemAnnouncementItem`

## 端点详情

### POST /api/v1/system/get_meta

获取当前 API 版本的元信息，包括废弃状态和下线计划。

- **认证**: 无
- **幂等**: 无

**说明**：各 API 版本路径下都应提供此端点；客户端启动时调用以检查当前版本状态。

**状态语义（供客户端参考）**：
- `active`: 正常使用，无需处理
- `deprecated`: 提示用户升级；可展示 `message` 并引导更新
- `sunset`: 强制升级或提示服务不可用

**Request**: `{}`

**Response**: `ApiResponse<ApiVersionMeta>`

```typescript
// API 版本元信息
interface ApiVersionMeta {
  version: string;                    // 当前 API 版本号，如 "v1"
  status: ApiVersionStatus;           // 版本状态
  deprecated_at?: string;             // 进入废弃状态时间（UTC ISO 8601）
  sunset_at?: string;                 // 计划下线时间（UTC ISO 8601）
  message?: string;                   // 面向客户端的提示信息
  recommended_version?: string;       // 推荐升级的版本号（如 "v2"）
}

// API 版本状态
type ApiVersionStatus = 
  | 'active'      // 正常使用中
  | 'deprecated'  // 已废弃，但仍可用
  | 'sunset';     // 即将下线或已停止
```

**错误码**: `10000`

---

### POST /api/v1/system/health

系统健康检查。

- **认证**: 无
- **幂等**: 无

**说明**：HTTP 非 200 表示未进入业务层；HTTP 200 时，`code=0` 表示系统可用（`data.status` 为 `healthy` 或 `degraded`），`code=10600` 表示系统不可用（`unhealthy`，`data` 省略，原因见 `message`）。

**Request**: `{}`

**Response**: `ApiResponse<HealthStatus>`（仅当 `code=0` 时返回 `data`）

```typescript
// 系统健康状态
type SystemHealthStatus = 'healthy' | 'degraded' | 'unhealthy';

interface HealthStatus {
  status: SystemHealthStatus;
  timestamp: string;
}
```

**错误码**: `10600`

---

### POST /api/v1/system/list_announcements

获取系统公告列表。

- **认证**: 无
- **幂等**: 无

**Request**: `{}`

**Response**: `ApiResponse<{ items: SystemAnnouncementItem[] }>`

**说明**：
- 返回当前有效的最新公告（"有效"= 未被管理员删除且状态为已发布），返回条数由服务端 `AppConfig.response_defaults.system_announcements_count` 控制，客户端不传数量参数
- 普通用户只需获取最新的 N 条公告，无需分页翻阅历史
- 管理员查看/搜索全部公告（含历史）请使用 `admin/announcements/search`（定义于 `api_spec_admin_moderation.md`）
- 按 `created_at` 倒序排列

**错误码**: `10000`

---

### POST /api/v1/system/get_client_config

获取客户端公开配置。

- **认证**: 无
- **幂等**: 无

**说明**：
- 返回的 `ClientConfig` 由 `AppConfig`（见 `api_spec_admin_core.md`）投影得来，包含功能开关、输入限制、联系方式
- 客户端应在启动时调用并缓存，配置变更频率低
- `defaults.default_location` 必须满足 `Location` 类型约束（尤其 `location_name` 非空）

**Request**: `{}`

**Response**: `ApiResponse<ClientConfig>`

```typescript
// 客户端公开配置（AppConfig 的可见投影；客户端启动时获取并缓存）
interface ClientConfig {
  features: FeatureSwitches;
  limits: AppLimits;
  contact: ContactInfo;
  defaults: ClientDefaults;
}

interface FeatureSwitches {
  enable_public_registration: boolean;
  enable_guest_login: boolean;
  enable_moments: boolean;
  enable_nearby_people: boolean;
}

interface AppLimits {
  max_group_members: number;
  max_moment_tags: number;
  max_moment_images: number;
  moment_description_max_length: number;
  comment_max_length: number;
  search_radius_max: number;  // 搜索半径上限（米），用于搜索和 Feed 相关接口
}

interface ContactInfo {
  support_email: string;
  privacy_policy_url: string;
  terms_of_service_url: string;
  official_website: string;
}

// 客户端可见的默认值
// 由管理员在后台配置（见 api_spec_admin_core.md 中 ContentDefaults）
interface ClientDefaults {
  default_location: Location;    // 默认位置（必须满足 Location 约束），Location 定义于 api_spec_types.md
  default_user_avatar: string;   // 默认用户头像 URL
  default_group_avatar: string;  // 默认群组头像 URL
}
```

**错误码**: `10000`
