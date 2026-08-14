# HotSpot API - 基础原则与约束

> 定义 HotSpot API 设计的基础原则、节点架构、命名规范、全局约束、Token 语义、分页与排序、幂等性、安全性实践等核心内容。

## 权威边界与冲突处理

- 若本文件与任何工程实现或其它 doc/rule 文件产生冲突，以本文件为准
- 引用本文件内容时仅使用小节标题或锚点标题，禁止使用数字章节号

## 术语约定（契约级）

- **未注册用户**：指没有任何有效会话身份的调用方（匿名）。只能调用 `/public/v1` 节点
- **注册用户**：指持有有效会话身份的调用方，包含**正式注册**（`user_type='normal'`）与**临时注册**（`user_type='temp'`，访客）两类；两者在权限上完全同权，差异仅在账号生命周期（`upgrade` 转正）。调用 `/api/v1` 节点
- **当前用户**：指已鉴权的调用方用户。个人视角操作由 Token 隐含标识，不设 `/users/me` 类资源域；单值读取用名词化端点（如 `auth/session`、`rbac/me`，见「无动词端点」清单）
- **自身**：指"对自己执行操作"的语义，用于错误码与规则描述
- **群成员**：指已加入群组的用户
- **会话参与者**：指私聊会话的双方用户
- **平台管理员**：指通过 `/admin/v1` 节点执行平台级管理操作的用户（superadmin 或绑定 AdminRole 的 admin）。平台管理员属于注册用户，在 `/api/v1` 上与普通用户完全同权
- **群主 / 群管理员 / 群成员**：指群组内的三种角色（`GroupRole`：owner / admin / member）。群管理员属于普通用户范畴，其群管理操作走 `/api/v1/groups/admin/` 守卫子树；与平台管理员是完全不同的两套体系，按节点与上下文分别判定、互不传递
- **术语纪律（必须）**：文档禁止裸称「管理员」；一律写「平台管理员」或「群主/群管理员」。禁止以任何方式暗示两套体系之间存在权限传递

重要契约约定（必须）：
- 所有时间字段使用 UTC，格式为 ISO 8601（如 `2024-01-15T08:30:00.000Z`）。毫秒部分可选：服务端可返回带或不带毫秒的格式，客户端必须兼容两种
- 客户端负责时区转换：接收到 UTC 时间后按用户本地时区展示
- 客户端发送时间字段也必须使用 UTC ISO 8601（禁止上传本地时区字符串）

---

## 节点与路径架构（契约级）

HotSpot 的 API 按**调用者身份**划分为三个全局节点 + 一个守卫子树。**节点即档位**：URL 前缀直接表达身份要求，无需查权限矩阵。契约只定义干净的划分与守卫；**该调用哪个端点由客户端按自身身份决定**，契约不做客户端的事。

| 位置 | 调用者 | 守卫 | 说明 |
|------|--------|------|------|
| `/public/v1/` | 未注册用户（纯匿名） | **完全不解析 Token**；无任何隐式身份增强 | 匿名浏览发现 + 账号生命周期入口 + 系统信息 |
| `/api/v1/` | 注册用户（正式 + 临时同权） | 必须携带有效 Token，否则 `10100`，**无一例外** | 客户端 API |
| `/api/v1/groups/admin/` | 群管理员（守卫子树） | 统一守卫：调用者必须是**请求目标群**的 owner/admin | 群管理 API，附属于 groups 资源 |
| `/admin/v1/` | 平台管理员（独立节点） | 必须 Token + 平台管理员身份 + `admin.*` 权限键 | 平台管理 API |

规则：
- 每个顶级节点对应**一种且仅一种**调用者身份；禁止在同一节点内混用多种身份要求（这正是节点划分的目的：看 URL 前缀即知身份要求）
- 只有**全局身份**才配顶级节点（`/public/v1`、`/admin/v1`）；群管理是群组资源的附属视角，挂在 `groups/admin` 守卫子树之下（群管理员属于普通用户范畴，故子树位于 `/api/v1` 内）
- 每个端点**单一角色、单一检查**：禁止「作者或管理员」「成员或管理员」等双角色端点；双角色语义一律拆分为两个端点（如消息撤回拆为 `group_messages/recall` 与 `groups/admin/messages/recall`）
- 同一功能面向匿名与登录两类身份时，**拆分为两个端点**（匿名版在 `/public/v1`，登录版在 `/api/v1`），且 Request/Response 类型**各自独立、禁止共用**（见「强类型要求」）
- 三处共用同一 token 体系、同一 envelope 与同一错误码空间；`/public/v1`（IP 维度）与 `/admin/v1`（独立反向代理）各自有限流策略（见「限流策略」）
- WebSocket 端点为 `/ws/v1`，独立于 HTTP 节点，仅服务于注册用户的实时事件（连接后必须通过 `Auth` 事件完成鉴权，见 `api_spec_websocket.md`）

### 统一路径结构

```
/public/v1/            # 未注册用户节点（纯匿名，不解析 Token）
├── auth/              # 账号生命周期入口（注册/登录/游客/刷新/密码重置）
├── users/             # 用户公开资料（搜索/详情/公开列表）
├── moments/           # 瞬间公开浏览（搜索/详情）
├── comments/          # 评论公开浏览（列表/详情）
├── groups/            # 群组公开浏览（搜索/详情/成员/公告）
├── feed/              # 公开信息流（附近用户/附近群组/热门）
├── search/            # 搜索建议
├── map/               # 地图渲染
├── share/             # 分享码落地
├── reactions/         # 反应公开列表
└── system/            # 系统（健康/版本/配置/公告/匿名冷启动聚合）

/api/v1/               # 注册用户节点（必须有效 Token）
├── auth/              # 会话自助管理（登出/改密/会话列表/撤销）
├── users/             # 用户相关（搜索/资料/关系/账户）
├── moments/           # 瞬间相关（搜索/CRUD/收藏）
├── comments/          # 评论相关
├── reactions/         # 表情反应（跨 message/moment/comment）
├── private_chats/     # 私聊管理
├── private_messages/  # 私聊消息
├── groups/            # 群组相关（搜索/详情/加入/成员/邀请/申请/公告）
│   └── admin/         # 群管理守卫子树（owner/admin 专属）
├── group_messages/    # 群聊消息
├── feed/              # 信息流（关注/附近/热门）
├── search/            # 搜索建议
├── reports/           # 举报
├── notify/            # 通知中心
├── media/             # 媒体上传
├── share/             # 分享系统
├── system/            # 系统（登录态冷启动聚合）
└── ...
/admin/v1/             # 平台管理员节点（RBAC/审计/配置/审核）
├── rbac/
├── audit/
├── app_config/
├── users/
├── groups/
├── reports/
├── tags/
└── announcements/
```

---

## 命名规范（契约级）

### 端点形态

- 端点形态统一为 `/{node}/{version}/{resource}/{action}`（node ∈ public / api / admin），`{resource}` 可含多级子资源路径（如 `groups/applications/fetch_submitted`、`rbac/roles/set_permissions`）；守卫子树为 `/{node}/{version}/{resource}/admin/{action}`（如 `groups/admin/members/set_mute`）
- **资源域单复数规则**：可数实体资源域用复数（users / moments / comments / reactions / groups / private_chats / private_messages / group_messages / reports）；服务域、动作域与不可数概念域用单数（auth / feed / search / map / share / media / notify / system）。禁止混用（如 `shares/`、`moment/`）
- 全小写 snake_case；ID 字段显式 `xxx_id` 命名（禁止裸 `id`）；所有 `xxx_id` 字段在 API 层统一为 `string`
- 两态参数统一 `active: boolean`（true=执行/开启，false=撤销/关闭）
- 所有 `*_url` 字段的值必须为**完整的绝对 URL**（含协议和域名），客户端无需拼接 base URL

### 权威动词表

通用资源动词：

| 动词 | 语义 |
|------|------|
| `create` | 创建资源 |
| `open_the_door` | 获取单资源详情（刻意保留的玩梗命名，见下方保护条款） |
| `fetch` / `fetch_<state>` | 列表（游标分页）；状态限定词见下 |
| `update` | 更新资源（Patch 语义） |
| `delete` | 删除/软删除资源 |
| `search` | 关键字/条件搜索 |
| `set_<state>` | 设置两态状态（set_follow / set_block / set_favorite / set_remark / set_role / set_mute / set_ban） |
| `batch_fetch` | 批量按 ID 获取 |

域专属动词：

| 动词 | 语义 | 专属域 |
|------|------|--------|
| `send` / `recall` / `mark_read` | 消息域动词 | private_messages / group_messages |
| `register` / `signin` / `signout` / `guest` | 账号生命周期（guest = 访客注册，与 register 对仗） | auth |
| `refresh_token` / `change_password` / `forgot_password` / `reset_password` | 凭证类动宾动词（忘记→重置两步对仗） | auth |
| `revoke_session` / `revoke_other_sessions` | 会话撤销（other_sessions 明确「除当前外」，含当前是 `signout` 的职责） | auth |
| `upgrade` | 访客转正（账号状态转移） | users |
| `enter` / `exit` / `dismiss` / `remove` | 成员进出与生命周期动词 | groups |
| `transfer_ownership` | 转让所有权 | groups/admin |
| `award` | 授予身份（revoke 见工作流动词） | admin/rbac |
| `handle` | 组合处理动作（body 参数判别具体子动作，如 resolve/dismiss） | admin/reports |
| `suggest` | 前缀联想 | search |
| `presign_upload` / `complete_upload` | 媒体两段式上传（数组参数 1..N 天然覆盖单/批，无独立批量端点） | media |
| `accept` / `reject` / `approve` / `cancel` / `revoke` | 工作流动词 | invites / applications / share |

**无动词端点**（名词直接做端点，两类规则）：
- **读取类名词化**（读单值/配置/当前对象，直接以名词命名）：`users/account`、`users/settings`、`auth/session`、`system/meta`、`system/client_config`、`notify/unread_count`、`rbac/me`、`groups/admin/settings`、`app_config`（裸域读取）
- **惯例例外**：`system/health`（健康检查行业惯例）、`system/bootstrap`（冷启动聚合，协议级入口）、`map/canvas`（地图画布资源）、`feed/` 流槽位名（`following` / `trending` / `nearby_moments` / `nearby_users` / `nearby_groups`，业界通用命名，非动作）

**动词审美规则（契约级，主观个案评估）**：动作词的观感以**整体字形与组合的协调感**为准，**不做机械的字母形状排除**——下伸部/上伸部字母本身不构成禁用理由（`presign` 即为无问题的反例，经评估保留原名）。已确认丑并废除的仅两类：`get_*` 前缀（全量名词化，改用「无动词端点」清单的名词化命名）与 `list_*` 前缀（全量更名 `fetch_*`）。新增动作词时按整体观感个案评估，存疑时向维护者确认（玩梗保护条款优先级更高）

**范围语义的表达约定（契约级）**：动作作用范围用 `scope` 参数表达（`notify/mark_read` 的 `'ids'|'all'`、`group_messages/mark_read` 的 `'group'|'all'`）；**仅当「全部」实际不含当前对象且行为链不同时**才拆独立端点并以名写实（`auth/revoke_other_sessions`——撤销除当前会话外的全部，含当前是 `signout` 的职责），禁止名实不符的 `*_all_*` 命名。

### 状态限定词（列表场景区分视角的唯一手段）

同一命名空间存在多视角列表时，用**状态限定词**区分，只描述资源状态、不标记视角。现有：`created / joined / submitted / sent / received / favorites / blocked / banned / reacted`，以及关系/子资源维度（`friends / following / followers / groups / moments`）。

- **`_my` 后缀与 `My` 类型前缀全量禁止**：命名空间天生私有（每人只见自己的数据）的列表直接 `fetch`（如 `reports/fetch`、`share/fetch`、`notify/fetch`）；不存在「非 my 版本」需要区分时不得加任何视角后缀
- 例：`groups/fetch_joined`（我加入的群）、`groups/applications/fetch_submitted`（我提交的申请）、`moments/fetch_created`（我创建的瞬间）、`groups/invites/fetch_sent`（我发出的邀请）

### 参数必显式（契约级）

- 语义敏感的参数一律**必传**，禁止用缺省值翻转语义：目标 ID（`user_id`/`user_ids`/`group_id`/`moment_id` 等）必传；影响行为范围的参数必须显式声明（如通知已读范围用 `scope: 'ids' | 'all'`）
- 仅允许「无歧义固定缺省」：分页 `limit`（默认 20）、`radius`（默认 5000）、`privacy_level`（默认 'public'）等单一固定值
- 禁止「缺省 = 当前用户」这类按调用者身份翻转的缺省语义

### 禁用词表

- 禁用 `notification`（统一 `notify`）；新增禁用词需维护者确认
- **玩梗/彩蛋命名保护条款**：禁止以"不专业/不规范"为由修改规范中刻意为之的**无害**玩梗/彩蛋命名（如 `open_the_door`），存疑时应向维护者确认；但冒犯性或针对特定群体的内容不属于保护范围

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

- 当端点的 Request 标注为 `{}`，或 Request 类型定义为空对象（如 `interface XxxRequest {}`）时，请求体必须为 JSON 对象
- 缺失请求体或非对象请求体（如数组、字符串、数字、布尔）时返回 `10000`（参数错误）
- **忽略未知字段**：空对象端点对请求体中出现的未知字段**静默忽略**（不返回错误）。严格未知键拒绝仅用于**有类型的请求体**

### 通用隐含错误码

部分错误码为**所有相关端点隐含支持**（清单见 `api_spec_errors.md`「隐含错误码（无需在端点重复列出）」，唯一权威）。各模块端点文档的"错误码"字段只列该端点**特有**的业务错误码，不得重复列出隐含码——若出现隐含码，视为文档缺陷，应删除。

---

## 请求与响应头（契约级）

- `X-Request-Id`：请求链路追踪 ID。客户端**可选**传入（用于跨端追踪关联）；服务端**始终**在响应头返回该字段——若请求携带则原样回显，未携带则由服务端生成。服务端是 `X-Request-Id` 的权威来源
- 服务端**不应**对客户端传入的 `X-Request-Id` 施加格式或熵值约束
- `Idempotency-Key`：仅用于需要幂等性的写操作，具体要求见"幂等性（契约级）"
- `If-None-Match`：条件请求，**仅 `system/bootstrap` 使用**（public 版与登录版均支持；携带上次响应的 `bootstrap_version` 做缓存校验）

说明：`X-Request-Id` 与 `Idempotency-Key` 相互独立；同一 `Idempotency-Key` 的多次请求可返回不同的 `X-Request-Id`。

### 响应头安全约束

服务端**禁止**返回以下可能泄露技术栈或部署架构的响应头：

| 禁止的 Header | 泄露风险 |
|---------------|---------|
| `Server` | 暴露 Web 服务器类型和版本 |
| `X-Powered-By` | 暴露后端框架信息 |
| `X-AspNet-Version` / `X-Runtime` 等 | 暴露运行时或框架版本 |

**实现要求**：在反向代理层和应用层均移除或覆盖上述响应头。

### 限流响应头

服务端**仅在触发限流时**返回 `Retry-After` 响应头（单位秒）。不暴露 `X-RateLimit-*` 等精确限流状态。

---

## Token 语义与节点身份（契约级）

### Web 端 Cookie 契约

Web 端使用以下**固定名称**的 HttpOnly Cookie 携带认证信息（**禁止**使用 `Authorization` Header）。此外使用一个独立的设备标识 Cookie（非鉴权凭证）：

| Cookie 名称 | 用途 | Path | 属性 | Max-Age |
|---|---|---|---|---|
| `access_token` | API 访问令牌 | `/` | `HttpOnly; Secure; SameSite=Strict` | 服务端动态设置（与 token 过期时间一致） |
| `refresh_token` | Token 刷新令牌 | `/public/v1/auth/refresh_token` | `HttpOnly; Secure; SameSite=Strict` | 服务端动态设置（与 token 过期时间一致） |
| `device_id` | 设备标识（非鉴权凭证） | `/` | `Secure; SameSite=Lax` | 10 年（长期，独立于登录态） |

**规则**：
- `access_token` 使用 `Path=/` 统一覆盖需要鉴权的 HTTP API（`/api/v1/`、`/admin/v1/`）和 WebSocket（`/ws/v1`）端点；`/public/v1/` 节点**不解析**该 Cookie（浏览器可能自动携带，服务端直接忽略，不影响响应）
- `refresh_token` 使用精确 Path 确保仅在刷新端点（`/public/v1/auth/refresh_token`）自动携带
- 鉴权 Cookie 由服务端通过 `Set-Cookie` 响应头设置和清除，前端 JavaScript 不可读取也不应手动操作
- 登出时服务端通过 `Max-Age=0` 清除两个鉴权 Cookie
- `device_id` 不带 `HttpOnly`（前端可读取以便在 App 内嵌 WebView 等场景与 App 共享）、`SameSite=Lax`；**下发时机（契约级）**：Web 端冷启动首个请求 `POST /public/v1/system/bootstrap` 时，若请求未携带 `device_id` Cookie，服务端生成 UUID v4 并随响应 `Set-Cookie` 下发（bootstrap 是冷启动第一端点，天然保证后续 `auth/register`/`auth/guest` 等从 Cookie 读取时已存在）；已携带则沿用。登出不清除

### CSRF 防护（Web 端）

Web 端的 CSRF 防护**完全依赖 auth Cookie 的 `SameSite=Strict` 属性**，不使用额外的 CSRF Token 机制。

### App 端 Header 认证

- `Authorization: Bearer <access_token>`
- Token 从响应体获取并存储在 Keychain（iOS）/ Keystore（Android）

### 节点身份语义（契约级）

身份要求由节点前缀直接表达，全规范无例外：

| 节点 | Token 处理 | 语义 |
|------|-----------|------|
| `/public/v1/*` | **完全不解析**（Cookie/Header 中的 Token 一律忽略） | 纯匿名视角：仅返回公开数据，响应形状与调用者是否恰好持有 Token 无关 |
| `/api/v1/*` | 必须携带有效 access_token | 缺少/无效/黑名单 → `10100`；封禁 → `10201` |
| `/admin/v1/*` | 必须携带有效 access_token + 平台管理员身份 | 未认证 → `10100`；非平台管理员 → `10200` + `not_admin`；细粒度权限见 `api_spec_authz.md` |
| `auth/refresh_token`（凭证特例） | 以请求体中的 refresh_token 为凭证 | 无效/过期/被撤销 → `10100`。该端点位于 `/public/v1`，但凭证语义独立于节点匿名语义 |

说明：
- 匿名与登录两种视角一律拆分为 `/public/v1` 匿名版 + `/api/v1` 登录版两个端点；不存在「同一端点按登录态返回不同数据」的隐式行为
- `/public/v1` 不解析 Token 意味着：即便请求携带了有效 Token，也**不会**返回 viewer_state / relation 等登录视角增强字段；需要增强视角时客户端必须调用 `/api/v1` 的对应端点
- 目标 ID 一律必传（禁止「缺省 = 当前用户」的身份缺省语义）；拉取自己的数据时客户端显式传入自己的 ID
- Web 端通过 Cookie 自动携带，无需前端代码手动处理；App 端在每个请求的 Header 中手动添加 `Authorization` 字段（`/public/v1` 请求可不携带）

**黑名单规则（契约级）**：
- access_token/refresh_token 的失效通过黑名单机制实现（登出/撤销即写入黑名单）
- 服务端验证 Token 时必须检查黑名单；黑名单条目有效期不得短于 Token 过期时间


---

## 隐私与访问控制（契约级）

**隐私等级判定**（`public` / `private`）：
- `public`：所有人可见（包括未登录用户）
- `private`：仅自己可见
- `/public/v1` 节点（纯匿名）：仅能访问 `privacy_level='public'` 的资源；`/api/v1` 节点按资源隐私等级 + 调用者关系判定

**隐私拒绝策略（统一三类）**：
- 资源**不存在**：返回 `10300`（资源不存在）
- 资源存在但**隐私不可见**（如 `private` 内容、资料完全隐藏、被拉黑等）：返回 `10300`（按不存在处理，不暴露资源存在性，防枚举）
- 资源存在且可见但**业务权限不足**（如非作者编辑、非群成员操作等）：返回 `10200`（权限不足）+ `data.forbidden_reason`（见 `api_spec_errors.md`）

**整表隐私拒绝（列表级）**：当整个列表因目标用户的隐私设置（如 `social_lists_visible=false`、`groups_visible=false`、`reacted_moments_visible=false`）而**整体不可见**时，返回 `10200` + `forbidden_reason='privacy_setting'`——区别于逐项过滤（整表隐藏返回空列表会泄露"数据存在但被隐藏"的信息）

**逐项过滤**：列表/搜索中逐项不可见资源按过滤处理，不返回不可见项（不因过滤返回错误）

---

## API 设计原则

### 核心设计理念

**首要原则：清晰明了 > 代码复用；一个端点 = 一种角色 + 一种固定形状。**

每个 API 的 response 类型应该：
1. **自解释**：类型名称清楚表达用途和场景
2. **精准**：只包含该 API 实际需要返回的字段
3. **稳定**：形状不随调用者权限/隐私变化（见「类型稳定性」）
4. **组合优于层级**：通过组合基础类型构建复杂对象，避免深层层级链

**端点风格（契约级）**：HotSpot 对外 HTTP API 仅允许使用 POST；对象存储直传的 PUT/POST 仅发生在存储服务域名上。理由：动作型社交产品 + 统一 envelope + 显式动词端点已完全覆盖语义，REST 风格不会带来额外收益，反而引入 HTTP 方法与状态码的语义分叉。

### 设计规范

- **创建操作 (create)**: 内容类创建（瞬间/评论/群组/公告等用户可见资源）统一返回完整对象（`XxxItem`）；非内容类创建（如分享码、角色）返回必要的标识字段即可
- **更新操作 (update)**: 内容类更新（瞬间/群组/公告/消息等）统一返回更新后的完整对象；偏好/设置类更新（settings/preferences/角色/禁言等状态设值）返回空 `ApiResponse`
- **列表/搜索/详情/Feed**: 统一使用固定公开类型（`XxxItem`）；不同视角使用**不同端点 + 不同扩展类型**（见「类型稳定性」）；`/admin/v1` 使用独立的 `AdminXxxItem` 扩展类型
- **上下文敏感已废除**：同一端点不按调用者返回不同形状——视角差异一律拆端点（注解废除声明见「类型稳定性」）

### 强类型要求

- 所有 request/response 必须强类型
- **每个端点必须有命名的 Request 类型**：空请求体也须 `interface XxxRequest {}`，禁止裸 `{}` 与匿名结构
- **禁止匿名响应容器**：响应必须引用命名类型（列表容器统一 `PaginatedData<T>` 或命名 `XxxList`）
- **public 节点类型完全隔离**：`/public/v1` 双版本端点的 Request/Response 主体类型使用专属 `Public` 前缀（`PublicXxxRequest` / `PublicXxxItem` 等，定义在 `api_spec_public.md`），**禁止**与 `/api/v1`、`/admin/v1` 的对偶类型互相引用。匿名视角类型**不含** viewer_state / relation / reacted_by_me / remark 等查看者状态字段——这些字段只存在于登录视角类型中；嵌套结构里含查看者字段的类型同样必须拆出 Public 版（如 `PublicReaction`）。**例外**：无视角的共享类型跨节点共用、不复制——Domain Primitives 与纯配置类型（`GeoPoint`、`PaginationParams`、`UserInfo`、`MediaAsset`、`ApiVersionMeta` 等，定义于 `api_spec_types.md`）；双版本端点的响应主体若本身即无查看者字段的共享类型（如 `reactions/fetch` 匿名版的 `ReactionItem`），同样直接共用；仅存在于 public 节点、无登录版对偶的类型（如 `RegisterRequest`、`AuthTokenPair`）保持常规命名
- 类型命名遵循：
  - 统一数据类型: `XxxItem`（公开固定形状）
  - 匿名视角类型: `PublicXxxItem`（/public/v1 专用，定义在 `api_spec_public.md`）
  - 管理员扩展类型: `AdminXxxItem`（/admin/v1 专用）
  - 拥有者扩展类型: `OwnerXxxItem`（拥有者视角端点专用）
  - 基础信息: `XxxInfo`（嵌入用）
  - 查看者状态: `XxxViewerState`（恒定可空字段的取值类型）
  - 非分页列表: `XxxList`
  - 组合响应: `XxxResult`

### 类型归属与复用规则（契约级）

- **嵌套引用也算引用**：若类型 A 被共享类型 B 作为字段使用，则所有引用 B 的模块都视为引用 A
- **多模块复用**：任一类型在 2 个及以上模块出现（直接或嵌套）必须定义在 `api_spec_types.md`
- **单模块归属**：只在单模块使用的类型必须定义在该模块文档
- **禁止同义重复**：已在 types 定义的类型，模块不得使用同义的字面量/结构硬编码

### 分层与依赖规则（契约级）

| 层 | 组成 | 规则 |
|----|------|------|
| L0 平台横切 | 认证/会话、守卫与权限判定、幂等、限流、存在性防护、地理计算 | 对 L1/L2 透明（中间件级保证，端点文档不重复声明） |
| L1 领域资源 | auth / users / moments / comments / reactions / chat / groups / media / share / reports / notify | 领域间禁止直接调用；协作仅经事件通道（WS + notify）或显式声明的类型/查询依赖 |
| L2 聚合发现 | feed / search / map / system.bootstrap / /admin/v1 | 只读聚合 L1（/admin/v1 例外：可读写，须经权限键 + 审计）；不得绕过 L1 的可见性规则 |

### 类型稳定性（契约级）

**原则：一个端点 = 一种角色 + 一个固定响应形状；不同视角 = 不同端点；字段只做数据选择与隐私过滤，不做视角翻转。**

- 视角分离：拥有者视角、群管理视角、平台管理视角分别由专属端点承载（如 `moments/fetch_created` 返回 `OwnerMomentItem`；`groups/admin/members/fetch` 返回 `AdminGroupMemberItem`；`/admin/v1` 返回 `AdminXxxItem`）
- **恒定 nullable 通则**：权限/隐私条件字段**恒定存在**，`null = 隐藏/不适用`。已废除「字段不存在即不可见」策略——字段缺失只在内容性可选字段（`images?` 等）上出现
  - 隐私受控字段（presence_status / last_online_at / location / stats 等）：恒在，null 表达隐藏/不适用；null 不泄露「隐藏 vs 不存在」差异
  - 查看者状态字段（relation / viewer_state / remark 等）：恒在，null 表达不适用（无关系/非成员）；查看者字段仅存在于登录视角类型（`/api/v1`、`/admin/v1`），public 节点类型不含这些字段
  - 查看者布尔字段：恒为布尔（如 `reacted_by_me` 恒返回，未反应时为 false）
- `next_cursor`：恒在，`next_cursor: string | null`（null 表示没有更多数据）
- 每个 nullable 字段的 null 语义必须在类型注释中写明
- `@auth-only`/`@owner-only`/`@group-admin-only`/`@member-only`/`@participant-only` 注解**全量废除**

### 客户端使用分层（契约级建议）

#### 未登录（Guest Browsing）→ 一律调用 `/public/v1`

**可以做**：浏览公开内容（瞬间/群组/用户资料公共字段）、搜索发现、地图浏览、系统配置、热门标签、分享码落地；完成注册/登录/游客创建/密码重置。

**响应形状**：`PublicXxxItem` 等匿名视角类型——不含任何查看者状态字段；只能访问 `public` 可见性资源。

#### 已登录（Personalized Usage）→ 一律调用 `/api/v1`

**额外能力**：查看者状态字段有值、个性化信息流、私聊/群聊/通知、所有写操作和互动操作；按角色访问 `groups/admin` 守卫子树；平台管理员身份另走 `/admin/v1`。

**登录态切换**：客户端在登录/登出时切换调用节点（同一浏览功能的两个版本端点路径仅节点前缀不同，如 `/public/v1/moments/open_the_door` ↔ `/api/v1/moments/open_the_door`，但类型独立）。**无需随登录态切换的端点**（仅 public 版，登录用户与匿名用户调用同一 URL）：`map/canvas`、`groups/announcements/fetch`、system 的 `meta`/`health`/`announcements/fetch`/`client_config`、auth 生命周期 6 端点（`/public/v1/auth/*`）、分享码解析（`/public/v1/share/fetch`；`/api/v1/share/fetch` 为我的分享列表，是不同功能）——客户端此类模块无需感知登录态。

---

## 字段校验约束（契约级）

本节定义跨模块复用的字段校验规则。各模块接口文档必须显式标注模块级校验约束。

说明：以下约束值为系统默认值，管理员可通过 `AppConfig.limits`（见 `api_spec_admin.md`）动态调整部分限制，客户端通过 `ClientConfig.limits`（见 `api_spec_system.md`）获取生效后的值。

**模块级覆盖**：各模块可根据业务语义收窄（不可放宽）本节定义的全局约束。覆盖时必须在模块 spec 的接口说明中显式标注覆盖后的约束值及原因。未显式标注的字段使用本节全局默认值。

### 通用字段约束

| 字段 | 约束 | 说明 |
|-----|------|------|
| `nickname` | 1-30 字符 | 用户昵称 |
| `password` | 8-128 字符，至少含字母和数字 | 密码复杂度 |
| `login_id` | 3-64 字符 | 登录标识（手机号/邮箱/用户名） |
| `bio` | 0-500 字符 | 个人简介 |
| `tags` | 每个标签 1-30 字符，默认最多 10 个 | 用户/群组/瞬间标签 |
| `description`（moment） | 1-2000 字符 | 瞬间描述 |
| `content`（comment） | 1-500 字符 | 评论内容 |
| `content`（message） | 1-5000 字符 | 聊天消息内容 |
| `image_file_ids` | 最多 9 个 | 瞬间图片 |
| `mentions` | 最多 20 个 | @ 提及的用户 ID 列表 |
| `description`（report） | 0-1000 字符 | 举报详细说明，`reason` 为 `'other'` 时建议填写 |
| `reason`（admin） | 0-500 字符 | 平台管理操作原因 |
| `remark` | 0-100 字符 | 用户备注 |
| `location_name` | 0-100 字符 | 位置名称 |

### 群组字段约束

| 字段 | 约束 | 说明 |
|-----|------|------|
| `name` | 1-50 字符 | 群组名称 |
| `description` | 0-500 字符 | 群组描述 |
| `password` | 8-32 字符 | 群组密码（设置时） |
| `max_members` | 2-2000 | 群组最大成员数 |
| `duration_minutes`（mute） | 1-43200（最多 30 天） | 禁言时长 |
| `message`（invite/apply） | 0-200 字符 | 邀请/申请附言 |

### 公告字段约束

| 字段 | 约束 | 说明 |
|-----|------|------|
| `title` | 1-100 字符 | 公告标题 |
| `content`（announcement） | 1-5000 字符 | 公告内容 |

### 搜索与地理约束

| 字段 | 约束 | 说明 |
|-----|------|------|
| `keyword` | 1-100 字符 | 搜索关键字 |
| `prefix` | 1-50 字符 | 搜索建议前缀 |
| `radius` | 100-50000 米（默认 `5000`） | 搜索半径 |
| `bounds`（map） | north/south ∈ [-90,90]；east/west ∈ [-180,180]；north>=south、east>=west | 地图视口 |

### 平台管理字段约束

| 字段 | 约束 | 说明 |
|-----|------|------|
| `role.name` | 1-50 字符 | RBAC 角色名称 |
| `role.description` | 0-200 字符 | RBAC 角色描述 |
| `duration_days`（ban） | 0-365（0=永久） | 封禁时长；超过 365 天应直接使用 `0=永久` |
| `note`（report handle） | 0-500 字符 | 举报处理备注 |
| `support_email` | 有效 Email 格式 | 支持邮箱 |
| `*_url`（contact） | 有效 URL 格式 | 联系信息 URL |

---

## 补充原则

### 分页与排序约定

#### 分页机制（契约级）

- 所有列表接口使用游标分页，禁止 offset 分页
- `cursor` 为**完全不透明字符串**（客户端禁止解析/构造/修改）。服务端必须对 cursor 进行签名或加密处理
- 无界列表接口统一使用 `PaginationParams` 与 `PaginatedData<T>`（类型定义见 `api_spec_types.md`"分页与元数据"小节）
- 有界/确定数量的数据（如搜索建议、热门标签、聊天最近消息）由服务端决定返回数量，具体值通过 `AppConfig.response_defaults` 配置，客户端不传分页或 limit 参数
- 任意使用 `PaginationParams` 的接口必须返回 `PaginatedData<T>`；不返回分页结构的接口不得使用 `PaginationParams`
- **无更多数据**：`next_cursor` 为 `null` 时表示没有更多数据

#### 分页默认值与模块级覆盖

全局默认值：`limit` default=20, max=100（定义于 `PaginationParams`）。

各模块可按业务特性覆盖 `limit` 的 default 和 max 值，但必须在接口文档的 `Request` 说明中显式标注。未显式标注的接口使用全局默认值。

#### 排序语义（契约级）

- 每个列表接口必须声明默认排序主序、稳定 tie-breaker、以及一致性级别（强一致/弱一致）
- 时间类列表必须提供稳定 tie-breaker（使用资源主键）
- 地理相关列表必须以服务端计算的 `distance` 作为主序，客户端不得本地计算或重排
- **tie-breaker 方向统一 ASC**（资源主键升序），保证跨端点一致

#### 去重（契约级）

客户端必须按资源主键去重（用户：`user_id`；群组：`group_id`；瞬间：`moment_id`；评论：`comment_id`；消息：`message_id`）。

#### 计数器一致性（契约级）

所有 `*_count` 字段（如 `comments_count`、`shares_count`、`members_count`、`followers_count`、`reaction.count`、`read_count` 等）均为**最终一致**，非强一致实时值：

- **更新策略**：服务端采用异步累计（写入队列 + 定时刷新到主存储 + 缓存），容忍秒级到分钟级延迟
- **客户端约束**：不得用于精确对账；不得在收到互动事件后立即期望本地 count +1 与服务端一致；不得轮询 count 字段
- **精度**：服务端保证最终收敛（无互动时 count 稳定），但不保证任意时刻与底层数据完全一致

### 密码安全

- 传输使用 HTTPS；客户端发送明文 (Over TLS)；服务端存储 Hash (Argon2/Bcrypt)

### 限流策略（契约级）

触发限流时返回限流类错误响应，并携带 `Retry-After` 响应头。实际阈值由管理员动态配置并即时生效。

| 接口类型 | 限流维度 | 策略 | 默认限制值 | `RateLimitConfig` 对应字段 |
|---------|---------|------|--------|--------------------------|
| 登录/注册/游客/重置 | IP | 滑动窗口 | 10次/10分钟 | `auth_signin_per_10minutes` 等 |
| 搜索类 | 用户（`/api/v1`）；IP（`/public/v1` 匿名版） | 令牌桶 | 60次/分钟 | `search_per_minute` |
| 通用写操作 | 用户 | 滑动窗口 | 30次/分钟 | `write_per_minute` |
| 业务写操作（动态/评论/举报） | 用户 | 滑动窗口 | 各不同 | `moments_create_per_minute` 等 |
| 文件上传 presign | 用户（按 files 内文件数计） | 滑动窗口 | 100次/小时 | `upload_presign_per_hour` |
| WebSocket 订阅 | 连接 | 固定窗口 | 10次/秒 | `ws_subscribe_per_second` |
| 平台管理查询（/admin/v1 读） | 用户 | 令牌桶 | 100次/分钟 | `admin_read_per_minute` |
| 平台管理写（/admin/v1 写） | 用户 | 滑动窗口 | 30次/分钟 | `admin_write_per_minute` |
| 全局限流（所有请求兜底） | IP+用户 | 固定窗口 | 服务端配置 | `global_per_hour` |
| 用户级全局限流 | 用户 | 固定窗口 | 服务端配置 | `user_per_hour` |

> 各字段的完整定义和取值范围见 `api_spec_admin.md` 的 `RateLimitConfig`。

**匿名请求限流**：`/public/v1` 节点请求（节点本身不解析 Token，一律按匿名处理）按 **IP 维度**限流，防止爬虫批量拉取公开数据。登录请求（`/api/v1`、`/admin/v1`）按用户维度限流；两种维度独立计费、分别触发。

**客户端真实 IP 解析**（IP 维度限流的前提）：

| 优先级 | Header | 说明 |
|--------|--------|------|
| 1 | `CF-Connecting-IP` | Cloudflare 注入的真实客户端 IP，**最可信**，始终为单个 IP |
| 2 | `X-Real-IP` | 备选，单个 IP |
| 3 | `X-Forwarded-For` | 最后手段，取第一个非内网 IP |

**实现要求**：
- 后端必须校验请求确实来自 Cloudflare IP 段，防止绕过 CDN 直连并伪造 `CF-Connecting-IP`
- 若所有 header 均缺失，回退到连接源 IP（仅开发/测试环境）
- `CF-IPCountry` header 可用于地理位置级别的异常检测

**客户端处理**：收到限流错误码后，可按端侧重试策略处理；如响应头包含 `Retry-After`，优先按该值重试。

### 幂等性（契约级）

幂等性用于防止网络重试导致的重复创建或重复副作用。

#### 判定原则

一个接口是否需要 `Idempotency-Key`，取决于**重发是否会真的产生重复结果**，而非「客户端是否会重试」：

- **需要**：重发会产生用户可见的重复内容/数据，且**无其他机制兜底去重**——客户端必须携带 `Idempotency-Key`，服务端据此去重并回放首次成功响应。
- **不需要**：以下三类天然幂等，契约不标注、客户端不传 key：
  - **状态设值/状态转移**：`set_mute`、`set_ban`、`set_role`、`set_favorite`、`reactions/set` 等——重复请求达成同一状态即 no-op 成功。
  - **删除/解散/更新**：删除已删除、解散已解散、更新相同字段，重发结果一致。
  - **有自然去重键的创建**：服务端存在唯一约束或一次性凭证，重发要么返回成功+已存在资源（无歧义）、要么因凭证失效/权限丧失而失败。

需要实现幂等的接口以本节「需要实现幂等的接口」清单为唯一权威。二元判定：要么「需要」、要么「不需要」——不存在「建议」中间档。

#### 需要实现幂等的接口

以下接口客户端必须携带 `Idempotency-Key`：

- 内容创建（无自然去重键，重发=重复内容）：`moments/create`、`comments/create`、`groups/create`、`groups/admin/announcements/create`、`/admin/v1/announcements/create`、`share/create`
- 消息发送（无自然去重键，重发=重复消息骚扰）：`private_messages/send`、`group_messages/send`
- 账号创建：`/public/v1/auth/guest`（每次调用创建独立访客账号）、`/public/v1/auth/register`（唯一约束特例，见下方）

> **唯一约束特例（`/public/v1/auth/register`）**：`login_id` 有唯一约束，重复注册返回 `10400`（login_id 已存在），表面看似自然去重。但这里的 `10400` 有**歧义**——是「我刚提交被自己重试占了」还是「该 login_id 早被别人占用」？客户端无法直接判断本次注册是否已成功，必须额外发起一次 `signin` 才能恢复。为消除这一歧义、让重试能回放首次的 `AuthTokenPair`，`register` 作为全规范唯一特例保留 `Idempotency-Key`。

#### 用自然去重键替代 Idempotency-Key 的接口（无需 key）

| 端点 | 自然去重键 | 重发行为 |
|------|----------|---------|
| `reports/create` | `(reporter_id, target_type, target_id, pending_status)` | 返回成功 + 已有 `report_id`（`is_duplicate: true`），不创建新举报单 |
| `groups/admin/invites/create` | `(group_id, invitee_id, pending)` | 返回已有邀请，不重复创建、不重复通知 |
| `groups/enter`（approval 路径） | `(group_id, applicant_id, pending)` | 返回已有 `application_id`，不重复创建；direct 路径成员关系唯一，天然幂等 |
| `media/complete_upload` | `upload_id` 一次性凭证 | 首次成功后凭证失效，重发返回相同 `file_id` |
| `/public/v1/auth/reset_password` | 一次性验证码 | 首次成功后验证码失效，重发 → `10300` |
| `auth/change_password` | 旧密码凭证 | 首次成功后旧密码失效，重发 → `10401` |
| `groups/admin/transfer_ownership` | 群主身份（调用方权限） | 首次后调用方失群主权限，重发 → `10200` |

#### Idempotency-Key 格式

- **Header 名称**：`Idempotency-Key`
- **格式**：`{client_id}-{timestamp_ms}-{random}`
- **长度**：32-64 字符；**字符集**：`[a-zA-Z0-9-_]`
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

- **重试**：必须使用**相同**的 `Idempotency-Key`；**新请求**：必须生成**新**的 key
- **收到 `22002`（处理中）**：等待 1-2 秒后使用相同 key 重试
- **收到 `22001`/`22000`**：生成新 key 后重试
- **收到缓存回放响应**：视为成功，直接使用返回数据
- 具体幂等错误码定义与客户端行为详见 `api_spec_errors.md`；各端点的「错误码」小节不应重复列出这三种幂等错误码

### 地理计算（契约级）

- `distance/bearing_degrees` 等地理计算结果必须由服务端返回，客户端禁止本地计算用于排序
- 服务端可基于隐私设置、关系与登录态对位置/距离做降精度或匿名化处理
- 客户端只用于展示，不得将距离/方位作为精确导航或本地排序依据
