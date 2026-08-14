# HotSpot API - 认证

> 本文档定义注册用户会话自助管理相关的 RPC 接口（`/api/v1/auth/`）。
> 账号生命周期端点（注册/登录/访客/刷新 Token/密码重置）位于 `/public/v1/auth/`，定义在 `api_spec_public.md`。

## 模块信息

- **命名空间**: `auth`
- **基础路径**: `/api/v1/auth/`（节点：`/api/v1`，必须有效 Token）
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 守卫与权限：`api_spec_authz.md`
  - 错误码：`api_spec_errors.md`
  - 账号生命周期端点与 Token 投递策略：`api_spec_public.md`

## 端点总览

| 端点 | 说明 | 节点 |
|------|------|------|
| `POST /api/v1/auth/signout` | 用户登出 | /api/v1 |
| `POST /api/v1/auth/session` | 查询当前会话状态 | /api/v1 |
| `POST /api/v1/auth/change_password` | 修改密码 | /api/v1 |
| `POST /api/v1/auth/fetch_sessions` | 获取会话列表 | /api/v1 |
| `POST /api/v1/auth/revoke_session` | 撤销会话 | /api/v1 |
| `POST /api/v1/auth/revoke_other_sessions` | 撤销所有其他会话 | /api/v1 |
| `POST /public/v1/auth/register` 等 6 个 | 账号生命周期（注册/登录/访客/刷新/密码重置），见 `api_spec_public.md` | /public/v1 |

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `ClientType`
- `DeviceId`（设备标识，见 `api_spec_public.md`「单设备单会话模型」）

Token 投递策略（`AuthTokenPair`、Web Cookie / App Header）与单设备单会话模型的权威定义在 `api_spec_public.md`（token 的产生端点全部位于 `/public/v1`）。

## 端点详情

### POST /api/v1/auth/signout

用户登出。

- **节点**: /api/v1（必须有效 Token）

**失效规则**：登出会将当前 access_token 与 refresh_token 写入黑名单，直至过期。

**Request**: `SignoutRequest`
```typescript
interface SignoutRequest {}
```

**Response**: `ApiResponse`

**Web 端行为**：
```http
Set-Cookie: access_token=; HttpOnly; Secure; SameSite=Strict; Path=/; Max-Age=0
Set-Cookie: refresh_token=; HttpOnly; Secure; SameSite=Strict; Path=/public/v1/auth/refresh_token; Max-Age=0
```
- 通过 `Max-Age=0` 立即清除 access_token 和 refresh_token Cookie；前端无需手动清除 Cookie

**App 端行为**：客户端需要自行清除存储的 access_token 和 refresh_token。

**WebSocket 处理**：
- 服务端在处理 logout 时会主动断开该用户当前会话关联的 WebSocket 连接
- 客户端也可以在调用 logout 之前先主动断开 WebSocket
- 无论哪种顺序，服务端都会确保 WebSocket 连接被清理

---

### POST /api/v1/auth/session

查询当前会话状态。

- **节点**: /api/v1（必须有效 Token；未登录/无效 Token 返回 `10100`——客户端可据此直接判定匿名态，无需单独的匿名探测端点）

**Request**: `SessionRequest`
```typescript
interface SessionRequest {}
```

**Response**: `ApiResponse<AuthStatus>`
```typescript
interface AuthStatus {
  user_id: string;    // 当前用户身份（App 冷启动校验免二次调用 account）
  expires_at: string; // 当前 access_token 的过期时间
}
```

---

### POST /api/v1/auth/change_password

修改密码（需要验证旧密码）。

- **节点**: /api/v1

**自然去重**：首次成功后旧密码立即失效，重发同一请求（仍带旧密码）返回 `10401`（旧密码错误），不会重复改密。无需 `Idempotency-Key`。

**Request**: `ChangePasswordRequest`
```typescript
interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}
```

**Response**: `ApiResponse`

**安全说明**：
- 修改密码成功后，服务端自动吊销当前会话之外的所有其他活跃会话
- 所有被吊销会话的 token 写入黑名单
- 被吊销设备的 WebSocket 连接收到 `SessionRevoked`（`reason='password_changed'`）事件后被服务端主动断开（见 `api_spec_websocket.md`）

**错误码**: `10401`（旧密码错误）

---

### POST /api/v1/auth/fetch_sessions

获取活跃会话列表。

- **节点**: /api/v1

**Request**: `FetchSessionsRequest`
```typescript
interface FetchSessionsRequest {} // 直接返回全部活跃会话，无需分页
```

**Response**: `ApiResponse<SessionList>`
```typescript
interface SessionList {
  items: SessionItem[];
}

interface SessionItem {
  session_id: string;           // 会话标识，用于展示和撤销操作
  device_id: DeviceId;          // 设备标识（会话主键的一部分）
  device_type: string | null;   // 设备类型展示（如 "iOS" / "Android" / "Web"），来自 client_type 与 UA 解析
  device_name: string | null;   // 设备名称展示（如 "iPhone 15 Pro" / "Chrome on macOS"），来自 UA 解析
  ip_address: string | null;    // 脱敏显示（如 "192.168.xxx.xxx"），防止被用于跟踪
  ip_location: string | null;   // IP 地理位置（如 "北京市"）
  created_at: string;
  last_active_at: string;
  is_current: boolean;          // 判定：session.session_id === 当前请求所属会话的 session_id
}
```

**说明**：
- 单设备单会话模型下，同一 `device_id` 最多一个活跃 session；活跃会话数 = 用户活跃设备数，无需分页
- 排序：当前会话置顶，其余按 `last_active_at` 倒序

---

### POST /api/v1/auth/revoke_session

撤销会话（设备下线）。

- **节点**: /api/v1

**安全说明**：
- 撤销操作会将对应会话的 token 写入黑名单，直至过期
- 不允许撤销当前会话（使用 logout 代替），返回 `10400`
- 被撤销会话关联的在线 WebSocket 连接会收到 `SessionRevoked` 事件后断开

**Request**: `RevokeSessionRequest`
```typescript
interface RevokeSessionRequest {
  session_id: string;  // 来自 SessionItem.session_id
}
```

**Response**: `ApiResponse`

**错误码**: `10300`（会话不存在） `10400`（不能撤销当前会话）

---

### POST /api/v1/auth/revoke_other_sessions

撤销当前会话之外的所有其他活跃会话。适用于账号被盗、密码泄露等紧急安全场景。

- **节点**: /api/v1

**安全说明**：
- 当前会话（发起请求的会话）不会被撤销
- 所有被撤销会话的 token 都会写入黑名单
- 操作不可逆，被撤销的设备需要重新登录；每个被撤销会话的在线连接收到 `SessionRevoked` 后断开

**Request**: `RevokeAllSessionsRequest`
```typescript
interface RevokeAllSessionsRequest {}
```

**Response**: `ApiResponse<RevokeAllResult>`
```typescript
interface RevokeAllResult {
  revoked_count: number;  // 被撤销的会话数量
}
```