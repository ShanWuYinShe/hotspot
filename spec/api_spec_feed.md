# HotSpot API - 信息流

> 本文档定义 HotSpot 信息流 (`feed`) 相关的 RPC 接口。标签类型（`TagItem`/`TagTrend`）定义于 `api_spec_types.md`，标签管理接口见 `api_spec_admin_moderation.md`。

## 模块信息

- **命名空间**: `feed`
- **基础路径**: `/api/v1/feed/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 错误码：`api_spec_errors.md`

## 端点总览

| 端点 | 说明 | 认证 |
|-----|------|------|

| `POST /api/v1/feed/following` | 关注动态（我关注的人发布的瞬间） | 必须 |
| `POST /api/v1/feed/nearby` | 附近瞬间（基于位置的发现流） | 必须 |
| `POST /api/v1/feed/nearby_users` | 附近的人推荐 | 可选 |
| `POST /api/v1/feed/nearby_groups` | 附近的群组推荐 | 可选 |
| `POST /api/v1/feed/trending` | 热门内容（发现页默认） | 可选 |
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
- **排序**: 按 `created_at DESC` 排序，`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `GetFeedFollowingRequest`
```typescript
interface GetFeedFollowingRequest {
  reference_point?: GeoPoint;  // 可选：用于 MomentItem.location 的距离/方位计算；不传时服务端按 api_spec_types.md「reference_point 解析规则」回退链解析
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

---

### POST /api/v1/feed/nearby

附近范围内的瞬间。当前用户的主要发现流入口。

- **认证**: 必须
- **排序**: 主序为服务端计算的 `distance` 升序（相对 `reference_point`）；仅当距离差在服务端定义的近似阈值内时，热度/活跃度作为次级信号参与排序；`moment_id DESC` 作为稳定 tie-breaker；一致性级别为弱一致

**Request**: `GetFeedNearbyRequest`
```typescript
interface GetFeedNearbyRequest {
  radius?: number;  // 搜索半径（米），约束见 api_spec_base.md 字段校验约束
  reference_point?: GeoPoint; // 浏览锚点位置（可选；不传时服务端按 api_spec_types.md「reference_point 解析规则」回退链解析）
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

---

### POST /api/v1/feed/nearby_users

获取附近的用户推荐。认证可选——未登录也可浏览附近用户，支持匿名发现（拉新关键路径）。

**隐私过滤（契约级）**：`location_searchable=false` 的用户不出现（位置发现被关闭）；`searchable=false` 的用户不出现（整体不被发现）。其余用户按各自 `presence_visible` 等隐私设置返回对应字段。

- **认证**: 可选
- **排序**: 
  - 主序：服务端计算的 `distance` 升序（相对 `reference_point`）
  - 次序：仅当距离差在服务端定义的近似阈值内时，允许综合"社交信号"（共同好友数、共同标签）与活跃度提升前排质量
  - tie-breaker：`user_id DESC`；一致性级别为弱一致

**Request**: `GetFeedNearbyUsersRequest`
```typescript
interface GetFeedNearbyUsersRequest {
  radius?: number;  // 搜索半径（米），约束见 api_spec_base.md 字段校验约束
  reference_point?: GeoPoint; // 浏览锚点位置（可选；不传时服务端按 api_spec_types.md「reference_point 解析规则」回退链解析）
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<UserItem>>`

---

### POST /api/v1/feed/nearby_groups

获取附近可加入的群组推荐。认证可选——未登录也可浏览附近群组。

- **认证**: 可选
- **排序**: 
  - 主序：服务端计算的 `distance` 升序（相对 `reference_point`）
  - 次序：仅当距离差在服务端定义的近似阈值内时，允许体现"社交信号"（共同好友/同标签）与热度/活跃度
  - tie-breaker：`group_id DESC`；一致性级别为弱一致

**Request**: `GetFeedNearbyGroupsRequest`
```typescript
interface GetFeedNearbyGroupsRequest {
  radius?: number;  // 搜索半径（米），约束见 api_spec_base.md 字段校验约束
  reference_point?: GeoPoint; // 浏览锚点位置（可选；不传时服务端按 api_spec_types.md「reference_point 解析规则」回退链解析）
  pagination?: PaginationParams;
}
```

**Response**: `ApiResponse<PaginatedData<GroupItem>>`

---

### POST /api/v1/feed/trending

获取热门瞬间（发现页默认内容，无输入场景）。

**与 search/suggest 的边界（契约级）**：
- `feed/trending`：**无输入**的"发现页默认内容"——用户刚进入 App/发现页时，平台主动推荐的热门瞬间。全局热门，不依赖用户输入或位置。
- `search/suggest`：**有输入前缀**的"搜索框联想"——用户在搜索框输入时，按前缀匹配的用户/群组/瞬间/标签建议。

- **认证**: 可选
- **排序**: 按热度降序（含时间衰减因子，避免老内容长期占榜；具体衰减算法为服务端实现细节），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为弱一致

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
