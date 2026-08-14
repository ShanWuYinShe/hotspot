# HotSpot API - 用户

> 本文档定义 HotSpot 用户相关的 RPC 接口。

## 模块信息

- **命名空间**: `users`
- **基础路径**: `/api/v1/users/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 守卫与权限：`api_spec_authz.md`
  - 错误码：`api_spec_errors.md`

## 端点总览

| 端点 | 说明 | 认证 |
|-----|------|------|
| `POST /api/v1/users/search` | 搜索用户（支持关键字/标签/位置组合） | optional |
| `POST /api/v1/users/open_the_door` | 用户资料详情（可选主页首屏聚合） | optional |
| `POST /api/v1/users/list_friends` | 好友列表（user_id 必传） | optional |
| `POST /api/v1/users/list_following` | 关注列表（user_id 必传） | optional |
| `POST /api/v1/users/list_followers` | 粉丝列表（user_id 必传） | optional |
| `POST /api/v1/users/list_groups` | 用户群组列表（user_id 必传，公开视角） | optional |
| `POST /api/v1/users/list_moments` | 用户瞬间历史（user_id 必传，公开视角） | optional |
| `POST /api/v1/users/list_reacted_moments` | 用户反应过的瞬间列表（user_id 必传） | optional |
| `POST /api/v1/users/batch_get` | 批量获取用户信息 | optional |
| `POST /api/v1/users/get_account` | 我的账户数据 | required |
| `POST /api/v1/users/update_account` | 更新我的账户 | required |
| `POST /api/v1/users/update_location` | 更新我的位置 | required |
| `POST /api/v1/users/upgrade` | 访客升级 | required |
| `POST /api/v1/users/get_settings` | 我的设置 | required |
| `POST /api/v1/users/update_settings` | 更新我的设置 | required |
| `POST /api/v1/users/delete_account` | 删除账户 | required |
| `POST /api/v1/users/set_follow` | 设置关注状态（两态统一） | required |
| `POST /api/v1/users/set_block` | 设置拉黑状态（两态统一） | required |
| `POST /api/v1/users/list_blocked` | 我的拉黑列表 | required |
| `POST /api/v1/users/set_remark` | 设置备注 | required |

> 表情反应（reaction）是跨 message/moment/comment 三类资源的统一动作，独立命名空间 `reactions/`（见 `api_spec_reaction.md`）。

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `PrivacyLevel`
- `UserStats`
- `UserItem`
- `UserInfo`
- `UserType`
- `UserPresenceStatus`
- `Location`
- `PaginationParams`
- `GeoPoint`
- `GroupItem`
- `MomentItem`
- `FollowSource`
- `ChatPerm`
- `UserSettings`

### 本模块类型

```typescript
// Gender 为本模块类型
// 'other' 为包容性取值，覆盖非二元/不愿透露等场景
type Gender = 'male' | 'female' | 'other';

// 用户关系列表项（list_friends / list_following / list_followers 共用，形状恒定）
// - followed_back_at：仅 list_friends 返回；其余端点恒为 null
// - remark：查看者状态字段（恒定 nullable：查他人列表时为 null）
// - presence_status / last_online_at：随 user: UserItem 返回，受对方 presence_visible 隐私控制（null = 隐藏）
interface RelatedUserItem {
  user: UserItem;              // 完整用户信息（隐私受控字段为恒定 nullable）
  followed_at: string;         // 关注时间（list_followers 为"对方关注我的时间"，即成为粉丝的时间）
  followed_back_at: string | null;   // 对方回关时间（成为好友的时间）；仅 list_friends 返回
  is_mutual: boolean;          // 是否互相关注（list_followers 语义：该粉丝是否也关注了我）
  remark: string | null;       // 备注；仅查询自己（user_id=当前用户）时返回，查他人恒为 null
}
```

## 端点详情

### POST /api/v1/users/search

统一搜索用户。支持关键字、标签、位置范围三种条件的自由组合。

**radius 语义（契约级）**：
- `radius` 仅**显式提供时**按半径过滤结果；缺省时不做半径过滤（keyword/tag 搜索为全局匹配）
- `sort_by='distance'` 的距离计算依赖 `reference_point`（按 `api_spec_types.md`「reference_point 解析规则」回退链解析）；`relevance`/`newest` 排序不依赖位置

**隐私过滤（契约级，searchable 系列开关）**：
- `searchable=false`：该用户不出现在任何搜索结果（本端点与 `feed/nearby_users`）
- `nickname_searchable=false`：昵称关键字搜索不命中该用户
- `phone_searchable=false`：手机号精确搜索不命中该用户
- `location_searchable=false`：位置/半径维度搜索与附近发现不返回该用户

- **认证**: optional
- **排序**: 
  - 默认 `sort_by='relevance'`：按相关性优先排序，同分时以最近活跃时间做稳定 tie-breaker
  - `sort_by='distance'`：按距离由近到远排序（依赖 reference_point 回退链），距离相近时体现关键词相关性
  - `sort_by='newest'`：按注册时间倒序
  - **稳定 tie-breaker**：任意排序模式下，当主序字段和次序均相同时，以 `user_id ASC` 作为最终稳定排序，保证游标分页一致性
  - 一致性级别为弱一致（同一 `cursor` 会话内保持排序基准一致）

**Request**: `SearchUsersRequest`
```typescript
interface SearchUsersRequest {
  keyword: string | null;    // 关键字搜索（匹配昵称、bio 等）
  tag: string | null;        // 标签筛选
  radius: number | null;     // 搜索半径（米）；显式提供时按半径过滤，缺省不过滤
  sort_by: 'relevance' | 'distance' | 'newest' | null;  // 默认: relevance（有意不提供 popular：用户无明确的公开热度指标）
  reference_point: GeoPoint | null; // 浏览锚点位置（不传时服务端按回退链解析）
  pagination: PaginationParams | null;
}
// 至少提供 keyword、tag、radius 中的一个；不满足时请求失败（10000）
// reference_point 不算搜索条件，仅用于距离计算和排序，不能单独作为搜索条件
```

**Response**: `ApiResponse<PaginatedData<UserItem>>`

---

### POST /api/v1/users/open_the_door

获取用户资料详情。认证 optional，返回恒定公开类型 `UserItem`（查看者状态字段 `relation`/`social_context` 匿名时为 null）。

**隐私说明**：用户存在但对其资料设置了完全隐藏（或调用方被对方拉黑等隐私限制）时，返回 `10300`（按不存在处理，不暴露存在性）；用户不存在时返回 `10300`。

**主页聚合能力**：`include_recent_moments=true` 时附加主页首屏数据（资料 + 近期瞬间），客户端一次请求拿到主页骨架。

- **认证**: optional
- **排序**: `recent_moments` 内部按 `created_at DESC`（最新在前），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**设计要点**：
- `user.stats` 详情场景返回（含 moments_count/friends_count/followers_count/following_count/groups_count），客户端据此渲染主页各 tab 的数量角标
- 好友/关注/粉丝/群组的**明细列表**用 `users/list_friends`、`users/list_following` 等按需懒加载
- `recent_moments` 条数受 `AppConfig.response_defaults.user_profile_moments_count` 控制（默认 6）；翻页加载更多时切换到 `users/list_moments`

**Request**: `OpenUserDoorRequest`
```typescript
interface OpenUserDoorRequest {
  user_id: string;
  include_recent_moments: boolean | null;  // 默认 false；true 时返回 recent_moments（主页首屏聚合）
}
```

**Response**: `ApiResponse<UserProfileResult>`
```typescript
interface UserProfileResult {
  user: UserItem;                                  // 资料 + stats（详情场景 stats 有值）
  recent_moments: PaginatedData<MomentItem> | null; // 仅 include_recent_moments=true 时返回；否则 null
}
```

**错误码**: `10300`

---

### POST /api/v1/users/list_friends

获取指定用户的好友列表（好友 = 双方互相关注的用户）。

- **认证**: optional（`user_id` **必传**；拉自己的列表时显式传自己的 user_id。查他人受 `social_lists_visible` 隐私设置控制）
- **排序**: 默认按 `followed_back_at` **时间倒序**（最近成为好友在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `ListFriendsRequest`
```typescript
interface ListFriendsRequest {
  user_id: string;   // 必传：要查询的目标用户
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<RelatedUserItem>>`（`followed_back_at` 仅本端点返回）

**可见性说明**：`remark` 仅查询自己（user_id=当前用户）时返回，查他人恒为 null；`presence_status`/`last_online_at` 随 `user` 返回、受对方 `presence_visible` 隐私控制（null = 隐藏）。

**错误码**: `10200`（对方 `social_lists_visible=false`，整表隐私拒绝，forbidden_reason=privacy_setting） `10300`（用户不存在）

---

### POST /api/v1/users/list_following

获取指定用户的关注列表。

- **认证**: optional（`user_id` 必传；查他人受 `social_lists_visible` 控制）
- **排序**: 默认按 `followed_at` **时间倒序**（最近关注在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `ListFollowingRequest`
```typescript
interface ListFollowingRequest {
  user_id: string;   // 必传：要查询的目标用户
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<RelatedUserItem>>`

**错误码**: `10200`（privacy_setting） `10300`（用户不存在）

---

### POST /api/v1/users/list_followers

获取指定用户的粉丝列表。

- **认证**: optional（`user_id` 必传；查他人受 `social_lists_visible` 控制）
- **排序**: 默认按 `followed_at` **时间倒序**（最近关注在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `ListFollowersRequest`
```typescript
interface ListFollowersRequest {
  user_id: string;   // 必传：要查询的目标用户
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<RelatedUserItem>>`

**错误码**: `10200`（privacy_setting） `10300`（用户不存在）

---

### POST /api/v1/users/list_groups

获取指定用户加入的公开群组（公开主页视角：仅 `is_searchable=true` 的群组）。

**隐私控制**：受 `groups_visible` 设置控制。本端点无 `reference_point` 参数，返回的 `GroupItem.location` 中 distance/bearing 为 null。

- **认证**: optional（`user_id` 必传）
- **排序**: 默认按 `joined_at` **时间倒序**（最近加入在前），`group_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `ListUserGroupsRequest`
```typescript
interface ListUserGroupsRequest {
  user_id: string;   // 必传
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<GroupItem>>`

**错误码**: `10200`（privacy_setting） `10300`（用户不存在）

---

### POST /api/v1/users/list_moments

获取指定用户发布的公开瞬间（公开视角，恒定类型）。

**说明**：本端点只返回 `privacy_level='public'` 的瞬间（未登录也可访问）。拥有者视角（含 private 与软删占位）请使用 `moments/list_created`（见 `api_spec_moment.md`）。

- **认证**: optional（`user_id` 必传）
- **排序**: 默认按 `created_at` **时间倒序**（最新瞬间在前），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `ListUserMomentsRequest`
```typescript
interface ListUserMomentsRequest {
  user_id: string;   // 必传
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

**错误码**: `10300`

---

### POST /api/v1/users/list_reacted_moments

获取指定用户反应过的瞬间（去重，一个瞬间无论被该用户用过几个 emoji 只出现一次）。

**隐私控制**：受 `reacted_moments_visible` 设置控制。

- **认证**: optional（`user_id` 必传）
- **排序**: 默认按 `last_reacted_at` **时间倒序**（最近反应在前），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `ListUserReactedMomentsRequest`
```typescript
interface ListUserReactedMomentsRequest {
  user_id: string;   // 必传
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

**错误码**: `10200`（privacy_setting） `10300`（用户不存在）

---

### POST /api/v1/users/batch_get

批量获取用户信息。用于列表/Feed/通知等场景中高效解析 `UserInfo`，消除逐个调用 `open_the_door` 的 N+1 问题。

- **认证**: optional

**Request**: `BatchGetUsersRequest`
```typescript
interface BatchGetUsersRequest {
  user_ids: string[];  // 必填，最多 100 个
}
```

**Response**: `ApiResponse<UsersBatchResult>`
```typescript
interface UsersBatchResult {
  users: Record<string, UserItem>;  // user_id → UserItem 的映射；不存在的 user_id 不出现在 map 中
}
```

---

### POST /api/v1/users/get_account

获取当前用户账户数据（含 `email`/`phone`/`user_type` 等仅自己可见的字段）。

- **认证**: required

**Request**: `GetAccountRequest`
```typescript
interface GetAccountRequest {}
```

**Response**: `ApiResponse<AccountData>`
```typescript
interface AccountData {
  user_id: string;
  nickname: string;
  avatar_url: string | null;
  user_type: UserType;          // 账号类型（temp/normal），仅表达账号生命周期；平台管理员身份通过 /admin/v1/rbac/get_me 获取
  bio: string | null;
  tags: string[];
  gender: Gender | null;        // Gender 定义于本模块「类型定义」小节
  email: string | null;         // 注册时 login_id 含 '@' 视为邮箱；否则为 null
  phone: string | null;         // 注册时 login_id 为纯数字视为手机号；否则为 null
  location: Location;           // 存储的原始位置，不含 distance/bearing（与 RelativeLocation 不同）
  presence_status: UserPresenceStatus;
  last_online_at: string;       // UTC ISO 8601，最后在线时间
  created_at: string;
  stats: UserStats;
}
```

---

### POST /api/v1/users/update_account

更新用户账户资料。

- **认证**: required

**Request**: `UpdateAccountRequest`
```typescript
interface UpdateAccountRequest {
  nickname: string | null;
  avatar_file_id: string | null;   // 来自 media/complete_upload 返回的 file_id；传空字符串 "" 表示清除头像
  bio: string | null;
  tags: string[] | null;
  gender: Gender | null;
}
```

**Patch 语义**：未传字段保持不变；`tags: []` 表示清空标签，传入非空数组表示替换。

**Response**: `ApiResponse`

**错误码**: `10300`（`avatar_file_id` 对应的媒体资源不存在或不属于当前用户） `10400`（`tags` 含被屏蔽标签）

---

### POST /api/v1/users/update_location

更新用户位置。

- **认证**: required

**Request**: `UpdateLocationRequest`
```typescript
interface UpdateLocationRequest {
  location: Location;
}
```

**Response**: `ApiResponse`

---

### POST /api/v1/users/upgrade

账户升级（访客升级为正式用户）。

**前置条件（契约级）**：仅 `user_type='temp'`（访客）可调用；已是 `normal` 的正式用户重复调用返回 `10400`（已是正式用户）。

**login_id 所有权验证**：`login_id` 为手机号/邮箱时，服务端须验证该凭证归属调用方（验证码流程）；验证码服务接入前，手机号/邮箱型 `login_id` 暂不支持升级，仅允许用户名型。防止访客抢占他人手机号/邮箱。

- **认证**: required

**Request**: `UpgradeAccountRequest`
```typescript
interface UpgradeAccountRequest {
  login_id: string;   // 用户名；手机号/邮箱需先接入验证码服务
  password: string;
}
```

**Response**: `ApiResponse<AccountData>`

升级成功后返回完整账户数据，客户端无需额外调 `get_account` 即可获取新的 `user_type`。

**Token/Session 语义**：升级**不吊销当前 session、不重发 token**——服务端将当前 session 的 `user_type` 更新为 `normal`，现有 token 继续有效。访客期资料随账户保留。

**错误码**: `10400`（`login_id` 已被占用 / 已是正式用户）

---

### POST /api/v1/users/get_settings

获取用户设置。

- **认证**: required

**Request**: `GetSettingsRequest`
```typescript
interface GetSettingsRequest {}
```

**Response**: `ApiResponse<UserSettings>`

`UserSettings` 定义于 `api_spec_types.md`。

---

### POST /api/v1/users/update_settings

更新用户设置。

- **认证**: required

**Request**: `UpdateSettingsRequest`
```typescript
interface UpdateSettingsRequest {
  settings: Partial<UserSettings>;
}
```

**Response**: `ApiResponse`

---

### POST /api/v1/users/delete_account

删除账户。

**数据保留策略（契约级）**：账户删除**不级联删除已发布内容**——该用户已发布的瞬间、评论、群聊消息等公开内容保留（维护其他用户视角的社交上下文与对话完整性），但发布者身份**匿名化**：昵称/头像等个人信息替换为"已注销用户"占位，`user_id` 保留用于数据关联但不再可被搜索/访问。个人数据（登录凭证、私聊记录、设置、收藏、关注关系等）随账户删除。删除不可逆。

- **认证**: required

**Request**: `DeleteAccountRequest`
```typescript
interface DeleteAccountRequest {
  confirmation: string;  // 必须精确匹配 "DELETE_MY_ACCOUNT"（区分大小写，客户端应明确提示用户输入）；不匹配返回 10000（参数错误）
}
```

**Response**: `ApiResponse`

**login_id 处理**：注销后 `login_id`（手机号/邮箱/用户名）**释放**，可被新用户重新注册（与多数产品一致，避免凭证永久占用）。

---

### POST /api/v1/users/set_follow

设置关注状态（两态统一端点）。

`active=true` 时：若对方已关注当前用户，自动成为好友，触发 WebSocket `MutualFollow` 事件。对方 `allow_follow=false` 时返回 `10200` + `privacy_setting`。不存在关注申请流程。

`active=false` 时：若原本是好友，则好友关系自动解除。

- **认证**: required

**Request**: `FollowRequest`
```typescript
interface FollowRequest {
  user_id: string;
  active: boolean;
  source: FollowSource | null;        // 关注来源（仅 active=true 有意义）
  source_moment_id: string | null;    // 通过瞬间关注时提供瞬间 ID
}
// 组合约束：source='moment' 时 source_moment_id 必填（缺失返回 10000）；
// 提供 source_moment_id 但 source != 'moment' 视为参数冲突（10000）
```

**Response**: `ApiResponse`

**错误码**: `10200`（对方 `allow_follow=false`） `10300`（目标用户不存在） `10400`（不能关注/取关自己）

---

### POST /api/v1/users/set_block

设置拉黑状态（两态统一端点）。

**级联语义**：拉黑时**自动解除双方关注关系**（`is_following`/好友关系清零）——被拉黑方无法再通过关注关系触达；解除拉黑**不自动恢复**关注关系（需重新关注）。

- **认证**: required

**Request**: `BlockRequest`
```typescript
interface BlockRequest {
  user_id: string;
  active: boolean;
  reason: string | null;   // 拉黑原因（仅 active=true 有意义）
}
```

**Response**: `ApiResponse`

**错误码**: `10300`（目标用户不存在） `10400`（不能拉黑/解除拉黑自己）

---

### POST /api/v1/users/list_blocked

获取拉黑列表（命名空间天生私有，仅自己的数据）。

- **认证**: required
- **排序**: 默认按 `blocked_at` **时间倒序**（最近拉黑在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `ListBlockedRequest`
```typescript
interface ListBlockedRequest {
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<BlockedUserItem>>`
```typescript
interface BlockedUserItem {
  user: UserInfo;
  blocked_at: string;
  reason: string | null;
}
```

---

### POST /api/v1/users/set_remark

设置用户备注（允许给我关注的任何人设置备注，方便记忆和管理）。

**权限要求**：必须已关注该用户。

- **认证**: required

**Request**: `SetRemarkRequest`
```typescript
interface SetRemarkRequest {
  user_id: string;
  remark: string;   // 传空字符串 "" 表示清除备注
}
```

**Response**: `ApiResponse`

**错误码**: `10300`（用户不存在） `10400`（未关注该用户）