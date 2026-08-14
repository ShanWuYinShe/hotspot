# HotSpot API - 认证

> 本文档定义 HotSpot 认证相关的 RPC 接口。

## 模块信息

- **命名空间**: `auth`
- **基础路径**: `/api/v1/auth/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 守卫与权限：`api_spec_authz.md`
  - 错误码：`api_spec_errors.md`

## 端点总览

| 端点 | 说明 | 认证 |
|------|------|------|
| `POST /api/v1/auth/register` | 用户注册 | public |
| `POST /api/v1/auth/login` | 用户登录 | public |
| `POST /api/v1/auth/logout` | 用户登出 | required |
| `POST /api/v1/auth/guest` | 创建访客用户 | public |
| `POST /api/v1/auth/refresh_token` | 刷新 Token | credential |
| `POST /api/v1/auth/get_status` | 检查认证状态 | optional |
| `POST /api/v1/auth/request_password_reset` | 请求密码重置 | public |
| `POST /api/v1/auth/reset_password` | 重置密码 | public |
| `POST /api/v1/auth/change_password` | 修改密码 | required |
| `POST /api/v1/auth/list_sessions` | 获取会话列表 | required |
| `POST /api/v1/auth/revoke_session` | 撤销会话 | required |
| `POST /api/v1/auth/revoke_all_sessions` | 撤销所有其他会话 | required |

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `ClientType`
- `DeviceId`（设备标识，见下方「单设备单会话模型」）

### 单设备单会话模型（契约级）

HotSpot 采用「**同一用户在同一设备上只有一个活跃 session**」的会话模型。`device_id`（`DeviceId` 类型）是设备身份的主键，与 `user_id` 一起构成 session 的唯一性约束 `(user_id, device_id)`。

**设备标识的稳定性**：
- **App**：客户端首次启动生成 UUID v4 并持久化到 Keychain（iOS）/ Keystore（Android）；卸载重装视为新设备（新 device_id）
- **Web**：由服务端通过独立 `device_id` Cookie 下发（见 `api_spec_base.md`「Web 端 Cookie 契约」），登出不清除；不同浏览器档案/无痕窗口天然对应不同 device_id
- 客户端**不得**自造或预测 device_id（App 端虽本地生成，但一经生成即稳定，禁止为绕过驱逐而轮换）

**驱逐规则**：
- `login` / `register` / `guest` / `reset_password`（自动登录）成功时，若 `(user_id, device_id)` 已有活跃 session，服务端立即将旧 session 的 access_token + refresh_token 写入黑名单，并向旧 session 的在线 WebSocket 连接推送 `SessionRevoked` 事件（见 `api_spec_websocket.md`），新 session 取而代之
- 驱逐写入黑名单**立即生效**，refresh 的 30 秒宽限期**不覆盖显式驱逐**（见 `refresh_token` 的「驱逐优先于宽限」）

**跨设备共存**：用户可在多个设备（手机、平板、Web）同时登录，各自 `device_id` 不同、互不驱逐。`list_sessions` 返回所有活跃设备的 session 列表。

**与既有失效机制的协同**：
- `change_password`：撤销除当前外所有 session（跨设备）——强度高于单设备驱逐，用于密码泄露场景
- `reset_password`：撤销所有 session（含当前）——最强，用于忘密码重置
- `logout`：仅撤销当前 session
- `revoke_session` / `revoke_all_sessions`：用户主动管理多设备

### 客户端类型与 Token 投递策略

> Cookie 名称与属性的完整定义见 `api_spec_base.md` 的 "Web 端 Cookie 契约" 章节。

```typescript
// 认证响应负载：按 client_type 拆分
// Web：仅通过 HttpOnly Cookie 传递 token，响应体不包含任何 token 字段
// App：access_token/refresh_token 出现在 body，客户端负责安全存储
// 
// JWT 设计原则：
// - JWT (access_token) 中只存储不可变信息：user_id
// - 可变信息（nickname、头像等）应通过 /api/v1/users/get_account 获取
// - Token 失效通过黑名单机制实现（登出/撤销时写入黑名单，直至过期）

// 判别字段：client_type 标明投递形态，客户端按类型分支解析（app 取 token 字段，web 取 Cookie）
interface AuthTokenPairApp {
  client_type: 'app';
  user_id: string;
  access_token: string;
  refresh_token: string;
  access_token_expires_at: string;
  refresh_token_expires_at: string;
}

interface AuthTokenPairWeb {
  client_type: 'web'; // 仅判别用；Token 通过 HttpOnly Cookie 传递，响应体不含任何 token 字段
}

type AuthTokenPair = AuthTokenPairApp | AuthTokenPairWeb;
```

#### Web 端（`client_type='web'`）

Token 通过服务端 `Set-Cookie` 响应头投递，**禁止**使用 `Authorization` Header。

```http
Set-Cookie: access_token=<token>; HttpOnly; Secure; SameSite=Strict; Path=/; Max-Age=<ttl>
Set-Cookie: refresh_token=<token>; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth/refresh_token; Max-Age=<ttl>
```

**响应体**：返回 `ApiResponse<AuthTokenPairWeb>`，`data` 仅含判别字段 `client_type: 'web'`（Token 仅在 Cookie 中传递，响应体不包含任何 token 值）。

#### App 端（`client_type='app'`）

Token 通过响应体投递：返回 `ApiResponse<AuthTokenPairApp>`，响应体中包含 `access_token`、`refresh_token` 及各自过期时间；客户端负责安全存储（Keychain / Keystore）。

#### Cookie 生命周期（Web 端）

| 场景 | `access_token` Cookie | `refresh_token` Cookie |
|---|---|---|
| 登录 / 注册 / 访客 | `Set-Cookie` 设置 | `Set-Cookie` 设置 |
| 刷新 Token | `Set-Cookie` 更新 | `Set-Cookie` 更新 |
| 登出 | `Max-Age=0` 清除 | `Max-Age=0` 清除 |
| 普通 API 请求 | 自动携带（`Path=/`） | **不**携带（`Path` 不匹配） |
| 调用 refresh_token API | 自动携带 | 自动携带（`Path` 匹配） |

## 端点详情

### POST /api/v1/auth/register

用户注册。

- **认证**: public
- **幂等**: 是（客户端必须携带 `Idempotency-Key`）；`auth/register` 是全规范唯一保留幂等键的「有唯一约束」接口，理由见 `api_spec_base.md`「唯一约束特例」

**Request**: `RegisterRequest`
```typescript
interface RegisterRequest {
  login_id: string;             // 手机号/邮箱/用户名（注册后作为登录凭证）
  nickname: string;
  password: string;
  client_type: ClientType;
  device_id: DeviceId;          // App 端必填；Web 端服务端从 device_id Cookie 读取
}
```

**Response**: `ApiResponse<AuthTokenPair>`

**错误码**: `10400`（login_id 已被占用）

### POST /api/v1/auth/login

用户登录（密码方式）。

- **认证**: public

**Request**: `LoginRequest`
```typescript
interface LoginRequest {
  login_id: string;            // 手机号/邮箱/用户名
  password: string;            // 密码
  client_type: ClientType;
  device_id: DeviceId;         // App 端必填；Web 端服务端从 device_id Cookie 读取
}
```

**Response**: `ApiResponse<AuthTokenPair>`

**单设备单会话驱逐**：登录成功时，服务端查找 `(user_id, device_id)` 已有的活跃 session，若存在则立即将其 access_token + refresh_token 写入黑名单（沿用 `change_password` 机制），并向被驱逐设备的在线 WebSocket 连接推送 `SessionRevoked` 事件（见 `api_spec_websocket.md`）。新 session 取代旧 session。

**安全说明**：用户不存在和密码错误统一返回 `10100`（认证失败），不区分具体原因，防止账号枚举攻击。

**错误码**: `10100`

---

### POST /api/v1/auth/logout

用户登出。

- **认证**: required

**失效规则**：登出会将当前 access_token 与 refresh_token 写入黑名单，直至过期。

**Request**: `LogoutRequest`
```typescript
interface LogoutRequest {}
```

**Response**: `ApiResponse`

**Web 端行为**：
```http
Set-Cookie: access_token=; HttpOnly; Secure; SameSite=Strict; Path=/; Max-Age=0
Set-Cookie: refresh_token=; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth/refresh_token; Max-Age=0
```
- 通过 `Max-Age=0` 立即清除 access_token 和 refresh_token Cookie；前端无需手动清除 Cookie

**App 端行为**：客户端需要自行清除存储的 access_token 和 refresh_token。

**WebSocket 处理**：
- 服务端在处理 logout 时会主动断开该用户当前会话关联的 WebSocket 连接
- 客户端也可以在调用 logout 之前先主动断开 WebSocket
- 无论哪种顺序，服务端都会确保 WebSocket 连接被清理

---

### POST /api/v1/auth/guest

创建访客用户。

- **认证**: public
- **幂等**: 是（客户端必须携带 `Idempotency-Key`）

**Request**: `GuestRequest`
```typescript
interface GuestRequest {
  nickname: string;
  client_type: ClientType;
  device_id: DeviceId;   // App 端必填；Web 端服务端从 device_id Cookie 读取
}
```

**Response**: `ApiResponse<AuthTokenPair>`

---

### POST /api/v1/auth/refresh_token

刷新 Token。

- **认证**: credential（仅限本端点：以 refresh_token 为凭证，无效/过期/被撤销返回 `10100`，不适用 optional 端点的匿名降级规则）

**刷新规则**：
- 刷新成功后：access_token 和 refresh_token 都会更新
- 此 API 允许在 access_token 过期或缺失时调用

**并发刷新策略（宽限期机制）**：
- 旧 refresh_token 在被轮换后，仍有 **30 秒宽限期**可用
- 宽限期内使用旧 refresh_token 刷新，服务端返回**与首次刷新相同的新 token 对**（不生成新的）
- 宽限期过后，旧 refresh_token 彻底失效
- 设计目的：解决多标签页/多并发请求同时触发刷新的竞态问题
- 客户端侧建议：使用刷新锁（mutex），同一时刻仅允许一个刷新请求执行

**驱逐优先于宽限（契约级）**：宽限期仅容忍"并发刷新竞态"。若旧 session 是被「单设备单会话驱逐」显式写入黑名单的（如同设备重新登录），则黑名单立即生效，**30 秒宽限不覆盖显式驱逐**——旧 session 的任何请求（含宽限期内 refresh）立即失败为 `10100`。

**client_type 约束**：
- `client_type='web'`：请求体不得包含 `refresh_token` 字段（若包含则请求失败）
- `client_type='app'`：请求体必须包含非空 `refresh_token`（缺少或为空则请求失败）

**Request**: `RefreshTokenRequest`
```typescript
interface RefreshTokenRequest {
  client_type: ClientType;
  device_id: DeviceId;     // 用于校验 refresh_token 归属当前设备，防止跨设备 token 挪用（Web 端服务端从 device_id Cookie 读取）
  refresh_token: string | null;  // App 端必填；Web 端为 null（使用 Cookie）
}
```

**Response**: `ApiResponse<AuthTokenPair>`

**错误码**: `10100`

---

### POST /api/v1/auth/get_status

检查认证状态。

- **认证**: optional

**说明**：Token 可选；未登录或携带无效/过期 Token 时返回 `is_valid: false`（不会触发 `10100`），仅当 Token 验证通过时返回 `is_valid: true` 并附 `expires_at`。

**Request**: `GetStatusRequest`
```typescript
interface GetStatusRequest {}
```

**Response**: `ApiResponse<AuthStatus>`
```typescript
interface AuthStatus {
  is_valid: boolean;
  user_id: string | null;    // 仅当 is_valid === true 时返回，客户端可直接获得身份（App 冷启动校验免二次调用 get_account）
  expires_at: string | null; // 仅当 is_valid === true 时返回，对应 access_token 的过期时间
}
```

---

### POST /api/v1/auth/request_password_reset

请求密码重置（发送验证码）。

- **认证**: public

**安全说明**：
- 无论 `login_id` 是否存在，均返回成功响应，不暴露账号是否注册
- 验证码发送频率由 `AppConfig.rate_limits.auth_reset_per_10minutes` 控制；验证码有效期由服务端内部配置决定（不对外暴露）

**Request**: `RequestPasswordResetRequest`
```typescript
interface RequestPasswordResetRequest {
  login_id: string;       // 注册时使用的邮箱或手机号
  client_type: ClientType;
}
```

**Response**: `ApiResponse`

---

### POST /api/v1/auth/reset_password

验证验证码并重置密码。重置成功后自动登录并返回 Token。

- **认证**: public

**自然去重**：验证码为一次性凭证，首次成功后即失效——重发同一请求返回 `10300`（验证码已用/过期），不会重复改密。无需 `Idempotency-Key`（判定见 `api_spec_base.md`「自然去重键」）。

**安全说明**：
- 重置密码成功后，服务端自动吊销该账号所有现有会话（与 `change_password` 一致）
- 所有被吊销会话的 token 写入黑名单
- 被吊销设备的 WebSocket 连接会被服务端主动断开
- 重置成功后自动登录：按 `(user_id, device_id)` 建立新会话；若该设备已有活跃会话则立即驱逐（与 `login` 的驱逐规则一致）

**Request**: `ResetPasswordRequest`
```typescript
interface ResetPasswordRequest {
  login_id: string;          // 注册时使用的手机号/邮箱
  verification_code: string;
  new_password: string;
  client_type: ClientType;
  device_id: DeviceId;       // App 端必填；Web 端服务端从 device_id Cookie 读取
}
```

**Response**: `ApiResponse<AuthTokenPair>`

**错误码说明**：
- `10300` 用于"重置请求不存在或已过期"（验证码已用/过期）与"login_id 不存在"——两者 message 一致，禁止返回"用户不存在"等暴露账号存在性的 message（与 `request_password_reset` 的安全策略一致，防账号枚举）

**错误码**: `10300`

---

### POST /api/v1/auth/change_password

修改密码（需要验证旧密码）。

- **认证**: required

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

### POST /api/v1/auth/list_sessions

获取活跃会话列表。

- **认证**: required

**Request**: `ListSessionsRequest`
```typescript
interface ListSessionsRequest {} // 直接返回全部活跃会话，无需分页
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

- **认证**: required

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

### POST /api/v1/auth/revoke_all_sessions

撤销当前会话之外的所有其他活跃会话。适用于账号被盗、密码泄露等紧急安全场景。

- **认证**: required

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