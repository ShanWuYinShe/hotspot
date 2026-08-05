# HotSpot API - 基础原则与约束

> 定义 HotSpot API 设计的基础原则、命名规范、全局约束、Token 语义、分页与排序、幂等性、安全性实践等核心内容。

## 权威边界与冲突处理

- 若本文件与任何工程实现或其它 doc/rule 文件产生冲突，以本文件为准
- 禁止使用数字章节号（如"第X章/第X节"）来引用本文件内容；引用时仅使用小节标题或锚点标题

## 术语约定（契约级）

- **当前用户**：指已鉴权的调用方用户。个人视角操作由 Token 隐含标识，无独立 `/me/` 路径段
- **自身**：指"对自己执行操作"的语义，用于错误码与规则描述
- **群成员**：指已加入群组的用户
- **会话参与者**：指私聊会话的双方用户

重要契约约定（必须）：
- 所有时间字段使用 UTC，格式为 ISO 8601（如 `2024-01-15T08:30:00.000Z`）。毫秒部分可选：服务端可返回带或不带毫秒的格式，客户端必须兼容两种
- 客户端负责时区转换：接收到 UTC 时间后按用户本地时区展示
- 客户端发送时间字段也必须使用 UTC ISO 8601（禁止上传本地时区字符串）

## 全局约束（必须/禁止）

本小节声明跨端必须一致遵守的契约级约束。

必须：
- 所有 HTTP 响应必须使用统一 envelope：`ApiResponse<T>`
- HTTP 状态码非 200 视为请求失败（网络/网关/服务器错误等），不进入业务层处理
- 仅当 HTTP 状态码为 200 时才解析 `ApiResponse`；业务成功与否只取决于 `ApiResponse.code`
- HotSpot 对外 HTTP API 仅允许使用 POST；对象存储直传的 PUT/POST 仅发生在存储服务域名上，不属于 HotSpot API 契约
- `cursor` 为不透明字符串：客户端只透传；服务端若需解析/编码，必须由服务端负责
- `distance/bearing_degrees` 等地理计算结果必须由服务端返回，客户端禁止本地计算用于排序
- 任意字段的可空性必须与类型定义一致
- 所有 ID 字段必须使用显式 `xxx_id` 命名（禁止裸 `id`）
- 所有 `xxx_id` 字段在 API 层统一为 `string` 类型（前端视为不透明字符串，不关心内部是数字还是 UUID）
- 所有 `*_url` 字段（如 `avatar_url`、`thumbnail_url`、`upload_url` 等）的值必须为**完整的绝对 URL**（含协议和域名，如 `https://cdn.example.com/path/to/image.jpg`），客户端无需拼接 base URL
- 幂等性：仅**重发会产生重复结果且无自然去重兜底**的写操作必须使用 `Idempotency-Key`；其余写操作（含状态设值、删除/解散/更新及有自然去重键的创建）无需该键（判定见「幂等性（契约级）」）。

禁止：
- 禁止在客户端生成或拼装 `cursor`
- 禁止返回本规范（`api_spec_base` / `api_spec_types` / 各模块 `api_spec_*`）中未定义的字段
- 禁止在任何文档中硬编码与本规范不一致的端点/字段名
- 禁止以"不专业/不规范"为由修改规范中刻意为之的**无害**玩梗/彩蛋命名，存疑时应向维护者确认；但冒犯性或针对特定群体的内容不属于保护范围

---

## 统一响应格式

```typescript
interface ApiResponse<T = void> {
  code: number;       // 0: Success, != 0: Error
  message?: string;   // code != 0 时必填（详见 api_spec_errors.md），code == 0 时可选
  data?: T;           // code == 0 且 T != void 时必填；code != 0 时必须省略（除非端点显式声明错误时 data 的结构）
}
```

**HTTP 语义**：HTTP 非 200 视为请求失败（未进入业务层）；仅 200 时返回 `ApiResponse`。

### 空请求体约定（契约级）

- 当端点的 Request 标注为 `{}`，或 Request 类型定义为空对象（如 `interface XxxRequest {}`）时，请求体必须为 JSON 对象。
- 缺失请求体或非对象请求体（如数组、字符串、数字、布尔）时返回 `10000`（参数错误）。
- **忽略未知字段**：空对象端点对请求体中出现的未知字段**静默忽略**（不返回错误）。原因是空对象端点无业务字段可校验，严格拒绝未知键会把客户端 SDK 注入的遥测字段、网关附加字段等无害变动变成硬失败，无安全或正确性收益。严格未知键拒绝仅用于**有类型的请求体**（字段语义需要校验时）。

### 通用隐含错误码

以下错误码为**所有端点隐含支持**，各模块端点文档中**无需重复列出**，仅列出该端点特有的业务错误码：

| 错误码 | 适用范围 |
|-------|---------|
| `10000` | 所有端点（参数错误） |
| `10100` | 所有"认证: 必须"的端点 |
| `10201` | 所有"认证: 必须"的端点（中间件级别检查，账号被封禁时任何已认证请求均可能返回） |
| `10500` | 所有端点 |
| `10600` | 所有端点 |

各错误码的类型、客户端行为及 message 示例详见 `api_spec_errors.md`。

因此，端点文档中的"**错误码**"字段仅需列出该端点**特有的业务错误码**（如 `10200`、`10300`、`10400` 等）；隐含错误码（`10000`、`10100`、`10201`、`10500`、`10600`）与幂等错误码（`22000`、`22001`、`22002`）无需重复列出。

> **权威清单**：哪些错误码属于"隐含/中间件级"（所有端点自动适用、模块文档禁止重复列出）的权威定义在 `api_spec_errors.md` 的「隐含错误码（无需在端点重复列出）」小节。模块文档的"错误码"字段只列该端点**特有**的业务错误码；若某端点的错误码小节出现了隐含码，视为文档缺陷，应删除。

## 请求与响应头（契约级）

- `X-Request-Id`：请求链路追踪 ID。客户端**可选**传入（用于跨端追踪关联）；服务端**始终**在响应头返回该字段——若请求携带则原样回显，未携带则由服务端生成。服务端是 `X-Request-Id` 的权威来源。
- 服务端**不应**对客户端传入的 `X-Request-Id` 施加格式或熵值约束（如"必须不可推测"）——这是无法验证且无法强制的要求。客户端值仅用于追踪关联，服务端始终返回一个有效值即可。
- `Idempotency-Key`：仅用于需要幂等性的写操作，具体要求见"幂等性（契约级）"

说明：`X-Request-Id` 与 `Idempotency-Key` 相互独立；同一 `Idempotency-Key` 的多次请求可返回不同的 `X-Request-Id`。

### 响应头安全约束

服务端**禁止**返回以下可能泄露技术栈或部署架构的响应头：

| 禁止的 Header | 泄露风险 |
|---------------|---------|
| `Server` | 暴露 Web 服务器类型和版本（如 `nginx/1.25.3`） |
| `X-Powered-By` | 暴露后端框架信息（如 `Express`、`Axum`） |
| `X-AspNet-Version` / `X-Runtime` 等 | 暴露运行时或框架版本 |

**实现要求**：在反向代理层和应用层均移除或覆盖上述响应头。

### 限流响应头

服务端**仅在触发限流时**返回 `Retry-After` 响应头：

| Header | 说明 |
|--------|------|
| `Retry-After` | 触发限流时返回，单位秒，表示客户端应等待的最短时间。根据实际限流窗口动态计算 |

**安全说明**：不暴露 `X-RateLimit-Limit` / `X-RateLimit-Remaining` / `X-RateLimit-Reset` 等精确限流状态。对于面向终端用户的社交应用，暴露精确额度会帮助爬虫/攻击者优化请求频率以规避限流

---

## 路径设计与 Token 语义（契约级）

本规范采用**统一路径 + Token 决定响应**的设计模式，简化 API 数量的同时保持灵活性。

### 统一路径结构

所有 API 统一使用 `/api/v1/` 前缀：

```
/api/v1/
├── auth/              # 认证（登录/注册/刷新 token）
├── users/             # 用户相关（搜索/资料/关系）
├── moments/           # 瞬间相关（搜索/CRUD/互动）
├── comments/          # 评论相关
├── private_chats/     # 私聊管理
├── private_messages/  # 私聊消息
├── groups/            # 群组相关（搜索/管理/成员/公告）
├── group_messages/    # 群聊消息
├── feed/              # 信息流（推荐/热门）
├── map/               # 地图渲染
├── search/            # 搜索与发现（综合建议）
├── reports/           # 举报
├── notify/            # 通知中心
├── media/             # 媒体上传
├── share/             # 分享系统
├── system/            # 系统（健康/版本/配置/公告）
└── admin/             # 管理员
```

### Token 处理规则

#### Web 端 Cookie 契约

Web 端使用以下**固定名称**的 HttpOnly Cookie 携带认证信息（**禁止**使用 `Authorization` Header）。此外使用一个独立的设备标识 Cookie（非鉴权凭证）：

| Cookie 名称 | 用途 | Path | 属性 | Max-Age |
|---|---|---|---|---|
| `access_token` | API 访问令牌 | `/` | `HttpOnly; Secure; SameSite=Strict` | 服务端动态设置（与 token 过期时间一致） |
| `refresh_token` | Token 刷新令牌 | `/api/v1/auth/refresh_token` | `HttpOnly; Secure; SameSite=Strict` | 服务端动态设置（与 token 过期时间一致） |
| `device_id` | 设备标识（非鉴权凭证，见下「Web 端 device_id Cookie」） | `/` | `Secure; SameSite=Lax` | 10 年（长期，独立于登录态） |

**规则**：
- `access_token` 和 `refresh_token` 是**固定的 Cookie 名称**，前端 / 后端必须一致使用，禁止使用其他名称（如 `token`、`jwt`、`session`）
- `access_token` 使用 `Path=/` 以统一覆盖 HTTP API（`/api/v1/`）和 WebSocket（`/ws/v1`）端点
- `refresh_token` 使用精确 Path 确保仅在刷新端点（`/api/v1/auth/refresh_token`）自动携带，普通 API 请求不会发送
- 鉴权 Cookie 由服务端通过 `Set-Cookie` 响应头设置和清除，前端 JavaScript 不可读取也不应手动操作
- 登出时服务端通过 `Max-Age=0` 清除两个鉴权 Cookie（`access_token`/`refresh_token`）

##### Web 端 device_id Cookie

`device_id` 是设备身份标识，**不是鉴权凭证**（`DeviceId` 类型语义见 `api_spec_types.md`），用于支撑「单设备单会话」模型（详见 `api_spec_auth.md`）。

- **属性差异**：不带 `HttpOnly`（前端可读取以便在 App 内嵌 WebView 等场景与 App 共享），`SameSite=Lax`（非凭证，无需 `Strict` 的 CSRF 保护，且需容忍从外部链接进入时携带）
- **生命周期**：长期有效（默认 10 年），**登出不清除**——保持设备身份连续，下次登录仍是同一设备
- **生成时机**：客户端首次访问（无 `device_id` Cookie）时由服务端生成 UUID v4 并通过 `Set-Cookie` 下发；客户端不可预测或自造
- **多浏览器档案**：不同浏览器档案（profile）/无痕窗口各自独立 Cookie 存储，天然对应不同 `device_id`，即「按浏览器档案 = 一台逻辑设备」

#### CSRF 防护（Web 端）

Web 端的 CSRF 防护**完全依赖 auth Cookie 的 `SameSite=Strict` 属性**，不使用额外的 CSRF Token（Double Submit Cookie）机制。

**设计依据**：
- `SameSite=Strict` 在所有现代浏览器中阻断跨站带凭证的 POST 请求，从根本上防御经典 CSRF（OWASP / Security StackExchange 共识）
- App 端无 Cookie，不受 CSRF 影响
- 此前的 Double Submit Cookie（`hs_csrf_token` / `X-CSRF-Token`）方案经评估为冗余——它防御的是已被 `SameSite=Strict` 中和的威胁，却带来每请求 header 注入、Cookie 生命周期管理、CSRF 不匹配误报等额外复杂度，故移除

#### App 端 Header 认证

App 端通过 Header 携带认证信息：
- `Authorization: Bearer <access_token>`
- Token 从响应体获取并存储在 Keychain（iOS）/ Keystore（Android）

#### Token 验证规则

| 场景 | Token 要求 | 响应差异 |
|-----|----------|---------|
| 查询类（搜索/详情/列表） | **可选** | 若类型定义含 `@auth-only` 字段，登录态返回这些字段 |
| 写操作（create/update/delete） | **必须** | 缺少 token 返回认证错误 |
| 互动操作（follow/block 等） | **必须** | 缺少 token 返回认证错误 |
| `/feed/*` 路径 | `following`/`nearby` 必须；`nearby_users`/`nearby_groups`/`trending` 可选 | 关注流和附近瞬间依赖身份；附近用户/群组/热门支持匿名浏览 |
| `/admin/*` 路径 | **必须**+权限 | 管理员功能 |

**认证档位说明**（端点"认证"字段统一取以下值之一）：
- **可选**：匿名与登录态均可访问；登录态按可见性返回额外字段
- **必须**：缺少或无效 Token 返回 `10100`
- **条件必须**：是否强制登录取决于请求参数。典型为"默认查自己（必须登录），显式指定他人 ID 时降级为可选"。**判定规则**：当请求未指定目标 ID（等价于查自己）且未携带有效 Token 时，返回 `10100`；其余情况按"可选"处理。各模块在该端点说明中需标注降级条件

**说明**：
- Web 端通过 Cookie 自动携带，无需前端代码手动处理
- App 端需要在每个请求的 Header 中手动添加 `Authorization` 字段

**认证可选端点的验证规则**：
- **任何时候**，只要请求携带了 Token，服务端都**必须验证** Token 的有效性，避免伪造或被撤销的凭证被继续使用。
- 对于标记为“认证: 可选”的端点，若 Token 验证失败（过期、被撤销、格式错误等），服务端必须视为未登录状态，忽略该 Token 并按匿名权限继续处理，仅返回公开数据（`@auth-only` 字段保持缺失），例如 `auth/check_status` 将返回 `is_valid: false`。
- 对于所有标记为“认证: 必须”的端点，缺少或无效 Token 仍返回 `10100`，客户端必须重新登录。
- 客户端若想显式以匿名身份访问，可不携带 Token；即便浏览器误带过期 Cookie，也不会阻塞公开接口。

**黑名单规则（契约级）**：
- access_token/refresh_token 的失效通过黑名单机制实现（登出/撤销即写入黑名单）
- 服务端验证 Token 时必须检查黑名单；黑名单条目有效期不得短于 Token 过期时间

### 字段可见性标注机制

> **重要**：`@xxx-only` 注解**仅用于字段级可见性控制**——标记同一类型中在不同上下文下是否返回某些字段。端点的访问权限通过错误码体系处理（如 `10200` 权限不足），不使用可见性注解。

#### @auth-only

在类型定义中，使用 `// @auth-only` 注释标记仅登录态返回的字段。

**客户端处理**：
- 未登录时：`@auth-only` 字段不存在于响应中（`undefined`）
- 已登录时：`@auth-only` 字段根据业务逻辑返回实际值
- **布尔型例外**：布尔型 `@auth-only` 字段（如 `Reaction.reacted_by_me`）在未登录时可返回 `false` 而非缺省，便于客户端直接判布尔而不必处理 `undefined`；`undefined` 与 `false` 语义等价

#### @owner-only

仅资源拥有者可见该字段（非拥有者视角不返回）。

#### @group-admin-only（群组管理员）

仅群组管理员可见该字段（群组内 role 为 `owner` 或 `admin`，非该视角不返回）。

**系统管理员豁免**：系统管理员通过 `/admin/*` 端点访问时（如 `AdminGroupItem`），视同拥有 `@group-admin-only`/`@member-only` 字段的查看权限，可完整返回这些字段。`@group-admin-only` 的成员资格限制仅约束普通用户视角的 API（群组内角色判定），不约束系统管理员的审计/管理视角。

说明：用于群组内的管理类字段，如成员管理详情、申请审核信息等。

#### @member-only

标记仅群成员视角返回的字段（等同于可选字段，非该视角下不返回）。

**可见性规则**：
- 仅群成员可见

#### @participant-only

标记仅会话参与者视角返回的字段（等同于可选字段，非该视角下不返回）。

### Token 可选端点的统一语义

认证为"可选"的端点遵循统一语义，各模块不再逐端点重复说明：

- **未登录（不带 token）**：仅返回公开可见内容；不返回 `@auth-only` 字段
- **已登录（带 token）**：在满足隐私与关系规则前提下，返回完整数据（包括 `@auth-only` 字段如 `relation`、`my_state`、`social_context` 等）

### 隐私与访问控制

**隐私等级判定**（`public` / `private`）：
- `public`：所有人可见（包括未登录用户）
- `private`：仅自己可见
- 未登录时：仅能访问 `privacy_level='public'` 的资源

**隐私字段可见性**：
- `privacy_level` 不对非资源拥有者暴露
- 仅资源拥有者与具备相应权限的系统管理员可查看 `privacy_level`

**隐私拒绝策略**：
- 资源不存在/不可见：返回资源不存在类错误
- 资源存在但无权限：返回权限不足类错误
- 列表/搜索：不可见资源按过滤处理，不返回

---

## API 设计原则

### 核心设计理念

**首要原则：清晰明了 > 代码复用**

每个 API 的 response 类型应该：
1. **自解释**：类型名称清楚表达用途和场景
2. **精准**：只包含该 API 实际需要返回的字段
3. **组合优于层级**：通过组合基础类型构建复杂对象，避免深层层级链

**设计规范**：

- **创建操作 (create)**: 根据场景选择响应模式——需要客户端立即渲染的资源（如瞬间、评论）返回完整对象以支持乐观更新；其他场景（如群组、举报）可仅返回资源 ID
- **列表/搜索/详情/Feed操作**: 普通 API 统一使用 `XxxItem` 类型，通过 `@auth-only`/`@group-admin-only` 等字段标注区分不同上下文的响应差异；`/admin/*` API 使用独立的 `AdminXxxItem` 扩展类型（通过 `extends` 继承普通类型），包含管理员视角的完整字段
- **上下文敏感**：同一路径下，区分"未登录（不带 token）"与"已登录（带 token）"的数据形态差异（通过 `@auth-only` 字段体现）

### 强类型要求

- 所有 request/response 必须强类型
- 类型命名遵循：
  - 统一数据类型: `XxxItem` (如 `UserItem`, `GroupItem`, `MomentItem` - 普通 API 使用，通过 `@auth-only`/`@group-admin-only` 等标注控制不同上下文的字段可见性)
  - 管理员扩展类型: `AdminXxxItem` (如 `AdminUserItem`, `AdminGroupItem` - `/admin/*` API 使用，通过 `extends XxxItem` 添加管理员专属字段)
  - 基础信息: `XxxInfo` (嵌入用，如 `UserInfo`, `GroupInfo`)
  - 我的状态: `MyXxxState` (如 `MyGroupState`)
  - 关系: `Relationship`

### 类型归属与复用规则（契约级）

- **嵌套引用也算引用**：若类型 A 被共享类型 B 作为字段使用，则所有引用 B 的模块都视为引用 A
- **多模块复用**：任一类型在 2 个及以上模块出现（直接或嵌套）必须定义在 `api_spec_types.md`
- **单模块归属**：只在单模块使用的类型必须定义在该模块文档
- **禁止同义重复**：已在 types 定义的类型，模块不得使用同义的字面量/结构硬编码
- **新增通用类型**：先改 types，再在模块引用，避免分叉

### 从用户使用场景出发的分层（契约级建议）

#### 未登录/冷启动（Guest Browsing）

目标：让用户**不注册也能理解产品**，同时获得流畅的浏览体验。

**可以做**：
- 浏览公开内容：瞬间/群组/用户资料页（公共字段）
- 搜索发现：按关键词/标签/地理位置搜索
- 地图浏览：公开地图渲染
- 系统配置：全局配置、公告、热门标签

**响应差异**：
- 详情页不返回 `relation`、`my_state`、`reactions[].reacted_by_me` 等 `@auth-only` 字段
- 搜索结果不返回互动状态字段
- 只能访问 `public` 可见性的资源

#### 已登录（Personalized Usage）

目标：提供完整功能与个性化信息。

**额外能力**：
- 详情页返回完整的 `@auth-only` 字段
- 可访问 `private` 资源（仅自己的资源）
- 个性化信息流（`/feed/*`）
- 私聊/群聊/通知等个人数据
- 所有写操作和互动操作

---

## 字段校验约束（契约级）

本节定义跨模块复用的字段校验规则。各模块接口文档必须显式标注模块级校验约束。

说明：以下约束值为系统默认值，管理员可通过 `AppConfig.limits`（见 `api_spec_admin_core.md`）动态调整部分限制，客户端通过 `ClientConfig.limits`（见 `api_spec_system.md`）获取生效后的值。

**模块级覆盖**：各模块可根据业务语义收窄（不可放宽）本节定义的全局约束。覆盖时必须在模块 spec 的接口说明中显式标注覆盖后的约束值及原因（如"分享场景 `location_name` 约束为 1-100 字符，必填"）。未显式标注的字段使用本节全局默认值。

### 通用字段约束

| 字段 | 约束 | 说明 |
|-----|------|------|
| `nickname` | 1-30 字符 | 用户昵称 |
| `password` | 8-128 字符，至少含字母和数字 | 密码复杂度 |
| `login_id` | 3-64 字符 | 登录标识（手机号/邮箱/用户名） |
| `bio` | 0-500 字符 | 个人简介 |
| `tags` | 每个标签 1-30 字符，默认最多 10 个（可由 `AppConfig.limits.max_moment_tags` 覆盖） | 用户/群组/瞬间标签 |
| `description`（moment） | 1-2000 字符 | 瞬间描述，可由 `AppConfig.limits.moment_description_max_length` 覆盖 |
| `content`（comment） | 1-500 字符 | 评论内容，可由 `AppConfig.limits.comment_max_length` 覆盖 |
| `content`（message） | 1-5000 字符 | 聊天消息内容 |
| `image_file_ids` | 最多 9 个 | 瞬间图片，可由 `AppConfig.limits.max_moment_images` 覆盖 |
| `mentions` | 最多 20 个 | @ 提及的用户 ID 列表 |
| `description`（report） | 0-1000 字符 | 举报详细说明，`reason` 为 `'other'` 时建议填写 |
| `reason`（admin） | 0-500 字符 | 管理员操作原因 |
| `remark` | 0-100 字符 | 用户备注 |
| `location_name` | 0-100 字符 | 位置名称 |

### 群组字段约束

| 字段 | 约束 | 说明 |
|-----|------|------|
| `name` | 1-50 字符 | 群组名称 |
| `description` | 0-500 字符 | 群组描述 |
| `password` | 8-32 字符 | 群组密码（设置时）；最小长度对齐账户密码，避免弱密码使 `has_password` 的隐藏保护失去意义 |
| `max_members` | 2-2000 | 群组最大成员数，可由 `AppConfig.limits.max_group_members` 覆盖 |
| `duration_minutes`（mute） | 1-43200（最多 30 天） | 禁言时长 |
| `message`（invite/apply） | 0-200 字符 | 邀请/申请附言 |

### 公告字段约束

| 字段 | 约束 | 说明 |
|-----|------|------|
| `title` | 1-100 字符 | 公告/群公告标题 |
| `content`（announcement） | 1-5000 字符 | 公告内容 |

### 搜索与地理约束

| 字段 | 约束 | 说明 |
|-----|------|------|
| `keyword` | 1-100 字符 | 搜索关键字 |
| `prefix` | 1-50 字符 | 搜索建议前缀 |
| `radius` | 100-50000 米 | 搜索半径，上限由 `AppConfig.limits.search_radius_max` 控制（`AppConfig` 见 `api_spec_admin_core.md`，`AppLimits` 结构见 `api_spec_system.md`） |
| `zoom`（map） | 1-20 | 地图缩放级别 |

### 管理员字段约束

| 字段 | 约束 | 说明 |
|-----|------|------|
| `role.name` | 1-50 字符 | RBAC 角色名称 |
| `role.description` | 0-200 字符 | RBAC 角色描述 |
| `duration_days`（ban） | 0-365（0=永久） | 封禁时长；超过 365 天（1 年）的封禁应直接使用 `0=永久`，避免"3650 天"这类实际等同永久却误导管理员以为是长期可逆封禁的取值 |
| `note`（report handle） | 0-500 字符 | 举报处理备注 |
| `support_email` | 有效 Email 格式 | 支持邮箱 |
| `*_url`（contact） | 有效 URL 格式 | 联系信息 URL |

---

## 补充原则

### 分页与排序约定

本节只描述"契约级"规则，不包含客户端实现建议。

#### 分页机制（契约级）

- 所有列表接口使用游标分页，禁止 offset 分页
- `cursor` 为**完全不透明字符串**（客户端禁止解析/构造/修改，格式可能随版本变化）。服务端必须对 cursor 进行签名或加密处理，防止客户端篡改或解码以推测内部排序逻辑和 ID 范围
- 无界列表接口统一使用 `PaginationParams` 与 `PaginatedData<T>` 进行游标分页（类型定义见 `api_spec_types.md`"分页与元数据"小节）
- 有界/确定数量的数据（如搜索建议、热门标签、聊天最近消息）由服务端决定返回数量，具体值通过 `AppConfig.response_defaults` 配置（见 `api_spec_admin_core.md`），客户端不传分页或 limit 参数
- 任意使用 `PaginationParams` 的接口必须返回 `PaginatedData<T>`；不返回分页结构的接口不得使用 `PaginationParams`
- **无更多数据**：当没有下一页时，响应中**省略 `next_cursor` 字段**（即 JSON 中不包含该 key）。客户端通过 `next_cursor` 字段是否存在来判断是否有更多数据，而非检查 `null`

#### 分页默认值与模块级覆盖

全局默认值：`limit` default=20, max=100（定义于 `PaginationParams`）。

各模块可按业务特性覆盖 `limit` 的 default 和 max 值，但必须在接口文档的 `Request` 说明中显式标注。未显式标注的接口使用全局默认值。

#### 排序语义（契约级）

- 每个列表接口必须声明默认排序主序、稳定 tie-breaker、以及一致性级别（强一致/弱一致）
- 时间类列表必须提供稳定 tie-breaker（使用资源主键）
- 地理相关列表必须以服务端计算的 `distance` 作为主序，客户端不得本地计算或重排

#### 去重（契约级）

- 客户端必须按资源主键去重（用户：`user_id`；群组：`group_id`；瞬间：`moment_id`；评论：`comment_id`；消息：`message_id`）

#### 计数器一致性（契约级）

所有 `*_count` 字段（如 `likes_count`、`comments_count`、`shares_count`、`members_count`、`followers_count`、`reaction.count`、`read_count` 等）均为**最终一致**，非强一致实时值：

- **更新策略**：服务端采用异步累计（写入队列 + 定时刷新到主存储 + 缓存），容忍秒级到分钟级延迟
- **客户端约束**：
  - 不得将 `*_count` 用于精确对账或与列表实际条数强校验（列表分页 + 计数器异步可能导致短暂不一致，属正常现象）
  - 不得在收到互动事件后立即期望本地 count +1 与服务端一致；应以服务端后续推送或下次拉取为准
  - 不得轮询 count 字段（高频拉取计数器视为滥用，触发限流）
- **精度**：服务端保证最终收敛（无互动时 count 稳定），但不保证任意时刻与底层数据完全一致

> 设计依据：社交产品的计数器（粉丝数、点赞数）在百万级活跃用户下，若每次互动都同步写主库会击穿 DB。业界（Twitter/微博/Discord）一致采用异步累计 + 缓存近似值。客户端 UI 应容忍短暂偏差，不显示"精确到个位"的承诺。

### 隐私与可见性（契约级）

- 隐私受控字段采用"字段不存在即不可见"的策略，禁止用 `null` 表达"隐藏"
- **隐私拒绝策略（统一三类）**：
  - 资源**不存在**：返回 `10300`（资源不存在）
  - 资源存在但**隐私不可见**（如 `private` 内容、资料完全隐藏、被拉黑等）：返回 `10300`（按不存在处理，不暴露资源存在性，防枚举）
  - 资源存在且可见但**业务权限不足**（如非作者编辑、非群成员操作等）：返回 `10200`（权限不足）
- 列表/搜索：不可见资源按过滤处理，不返回不可见项（不因过滤返回错误）
- 各模块端点若与上述分类不符，以本小节为准

### 密码安全

- 传输使用 HTTPS
- 客户端发送明文 (Over TLS)
- 服务端存储 Hash (Argon2/Bcrypt)

### 限流策略（契约级）

本小节定义限流**默认策略**，实际阈值可由管理员动态配置并即时生效。触发限流时返回限流类错误响应。

| 接口类型 | 限流维度 | 策略 | 默认限制值 | `RateLimitConfig` 对应字段 |
|---------|---------|------|--------|--------------------------|
| 登录/注册/游客/重置 | IP | 滑动窗口 | 10次/10分钟 | `auth_login_per_10minutes` 等 |
| 搜索类 | 用户 | 令牌桶 | 60次/分钟 | `search_per_minute` |
| 通用写操作（创建/更新/删除） | 用户 | 滑动窗口 | 30次/分钟 | `write_per_minute` |
| 业务写操作（动态/评论/举报） | 用户 | 滑动窗口 | 各不同 | `moments_create_per_minute` 等 |
| 文件上传 presign | 用户 | 滑动窗口 | 100次/小时 | `upload_presign_per_hour` |
| WebSocket 订阅 | 连接 | 固定窗口 | 10次/秒 | `ws_subscribe_per_second` |
| 管理员查询（`/admin/*` 读） | 用户 | 令牌桶 | 100次/分钟 | `admin_read_per_minute` |
| 管理员写操作（`/admin/*` 写） | 用户 | 滑动窗口 | 30次/分钟 | `admin_write_per_minute` |
| 全局限流（所有请求兜底） | IP+用户 | 固定窗口 | 服务端配置 | `global_per_hour` |
| 用户级全局限流（所有请求兜底） | 用户 | 固定窗口 | 服务端配置 | `user_per_hour` |

> 各字段的完整定义和取值范围见 `api_spec_admin_core.md` 的 `RateLimitConfig`。

**限流策略说明**：
- **滑动窗口**：按实际时间窗口统计，平滑限流
- **固定窗口**：按自然时间窗口（如整分钟）统计
- **令牌桶**：允许突发流量，但平均速率受限

**客户端真实 IP 解析**（IP 维度限流的前提）：

项目通过 Cloudflare CDN 代理所有流量，后端**禁止**使用 TCP 连接源 IP（那是 CF 边缘节点 IP）。IP 解析优先级：

| 优先级 | Header | 说明 |
|--------|--------|------|
| 1 | `CF-Connecting-IP` | Cloudflare 注入的真实客户端 IP，**最可信**，始终为单个 IP |
| 2 | `X-Real-IP` | 备选（部分反向代理配置），单个 IP |
| 3 | `X-Forwarded-For` | 最后手段，取第一个非内网 IP（注意：可被客户端伪造前缀，仅在 CF 层保证最后一段可信） |

**实现要求**：
- 后端必须校验请求确实来自 Cloudflare IP 段（[CF IP Ranges](https://www.cloudflare.com/ips/)），防止攻击者绕过 CDN 直连后端并伪造 `CF-Connecting-IP`
- 若所有 header 均缺失，回退到连接源 IP（仅开发/测试环境）
- `CF-IPCountry` header 可用于地理位置级别的异常检测（如同一账号短时间内跨大洲登录）

**客户端处理**：
- 收到限流错误码后，可按端侧重试策略处理
- 如响应头包含 `Retry-After`，优先按该值重试

### 幂等性（契约级）

幂等性用于防止网络重试导致的重复创建或重复副作用。

#### 判定原则

一个接口是否需要 `Idempotency-Key`，取决于**重发是否会真的产生重复结果**，而非「客户端是否会重试」：

- **需要**：重发会产生用户可见的重复内容/数据，且**无其他机制兜底去重**——客户端必须携带 `Idempotency-Key`，服务端据此去重并回放首次成功响应。
- **不需要**：以下三类天然幂等，契约不标注、客户端不传 key：
  - **状态设值/状态转移**：`set_mute`、`ban`/`unban`、`set_role`、`favorite`、`reactions/set` 等——重复请求达成同一状态即 no-op 成功。
  - **删除/解散/更新**：删除已删除、解散已解散、更新相同字段，重发结果一致。
  - **有自然去重键的创建**：服务端存在唯一约束或一次性凭证，重发要么返回成功+已存在资源（无歧义）、要么因凭证失效/权限丧失而失败——这本身就是幂等保护，无需额外 key。

需要实现幂等的接口以本节「需要实现幂等的接口」清单为唯一权威。二元判定：要么「需要」、要么「不需要」——不存在「建议」中间档。

#### 需要实现幂等的接口

以下接口客户端必须携带 `Idempotency-Key`：

- 内容创建（无自然去重键，重发=重复内容）：`moments/create`、`comments/create`、`groups/create`、`groups/announcements/create`、`admin/announcements/create`、`share/create`
- 消息发送（无自然去重键，重发=重复消息骚扰）：`private_messages/send`、`group_messages/send`
- 账号创建：`auth/guest`（无唯一约束，每次调用创建独立访客账号）、`auth/register`（唯一约束特例，见下方）

> **唯一约束特例（`auth/register`）**：`login_id` 有唯一约束，重复注册返回 `10400`（login_id 已存在），表面看似自然去重。但与下面「自然去重键」白名单的接口不同：这里的 `10400` 有**歧义**——是「我刚提交被自己重试占了」还是「该 login_id 早被别人占用」？客户端收到后无法直接判断本次注册是否已成功，必须额外发起一次 `login` 才能恢复。为消除这一歧义、让重试能回放首次的 `AuthTokenPair`，`register` 作为全规范唯一特例保留 `Idempotency-Key`。

#### 用自然去重键替代 Idempotency-Key 的接口（无需 key）

以下接口虽为创建/写操作，但服务端存在自然去重机制——重发不会产生重复数据，因此**不需要** `Idempotency-Key`：

| 端点 | 自然去重键 | 重发行为 |
|------|----------|---------|
| `reports/create` | `(reporter_id, target_type, target_id, pending_status)` | 返回成功 + 已有 `report_id`（`is_duplicate: true`），不创建新举报单 |
| `groups/members/invite` | `(group_id, invitee_id, pending)` | 返回已有邀请，不重复创建、不重复通知 |
| `groups/join`（approval 路径） | `(group_id, applicant_id, pending)` | 返回已有 `application_id`，不重复创建、不重复通知管理员；direct/invite 路径成员关系唯一，天然幂等 |
| `auth/reset_password` | 一次性验证码 | 首次成功后验证码失效，重发 → `10300`（验证码已用/过期） |
| `auth/change_password` | 旧密码凭证 | 首次成功后旧密码失效，重发 → `10100`（认证失败） |
| `groups/transfer_ownership` | 群主身份（调用方权限） | 首次后调用方失群主权限，重发 → `10200`（权限不足） |

#### Idempotency-Key 格式

- **Header 名称**：`Idempotency-Key`
- **格式**：`{client_id}-{timestamp_ms}-{random}`
- **长度**：32-64 字符
- **字符集**：`[a-zA-Z0-9-_]`
- **要求**：同一用户对同一接口的同一次重试必须复用同一个 `Idempotency-Key`

#### 服务端行为（契约级）

- `Idempotency-Key` 绑定当前用户与具体接口（同一 key 不同用户视为不同请求）
- Key 有效期：服务端保证至少 24 小时内有效（具体实现细节不对外暴露）
- 对于同一 key：返回首次成功响应的**完整副本**（包括 code/data）

#### 响应行为

| 场景 | 响应内容 |
|-----|---------|
| 首次请求 | 正常处理并缓存响应 |
| Key 命中（已处理完成） | 返回缓存的原响应（等价成功） |
| Key 命中（处理中） | 返回 `22002`，客户端应等待后使用相同 key 重试 |
| Key 缓存失效 | 返回 `22001`，客户端应使用新 key 重试 |
| Key 不可恢复冲突（如格式/上下文不匹配） | 返回 `22000`，客户端应使用新 key 重试 |

#### 客户端要求

- **重试**：必须使用**相同**的 `Idempotency-Key`
- **新请求**：必须生成**新**的 `Idempotency-Key`
- **收到 `22002`（处理中）**：等待 1-2 秒后使用相同 key 重试
- **收到 `22001`（缓存失效）或 `22000`（不可恢复冲突）**：生成新 key 后重试
- **收到缓存回放响应（命中已处理完成）**：视为成功，直接使用返回数据
- 具体幂等错误码（`22000`/`22001`/`22002`）的定义与客户端行为详见 `api_spec_errors.md`
- **错误码约定**：需要实现幂等的端点除特别说明外均可能返回 `22000`/`22001`/`22002`；各端点的「错误码」小节**不应重复列出**这三种幂等错误码，只需补充额外的业务错误码。
