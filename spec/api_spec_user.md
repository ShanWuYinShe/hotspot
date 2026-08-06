# HotSpot API - 用户

> 本文档定义 HotSpot 用户相关的 RPC 接口。

## 模块信息

- **命名空间**: `users`
- **基础路径**: `/api/v1/users/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 错误码：`api_spec_errors.md`

## 端点总览

| 端点 | 说明 | 认证 |
|-----|------|------|

| `POST /api/v1/users/search` | 搜索用户（支持关键字/标签/位置组合） | 可选 |
| `POST /api/v1/users/open_the_door` | 用户资料详情 | 可选 |
| `POST /api/v1/users/get_profile` | 用户主页聚合（资料 + 近期瞬间，首屏骨架） | 可选 |
| `POST /api/v1/users/list_friends` | 好友列表（user_id 可选，默认=当前用户） | 条件必须 |
| `POST /api/v1/users/list_following` | 关注列表（user_id 可选，默认=当前用户） | 条件必须 |
| `POST /api/v1/users/list_followers` | 粉丝列表（user_id 可选，默认=当前用户） | 条件必须 |
| `POST /api/v1/users/list_groups` | 用户群组列表 | 可选 |
| `POST /api/v1/users/list_moments` | 用户瞬间历史 | 可选 |
| `POST /api/v1/users/list_reacted_moments` | 用户反应过的瞬间列表 | 可选 |
| `POST /api/v1/users/batch_get` | 批量获取用户信息 | 可选 |
| `POST /api/v1/users/get_account` | 我的账户数据 | 必须 |
| `POST /api/v1/users/update_account` | 更新我的账户 | 必须 |
| `POST /api/v1/users/update_location` | 更新我的位置 | 必须 |
| `POST /api/v1/users/upgrade` | 访客升级 | 必须 |
| `POST /api/v1/users/get_settings` | 我的设置 | 必须 |
| `POST /api/v1/users/update_settings` | 更新我的设置 | 必须 |
| `POST /api/v1/users/delete_account` | 删除账户 | 必须 |
| `POST /api/v1/users/set_follow` | 设置关注状态（两态统一） | 必须 |
| `POST /api/v1/users/set_block` | 设置拉黑状态（两态统一） | 必须 |
| `POST /api/v1/users/list_blocked` | 我的拉黑列表 | 必须 |
| `POST /api/v1/users/set_remark` | 设置备注 | 必须 |
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
```

## 端点详情

### POST /api/v1/users/search

统一搜索用户。支持关键字、标签、位置范围三种条件的自由组合。

- **认证**: 可选
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
  radius?: number;     // 搜索半径（米），约束见 api_spec_base.md 字段校验约束
  sort_by?: 'relevance' | 'distance' | 'newest';  // 默认: relevance（有意不提供 popular：用户无明确的公开热度指标，与 moments/groups 不同）
  reference_point?: GeoPoint; // 浏览锚点位置（可选；不传时服务端按 api_spec_types.md「reference_point 解析规则」回退链解析）
  pagination?: PaginationParams;
}
// 至少提供 keyword、tag、radius 中的一个；不满足时请求失败
// reference_point 不算搜索条件，仅用于距离计算和排序，不能单独作为搜索条件
// 提供 keyword/tag 但缺省 radius 时，服务端按默认半径计算
```

**Response**: `ApiResponse<PaginatedData<UserItem>>`

---

### POST /api/v1/users/open_the_door

获取用户资料详情。认证可选，字段可见性规则见 `api_spec_base.md`。

**隐私说明**：用户存在但对其资料设置了完全隐藏（或调用方被对方拉黑等隐私限制）时，返回 `10300`（按不存在处理，不暴露存在性，见 `api_spec_base.md` 隐私拒绝策略）；用户不存在时返回 `10300`。

- **认证**: 可选

**Request**: `UserOpenTheDoorRequest`
```typescript
interface UserOpenTheDoorRequest {
  user_id: string;
}
```

**Response**: `ApiResponse<UserItem>`

**错误码**: `10300`

---

### POST /api/v1/users/get_profile

获取用户主页首屏聚合数据（资料 + 近期瞬间）。一次请求拿到主页骨架，消除"资料 + 瞬间"的多次往返。

- **认证**: 可选（与 `users/open_the_door` 对齐，字段可见性规则见 `api_spec_base.md`）
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

**错误码**: `10300`

---

### POST /api/v1/users/list_friends

获取好友列表（好友 = 双方互相关注的用户）。

**统一端点**：`user_id` 可选。不传或传入当前用户 ID 时查询自己的好友列表，返回 `@owner-only` 字段（如 `remark`）。传入其他用户 ID 时查询对方的好友列表，受 `social_lists_visible` 隐私设置控制。

- **认证**: 条件必须（见 `api_spec_base.md` 认证档位说明：未指定 `user_id` 等价于查自己，必须登录；指定他人 ID 时为可选）
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

**错误码**: `10200` `10300`

---

### POST /api/v1/users/list_following

获取关注列表。

**统一端点**：`user_id` 可选。不传或传入当前用户 ID 时查询自己的关注列表。传入其他用户 ID 时查询对方的关注列表，受 `social_lists_visible` 隐私设置控制。

- **认证**: 条件必须（见 `api_spec_base.md` 认证档位说明：未指定 `user_id` 等价于查自己，必须登录；指定他人 ID 时为可选）
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
  remark?: string;             // @owner-only：仅查询自己时返回
  presence_status?: UserPresenceStatus;  // 隐私受控：对方隐私设置时不返回
  last_online_at?: string;
}
```

**可见性说明**：`remark` 为 `@owner-only`（仅查询自己时返回）；`presence_status`/`last_online_at` 受对方 `presence_visible` 隐私设置控制。

**错误码**: `10200` `10300`

---

### POST /api/v1/users/list_followers

获取粉丝列表。

**统一端点**：`user_id` 可选。不传或传入当前用户 ID 时查询自己的粉丝列表。传入其他用户 ID 时查询对方的粉丝列表，受 `social_lists_visible` 隐私设置控制。

- **认证**: 条件必须（见 `api_spec_base.md` 认证档位说明：未指定 `user_id` 等价于查自己，必须登录；指定他人 ID 时为可选）
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
  is_mutual: boolean;          // 是否互相关注（该粉丝是否也关注了我），与 FollowingItem.is_mutual 语义一致
  presence_status?: UserPresenceStatus;  // 隐私受控：对方隐私设置时不返回
  last_online_at?: string;
}
```

**可见性说明**：`presence_status`/`last_online_at` 受对方 `presence_visible` 隐私设置控制。

**错误码**: `10200` `10300`

---

### POST /api/v1/users/list_groups

获取指定用户加入的公开群组。

**隐私控制**：受 `groups_visible` 设置控制。仅返回 `is_searchable=true` 的公开群组。本端点无 `reference_point` 参数，返回的 `GroupItem.location` 中 `distance` 和 `bearing_degrees` 为 `undefined`。

- **认证**: 可选
- **排序**: 默认按 `joined_at` **时间倒序**（最近加入在前），`group_id ASC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `UserGroupsRequest`
```typescript
interface UserGroupsRequest {
  user_id: string;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<GroupItem>>`

**错误码**: `10200` `10300`

---

### POST /api/v1/users/list_moments

获取指定用户生活瞬间历史。

**说明**：生活瞬间支持 `privacy_level` 控制可见性（`public` / `private`）。未登录用户仅能访问 `public` 的瞬间；登录态可额外看到自己的所有瞬间（包括 `private`）。认证可选，字段可见性规则见 `api_spec_base.md`。

- **认证**: 可选
- **排序**: 默认按 `created_at` **时间倒序**（最新瞬间在前），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `UserMomentsRequest`
```typescript
interface UserMomentsRequest {
  user_id: string;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

**错误码**: `10300`

---

### POST /api/v1/users/list_reacted_moments

获取指定用户反应过的瞬间（去重，一个瞬间无论被该用户用过几个 emoji 只出现一次）。

**隐私控制**：受 `reacted_moments_visible` 设置控制。

- **认证**: 可选
- **排序**: 默认按 `last_reacted_at` **时间倒序**（最近反应在前），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `UserReactedMomentsRequest`
```typescript
interface UserReactedMomentsRequest {
  user_id: string;
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

**错误码**: `10200` `10300`

---

### POST /api/v1/users/batch_get

批量获取用户信息。用于列表/Feed/通知等场景中高效解析 `UserInfo`，消除逐个调用 `open_the_door` 的 N+1 问题。

- **认证**: 可选（已登录返回 `@auth-only` 字段）

**Request**: `BatchGetUsersRequest`
```typescript
interface BatchGetUsersRequest {
  user_ids: string[];  // 最多 100 个
}
```

**Response**: `ApiResponse<{ users: Record<string, UserItem> }>`
```typescript
// users 为 user_id → UserItem 的映射；不存在的 user_id 不出现在 map 中
```

---

### POST /api/v1/users/get_account

获取当前用户账户数据（含 `email`/`phone`/`user_type` 等仅自己可见的字段）。

- **认证**: 必须

**Request**: `{}`

**Response**: `ApiResponse<AccountData>`
```typescript
interface AccountData {
  user_id: string;
  nickname: string;
  avatar_url?: string;
  user_type: UserType;          // 账号类型（temp/normal），仅表达账号生命周期；管理员身份通过 admin/rbac/get_me 获取
  bio?: string;
  tags?: string[];
  gender?: Gender;              // Gender 定义于本模块「类型定义」小节
  email?: string;               // 邮箱：注册时 login_id 按格式判定——含 '@' 视为邮箱（存 email），否则省略；后续通过邮箱变更流程修改（变更接口暂未实现，见 api_spec_index.md 预留）
  phone?: string;               // 手机号：注册时 login_id 为纯数字视为手机号（存 phone），否则省略；用户名注册时两字段均省略
  location: Location;           // 存储的原始位置，不含 distance/bearing_degrees（与 RelativeLocation 不同）
  presence_status: UserPresenceStatus;
  last_online_at: string;       // UTC ISO 8601，最后在线时间
  created_at: string;
  stats: UserStats;
}
```

---

### POST /api/v1/users/update_account

更新用户账户资料。

- **认证**: 必须

**Request**: `UpdateAccountRequest`
```typescript
interface UpdateAccountRequest {
  nickname?: string;
  avatar_file_id?: string;   // 来自 media/complete_upload 返回的 file_id
  bio?: string;
  tags?: string[];
  gender?: Gender;
}
```

**Patch 语义**：未传字段保持不变；`tags: []` 表示清空标签，传入非空数组表示替换。

**Response**: `ApiResponse`

**错误码**: `10300`

---

### POST /api/v1/users/update_location

更新用户位置。

- **认证**: 必须

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

- **认证**: 必须

**Request**: `UpgradeAccountRequest`
```typescript
interface UpgradeAccountRequest {
  login_id: string;   // 手机号/邮箱/用户名
  password: string;
}
```

**Response**: `ApiResponse<AccountData>`

升级成功后返回完整账户数据，客户端无需额外调 `get_account` 即可获取新的 `user_type`。

**Token/Session 语义**：升级**不吊销当前 session、不重发 token**——服务端将当前 session 的 `user_type` 更新为 `normal`，现有 access_token/refresh_token 继续有效，客户端无需重新登录。访客期资料（昵称/头像/bio 等）随账户保留。

**错误码**: `10400`（`login_id` 已被占用）

---

### POST /api/v1/users/get_settings

获取用户设置。

- **认证**: 必须

**Request**: `{}`

**Response**: `ApiResponse<UserSettings>`

`UserSettings` 定义于 `api_spec_types.md`。

---

### POST /api/v1/users/update_settings

更新用户设置。

- **认证**: 必须

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

- **认证**: 必须

**Request**: `DeleteAccountRequest`
```typescript
interface DeleteAccountRequest {
  confirmation: string;  // 必须精确匹配 "DELETE_MY_ACCOUNT"（区分大小写，客户端应明确提示用户输入）
}
```

**Response**: `ApiResponse`

**login_id 处理**：注销后 `login_id`（手机号/邮箱/用户名）**释放**，可被新用户重新注册（与多数产品一致，避免凭证永久占用）。

**错误码**: `10400`

---

### POST /api/v1/users/set_follow

设置关注状态（两态统一端点）。

`active=true` 时：若对方已关注当前用户，自动成为好友，触发 WebSocket `MutualFollow` 事件。对方 `allow_follow=false` 时返回 `10200`。不存在关注申请流程。

`active=false` 时：若原本是好友，则好友关系自动解除。

- **认证**: 必须

**Request**: `FollowRequest`
```typescript
interface FollowRequest {
  user_id: string;
  active: boolean;
  source?: FollowSource;        // 可选：关注来源（用于来源归因，仅 active=true 有意义）
  source_moment_id?: string;    // 可选：通过瞬间关注时提供瞬间 ID
}
```

**Response**: `ApiResponse`

**错误码**: `10200` `10300` `10400`（不能关注/取关自己）

---

### POST /api/v1/users/set_block

设置拉黑状态（两态统一端点）。

**级联语义**：拉黑时**自动解除双方关注关系**（`is_following`/好友关系清零，与微博/Instagram 等主流产品一致）——被拉黑方无法再通过关注关系触达；解除拉黑**不自动恢复**关注关系（需重新关注）。

- **认证**: 必须

**Request**: `BlockRequest`
```typescript
interface BlockRequest {
  user_id: string;
  active: boolean;
  reason?: string;   // 可选：拉黑原因（仅 active=true 有意义）
}
```

**Response**: `ApiResponse`

**错误码**: `10300` `10400`（不能拉黑/解除拉黑自己）

---

### POST /api/v1/users/list_blocked

获取拉黑列表。

- **认证**: 必须
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

---

### POST /api/v1/users/set_remark

设置用户备注（允许给我关注的任何人设置备注，方便记忆和管理）。

**权限要求**：必须已关注该用户。

- **认证**: 必须

**Request**: `SetRemarkRequest`
```typescript
interface SetRemarkRequest {
  user_id: string;
  remark: string;   // 传空字符串 "" 表示清除备注
}
```

**Response**: `ApiResponse`

**错误码**: `10300` `10400`（未关注该用户）
