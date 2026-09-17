# HotSpot API - 未注册用户节点（public）

> 本文档定义 `/public/v1` 节点的全部 RPC 接口：匿名浏览发现、账号生命周期入口与系统信息。
> 会话自助管理端点（登出/改密/会话撤销）位于 `/api/v1/auth/`（见 `api_spec_auth.md`）。

## 模块信息

- **节点**: `/public/v1`（未注册用户，纯匿名）
- **引用**:
  - 基础规范：`api_spec_base.md`（节点身份语义、命名规范、类型稳定性）
  - 通用类型：`api_spec_types.md`
  - 守卫与权限：`api_spec_authz.md`
  - 错误码：`api_spec_errors.md`

## 节点级规则（契约级）

- **纯匿名**：本节点**完全不解析 Token**（Cookie/Header 中的 Token 一律忽略）。响应形状与调用者是否恰好持有 Token 无关；不存在任何隐式身份增强
- **无身份类资源约束**：本节点仅适用可见性过滤（privacy_level / searchable 系列 / social_lists_visible 等），无 owner/participant/member/self 类约束
- **限流**：本节点请求一律按 **IP 维度**限流（见 `api_spec_base.md`「限流策略」）
- **类型隔离**：本节点端点的 Request/Response 使用专属类型，与 `/api/v1`（含 groups/admin 子树）、`/admin/v1` **禁止共用**（规则见下方「类型体系」）
- **镜像判定原则**：仅当登录版存在真实增量——查看者状态字段，或可见性判定依赖身份（如 `reactions/fetch` 的私聊消息仅参与者可见）——才在 `/api/v1` 设登录镜像版；无任何增量的端点仅在本节点提供（如 `map/canvas`、`share/open_the_door`、`groups/announcements/fetch`、system 的 meta/health/announcements/fetch/client_config）

## 类型体系（契约级）

**双版本端点的类型隔离**：同一功能的匿名版（本节点）与登录版（`/api/v1`）是两个独立端点，Request/Response 主体类型**各自独立**：

- 匿名版类型一律 `Public` 前缀（`PublicXxxRequest` / `PublicXxxItem` / `PublicXxxResult`），定义于本文档
- 登录版保持原名（`XxxRequest` / `XxxItem`），定义于对应模块文档
- 匿名版类型**不含查看者状态字段**（`relation` / `social_context` / `viewer_state` / `is_favorited` / `reacted_by_me` / `remark` / `friends_in_group` / `recommendation` 等）——这些字段只存在于登录版类型中；嵌套结构里含查看者字段的类型同样必须拆出 Public 版（如 `PublicReaction`）

**无对偶类型不加前缀**：仅存在于本节点、没有登录版对偶的端点类型保持常规命名（如 `RegisterRequest`、`AuthTokenPair`，均定义于本文档；`ApiVersionMeta` 定义于 `api_spec_types.md`）。

**共享 Domain Primitives**：无视角的基础值对象（`GeoPoint` / `Location` / `RelativeLocation` / `PaginationParams` / `PaginatedData` / `UserInfo` / `UserStats` / `MediaAsset` / `TagItem` / `GroupInfo` 等，定义于 `api_spec_types.md`）跨节点共享，不复制；双版本端点的响应主体若本身即无查看者字段的共享类型（如 `reactions/fetch` 匿名版的 `ReactionItem`），同样直接共用。

## 单设备单会话模型（契约级）

HotSpot 采用「**同一用户在同一设备上只有一个活跃 session**」的会话模型。`device_id`（`DeviceId` 类型）是设备身份的主键，与 `user_id` 一起构成 session 的唯一性约束 `(user_id, device_id)`。

**设备标识的稳定性**：
- **App**：客户端首次启动生成 UUID v4 并持久化到 Keychain（iOS）/ Keystore（Android）；卸载重装视为新设备（新 device_id）
- **Web**：由服务端通过独立 `device_id` Cookie 下发（见 `api_spec_base.md`「Web 端 Cookie 契约」），登出不清除；不同浏览器档案/无痕窗口天然对应不同 device_id
- 客户端**不得**自造或预测 device_id（App 端虽本地生成，但一经生成即稳定，禁止为绕过驱逐而轮换）

**驱逐规则**：
- `signin` / `register` / `guest` / `reset_password`（自动登录）成功时，若 `(user_id, device_id)` 已有活跃 session，服务端立即将旧 session 的 access_token + refresh_token 写入黑名单，并向旧 session 的在线 WebSocket 连接推送 `SessionRevoked` 事件（见 `api_spec_websocket.md`），新 session 取而代之
- 驱逐写入黑名单**立即生效**，refresh 的 30 秒宽限期**不覆盖显式驱逐**（见 `refresh_token` 的「驱逐优先于宽限」）

**跨设备共存**：用户可在多个设备（手机、平板、Web）同时登录，各自 `device_id` 不同、互不驱逐。`/api/v1/auth/fetch_sessions` 返回所有活跃设备的 session 列表。

**与既有失效机制的协同**：
- `change_password`：撤销除当前外所有 session（跨设备）——强度高于单设备驱逐，用于密码泄露场景
- `reset_password`：撤销所有 session（含当前）——最强，用于忘密码重置
- `signout`：仅撤销当前 session
- `revoke_session` / `revoke_other_sessions`：用户主动管理多设备

## 客户端类型与 Token 投递策略

> Cookie 名称与属性的完整定义见 `api_spec_base.md` 的 "Web 端 Cookie 契约" 章节。

```typescript
// 认证响应负载：按 client_type 拆分
// Web：仅通过 HttpOnly Cookie 传递 token，响应体不包含任何 token 字段
// App：access_token/refresh_token 出现在 body，客户端负责安全存储
// 
// JWT 设计原则：
// - JWT (access_token) 中只存储不可变信息：user_id
// - 可变信息（nickname、头像等）应通过 /api/v1/users/account 获取
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

### Web 端（`client_type='web'`）

Token 通过服务端 `Set-Cookie` 响应头投递，**禁止**使用 `Authorization` Header。Cookie 名称、Path 与属性见 `api_spec_base.md`「Web 端 Cookie 契约」。

**响应体**：返回 `ApiResponse<AuthTokenPairWeb>`，`data` 仅含判别字段 `client_type: 'web'`（Token 仅在 Cookie 中传递，响应体不包含任何 token 值）。

### App 端（`client_type='app'`）

Token 通过响应体投递：返回 `ApiResponse<AuthTokenPairApp>`，响应体中包含 `access_token`、`refresh_token` 及各自过期时间；客户端负责安全存储（Keychain / Keystore）。后续调用 `/api/v1`、`/admin/v1` 端点时以 `Authorization: Bearer <token>` 携带。

### Cookie 更新时机（Web 端）

| 场景 | `access_token` Cookie | `refresh_token` Cookie |
|---|---|---|
| 登录 / 注册 / 访客 | `Set-Cookie` 设置 | `Set-Cookie` 设置 |
| 刷新 Token | `Set-Cookie` 更新 | `Set-Cookie` 更新 |
| 登出（`/api/v1/auth/signout`） | `Max-Age=0` 清除 | `Max-Age=0` 清除 |

Cookie 的 Path 覆盖与自动携带规则（`/public/v1` 忽略、refresh 精确 Path 等）见 `api_spec_base.md`「Web 端 Cookie 契约」。

## 端点总览

| 端点 | 说明 |
|------|------|
| `POST /public/v1/auth/register` | 用户注册（手机号/邮箱型需验证码） |
| `POST /public/v1/auth/send_register_code` | 发送注册验证码（手机号/邮箱所有权验证） |
| `POST /public/v1/auth/signin` | 用户登录 |
| `POST /public/v1/auth/guest` | 创建访客用户（临时注册） |
| `POST /public/v1/auth/refresh_token` | 刷新 Token（凭证特例） |
| `POST /public/v1/auth/forgot_password` | 忘记密码（发送重置验证码） |
| `POST /public/v1/auth/reset_password` | 验证码重置密码（自动登录） |
| `POST /public/v1/users/search` | 搜索用户（关键字/标签/位置组合） |
| `POST /public/v1/users/open_the_door` | 用户资料详情（可选主页首屏聚合） |
| `POST /public/v1/users/fetch_friends` | 好友列表（user_id 必传） |
| `POST /public/v1/users/fetch_following` | 关注列表（user_id 必传） |
| `POST /public/v1/users/fetch_followers` | 粉丝列表（user_id 必传） |
| `POST /public/v1/users/fetch_groups` | 用户群组列表（user_id 必传，公开视角） |
| `POST /public/v1/users/fetch_moments` | 用户瞬间历史（user_id 必传，公开视角） |
| `POST /public/v1/users/fetch_reacted_moments` | 用户反应过的瞬间列表（user_id 必传） |
| `POST /public/v1/users/batch_fetch` | 批量获取用户公开信息 |
| `POST /public/v1/moments/search` | 搜索瞬间（关键字/标签；热门模式承接发现页首屏） |
| `POST /public/v1/moments/open_the_door` | 瞬间详情 |
| `POST /public/v1/moments/fetch_nearby` | 附近公开瞬间（位置发现流，拉新路径） |
| `POST /public/v1/comments/fetch` | 顶层评论列表 |
| `POST /public/v1/comments/fetch_replies` | 子评论列表 |
| `POST /public/v1/comments/open_the_door` | 评论详情 |
| `POST /public/v1/groups/search` | 搜索群组（关键字/标签/位置组合） |
| `POST /public/v1/groups/open_the_door` | 群组详情 |
| `POST /public/v1/groups/announcements/fetch` | 群公告列表（仅 public 版，无登录镜像） |
| `POST /public/v1/groups/members/fetch` | 成员列表 |
| `POST /public/v1/groups/members/search` | 搜索成员 |
| `POST /public/v1/search/suggest` | 综合搜索建议 |
| `POST /public/v1/map/canvas` | 地图视口渲染（热力/热点；仅 public 版，无登录镜像） |
| `POST /public/v1/share/open_the_door` | 分享码落地（公开访问；仅 public 版，无登录镜像） |
| `POST /public/v1/reactions/fetch` | 反应列表（统一三类目标） |
| `POST /public/v1/system/meta` | API 版本元信息 |
| `POST /public/v1/system/health` | 健康检查 |
| `POST /public/v1/system/announcements/fetch` | 系统公告列表 |
| `POST /public/v1/system/client_config` | 客户端公开配置 |
| `POST /public/v1/system/bootstrap` | 匿名冷启动聚合 |

## Public 类型定义

以下类型为本节点专属（登录版对应类型见各模块文档）：

```typescript
// 匿名视角反应聚合（Reaction 去 reacted_by_me——查看者字段仅存在于登录版 Reaction）
interface PublicReaction {
  emoji: string;
  count: number;
}

// 用户公开信息（UserItem 去 relation / social_context）
interface PublicUserItem {
  user: UserInfo;
  bio: string | null;
  tags: string[];

  // === 隐私受控字段（恒定 nullable：null = 对方隐私设置隐藏；与登录态无关） ===
  presence_status: UserPresenceStatus | null;
  last_online_at: string | null;
  location: RelativeLocation | null;
  stats: UserStats | null;
}

// 关系列表项（RelatedUserItem 去 remark / is_mutual——两者均为查看者字段；
// 互关可由登录版 user.relation 计算，匿名视角无需表达）
// - followed_back_at：仅 fetch_friends 返回；其余端点恒为 null
interface PublicRelatedUserItem {
  user: PublicUserItem;
  followed_at: string;
  followed_back_at: string | null;
}

// 瞬间公开类型（MomentItem 去 is_favorited / recommendation；reactions 用 PublicReaction）
interface PublicMomentItem {
  moment_id: string;
  description: string;         // 软删除时为空字符串，客户端展示"内容已删除"占位
  images: MediaAsset[];
  tags: string[];
  mentions: string[];
  created_at: string;
  comments_count: number;
  shares_count: number;
  reactions: PublicReaction[];
  user: UserInfo;
  is_deleted: boolean;         // 公开视角恒 false（软删瞬间在匿名端点不可达，字段保留以与登录版同构）
  location: RelativeLocation | null;
}

// 评论公开类型（CommentItem 的 reactions 换为 PublicReaction）
interface PublicCommentItem {
  comment_id: string;
  user: UserInfo;
  content: string;             // 软删除时为空，客户端展示"该评论已删除"占位
  parent_id: string | null;
  mentions: string[];
  created_at: string;
  replies_count: number;
  reactions: PublicReaction[];
  is_deleted: boolean;
  top_replies: PublicCommentItem[];  // 顶层评论列表场景返回前 N 条回复预览
}

// 群组公开类型（GroupItem 去 viewer_state / friends_in_group）
interface PublicGroupItem {
  group: GroupInfo;
  description: string | null;
  tags: string[];

  // 加入门槛（对所有人可见——用户加入前必须知晓加入方式与是否需审核）
  join_method: GroupJoinMethod;
  requires_approval: boolean;

  location: RelativeLocation | null;
  last_activity_at: string | null;
  online_members_count: number | null;
  next_activity_at: string | null;
  created_at: string | null;
  owner_id: string | null;
}

// 公开成员列表项（members/fetch 与 members/search 共用；字段与管理视角无关联，独立定义）
interface PublicGroupMemberItem {
  user: UserInfo;
  role: GroupRole;
  joined_at: string;
  presence_status: UserPresenceStatus | null;  // null = 成员 presence_visible=false
  last_online_at: string | null;
}

// 群公告列表（条目 GroupAnnouncementItem 定义于 api_spec_types.md；无登录版对偶——公告为公开读、无查看者字段）
interface GroupAnnouncementList {
  items: GroupAnnouncementItem[];
}

// 用户资料详情组合响应
interface PublicUserProfileResult {
  user: PublicUserItem;
  recent_moments: PaginatedData<PublicMomentItem> | null;  // 仅 include_recent_moments=true 时返回；否则 null
}

// 批量获取用户响应：与登录版共用 UsersBatchResult（Record<string, UserInfo>，无查看者字段的共享类型，
// 定义于 api_spec_types.md）

// 搜索建议响应（user/group/moment 分支使用 Public 载荷；tag 无视角共享 TagItem）
interface PublicSearchSuggestResponse {
  items: PublicSearchSuggestItem[];
}

type PublicSearchSuggestItem =
  | { type: 'tag'; payload: TagItem }
  | { type: 'user'; payload: PublicUserItem }
  | { type: 'group'; payload: PublicGroupItem }
  | { type: 'moment'; payload: PublicMomentSuggestItem };

// 瞬间建议项（MomentSuggestItem 的 Public 版）
interface PublicMomentSuggestItem extends MomentSummary {
  author: UserInfo;
  location: RelativeLocation | null;
}

// 分享落地响应（share/payload/preview 结构见 api_spec_types.md；无登录版对偶）
interface ShareDetail {
  share: ShareBase;
  payload: SharePayload;
  preview: SharePreview | null;   // 目标资源不可访问时为 null；分享撤销/过期整体 10300
}

// 匿名冷启动聚合响应（仅公共数据；登录态字段在 /api/v1/system/bootstrap）
interface PublicBootstrapData {
  bootstrap_version: string;
  // === 缓存命中时以下键整体省略（客户端使用缓存）；未命中/无缓存时必填 ===
  // 命中判定：键不存在（而非 === null）。本接口是 base「类型稳定性」
  // 「字段缺失仅限内容性可选字段」通则的显式例外。
  meta?: ApiVersionMeta;
  client_config?: ClientConfig;
}
```

## 端点详情

## 认证（auth）

### POST /public/v1/auth/send_register_code

发送注册验证码（手机号/邮箱所有权验证）。与 `register` 对仗：先验证所有权，再凭码注册。

- **节点**: /public/v1（纯匿名）

**安全说明**：
- 无论 `login_id` 是否已被注册，均返回成功响应，不暴露账号存在性（与 `forgot_password` 防枚举策略一致）；已占用的 `login_id` 对应验证码无效，注册最终仍以 `10400` 拒绝
- 仅手机号/邮箱型 `login_id` 支持本端点；用户名型（不含 '@' 且非纯数字）请求失败（10000）
- 发送频率由 `AppConfig.rate_limits.auth_register_per_10minutes` 控制（与注册共用限流桶）；验证码有效期由服务端内部配置决定（不对外暴露）

**Request**: `SendRegisterCodeRequest`
```typescript
interface SendRegisterCodeRequest {
  login_id: string;       // 注册时使用的手机号/邮箱
  client_type: ClientType;
}
```

**Response**: `ApiResponse`

---

### POST /public/v1/auth/register

用户注册。

- **节点**: /public/v1（纯匿名）
- **幂等**: 是（客户端必须携带 `Idempotency-Key`）；`auth/register` 是全规范唯一保留幂等键的「有唯一约束」接口，理由见 `api_spec_base.md`「唯一约束特例」

**所有权验证（契约级）**：
- `login_id` 为手机号/邮箱型时，必须携带有效 `verification_code`（经 `auth/send_register_code` 发送到该手机号/邮箱），防止抢占他人联系方式注册；验证码错误或过期返回 `10401`（旧凭证校验失败，不触发登录态流程）
- `login_id` 为用户名型时 `verification_code` 必须为 null（同传视为参数冲突 10000）
- **过渡约束**：验证码服务接入前，手机号/邮箱型注册暂不可用（返回 10000），仅支持用户名型——与 `users/upgrade` 的同款过渡声明；接入后本约束解除

**Request**: `RegisterRequest`
```typescript
interface RegisterRequest {
  login_id: string;              // 手机号/邮箱/用户名（注册后作为登录凭证；格式判定规则见 api_spec_base.md「通用字段约束」login_id）
  verification_code: string | null;  // 手机号/邮箱型必填（所有权验证）；用户名型为 null
  nickname: string;
  password: string;
  client_type: ClientType;
  device_id: DeviceId | null;    // App 端必填非 null；Web 端为 null（服务端从 device_id Cookie 读取）
}
```

**Response**: `ApiResponse<AuthTokenPair>`

**平台开关（契约级）**：`features.enable_public_registration`（`api_spec_types.md` FeatureSwitches）由平台管理员配置，**服务端强制**——关闭时本端点直接返回 `10400`（message「当前未开放注册」），客户端开关仅作入口隐藏，不能替代服务端校验。

**错误码**: `10400`（login_id 已被占用 / 注册未开放） `10401`（验证码错误/过期）

---

### POST /public/v1/auth/signin

用户登录（密码方式）。

- **节点**: /public/v1（纯匿名）

**Request**: `SigninRequest`
```typescript
interface SigninRequest {
  login_id: string;            // 手机号/邮箱/用户名
  password: string;            // 密码
  client_type: ClientType;
  device_id: DeviceId | null;  // App 端必填非 null；Web 端为 null（服务端从 device_id Cookie 读取）
}
```

**Response**: `ApiResponse<AuthTokenPair>`

**单设备单会话驱逐**：登录成功时，服务端查找 `(user_id, device_id)` 已有的活跃 session，若存在则立即将其 access_token + refresh_token 写入黑名单（沿用 `change_password` 机制），并向被驱逐设备的在线 WebSocket 连接推送 `SessionRevoked` 事件（见 `api_spec_websocket.md`）。新 session 取代旧 session。

**安全说明**：用户不存在和密码错误统一返回 `10100`（认证失败），不区分具体原因，防止账号枚举攻击。

**错误码**: `10100`

---

### POST /public/v1/auth/guest

创建访客用户（临时注册，`user_type='temp'`；与正式注册用户在权限上完全同权，仅账号生命周期不同，可通过 `/api/v1/users/upgrade` 转正）。

- **节点**: /public/v1（纯匿名）
- **幂等**: 是（客户端必须携带 `Idempotency-Key`）

**Request**: `GuestRequest`
```typescript
interface GuestRequest {
  nickname: string | null;  // 可选；null 时服务端自动生成默认昵称（如「游客·8k2f」），进入后引导修改，降低访客模式进入摩擦
  client_type: ClientType;
  device_id: DeviceId | null;  // App 端必填非 null；Web 端为 null（服务端从 device_id Cookie 读取）
}
```

**Response**: `ApiResponse<AuthTokenPair>`

**平台开关（契约级）**：`features.enable_guest_register` 关闭时本端点返回 `10400`（message「当前未开放访客模式」），服务端强制（同 register）。

---

### POST /public/v1/auth/refresh_token

刷新 Token。

- **节点**: /public/v1（凭证特例：以请求体中的 refresh_token 为凭证，无效/过期/被撤销返回 `10100`；节点本身不解析 access_token）

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

**client_type 约束**（服务端按 `refresh_token` 值是否非空判定）：
- `client_type='web'`：`refresh_token` 值必须为空——推荐缺省该键，显式传 `null` 亦可接受；传非空字符串则请求失败（`10000`）
- `client_type='app'`：请求体必须包含非空 `refresh_token`（缺少或为空则请求失败）

**Request**: `RefreshTokenRequest`
```typescript
interface RefreshTokenRequest {
  client_type: ClientType;
  device_id: DeviceId | null;  // 用于校验 refresh_token 归属当前设备，防止跨设备 token 挪用。App 端必填非 null；Web 端为 null（服务端从 device_id Cookie 读取）
  refresh_token: string | null;  // App 端必填非空；Web 端必须为空值（推荐缺省，null 亦可；使用 Cookie）
}
```

**Response**: `ApiResponse<AuthTokenPair>`

**错误码**: `10100`

---

### POST /public/v1/auth/forgot_password

忘记密码（发送重置验证码）。与 `reset_password` 对仗：忘记 → 重置。

- **节点**: /public/v1（纯匿名）

**安全说明**：
- 无论 `login_id` 是否存在，均返回成功响应，不暴露账号是否注册
- 验证码发送频率由 `AppConfig.rate_limits.auth_reset_per_10minutes` 控制；验证码有效期由服务端内部配置决定（不对外暴露）

**Request**: `ForgotPasswordRequest`
```typescript
interface ForgotPasswordRequest {
  login_id: string;       // 注册时使用的邮箱或手机号
  client_type: ClientType;
}
```

**Response**: `ApiResponse`

---

### POST /public/v1/auth/reset_password

验证验证码并重置密码。重置成功后自动登录并返回 Token。

- **节点**: /public/v1（纯匿名）

**自然去重**：验证码为一次性凭证，首次成功后即失效——重发同一请求返回 `10300`（验证码已用/过期），不会重复改密。无需 `Idempotency-Key`（判定见 `api_spec_base.md`「自然去重键」）。

**安全说明**：
- 重置密码成功后，服务端自动吊销该账号所有现有会话（与 `change_password` 一致）
- 所有被吊销会话的 token 写入黑名单
- 被吊销设备的 WebSocket 连接会收到 `SessionRevoked`（`reason='password_changed'`，与 `change_password` 一致，见 `api_spec_websocket.md`）后被服务端主动断开
- 重置成功后自动登录：按 `(user_id, device_id)` 建立新会话；若该设备已有活跃会话则立即驱逐（与 `signin` 的驱逐规则一致）

**Request**: `ResetPasswordRequest`
```typescript
interface ResetPasswordRequest {
  login_id: string;          // 注册时使用的手机号/邮箱
  verification_code: string;
  new_password: string;
  client_type: ClientType;
  device_id: DeviceId | null; // App 端必填非 null；Web 端为 null（服务端从 device_id Cookie 读取）
}
```

**Response**: `ApiResponse<AuthTokenPair>`

**错误码说明**：
- `10300` 用于"重置请求不存在或已过期"（验证码已用/过期）与"login_id 不存在"——两者 message 一致，禁止返回"用户不存在"等暴露账号存在性的 message（与 `forgot_password` 的安全策略一致，防账号枚举）

**错误码**: `10300`

---

## 用户公开资料（users）

> 与 `/api/v1/users/` 同名端点为双版本关系：本节点返回匿名视角（`PublicUserItem`，无 relation / social_context）；登录视角（含查看者状态）见 `api_spec_user.md`。

### POST /public/v1/users/search

统一搜索用户。支持关键字、标签、位置范围三种条件的自由组合。

**radius 语义（契约级）**：
- `radius` 仅**显式提供时**按半径过滤结果；缺省时不做半径过滤（keyword/tag 搜索为全局匹配）
- `sort_by='distance'` 的距离计算依赖 `reference_point`（按 `api_spec_types.md`「reference_point 解析规则」回退链解析；匿名请求跳过「登录用户最近位置」一环）；`relevance`/`newest` 排序不依赖位置

**隐私过滤（契约级，searchable 系列开关）**：与登录版一致（`searchable` / `nickname_searchable` / `phone_searchable` / `location_searchable=false` 的用户不命中对应维度）。

- **排序**:
  - 默认 `sort_by='relevance'`：按相关性优先排序，同分时以最近活跃时间做稳定 tie-breaker
  - `sort_by='distance'`：按距离由近到远排序，距离相近时体现关键词相关性
  - `sort_by='newest'`：按注册时间倒序
  - **稳定 tie-breaker**：任意排序模式下，当主序字段和次序均相同时，以 `user_id ASC` 作为最终稳定排序
  - 一致性级别为弱一致

**Request**: `PublicSearchUsersRequest`
```typescript
interface PublicSearchUsersRequest {
  keyword: string | null;    // 关键字搜索（匹配昵称、bio 等；为手机号格式时同时按手机号精确匹配，受 phone_searchable 控制，与登录版一致）
  tag: string | null;        // 标签筛选
  radius: number | null;     // 搜索半径（米）；显式提供时按半径过滤，缺省不过滤
  sort_by: 'relevance' | 'distance' | 'newest' | null;  // 默认: relevance
  reference_point: GeoPoint | null;
  pagination: PaginationParams | null;
}
// 至少提供 keyword、tag、radius 中的一个；不满足时请求失败（10000）
```

**Response**: `ApiResponse<PaginatedData<PublicUserItem>>`

---

### POST /public/v1/users/open_the_door

获取用户公开资料详情（可选主页首屏聚合）。

**隐私说明**：用户存在但对其资料设置了完全隐藏（或调用方被对方拉黑等隐私限制）时，返回 `10300`（按不存在处理，不暴露存在性）；用户不存在时返回 `10300`。

**主页聚合能力**：`include_recent_moments=true` 时附加主页首屏数据（资料 + 近期公开瞬间）。

- **排序**: `recent_moments` 内部按 `created_at DESC`（最新在前），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**设计要点**：
- `user.stats` 详情场景返回（含 moments_count/friends_count/followers_count/following_count/groups_count），客户端据此渲染主页各 tab 的数量角标
- 好友/关注/粉丝/群组的**明细列表**用本节点 `users/fetch_friends`、`users/fetch_following` 等按需懒加载
- `recent_moments` 条数控制与翻页策略与登录版一致（见 `api_spec_user.md`）

**Request**: `PublicOpenUserDoorRequest`
```typescript
interface PublicOpenUserDoorRequest {
  user_id: string;
  include_recent_moments: boolean | null;  // 默认 false；true 时返回 recent_moments（主页首屏聚合）
}
```

**Response**: `ApiResponse<PublicUserProfileResult>`

**错误码**: `10300`

---

### POST /public/v1/users/fetch_friends

获取指定用户的好友列表（好友 = 双方互相关注的用户）。

- **排序**: 默认按 `followed_back_at` **时间倒序**（最近成为好友在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `PublicFetchFriendsRequest`
```typescript
interface PublicFetchFriendsRequest {
  user_id: string;   // 必传：要查询的目标用户
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<PublicRelatedUserItem>>`（`followed_back_at` 仅本端点返回）

**错误码**: `10200`（对方 `social_lists_visible=false`，整表隐私拒绝，forbidden_reason=privacy_setting） `10300`（用户不存在）

---

### POST /public/v1/users/fetch_following

获取指定用户的关注列表。

- **排序**: 默认按 `followed_at` **时间倒序**（最近关注在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `PublicFetchFollowingRequest`
```typescript
interface PublicFetchFollowingRequest {
  user_id: string;   // 必传
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<PublicRelatedUserItem>>`

**错误码**: `10200`（privacy_setting） `10300`（用户不存在）

---

### POST /public/v1/users/fetch_followers

获取指定用户的粉丝列表。

- **排序**: 默认按 `followed_at` **时间倒序**（最近关注在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `PublicFetchFollowersRequest`
```typescript
interface PublicFetchFollowersRequest {
  user_id: string;   // 必传
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<PublicRelatedUserItem>>`

**错误码**: `10200`（privacy_setting） `10300`（用户不存在）

---

### POST /public/v1/users/fetch_groups

获取指定用户加入的公开群组（公开主页视角：仅 `is_searchable=true` 的群组）。

**隐私控制**：受 `groups_visible` 设置控制。本端点无 `reference_point` 参数，返回的 `PublicGroupItem.location` 中 distance/bearing 为 null。

- **排序**: 默认按 `joined_at` **时间倒序**（最近加入在前），`group_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `PublicFetchUserGroupsRequest`
```typescript
interface PublicFetchUserGroupsRequest {
  user_id: string;   // 必传
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<PublicGroupItem>>`

**错误码**: `10200`（privacy_setting） `10300`（用户不存在）

---

### POST /public/v1/users/fetch_moments

获取指定用户发布的公开瞬间（匿名视角，仅 `privacy_level='public'`）。

- **排序**: 默认按 `created_at` **时间倒序**（最新瞬间在前），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `PublicFetchUserMomentsRequest`
```typescript
interface PublicFetchUserMomentsRequest {
  user_id: string;   // 必传
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<PublicMomentItem>>`

**错误码**: `10300`

---

### POST /public/v1/users/fetch_reacted_moments

获取指定用户反应过的瞬间（去重，一个瞬间无论被该用户用过几个 emoji 只出现一次）。

**隐私控制**：受 `reacted_moments_visible` 设置控制。

- **排序**: 默认按 `last_reacted_at` **时间倒序**（最近反应在前），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `PublicFetchUserReactedMomentsRequest`
```typescript
interface PublicFetchUserReactedMomentsRequest {
  user_id: string;   // 必传
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<PublicMomentItem>>`

**错误码**: `10200`（privacy_setting） `10300`（用户不存在）

---

### POST /public/v1/users/batch_fetch

批量获取用户公开信息。用于列表等场景中高效解析 `UserInfo`（昵称/头像），消除逐个调用 `open_the_door` 的 N+1 问题。与登录版共用无查看者字段的共享类型 `UsersBatchResult`（定义于 `api_spec_types.md`）。

**Request**: `PublicBatchFetchUsersRequest`
```typescript
interface PublicBatchFetchUsersRequest {
  user_ids: string[];  // 必填，最多 100 个
}
```

**Response**: `ApiResponse<UsersBatchResult>`

---

## 瞬间公开浏览（moments）

> 双版本关系同上：本节点返回 `PublicMomentItem`；登录视角见 `api_spec_moment.md`。

### POST /public/v1/moments/search

统一搜索生活瞬间（关键字/标签）。**不承担附近发现职责**——按位置发现瞬间使用 `moments/fetch_nearby`（见下方，匿名可用）；无输入的热门发现由本端点的热门模式承担。

- **排序**:
  - `sort_by='relevance'`（默认）：按相关性优先；同分时按 `created_at` 倒序，再以 `moment_id DESC` 作为稳定 tie-breaker
  - `sort_by='newest'`：按发布时间倒序
  - `sort_by='popular'`：按互动热度排序（含时间衰减因子；具体衰减算法为服务端实现细节），`moment_id DESC` 作为稳定 tie-breaker
  - 一致性级别为弱一致

**Request**: `PublicSearchMomentsRequest`
```typescript
interface PublicSearchMomentsRequest {
  keyword: string | null;    // 关键字搜索（匹配描述等）
  tag: string | null;        // 标签筛选
  sort_by: 'relevance' | 'newest' | 'popular' | null;  // 默认: relevance
  pagination: PaginationParams | null;
}
// 至少提供 keyword、tag、sort_by='popular' 中的一个，不满足时请求失败（10000）：
// - keyword/tag 均缺省且显式传 sort_by='popular' 时进入「热门模式」（无输入条件的热门公开瞬间，
//   承接匿名发现页首屏，与登录版口径一致）
// - 热门模式下 keyword/tag 必须缺省（与输入条件互斥，同传视为参数冲突 10000）
```

**Response**: `ApiResponse<PaginatedData<PublicMomentItem>>`

---

### POST /public/v1/moments/open_the_door

获取公开瞬间详情。

**隐私说明**：`privacy_level='private'` 的瞬间仅拥有者可见；非拥有者访问 private 瞬间统一返回 `10300`（按不存在处理，不通过 10200 暴露瞬间存在性）。

**Request**: `PublicOpenMomentDoorRequest`
```typescript
interface PublicOpenMomentDoorRequest {
  moment_id: string;
}
```

**Response**: `ApiResponse<PublicMomentItem>`

**错误码**: `10300`

---

### POST /public/v1/moments/fetch_nearby

附近公开瞬间（基于位置的发现流，匿名拉新路径；原 `feed/nearby_moments` 迁移并开放匿名版）。

> 双版本关系：登录镜像 `/api/v1/moments/fetch_nearby`（登录视角见 `api_spec_moment.md`）。附近公开瞬间的可见性不依赖身份（仅 `privacy_level='public'` 且未被过滤），故提供匿名版。

- **排序**: 主序为服务端计算的 `distance` 升序（相对 `reference_point`）；仅当距离差在服务端定义的近似阈值内时，热度/活跃度作为次级信号参与排序；`moment_id DESC` 作为稳定 tie-breaker；一致性级别为弱一致

**Request**: `PublicFetchNearbyMomentsRequest`
```typescript
interface PublicFetchNearbyMomentsRequest {
  radius: number | null;  // 搜索半径（米）；不传默认 5000 并受 search_radius_max 裁剪（位置发现类缺省口径，见 api_spec_base.md「搜索与地理约束」）
  reference_point: GeoPoint | null;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<PublicMomentItem>>`

---

## 评论公开浏览（comments）

### POST /public/v1/comments/fetch

获取指定瞬间的顶层评论列表。

- **排序**: 评论列表默认按 `created_at` **时间倒序**（最新在前），`comment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `PublicFetchCommentsRequest`
```typescript
interface PublicFetchCommentsRequest {
  moment_id: string;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<PublicCommentItem>>`

**错误码**: `10300`

---

### POST /public/v1/comments/fetch_replies

获取指定评论的子评论列表。

- **排序**: 回复列表默认按 `created_at` **时间正序**（最早在前，便于对话阅读），`comment_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `PublicFetchCommentRepliesRequest`
```typescript
interface PublicFetchCommentRepliesRequest {
  comment_id: string;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<PublicCommentItem>>`

**错误码**: `10300`

---

### POST /public/v1/comments/open_the_door

获取单条评论详情（用于分享/举报等场景的评论定位）。

**Request**: `PublicOpenCommentDoorRequest`
```typescript
interface PublicOpenCommentDoorRequest {
  comment_id: string;
}
```

**Response**: `ApiResponse<PublicCommentItem>`

**错误码**: `10300`

---

## 群组公开浏览（groups）

> 本节点仅含公开读；加入/退出/申请/邀请等成员操作位于 `/api/v1/groups/`（见 `api_spec_group.md`），群管理操作位于 `/api/v1/groups/admin/`。

### POST /public/v1/groups/search

统一搜索群组。支持关键字、标签、位置范围三种条件的自由组合。

**radius 语义（契约级）**：`radius` 仅**显式提供时**按半径过滤结果；缺省时不做半径过滤。`sort_by='distance'` 的距离计算依赖 `reference_point`（按 `api_spec_types.md`「reference_point 解析规则」回退链解析）。

- **排序**:
  - 默认 `sort_by='relevance'`：按相关性优先排序
  - `sort_by='distance'`：按距离由近到远排序
  - `sort_by='newest'`：按创建时间倒序
  - `sort_by='popular'`：按成员数/活跃度排序
  - **稳定 tie-breaker**：`group_id ASC`
  - 一致性级别为弱一致

**Request**: `PublicSearchGroupsRequest`
```typescript
interface PublicSearchGroupsRequest {
  keyword: string | null;    // 关键字搜索（匹配名称、描述等）
  tag: string | null;        // 标签筛选
  radius: number | null;     // 搜索半径（米）；显式提供时按半径过滤，缺省不过滤
  sort_by: 'relevance' | 'distance' | 'newest' | 'popular' | null;  // 默认: relevance
  reference_point: GeoPoint | null;
  pagination: PaginationParams | null;
}
// 至少提供 keyword、tag、radius 中的一个；不满足时请求失败（10000）
```

**Response**: `ApiResponse<PaginatedData<PublicGroupItem>>`

---

### POST /public/v1/groups/open_the_door

获取群组公开详情。

**Request**: `PublicOpenGroupDoorRequest`
```typescript
interface PublicOpenGroupDoorRequest {
  group_id: string;
}
```

**Response**: `ApiResponse<PublicGroupItem>`

**错误码**: `10300`

---

### POST /public/v1/groups/announcements/fetch

获取群公告列表（公开读；管理操作在 `/api/v1/groups/admin/announcements/*`）。

**仅 public 版说明**：`GroupAnnouncementItem` 无查看者字段、公告可见性不依赖调用者身份，登录用户与匿名用户行为一致——故不设登录镜像（镜像判定原则见「节点级规则」）。

- **排序**: 按 `is_pinned DESC, created_at DESC` 排序，`announcement_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FetchGroupAnnouncementsRequest`
```typescript
interface FetchGroupAnnouncementsRequest {
  group_id: string;
}
// 返回条数由 AppConfig.response_defaults.group_announcements_count 控制
```

**Response**: `ApiResponse<GroupAnnouncementList>`

**错误码**: `10300`

---

### POST /public/v1/groups/members/fetch

获取成员列表（公开成员信息；管理字段见 `/api/v1/groups/admin/members/fetch`）。

- **可见性**: `members_searchable=false` 时非成员调用返回 `10200` + `not_group_member`
- **排序**: 按 `joined_at` 时间正序（最早加入在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `PublicFetchGroupMembersRequest`
```typescript
interface PublicFetchGroupMembersRequest {
  group_id: string;
  role: GroupRole | null;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<PublicGroupMemberItem>>`

**错误码**: `10200`（not_group_member） `10300`

---

### POST /public/v1/groups/members/search

搜索群组成员。用于大群场景中快速定位特定成员。

- **可见性**: `members_searchable=false` 时非成员调用返回 `10200` + `not_group_member`
- **排序**: 按匹配度优先，同匹配度按 `joined_at` 正序，`user_id ASC` 作为稳定 tie-breaker；一致性级别为弱一致

**Request**: `PublicSearchGroupMembersRequest`
```typescript
interface PublicSearchGroupMembersRequest {
  group_id: string;
  keyword: string;    // 按昵称搜索
  role: GroupRole | null;   // 可选：按角色过滤
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<PublicGroupMemberItem>>`

**错误码**: `10200`（not_group_member） `10300`

---

## 搜索建议（search）

### POST /public/v1/search/suggest

综合搜索建议（标签、用户、群组、瞬间）。

> 双版本关系：登录镜像 `/api/v1/search/suggest`（登录视角见 `api_spec_discovery.md`）。

- **排序**:
  - 完全匹配优先
  - 在匹配度相近时按距离优先，热度/活跃度作为辅助信号
  - 各类型结果内部以对应资源 ID（`user_id`/`group_id`/`moment_id`/tag name）作为稳定 tie-breaker；一致性级别为弱一致

**说明**：
- 建议用于搜索框输入提示与全局发现入口
- **LBS 优化**：`reference_point` 可选，不传时服务端按 `api_spec_types.md`「reference_point 解析规则」回退链解析（匿名请求跳过「登录用户最近位置」一环）
- **类型过滤**：通过 `types` 参数可限定返回特定类型的结果

**Request**: `PublicSearchSuggestRequest`
```typescript
interface PublicSearchSuggestRequest {
  prefix: string;
  reference_point: GeoPoint | null;
  types: SearchSuggestType[];  // 空数组表示返回所有类型
  radius: number | null;  // 搜索半径（米）；不传默认 5000 并受 search_radius_max 裁剪
}
```

**Response**: `ApiResponse<PublicSearchSuggestResponse>`

> 综合搜索结果（用户 + 群组 + 瞬间）由客户端分别调用 `users/search`、`groups/search`、`moments/search` 拼装，各自独立分页，不提供聚合端点。

---

## 地图渲染（map）

### 设计理念

地图模块不依赖任何行政区或 region 概念，完全基于 S2 网格进行空间划分与聚合：

- 所有带位置的实体（用户、群组、动态）在服务端被投影到服务端选定层级的 S2 网格单元（层级由视口跨度与数据密度自动决定，对客户端不透明）
- 服务端根据当前视口的物理跨度和预估结果规模，自动选择合适的 S2 层级（`level`），并使用 S2 覆盖算法确定视口内的网格集合
- 在大尺度视野下返回**网格热力统计**，在小尺度视野下返回**具体热点对象列表**，对客户端透明
- 查询实现依赖预聚合的 S2 统计表，而不是直接扫描明细记录

### 类型定义（map 域专属）

```typescript
// === 地图渲染（S2 网格）===

// 过滤器：限定参与统计/热点的类别（'all' 或未传：用户/群组/动态都参与）
type MapDataType = 'user' | 'group' | 'moment' | 'all';

interface Bounds {
  north: number;  // [-90, 90]
  south: number;  // [-90, 90]
  east: number;   // [-180, 180]
  west: number;   // [-180, 180]
}
// 约束：north >= south；east >= west（允许退化视口）。
// 暂不支持跨国际日期变更线视口（需由客户端拆分请求）。

// 地图视口：仅由经纬度边界决定，不需要客户端提供 zoom
interface MapViewport {
  bounds: Bounds;
}

interface TimeRangeParams {
  start_time: string | null;  // UTC ISO 8601
  end_time: string | null;    // UTC ISO 8601；若同时提供，必须满足 start_time <= end_time
}

interface MapHeatBreakdown {
  user: number | null;   // 该网格内用户数量
  group: number | null;  // 该网格内群组数量
  moment: number | null; // 该网格内动态数量
}

// S2 网格的标识。为避免 JS 精度问题，对外暴露的是 S2 cell 的字符串 token。
type S2CellToken = string;

interface MapHeatBucket {
  cell_token: S2CellToken;  // S2 单元格的字符串 token，可用于前端作为 key 或调试
  level: number;            // 该单元格所在的 S2 level
  location: GeoPoint;       // 聚合单元的代表坐标（例如 S2 单元格的中心点）
  total_count: number;      // 当前过滤条件下，该网格内参与统计实体的总数
  breakdown: MapHeatBreakdown | null;  // 按类型拆分；当 data_type = 'all' 时返回完整三分类拆分
}

interface MapHeatCanvas {
  items: MapHeatBucket[];
}

type HotspotObjectType = 'user' | 'group' | 'moment';

interface MapHotspotItem {
  object_type: HotspotObjectType; // 对象类型
  object_id: string;              // 业务 ID（user_id / group_id / moment_id）
  title: string;                  // 昵称 / 群名 / 动态标题
  subtitle: string | null;        // 简短描述 / 地点名等
  cover_url: string | null;       // 头像 / 群头像 / 动态封面
  // 位置统一使用 RelativeLocation（含 distance/bearing），与 search 各端点对齐
  // distance/bearing 相对于视口几何中心计算，由服务端计算
  location: RelativeLocation;
  created_at: string | null;      // 对于动态等有时间语义的对象可返回创建时间
}

interface MapCanvasFilter {
  data_type: MapDataType | null;
  time_range: TimeRangeParams | null;
}
```

### POST /public/v1/map/canvas

根据视口范围和 S2 网格预聚合，自动返回热力统计（`heatmap`）或热点明细（`hotspots`）中的一种。

**仅 public 版说明**：地图聚合仅含公开可见性对象，响应无查看者字段、可见性过滤不依赖调用者身份，登录用户与匿名用户行为一致——故不设 `/api/v1` 登录镜像（镜像判定原则见「节点级规则」）。

**Request**: `MapCanvasRequest`
```typescript
interface MapCanvasRequest {
  viewport: MapViewport;         // 覆盖区域由 bounds 决定
  filter: MapCanvasFilter | null;
  // pagination 仅在返回 hotspots 时使用；limit 覆盖全局默认为 default=50、max=200
  pagination: PaginationParams | null;
  // 注：无 reference_point 字段。distance/bearing 的计算锚点 = viewport 几何中心。
}
```

**Response**: `ApiResponse<MapCanvasResponse>`
```typescript
// 响应为 tagged union：模式由服务端按视口尺度与数据密度决定（大尺度→heatmap，小尺度→hotspots）
type MapCanvasResponse =
  | { mode: 'heatmap'; heatmap: MapHeatCanvas }
  | { mode: 'hotspots'; hotspots: PaginatedData<MapHotspotItem> }
  | { mode: 'empty' };   // 视口内无足够数据时返回空态
```

**行为约束（契约级）**：

- 服务端基于 `viewport.bounds` 计算视口物理跨度，并结合预估结果规模，自动决定返回统计还是明细
- **互斥保证**：同一次响应中 `heatmap` 与 `hotspots` 最多出现一个字段，不会同时返回
- 当视口范围内无任何已知对象时，既不返回 `heatmap` 也不返回 `hotspots`（返回 `mode: 'empty'`）
- 仅在返回 `hotspots` 时支持游标分页（`pagination` 生效）；`limit` 默认 50、最大 200
- **排序**：主序为服务端计算的 `distance` 升序（相对视口几何中心），`object_type + object_id` 组合作为稳定 tie-breaker；一致性级别为弱一致
- 聚合与查询必须基于 S2 网格与视口范围，不得引入行政区 / region 概念作为契约维度

---

## 分享落地（share）

### POST /public/v1/share/open_the_door

解析分享码换取目标详情（分享的创建/管理位于 `/api/v1/share/`，见 `api_spec_share.md`）。

**仅 public 版说明**：目标可见性按资源状态判定（目标被删除/隐私变更时 `preview` 为 null），不依赖查看者身份，登录用户与匿名用户行为一致——故不设登录镜像（镜像判定原则见「节点级规则」）。

**反滥用要求（契约级）**：
- 本端点为公开端点，必须启用 IP 维度限流，超限时返回限流类错误并携带 `Retry-After` 响应头
- 对连续失败（share_code 无效/过期）的访问需有递进式惩罚策略（如临时封禁或要求验证码）
- 失败请求应记录日志，用于离线分析潜在暴力破解行为

**Request**: `OpenShareDoorRequest`
```typescript
interface OpenShareDoorRequest {
  share_code: string;         // 不区分大小写，服务端统一转换为大写
}
```

**Response**: `ApiResponse<ShareDetail>`

**错误码**: `10300`（分享不存在/已撤销/已过期，防枚举统一语义）

---

## 反应公开列表（reactions）

### POST /public/v1/reactions/fetch

获取表情反应列表（统一 moment / comment / message 三类目标）。按反应时间倒序，支持按 emoji 过滤。

**双版本说明**：虽然 `ReactionItem` 无查看者字段，但**私聊消息**的反应可见性依赖会话参与者身份（匿名调用恒为不可见）——存在身份相关可见性，故保留 `/api/v1/reactions/fetch` 登录镜像（镜像判定原则见「节点级规则」）。

- **排序**: 默认按 `reacted_at` **时间倒序**（最近反应在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `PublicFetchReactionsRequest`
```typescript
interface PublicFetchReactionsRequest {
  target_type: ReactTarget;  // 'message' | 'moment' | 'comment'
  target_id: string;
  emoji: string | null;      // 可选：按特定 emoji 过滤；不传则返回所有 emoji 的反应者
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<ReactionItem>>`（`ReactionItem` 无查看者字段，定义于 `api_spec_types.md`）

**目标可见性**：目标资源存在但调用方不可见（私聊非参与者、非群成员、私有瞬间等）统一返回 `10300`。

**错误码**: `10300`

---

## 系统（system）

> 系统信息为公开资源，全部位于本节点。登录态冷启动聚合（含 profile/unread）位于 `/api/v1/system/bootstrap`。

### POST /public/v1/system/meta

获取当前 API 版本的元信息，包括废弃状态和下线计划。

**说明**：各 API 版本路径下都应提供此端点；客户端启动时调用以检查当前版本状态。

**状态语义（供客户端参考）**：
- `active`: 正常使用，无需处理
- `deprecated`: 提示用户升级；可展示 `message` 并引导更新
- `sunset`: 强制升级或提示服务不可用

**Request**: `MetaRequest`
```typescript
interface MetaRequest {}
```

**Response**: `ApiResponse<ApiVersionMeta>`（类型定义于 `api_spec_types.md`）

---

### POST /public/v1/system/health

系统健康检查。

**说明**：HTTP 非 200 表示未进入业务层；HTTP 200 时，`code=0` 表示系统可用（`data.status` 为 `healthy` 或 `degraded`），`code=10600` 表示系统不可用（`unhealthy`，`data` 省略，原因见 `message`）。`data.version`/`data.commit`/`data.build_time` 为构建期注入的版本指纹（见 api_spec_types.md HealthStatus），线上排障时据此核对当前运行产物；Web 前端产物对应 `/version.json` 静态文件。

**Request**: `HealthRequest`
```typescript
interface HealthRequest {}
```

**Response**: `ApiResponse<HealthStatus>`（类型定义于 `api_spec_types.md`）

---

### POST /public/v1/system/announcements/fetch

获取系统公告列表。

**Request**: `FetchSystemAnnouncementsRequest`
```typescript
interface FetchSystemAnnouncementsRequest {}
```

**Response**: `ApiResponse<SystemAnnouncementList>`
```typescript
interface SystemAnnouncementList {
  items: SystemAnnouncementItem[];
}
```

**说明**：
- 返回当前有效的公告（"有效"= 未被平台管理员删除的公告；create 即发布，delete 即下线），返回条数由服务端 `AppConfig.response_defaults.system_announcements_count` 控制，客户端不传数量参数
- 普通用户只需获取最新的 N 条公告，无需分页翻阅历史
- 平台管理员查看/搜索全部公告（含历史）请使用 `/admin/v1/announcements/fetch`（定义于 `api_spec_admin.md`）
- **排序**: 按 `created_at` 倒序，`announcement_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

---

### POST /public/v1/system/client_config

获取客户端公开配置。

**说明**：
- 返回的 `ClientConfig` 由 `AppConfig`（见 `api_spec_admin.md`）投影得来，包含功能开关、输入限制、联系方式
- 客户端应在启动时调用并缓存，配置变更频率低
- `defaults.default_location` 必须满足 `Location` 类型约束（尤其 `location_name` 非空）

**Request**: `ClientConfigRequest`
```typescript
interface ClientConfigRequest {}
```

**Response**: `ApiResponse<ClientConfig>`（类型定义于 `api_spec_types.md`）

---

### POST /public/v1/system/bootstrap

匿名冷启动聚合端点。一次请求拉齐匿名客户端冷启动所需的基础数据，替代分别调用 `meta` + `client_config` 的多次往返。登录态冷启动（含 profile/unread_summary）使用 `/api/v1/system/bootstrap`。

- **缓存契约（防惊群）**：与登录版一致——响应携带 `bootstrap_version`（服务端在 config/version 变更时递增）。客户端可缓存响应并在后续冷启动时通过 `If-None-Match: <bootstrap_version>` 请求头询问；服务端无变化时返回精简响应（`code=0` + `data.bootstrap_version` 不变 + 其余字段省略，由客户端使用缓存）。
- 客户端应加随机 jitter 延迟后刷新，避免 `bootstrap_version` 变更导致集中请求。`meta`/`client_config` 为公共数据可做 CDN 缓存。

**Request**: `PublicBootstrapRequest`
```typescript
interface PublicBootstrapRequest {
  client_type: ClientType;
  device_id: DeviceId | null;        // 可选：携带设备标识（App 端通常已有，便于服务端记录设备绑定）
}
```

**device_id Cookie 下发时机（Web 端，契约级）**：本端点是 Web 端冷启动的第一个请求——若请求未携带 `device_id` Cookie，服务端生成 UUID v4 并随本响应 `Set-Cookie` 下发（见 `api_spec_base.md`「Web 端 Cookie 契约」）；保证后续 `auth/register`、`auth/guest`（Web 端从 Cookie 读取 device_id）调用时设备标识已就绪。

**Response**: `ApiResponse<PublicBootstrapData>`

**说明**：这是**聚合端点**，不引入新数据，仅合并现有端点的响应；各字段语义与原端点完全一致。客户端应优先调用此端点；各原子端点（`meta`/`client_config` 等）仍保留，供需要单项刷新的场景。
