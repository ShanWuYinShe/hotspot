# HotSpot API - 系统与全局配置

> 本文档定义 `/api/v1/system/` 的登录态冷启动聚合端点。
> 系统公开信息（meta / health / announcements / client_config / 匿名 bootstrap）位于 `/public/v1/system/`，定义于 `api_spec_public.md`；其共享类型（`ApiVersionMeta`/`ClientConfig`/`HealthStatus` 等）定义于 `api_spec_types.md`。

## 模块信息

- **命名空间**: `system`
- **基础路径**: `/api/v1/system/`
- **引用**:
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 守卫与权限：`api_spec_authz.md`
  - 错误码：`api_spec_errors.md`
  - 匿名版 bootstrap：`api_spec_public.md`

## 端点总览

| 端点 | 说明 | 节点 |
|-----|------|------|
| `POST /api/v1/system/bootstrap` | 登录态冷启动聚合（config + meta + profile + unread） | /api/v1 |
| `POST /public/v1/system/meta` 等 | 系统公开信息（匿名版），见 `api_spec_public.md` | /public/v1 |

## 端点详情

### POST /api/v1/system/bootstrap

登录态冷启动聚合端点。一次请求拉齐客户端冷启动所需的全部基础数据，替代分别调用 `/public/v1/system/meta` + `/public/v1/system/client_config` + `users/account` + `notify/unread_summary` 的多次往返。

- **节点**: /api/v1（双版本端点：匿名版 `/public/v1/system/bootstrap` 见 `api_spec_public.md`；必须有效 Token）

**缓存契约（防惊群）**：响应携带 `bootstrap_version`（服务端在 config/version 变更时递增）。客户端可缓存响应并在后续冷启动时通过 `If-None-Match: <bootstrap_version>` 请求头询问；服务端无变化时返回精简响应（`code=0` + `data.bootstrap_version` 不变 + **仅 `meta`/`client_config` 省略**，由客户端使用缓存；`profile`/`unread_summary` 为用户私有数据、无缓存语义，命中时仍恒返回最新值——`bootstrap_version` 不因用户资料/未读数变化而递增）。`bootstrap_version` 的失效由服务端在平台管理员修改 `AppConfig` 或发布新版本/公告时主动递增。

> **防惊群**：`bootstrap_version` 变更会让所有客户端同时 miss 缓存并全量重拉，客户端应加随机 jitter 延迟后刷新，避免集中请求。`meta`/`client_config` 为公共数据可做 CDN 缓存；`profile`/`unread_summary` 为用户私有数据，不可缓存。

**Request**: `BootstrapRequest`
```typescript
interface BootstrapRequest {
  client_type: ClientType;
  device_id: DeviceId | null;        // 可选：携带设备标识（App 端通常已有，便于服务端记录设备绑定）
}
```

**Response**: `ApiResponse<BootstrapData>`
```typescript
interface BootstrapData {
  bootstrap_version: string;        // 缓存版本号，客户端下次冷启动透传给 If-None-Match
  // === 缓存命中时以下键整体省略（客户端使用缓存）；未命中/无缓存时必填 ===
  // 命中判定：键不存在（而非 === null）。本接口是 base「类型稳定性」
  // 「字段缺失仅限内容性可选字段」通则的显式例外。
  meta?: ApiVersionMeta;                   // 同 /public/v1/system/meta 响应
  client_config?: ClientConfig;            // 同 /public/v1/system/client_config 响应

  // === 登录态字段（未命中时必填且恒非 null；命中时同样恒返回，不省略） ===
  profile?: AccountData;                   // 同 users/account（AccountData 定义于 api_spec_user.md）
  unread_summary?: UnreadSummary;          // 未读摘要（聚合私聊/群聊/通知）
}

// 未读摘要（聚合各域未读，冷启动时一次性渲染角标）
// 通知部分的字段与 notify/unread_summary 的 UnreadSummaryResult 对齐（定义见 api_spec_notify.md）
interface UnreadSummary {
  notify_total: number;               // 通知未读总数（同 notify/unread_summary.total）
  notify_by_type: Record<NotifyCategory, number>;  // 按分类的未读数（同 notify/unread_summary.by_type）
  private_chat_unread_count: number;  // 私聊会话未读消息总数
  group_chat_unread_count: number;    // 群聊会话未读消息总数
  total_badge: number;                // 综合角标数 = notify_total + private_chat_unread_count + group_chat_unread_count
}
```

**说明**：
- 这是**聚合端点**，不引入新数据，仅合并现有端点的响应；各字段语义与原端点完全一致
- `total_badge` 语义说明（F10）：群聊 @（group_mentioned）等通知会同时计入 `notify_total` 与该群的 `group_chat_unread_count`，故 `total_badge` 是「各域角标之和」而非「互斥未读条数」，允许同一内容在两域各计一次；客户端仅用于角标展示，不做精确对账
- 客户端应优先调用此端点；各原子端点（`meta`/`client_config` 等）仍保留于 `/public/v1/system/`，供需要单项刷新的场景
