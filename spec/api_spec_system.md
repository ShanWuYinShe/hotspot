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
| `POST /api/v1/system/bootstrap` | 冷启动聚合（config + meta + profile + unread） | 可选 | 无 |

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `SystemAnnouncementItem`
- `FeatureSwitches`
- `AppLimits`
- `ContactInfo`
- `Location`

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
// FeatureSwitches / AppLimits / ContactInfo 定义于 api_spec_types.md
interface ClientConfig {
  features: FeatureSwitches;
  limits: AppLimits;
  contact: ContactInfo;
  defaults: ClientDefaults;
}

// 客户端可见的默认值
// 由管理员在后台配置（见 api_spec_admin_core.md 中 ContentDefaults）
interface ClientDefaults {
  default_location: Location;        // 默认位置（必须满足 Location 约束），Location 定义于 api_spec_types.md
  default_user_avatar_url: string;   // 默认用户头像 URL（原 default_user_avatar，加 _url 后缀统一）
  default_group_avatar_url: string;  // 默认群组头像 URL（原 default_group_avatar）
}
```

**错误码**: `10000`

---

### POST /api/v1/system/bootstrap

冷启动聚合端点。一次请求拉齐客户端冷启动所需的全部基础数据，替代分别调用 `get_meta` + `get_client_config` + `users/get_account` + `notify/get_unread_count` 的多次往返。

- **认证**: 可选（未登录时 `profile`/`unread_summary` 省略，仅返回 `meta` + `client_config`）
- **幂等**: 无

**缓存契约（防惊群）**：响应携带 `bootstrap_version`（服务端在 config/version 变更时递增）。客户端可缓存响应并在后续冷启动时通过 `If-None-Match: <bootstrap_version>` 请求头询问；服务端无变化时返回精简响应（`code=0` + `data.bootstrap_version` 不变 + 其余字段省略，由客户端使用缓存）。`bootstrap_version` 的失效由服务端在 admin 修改 `AppConfig` 或发布新版本/公告时主动递增。

> **惊群防护（生产必需）**：`bootstrap_version` 变更时，所有客户端下次冷启动会同时 miss 缓存并全量重拉 bootstrap。若不加防护，单次 admin 配置变更会瞬间产生百万级全量请求击穿该端点。**强制要求**：
> - **客户端**：检测到 `bootstrap_version` 变更后，**不得立即**发起全量请求；须加随机 jitter 延迟（建议 0-300 秒均匀分布）后再刷新，将集中请求打散到数分钟窗口
> - **服务端**：`bootstrap_version`、`meta`、`client_config` 三个字段为**公共数据**，可按 `bootstrap_version` 做 CDN/反向代理缓存 key 并启用 stale-while-revalidate
> - **注意**：`profile` 和 `unread_summary` 为用户私有数据，**不可被 CDN 缓存**。已登录用户的 bootstrap 响应天然不可缓存——jitter 是唯一防线；CDN 缓存仅对匿名用户流量生效

**Request**: `BootstrapRequest`
```typescript
interface BootstrapRequest {
  client_type: ClientType;
  device_id?: DeviceId;        // 可选：携带设备标识（App 端通常已有，便于服务端记录设备绑定）
}
```

**Response**: `ApiResponse<BootstrapData>`
```typescript
interface BootstrapData {
  bootstrap_version: string;        // 缓存版本号，客户端下次冷启动透传给 If-None-Match
  meta: ApiVersionMeta;             // 同 get_meta 响应（ApiVersionMeta 定义于本文件 get_meta 段）
  client_config: ClientConfig;      // 同 get_client_config 响应

  // === @auth-only 以下字段仅登录态返回 ===
  profile?: AccountData;         // 同 users/get_account（AccountData 定义于 api_spec_user.md）
  unread_summary?: UnreadSummary;   // 未读摘要（聚合私聊/群聊/通知，替代多次 unread_count 调用）
}

// 未读摘要（聚合各域未读，冷启动时一次性渲染角标）
// 通知部分的字段与 notify/get_unread_count 的 UnreadCountResult 对齐
interface UnreadSummary {
  notification_total: number;               // 通知未读总数（同 notify/get_unread_count.total）
  notification_by_type: Record<NotifyCategory, number>;  // 按分类的未读数（同 notify/get_unread_count.by_type）
  private_chat_unread_count: number;        // 私聊会话未读消息总数
  group_chat_unread_count: number;          // 群聊未读消息总数
  total_badge: number;                      // 综合角标数（客户端展示用，服务端按业务规则聚合）
}
```

**说明**：
- 这是**聚合端点**，不引入新数据，仅合并现有端点的响应；各字段语义与原端点完全一致
- 登录态下若 `profile` 所在的 token 已过期，按"认证可选"语义降级为匿名（返回 `meta` + `client_config`，省略 `profile`/`unread_summary`），不返回 `10100`
- 客户端应优先调用此端点；各原子端点（`get_meta`/`get_client_config` 等）仍保留，供需要单项刷新的场景

**错误码**: `10000`
