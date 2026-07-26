# HotSpot API - 用户

> 本文档定义 HotSpot 用户相关的 RPC 接口。

## 模块信息

- **命名空间**: `users`
- **基础路径**: `/api/v1/users/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 错误码：`api_spec_errors.md`
  - 搜索建议：`api_spec_search.md`（per-module suggest 已移除，使用全局 `search/suggest`）

## 端点总览

| 端点 | 说明 | 认证 | 幂等 |
|-----|------|------|------|
| `POST /api/v1/users/search` | 搜索用户（支持关键字/标签/位置组合） | 可选 | 无 |
| `POST /api/v1/users/get_detail` | 用户资料详情 | 可选 | 无 |
| `POST /api/v1/users/profile_get` | 用户主页聚合（资料 + 近期瞬间，首屏骨架） | 可选 | 无 |
| `POST /api/v1/users/list_friends` | 好友列表（user_id 可选，默认=当前用户） | 条件必须 | 无 |
| `POST /api/v1/users/list_following` | 关注列表（user_id 可选，默认=当前用户） | 条件必须 | 无 |
| `POST /api/v1/users/list_followers` | 粉丝列表（user_id 可选，默认=当前用户） | 条件必须 | 无 |
| `POST /api/v1/users/list_groups` | 用户群组列表 | 可选 | 无 |
| `POST /api/v1/users/list_moments` | 用户瞬间历史 | 可选 | 无 |
| `POST /api/v1/users/list_reacted_moments` | 用户反应过的瞬间列表 | 可选 | 无 |
| `POST /api/v1/users/me/profile/get` | 我的资料 | 必须 | 无 |
| `POST /api/v1/users/me/profile/update` | 更新我的资料 | 必须 | 幂等语义 |
| `POST /api/v1/users/me/location/update` | 更新我的位置 | 必须 | 幂等语义 |
| `POST /api/v1/users/me/upgrade` | 访客升级 | 必须 | 无 |
| `POST /api/v1/users/me/settings/get` | 我的设置 | 必须 | 无 |
| `POST /api/v1/users/me/settings/update` | 更新我的设置 | 必须 | 幂等语义 |
| `POST /api/v1/users/me/delete` | 删除账户 | 必须 | SHOULD |
| `POST /api/v1/users/me/set_follow` | 设置关注状态（两态统一） | 必须 | 幂等语义 |
| `POST /api/v1/users/me/set_block` | 设置拉黑状态（两态统一） | 必须 | 幂等语义 |
| `POST /api/v1/users/me/list_blocked` | 我的拉黑列表 | 必须 | 无 |
| `POST /api/v1/users/me/set_remark` | 设置备注 | 必须 | 幂等语义 |
| `POST /api/v1/users/me/email/request_change` | 请求变更邮箱 (⏳ 延期) | 必须 | MUST |
| `POST /api/v1/users/me/email/confirm_change` | 确认变更邮箱 (⏳ 延期) | 必须 | MUST |
| `POST /api/v1/users/me/phone/request_change` | 请求变更手机 (⏳ 延期) | 必须 | MUST |
| `POST /api/v1/users/me/phone/confirm_change` | 确认变更手机 (⏳ 延期) | 必须 | MUST |

> 表情反应（点赞/emoji）是跨 message/moment/comment 三类资源的统一动作，由全局端点 `POST /api/v1/me/set_reaction` 处理（见下方「表情反应 API」）。它不属于 users 命名空间，但因其「目标可为用户发布的资源」而在此一并说明。
| `POST /api/v1/users/me/email/request_change` | 请求变更邮箱 (⏳ 延期) | 必须 | MUST |
| `POST /api/v1/users/me/email/confirm_change` | 确认变更邮箱 (⏳ 延期) | 必须 | MUST |
| `POST /api/v1/users/me/phone/request_change` | 请求变更手机 (⏳ 延期) | 必须 | MUST |
| `POST /api/v1/users/me/phone/confirm_change` | 确认变更手机 (⏳ 延期) | 必须 | MUST |

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
- `Gender`
- `PrivateChatPermission`
- `UserSettings`

## 端点详情

### POST /api/v1/users/search

统一搜索用户。支持关键字、标签、位置范围三种条件的自由组合。

- **认证**: 可选
- **幂等**: 无
- **排序**: 
  - 默认 `sort_by='relevance'`：按相关性优先排序，同分时以最近活跃时间做稳定 tie-breaker
  - `sort_by='distance'`：按距离由近到远排序（未提供 `radius` 时按默认 `5_000` 米计算），距离相近时体现关键词相关性
  - `sort_by='newest'`：按注册时间倒序
  - **稳定 tie-breaker**：任意排序模式下，当主序字段和次序均相同时，以 `user_id ASC` 作为最终稳定排序，保证游标分页一致性
  - 一致性级别为弱一致（搜索场景允许短暂延迟；同一 `cursor` 会话内保持排序基准一致，跨会话允许排序轻微波动）

**Request**: `SearchUsersRequest`
```typescript
interface SearchUsersRequest {
  keyword?: string;    // 关键字搜索（匹配昵称、bio 等）
  tag?: string;        // 标签筛选
  radius?: number;     // 位置范围（米），100-50000，配合 reference_point 使用；未提供时默认 5_000 米，并受 AppConfig.limits.search_radius_max 上限约束
  sort_by?: 'relevance' | 'distance' | 'newest';  // 默认: relevance（有意不提供 popular：用户无明确的公开热度指标，与 moments/groups 不同）
  reference_point: GeoPoint; // 浏览锚点位置（必填；拒绝定位时使用 ClientConfig.defaults.default_location.point）
  pagination?: PaginationParams;
}
// 至少提供 keyword、tag、radius 中的一个；不满足时返回 10000（参数错误）
// reference_point 不算搜索条件，仅用于距离计算和排序，不能单独作为搜索条件
// 当仅缺省 radius 且提供了 keyword/tag 时，服务端按默认 5_000 米计算，并受 AppConfig.limits.search_radius_max 上限约束
```

**Response**: `ApiResponse<PaginatedData<UserItem>>`

**错误码**: `10000`

---

### POST /api/v1/users/get_detail

获取用户资料详情。认证可选，字段可见性规则见 `api_spec_base.md`。

**隐私说明**：用户存在但对其资料设置了完全隐藏（或调用方被对方拉黑等隐私限制）时，返回 `10200`（权限不足）；用户不存在时返回 `10300`。

- **认证**: 可选
- **幂等**: 无

**Request**: `UserOpenTheDoorRequest`
```typescript
interface UserOpenTheDoorRequest {
  user_id: string;
}
```

**Response**: `ApiResponse<UserItem>`

**错误码**: `10000` `10200` `10300`

---

### POST /api/v1/users/profile_get

获取用户主页首屏聚合数据（资料 + 近期瞬间）。一次请求拿到主页骨架，消除"资料 + 瞬间"的多次往返。

- **认证**: 可选（与 `users/get_detail` 对齐，字段可见性规则见 `api_spec_base.md`）
- **幂等**: 无
- **排序**: `recent_moments` 内部按 `created_at DESC`（最新在前），与 `users/list_moments` 一致

**设计要点**：
- `user.stats` 始终返回（含 `moments_count`/`friends_count`/`followers_count`/`following_count`/`groups_count`），客户端据此渲染主页各 tab 的数量角标，无需单独拉取列表
- 好友/关注/粉丝/群组的**明细列表**仍用现有 `users/list_friends`、`users/list_following` 等按需懒加载（用户点进对应 tab 时再拉）
- `recent_moments` 条数受 `AppConfig.response_defaults.user_profile_moments_count` 控制（默认 6）；客户端翻页加载更多瞬间时切换到 `users/list_moments`

**Request**: `UserProfileBundleRequest`
```typescript
interface UserProfileBundleRequest {
  user_id: string;
}
```

**Response**: `ApiResponse<UserProfileBundle>`
```typescript
interface UserProfileBundle {
  user: UserItem;                                  // 资料 + stats（stats 始终返回）
  recent_moments: PaginatedData<MomentItem>;       // 近期瞬间首屏；条数受 response_defaults.user_profile_moments_count 控制
}
```

**错误码**: `10000` `10200` `10300`

---

### POST /api/v1/users/list_friends

获取好友列表（好友 = 双方互相关注的用户）。

**统一端点**：`user_id` 可选。不传或传入当前用户 ID 时查询自己的好友列表，返回 `@owner-only` 字段（如 `remark`）。传入其他用户 ID 时查询对方的好友列表，受 `social_lists_visible` 隐私设置控制。

- **认证**: 条件必须（见 `api_spec_base.md` 认证档位说明：未指定 `user_id` 等价于查自己，必须登录；指定他人 ID 时为可选）
- **幂等**: 无
- **排序**: 默认按 `followed_back_at` **时间倒序**（最近成为好友在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `ListFriendsRequest`
```typescript
interface ListFriendsRequest {
  user_id?: string;   // 可选，不传时默认当前用户
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<FriendItem>>`
```typescript
interface FriendItem {
  user: UserInfo;
  presence_status?: UserPresenceStatus;  // 隐私受控：对方隐私设置时不返回
  last_online_at?: string;
  remark?: string;             // @owner-only：仅查询自己时返回
  followed_at: string;         // 我关注对方的时间
  followed_back_at: string;    // 对方回关我的时间（成为好友的时间）
}
```

**可见性说明**：`remark` 为 `@owner-only`（仅查询自己时返回），`presence_status`/`last_online_at` 受对方 `presence_visible` 隐私设置控制。

**错误码**: `10000` `10200` `10300`

---

### POST /api/v1/users/list_following

获取关注列表。

**统一端点**：`user_id` 可选。不传或传入当前用户 ID 时查询自己的关注列表。传入其他用户 ID 时查询对方的关注列表，受 `social_lists_visible` 隐私设置控制。

- **认证**: 条件必须（见 `api_spec_base.md` 认证档位说明：未指定 `user_id` 等价于查自己，必须登录；指定他人 ID 时为可选）
- **幂等**: 无
- **排序**: 默认按 `followed_at` **时间倒序**（最近关注在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `ListFollowingRequest`
```typescript
interface ListFollowingRequest {
  user_id?: string;   // 可选，不传时默认当前用户
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<FollowingItem>>`
```typescript
interface FollowingItem {
  user: UserInfo;
  followed_at: string;
  is_mutual: boolean;
  presence_status?: UserPresenceStatus;  // 隐私受控：对方隐私设置时不返回
  last_online_at?: string;
}
```

**可见性说明**：`presence_status`/`last_online_at` 受对方 `presence_visible` 隐私设置控制。

**错误码**: `10000` `10200` `10300`

---

### POST /api/v1/users/list_followers

获取粉丝列表。

**统一端点**：`user_id` 可选。不传或传入当前用户 ID 时查询自己的粉丝列表。传入其他用户 ID 时查询对方的粉丝列表，受 `social_lists_visible` 隐私设置控制。

- **认证**: 条件必须（见 `api_spec_base.md` 认证档位说明：未指定 `user_id` 等价于查自己，必须登录；指定他人 ID 时为可选）
- **幂等**: 无
- **排序**: 默认按 `followed_at` **时间倒序**（最近关注在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `ListFollowersRequest`
```typescript
interface ListFollowersRequest {
  user_id?: string;   // 可选，不传时默认当前用户
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<FollowerItem>>`
```typescript
interface FollowerItem {
  user: UserInfo;
  followed_at: string;
  is_following_back: boolean;
  presence_status?: UserPresenceStatus;  // 隐私受控：对方隐私设置时不返回
  last_online_at?: string;
}
```

**可见性说明**：`presence_status`/`last_online_at` 受对方 `presence_visible` 隐私设置控制。

**错误码**: `10000` `10200` `10300`

---

### POST /api/v1/users/list_groups

获取指定用户加入的公开群组。

**隐私控制**：受 `groups_visible` 设置控制。仅返回 `is_searchable=true` 的公开群组。本端点无 `reference_point` 参数，返回的 `GroupItem.location` 中 `distance` 和 `bearing_degrees` 为 `undefined`。

- **认证**: 可选
- **幂等**: 无
- **排序**: 默认按 `joined_at` **时间倒序**（最近加入在前），`group_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `UserGroupsRequest`
```typescript
interface UserGroupsRequest {
  user_id: string;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<GroupItem>>`

**错误码**: `10000` `10200` `10300`

---

### POST /api/v1/users/list_moments

获取指定用户生活瞬间历史。

**说明**：生活瞬间支持 `privacy_level` 控制可见性（`public` / `private`）。未登录用户仅能访问 `public` 的瞬间；登录态可额外看到自己的所有瞬间（包括 `private`）。认证可选，字段可见性规则见 `api_spec_base.md`。

- **认证**: 可选
- **幂等**: 无
- **排序**: 默认按 `created_at` **时间倒序**（最新瞬间在前），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `UserMomentsRequest`
```typescript
interface UserMomentsRequest {
  user_id: string;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

**错误码**: `10000` `10300`

---

### POST /api/v1/users/list_reacted_moments

获取指定用户反应过的瞬间（去重，一个瞬间无论被该用户用过几个 emoji 只出现一次）。

**隐私控制**：受 `liked_moments_visible` 设置控制（语义不变：控制"他人能否看到我反应过的瞬间"）。

- **认证**: 可选
- **幂等**: 无
- **排序**: 默认按 `last_reacted_at` **时间倒序**（最近反应在前），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `UserReactedMomentsRequest`
```typescript
interface UserReactedMomentsRequest {
  user_id: string;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

**错误码**: `10000` `10200` `10300`

---

### POST /api/v1/users/me/profile/get

获取当前用户资料。

**说明**：`/me/*` 端点返回的是当前用户的资料字段，永远可见，不受隐私设置影响。

- **认证**: 必须
- **幂等**: 无

**Request**: `{}`

**Response**: `ApiResponse<MyProfileState>`
```typescript
// Gender 为本模块类型（原定义于 api_spec_types.md，因仅 user 模块使用已下放至此）
// 'other' 为包容性取值，覆盖非二元/不愿透露等场景
type Gender = 'male' | 'female' | 'other';

interface MyProfileState {
  user_id: string;
  nickname: string;
  avatar_url?: string;
  user_type: UserType;          // 账号类型（temp/normal），仅表达账号生命周期；管理员身份通过 admin/rbac/get_me 获取
  bio?: string;
  tags?: string[];
  gender?: Gender;
  email?: string;               // 通过 users/me/email/request_change + confirm_change 变更
  phone?: string;               // 通过 users/me/phone/request_change + confirm_change 变更
  location: Location;           // 存储的原始位置，不含 distance/bearing_degrees（与 RelativeLocation 不同）
  presence_status: UserPresenceStatus;
  last_online_at: string;       // UTC ISO 8601，最后在线时间
  created_at: string;
  stats: UserStats;
}
```

**错误码**: `10000`

---

### POST /api/v1/users/me/profile/update

更新用户资料。

- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `UpdateProfileRequest`
```typescript
interface UpdateProfileRequest {
  nickname?: string;
  avatar_file_id?: string;
  bio?: string;
  tags?: string[];
  gender?: Gender;
}
```

**Patch 语义**：未传字段保持不变；`tags: []` 表示清空标签，传入非空数组表示替换。

**Response**: `ApiResponse`

**错误码**: `10000` `10300`

---

### POST /api/v1/users/me/location/update

更新用户位置。

- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `UpdateLocationRequest`
```typescript
interface UpdateLocationRequest {
  location: Location;
}
```

**Response**: `ApiResponse`

**错误码**: `10000`

---

### POST /api/v1/users/me/upgrade

账户升级（访客升级为正式用户）。

- **认证**: 必须
- **幂等**: 无

**Request**: `UpgradeAccountRequest`
```typescript
interface UpgradeAccountRequest {
  login_id: string;
  password: string;
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10400`

---

### POST /api/v1/users/me/settings/get

获取用户设置。

- **认证**: 必须
- **幂等**: 无

**Request**: `{}`

**Response**: `ApiResponse<UserSettings>`

`UserSettings` 定义于 `api_spec_types.md`。

**错误码**: `10000`

---

### POST /api/v1/users/me/settings/update

更新用户设置。

- **认证**: 必须
- **幂等**: 幂等语义（重复请求返回一致结果）

**Request**: `UpdateSettingsRequest`
```typescript
interface UpdateSettingsRequest {
  settings: Partial<UserSettings>;
}
```

**Response**: `ApiResponse`

**错误码**: `10000`

---

### POST /api/v1/users/me/delete

删除账户。

- **认证**: 必须
- **幂等**: SHOULD（建议客户端携带 `Idempotency-Key`）

**Request**: `DeleteAccountRequest`
```typescript
interface DeleteAccountRequest {
  confirmation: string;  // 必须精确匹配 "DELETE_MY_ACCOUNT"（区分大小写，客户端应明确提示用户输入）
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10400`

---

### POST /api/v1/users/me/set_follow

设置关注状态（两态统一端点，替代原 follow/unfollow 成对动词）。

`active=true` 时：若对方已关注当前用户，自动成为好友，触发 WebSocket `MutualFollow` 事件。对方 `allow_follow=false` 时返回 `10200`。不存在关注申请流程。

`active=false` 时：若原本是好友，则好友关系自动解除。

- **认证**: 必须
- **幂等**: 幂等语义（`active` 与当前状态一致时视为 no-op，返回成功）

**Request**: `SetFollowRequest`
```typescript
interface SetFollowRequest {
  user_id: string;
  active: boolean;
  source?: FollowSource;        // 可选：关注来源（用于来源归因，仅 active=true 有意义）
  source_moment_id?: string;    // 可选：通过瞬间关注时提供瞬间 ID
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10300` `10400`（不能关注/取关自己）

---

### POST /api/v1/users/me/set_block

设置拉黑状态（两态统一端点，替代原 block/unblock 成对动词）。

- **认证**: 必须
- **幂等**: 幂等语义（`active` 与当前状态一致时视为 no-op，返回成功）

**Request**: `SetBlockRequest`
```typescript
interface SetBlockRequest {
  user_id: string;
  active: boolean;
  reason?: string;   // 可选：拉黑原因（仅 active=true 有意义）
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10300` `10400`（不能拉黑/解除拉黑自己）

---

## 表情反应 API（全局 `/me/` 端点）

> 此端点位于全局 `/api/v1/me/` 命名空间（非 `users/`），因统一处理 message/moment/comment 三类资源，故定义于此处便于跨模块引用。类型 `ReactionTargetType` / `Reaction` / `ReactionItem` 见 `api_spec_types.md`。

### POST /api/v1/me/set_reaction

设置表情反应（两态统一端点）。覆盖消息、瞬间、评论三类目标。

- **认证**: 必须
- **幂等**: 幂等语义（`active` 与当前状态一致时视为 no-op，返回成功；同一目标切换 emoji 视为更新而非新增）

**Request**: `SetReactionRequest`
```typescript
interface SetReactionRequest {
  target_type: ReactionTargetType;  // 'message' | 'moment' | 'comment'
  target_id: string;                // message_id / moment_id / comment_id
  emoji: string;                    // Unicode 码点字符串，须在 AppConfig.limits.reaction_emojis 白名单内
  active: boolean;                  // true=添加/切换反应，false=取消反应
}
```

**Response**: `ApiResponse`

**行为细节**：
- `active=true` 且用户此前对该目标用过其他 emoji：视为切换（旧 emoji 计数 -1，新 emoji 计数 +1），不分多次调用
- `active=true` 且 emoji 与当前一致：no-op 成功
- `active=false`：移除当前用户对该目标的反应（若 emoji 与当前不一致仍视为成功，幂等）
- 目标资源不存在返回 `10300`；emoji 不在白名单返回 `10000`
- 私聊/群聊消息场景：若被对方拉黑或对方 `allow_private_chat='nobody'`，仍允许 reaction（reaction 是公开互动信号，不受私聊权限约束——与点赞一致）

**WS 推送**：成功后触发 `ReactionAdded` / `ReactionRemoved` 事件（见 `api_spec_websocket.md`），并向被反应资源作者推送 `reaction_added` 通知（若作者非当前用户）。

**错误码**: `10000` `10300`

---

### POST /api/v1/users/me/list_blocked

获取拉黑列表。

- **认证**: 必须
- **幂等**: 无
- **排序**: 默认按 `blocked_at` **时间倒序**（最近拉黑在前），`user_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `MyBlockedRequest`
```typescript
interface MyBlockedRequest {
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<BlockedUserItem>>`
```typescript
interface BlockedUserItem {
  user: UserInfo;
  blocked_at: string;
  reason?: string;
}
```

**错误码**: `10000`

---

### POST /api/v1/users/me/set_remark

设置用户备注（允许给我关注的任何人设置备注，方便记忆和管理）。

**权限要求**：必须已关注该用户。

- **认证**: 必须
- **幂等**: 幂等语义（重复设置相同备注返回成功）

**Request**: `SetRemarkRequest`
```typescript
interface SetRemarkRequest {
  user_id: string;
  remark: string;   // 传空字符串 "" 表示清除备注
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10300` `10400`（未关注该用户）

---

### POST /api/v1/users/me/email/request_change

> ⏳ **延期实现**：依赖邮件验证码发送服务，待服务接入后实现。

请求变更邮箱。发送验证码到新邮箱地址。

**安全设计**：需要当前密码验证身份，防止账号被盗后变更邮箱。

- **认证**: 必须
- **幂等**: MUST（客户端必须携带 `Idempotency-Key`）

**Request**: `RequestEmailChangeRequest`
```typescript
interface RequestEmailChangeRequest {
  new_email: string;       // 新邮箱地址
  password: string;        // 当前密码（身份验证）
}
```

**Response**: `ApiResponse<VerificationSent>`
```typescript
interface VerificationSent {
  expires_at: string;      // 验证码过期时间（UTC ISO 8601）
}
```

**错误码**: `10000` `10200` `10400`

---

### POST /api/v1/users/me/email/confirm_change

> ⏳ **延期实现**：依赖邮件验证码发送服务，待服务接入后实现。

确认变更邮箱。提交发送到新邮箱的验证码完成变更。

- **认证**: 必须
- **幂等**: MUST（客户端必须携带 `Idempotency-Key`）

**Request**: `ConfirmEmailChangeRequest`
```typescript
interface ConfirmEmailChangeRequest {
  new_email: string;             // 与 request_change 一致的新邮箱
  verification_code: string;     // 验证码
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10400`

---

### POST /api/v1/users/me/phone/request_change

> ⏳ **延期实现**：依赖短信验证码发送服务，待服务接入后实现。

请求变更手机号。发送验证码到新手机号。

**安全设计**：需要当前密码验证身份，防止账号被盗后变更手机号。

- **认证**: 必须
- **幂等**: MUST（客户端必须携带 `Idempotency-Key`）

**Request**: `RequestPhoneChangeRequest`
```typescript
interface RequestPhoneChangeRequest {
  new_phone: string;       // 新手机号
  password: string;        // 当前密码（身份验证）
}
```

**Response**: `ApiResponse<VerificationSent>`

**错误码**: `10000` `10200` `10400`

---

### POST /api/v1/users/me/phone/confirm_change

> ⏳ **延期实现**：依赖短信验证码发送服务，待服务接入后实现。

确认变更手机号。提交发送到新手机号的验证码完成变更。

- **认证**: 必须
- **幂等**: MUST（客户端必须携带 `Idempotency-Key`）

**Request**: `ConfirmPhoneChangeRequest`
```typescript
interface ConfirmPhoneChangeRequest {
  new_phone: string;             // 与 request_change 一致的新手机号
  verification_code: string;     // 验证码
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10200` `10400`
