# HotSpot API - 分享系统

> 本文档定义 HotSpot 分享系统 (`share`) 相关的 RPC 接口。

## 模块信息

- **命名空间**: `share`
- **基础路径**: `/api/v1/share/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 错误码：`api_spec_errors.md`

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

以下类型定义于 `api_spec_types.md`：
- `UserInfo`
- `Location`
- `PaginationParams`

### 本模块类型

```typescript
// 分享目标类型（原定义于 api_spec_types.md，因仅 share 模块使用已下放至此）
type ShareType = 'user' | 'group' | 'moment' | 'location';
```

**说明**：`preview` 为展示用摘要。分享存在但目标资源不可访问时（如目标被删除/隐私变更），不返回 `preview`，仅返回 `share` 与 `payload`（让客户端知道分享类型与目标 ID，可提示"内容已不可访问"）；分享本身已撤销/过期时整体返回 `10300`（见「有效期规则」）。

## 分享码规则

- **字符集**：`[A-HJ-NP-Z2-9]` 共 32 个字符（排除易混淆字符 0/O/1/I）
- **长度**：12 位
- **组合数**：32^12 ≈ 1.15 × 10^18 种
- **唯一性**：数据库唯一索引保证
- **大小写**：不区分大小写，服务端统一转换为大写
- **校验**：必须完全匹配字符集与长度，禁止空格、连字符、前后缀

**反滥用要求（契约级）**：
- `share/get` 为公开端点，必须启用多维度限流（IP 维度 + 用户维度），具体阈值由服务端 `RateLimitConfig` 配置（见 `api_spec_admin_core.md`），超限返回 `10500` 并携带 `Retry-After`
- 对连续失败（share_code 无效/过期）的访问需有递进式惩罚策略（如临时封禁或要求验证码），具体阈值由服务端安全策略决定
- 失败请求应记录日志，用于离线分析潜在暴力破解行为
- 限流策略统一遵循 `api_spec_base.md`「限流策略（契约级）」，不在本模块重复硬编码阈值

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
// ShareType 定义于本模块「类型定义」小节

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
- 对于"资源存在但不可访问"场景，统一返回 `10300`（按资源不存在处理，避免泄漏隐私可见性）
- `location`：校验经纬度范围；当 `share_type='location'` 时，`location_name` 约束覆盖为 **1-100 字符（必填）**，要求位置名称非空（全局默认 0-100，见 `api_spec_base.md` 字段校验约束）
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
  preview?: SharePreview;   // 目标资源不可访问时不返回（见本模块说明）；分享撤销/过期整体 10300
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
  share_id?: string;    // 分享 ID（来自 share/create 或 share/me/list）
  share_code?: string;  // 或分享码（创建者可能仅保留了复制出的链接码）
}
// 二选一：必须且只能传 share_id 或 share_code 其中之一；同时传或都不传返回 10000
```

**Response**: `ApiResponse`

**错误码**: `10000` `10300`

> 隐私一致性：非创建者撤销（share 存在但无权限）统一返回 `10300`，与本模块「资源存在但不可访问按不存在处理」规则一致（见上方隐私规则），不通过 `10200` 泄露 share 的存在性。
> 设计说明：revoke 同时支持 share_id 与 share_code，消除"创建者只保留了链接码却无法撤销"的寻址不对称。
