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
| `POST /api/v1/feed/following` | 关注动态（我关注的人发布的瞬间） | 必须 | 无 |
| `POST /api/v1/feed/nearby` | 附近瞬间（基于位置的发现流） | 必须 | 无 |
| `POST /api/v1/feed/nearby_users` | 附近的人推荐 | 可选 | 无 |
| `POST /api/v1/feed/nearby_groups` | 附近的群组推荐 | 可选 | 无 |
| `POST /api/v1/feed/trending` | 热门内容（发现页默认） | 可选 | 无 |

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `MomentItem`（Feed 场景下 `my_state` 和 `recommendation` 字段会被填充）
- `GroupItem`（Feed 场景下 `my_state` 和 `friends_in_group` 字段会被填充）
- `UserItem`（Feed 场景下 `relation` 和 `social_context` 字段会被填充）
- `PaginationParams`
- `GeoPoint`

## 端点详情

### POST /api/v1/feed/following

关注的人发布的瞬间，按时间倒序排列。

- **认证**: 必须
- **幂等**: 无
- **排序**: 按 `created_at DESC` 排序，`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `GetFeedFollowingRequest`
```typescript
interface GetFeedFollowingRequest {
  reference_point?: GeoPoint;  // 可选：用于 MomentItem.location 的距离/方位计算；不传则 distance/bearing 为 undefined
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

**错误码**: `10000`

---

### POST /api/v1/feed/nearby

附近范围内的瞬间，按距离和热度综合排序。当前用户的主要发现流入口。

- **认证**: 必须
- **幂等**: 无
- **排序**: 以地理位置相关信号为主，`moment_id DESC` 作为稳定 tie-breaker；一致性级别为弱一致

**Request**: `GetFeedNearbyRequest`
```typescript
interface GetFeedNearbyRequest {
  radius?: number;  // 位置范围（米），100-50000，可由 AppConfig.limits.search_radius_max 覆盖
  reference_point: GeoPoint; // 浏览锚点位置（必填；拒绝定位时使用 ClientConfig.defaults.default_location.point）
  pagination?: PaginationParams;
}
```

**半径默认值**：未提供 `radius` 时默认使用 `5_000` 米（5 km），服务端按 `AppConfig.limits.search_radius_max` 上限裁剪。

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

**错误码**: `10000`

---

### POST /api/v1/feed/nearby_users

获取附近的用户推荐。认证可选——未登录也可浏览附近用户，支持匿名发现（拉新关键路径）。

- **认证**: 可选
- **幂等**: 无
- **排序**: 
  - 主序：以地理位置相关信号为主（距离/活跃度的组合）
  - 次序：允许综合"社交信号"（共同好友数、共同标签）提升前排质量
  - tie-breaker：`user_id DESC`；一致性级别为弱一致

**Request**: `GetFeedNearbyUsersRequest`
```typescript
interface GetFeedNearbyUsersRequest {
  radius?: number;  // 位置范围（米），100-50000，可由 AppConfig.limits.search_radius_max 覆盖
  reference_point: GeoPoint; // 浏览锚点位置（必填；拒绝定位时使用 ClientConfig.defaults.default_location.point）
  pagination?: PaginationParams;
}
```

**半径默认值**：未提供 `radius` 时默认使用 `5_000` 米（5 km），并受 `AppConfig.limits.search_radius_max` 上限约束。

**Response**: `ApiResponse<PaginatedData<UserItem>>`

**错误码**: `10000`

---

### POST /api/v1/feed/nearby_groups

获取附近可加入的群组推荐。认证可选——未登录也可浏览附近群组。

- **认证**: 可选
- **幂等**: 无
- **排序**: 
  - 主序：以地理位置相关信号为主（距离/热度/活跃度的组合）
  - 次序：允许在距离接近时体现"社交信号"（共同好友/同标签）
  - tie-breaker：`group_id DESC`；一致性级别为弱一致

**Request**: `GetFeedNearbyGroupsRequest`
```typescript
interface GetFeedNearbyGroupsRequest {
  radius?: number;  // 位置范围（米），100-50000，可由 AppConfig.limits.search_radius_max 覆盖
  reference_point: GeoPoint; // 浏览锚点位置（必填；拒绝定位时使用 ClientConfig.defaults.default_location.point）
  pagination?: PaginationParams;
}
```

**半径默认值**：未提供 `radius` 时默认使用 `5_000` 米（5 km），并受 `AppConfig.limits.search_radius_max` 上限约束。

**Response**: `ApiResponse<PaginatedData<GroupItem>>`

**错误码**: `10000`

---

### POST /api/v1/feed/trending

获取热门瞬间（发现页默认内容，无输入场景）。

**与 search/suggest 的边界（契约级）**：
- `feed/trending`：**无输入**的"发现页默认内容"——用户刚进入 App/发现页时，平台主动推荐的热门瞬间。全局热门，不依赖用户输入或位置。
- `search/suggest`：**有输入前缀**的"搜索框联想"——用户在搜索框输入时，按前缀匹配的用户/群组/瞬间/标签建议。

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
