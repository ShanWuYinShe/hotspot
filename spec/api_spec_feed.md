# HotSpot API - 信息流

> 本文档定义 HotSpot 信息流 (`feed`) 相关的 RPC 接口。标签相关 API 定义于 `api_spec_search.md`。

## 模块信息

- **命名空间**: `feed`
- **基础路径**: `/api/v1/feed/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 错误码：`api_spec_errors.md`

## 端点总览

| 端点 | 说明 | 认证 | 幂等 |
|-----|------|------|------|
| `POST /api/v1/feed/list_moments` | 获取瞬间信息流 | 必须 | 无 |
| `POST /api/v1/feed/list_groups` | 获取群组推荐 | 必须 | 无 |
| `POST /api/v1/feed/list_users` | 获取用户推荐 | 必须 | 无 |
| `POST /api/v1/feed/list_trending` | 获取热门内容 | 可选 | 无 |
| `POST /api/v1/feed/me/dismiss` | 标记不感兴趣 (⏳ 延期实现，见详情) | 必须 | 幂等语义 |

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `MomentItem`（Feed 场景下 `my_state` 和 `recommendation` 字段会被填充）
- `GroupItem`（Feed 场景下 `my_state` 和 `friends_in_group` 字段会被填充）
- `UserItem`（Feed 场景下 `relation` 和 `social_context` 字段会被填充）
- `PaginationParams`
- `GeoPoint`
- `TagItem`、`TagTrend`（定义于 `api_spec_types.md`）

## 端点详情

### POST /api/v1/feed/list_moments

获取瞬间信息流，支持多种来源筛选。

- **认证**: 必须
- **幂等**: 无
- **排序**: 
  - `source='following'`：按 `created_at DESC` 排序，`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致
  - `source='nearby'`：以地理位置相关信号为主，`moment_id DESC` 作为稳定 tie-breaker；一致性级别为弱一致
  - `source='all'`：综合推荐流（允许排序波动），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为弱一致

**Request**: `GetFeedMomentsRequest`
```typescript
// 信息流来源
type FeedSource = 'following' | 'nearby' | 'all';

interface GetFeedMomentsRequest {
  source: FeedSource;
  radius?: number;  // 位置范围（米），100-50000，可由 AppConfig.limits.search_radius_max 覆盖
  reference_point: GeoPoint; // 浏览锚点位置（必填；拒绝定位时使用 ClientConfig.defaults.default_location.point）
  pagination?: PaginationParams;
}
```

**半径默认值**：未提供 `radius` 时默认使用 `5_000` 米（5 km）作为搜索范围；服务端仍按 `AppConfig.limits.search_radius_max` 上限裁剪。

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

**来源说明**：
- `following`：仅关注的人发布的瞬间，按时间倒序（需要 `reference_point`，用于 `MomentItem.location` 的距离/方位计算）
- `nearby`：附近范围内的瞬间，按距离和热度排序（需要 reference_point）
- `all`：综合推荐，混合关注动态、附近热门、个性化推荐（需要 reference_point）

**错误码**: `10000`

---

### POST /api/v1/feed/list_groups

获取附近可加入的群组推荐。

- **认证**: 必须
- **幂等**: 无
- **排序**: 
  - 主序：以地理位置相关信号为主（距离/热度/活跃度的组合）
  - 次序：允许在距离接近时体现"社交信号"（共同好友/同标签）
  - tie-breaker：`group_id DESC`；一致性级别为弱一致

**Request**: `GetFeedGroupsRequest`
```typescript
interface GetFeedGroupsRequest {
  radius?: number;  // 位置范围（米），100-50000，可由 AppConfig.limits.search_radius_max 覆盖
  reference_point: GeoPoint; // 浏览锚点位置（必填；拒绝定位时使用 ClientConfig.defaults.default_location.point）
  pagination?: PaginationParams;
}
```

**半径默认值**：未提供 `radius` 时默认使用 `5_000` 米（5 km），并受 `AppConfig.limits.search_radius_max` 上限约束。

**Response**: `ApiResponse<PaginatedData<GroupItem>>`

**错误码**: `10000`

---

### POST /api/v1/feed/list_users

获取附近的用户推荐。

- **认证**: 必须
- **幂等**: 无
- **排序**: 
  - 主序：以地理位置相关信号为主（距离/活跃度的组合）
  - 次序：允许综合"社交信号"（共同好友数、共同标签）提升前排质量
  - tie-breaker：`user_id DESC`；一致性级别为弱一致

**Request**: `GetFeedUsersRequest`
```typescript
interface GetFeedUsersRequest {
  radius?: number;  // 位置范围（米），100-50000，可由 AppConfig.limits.search_radius_max 覆盖
  reference_point: GeoPoint; // 浏览锚点位置（必填；拒绝定位时使用 ClientConfig.defaults.default_location.point）
  pagination?: PaginationParams;
}
```

**半径默认值**：未提供 `radius` 时默认使用 `5_000` 米（5 km），并受 `AppConfig.limits.search_radius_max` 上限约束。

**Response**: `ApiResponse<PaginatedData<UserItem>>`

**错误码**: `10000`

---

### POST /api/v1/feed/list_trending

获取热门瞬间（发现页默认内容，无输入场景）。

**与 search/suggest 的边界（契约级）**：
- `feed/list_trending`：**无输入**的"发现页默认内容"——用户刚进入 App/发现页时，平台主动推荐的热门瞬间。全局热门，不依赖用户输入或位置。
- `search/suggest`：**有输入前缀**的"搜索框联想"——用户在搜索框输入时，按前缀匹配的用户/群组/瞬间/标签建议。
- 标签的热门榜（trending tags）归 `search/suggest` 的 tag 类别，**不在本端点返回**（原 `trending_tags` 已移除以消除与 suggest 的职责重叠）。

- **认证**: 可选
- **幂等**: 无

**Request**: `GetFeedTrendingRequest`
```typescript
interface GetFeedTrendingRequest {}
// 返回条数由 AppConfig.response_defaults.hot_moments_count 控制
```

**Response**: `ApiResponse<TrendingBundle>`
```typescript
interface TrendingBundle {
  hot_moments: MomentItem[];   // 热门瞬间列表；@auth-only 字段（如 reactions[].reacted_by_me）按登录态返回
}
```

**错误码**: `10000`

---

### POST /api/v1/feed/me/dismiss

> ⏳ **延期实现**：推荐算法优化为后续迭代重点，当前阶段优先完成核心 Feed 流。

标记内容为"不感兴趣"，用于改善推荐质量。支持对瞬间、用户、群组的负反馈。

- **认证**: 必须
- **幂等**: 幂等语义（重复标记同一内容返回成功）

**Request**: `DismissFeedItemRequest`
```typescript
// 可标记不感兴趣的内容类型
type DismissTargetType = 'moment' | 'user' | 'group';

interface DismissFeedItemRequest {
  target_type: DismissTargetType;
  target_id: string;               // moment_id / user_id / group_id
}
```

**Response**: `ApiResponse`

**错误码**: `10000` `10300`
