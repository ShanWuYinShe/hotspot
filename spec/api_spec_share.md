---
alwaysApply: false
---
# HotSpot API - 分享系统

> 本文档定义 HotSpot 分享系统 (`share`) 相关的 RPC 接口。

## 模块信息

- **命名空间**: `share`
- **基础路径**: `/api/v1/share/`
- **引用**: 
  - 基础规范：`@spec/api_spec_base.md`
  - 通用类型：`@spec/api_spec_types.md`
  - 错误码：`@spec/api_spec_errors.md`

## 设计理念

- **统一承载**：用户/群组/动态/锚点位置均使用同一分享机制
- **一次创建，多端复用**：分享码可用于 Web/App 统一解析
- **最小信息暴露**：分享码仅暴露必要目标信息，敏感字段不出现在分享记录中

## 端点总览

| 端点 | 说明 | 认证 | 幂等 |
|-----|------|------|------|
| `POST /api/v1/share/create` | 创建分享 | 必须 | MUST |
| `POST /api/v1/share/get` | 获取分享详情 | 可选 | 无 |
| `POST /api/v1/share/me/list` | 我的分享列表 | 必须 | 无 |
| `POST /api/v1/share/revoke` | 撤销分享 | 必须 | 幂等语义 |

## 类型定义

以下类型定义于 `@spec/api_spec_types.md`：
- `UserInfo`
- `Location`
- `PaginationParams`

**说明**：`preview` 为展示用摘要。已撤销、已过期或目标不可访问时不返回 preview。

## 分享码规则

- **字符集**：`[A-HJ-NP-Z2-9]` 共 32 个字符（排除易混淆字符 0/O/1/I）
- **长度**：12 位
- **组合数**：32^12 ≈ 1.15 × 10^18 种
- **唯一性**：数据库唯一索引保证
- **大小写**：不区分大小写，服务端统一转换为大写
- **校验**：必须完全匹配字符集与长度，禁止空格、连字符、前后缀

**反滥用要求**：
- 对 `share/get` 启用多维度限流：建议 IP 级阈值为 10 次/分钟、用户级阈值为 30 次/分钟，超限返回 `10500` 并携带 `Retry-After`。
- 需跟踪连续失败次数（如 share_code 无效/过期）。当单个 IP 或用户在 5 分钟内连续失败 ≥5 次时，至少封禁 1 小时或要求验证码后再允许请求。
- 日志中记录所有失败的 share_code，以便离线分析潜在暴力破解行为。

## 有效期规则

- 分享可设置有效期（`expires_at`）
- 过期分享在 `get` 中返回 `10300`

## 端点详情

### POST /api/v1/share/create

创建分享。

- **认证**: 必须
- **幂等**: MUST（客户端必须携带 `Idempotency-Key`）

**Request**: `CreateShareRequest`
```typescript
// ShareType 定义于 @spec/api_spec_types.md

interface UserSharePayload {
  shared_user_id: string;
}

interface GroupSharePayload {
  shared_group_id: string;
}

interface MomentSharePayload {
  shared_moment_id: string;
}

interface LocationSharePayload {
  shared_location: Location;
}

type SharePayload =
  | UserSharePayload
  | GroupSharePayload
  | MomentSharePayload
  | LocationSharePayload;

interface CreateShareRequest {
  share_type: ShareType;
  payload: SharePayload;
  expires_in_seconds?: number; // 可选，默认由系统配置
}
```

**Response**: `ApiResponse<ShareCreated>`
```typescript
interface ShareCreated {
  share_id: string;
  share_code: string;
  expires_at?: string;
}
```

**业务规则**：
- 校验 payload 与 share_type 匹配
- 校验目标资源存在且可访问
- `location`：校验经纬度范围；`location_name` 约束覆盖为 **1-100 字符（必填）**，分享场景要求位置名称非空（全局默认 0-100，见 `@spec/api_spec_base.md` 字段校验约束）
- `expires_in_seconds` 若提供，必须为正整数且不超过系统允许上限（默认上限=30天，即 2_592_000 秒；超过上限或 <=0 返回 `10000` 参数错误）

**错误码**: `10000` `10300`

---

### POST /api/v1/share/get

获取分享详情。

- **认证**: 可选（公开访问）
- **幂等**: 无

**Request**: `GetShareRequest`
```typescript
interface GetShareRequest {
  share_code: string;         // 不区分大小写，服务端统一转换为大写
}
```

**Response**: `ApiResponse<ShareDetail>`
```typescript
interface ShareBase {
  share_id: string;
  share_code: string;
  share_type: ShareType;
  creator: UserInfo;
  created_at: string;
  expires_at?: string;
}

interface SharePreview {
  title: string;
  subtitle?: string;
  cover_url?: string;
}

interface ShareDetail {
  share: ShareBase;
  payload: SharePayload;
  preview?: SharePreview;
}
```

**错误码**: `10000` `10300`

---

### POST /api/v1/share/me/list

获取当前用户创建的分享列表。

- **认证**: 必须
- **幂等**: 无
- **排序**: 默认按照 `created_at DESC` 排序，时间相同以 `share_id DESC` 作为稳定 tie-breaker；一致性级别为强一致。

**Request**: `ShareListRequest`
```typescript
interface ShareListRequest {
  share_type?: ShareType;           // 按类型筛选
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<ShareListItem>>`
```typescript
interface ShareListItem {
  share_id: string;
  share_code: string;
  share_type: ShareType;
  created_at: string;
  expires_at?: string;
  is_revoked: boolean;
  preview?: SharePreview;           // 分享目标的摘要预览
}
```

**错误码**: `10000`

---

### POST /api/v1/share/revoke

撤销分享。撤销后分享码不可再用于 `get`。

- **认证**: 必须（仅分享创建者可撤销）
- **幂等**: 幂等语义（重复撤销返回一致结果）

**Request**: `RevokeShareRequest`
```typescript
interface RevokeShareRequest {
  share_id: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300`
