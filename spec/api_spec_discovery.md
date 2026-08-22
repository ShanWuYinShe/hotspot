# HotSpot API - 发现与聚合

> 本文档定义 HotSpot L2 聚合发现层中**依赖登录视角**的 RPC 接口：信息流 (`feed`) 与登录版搜索建议 (`search`)。
> 匿名版发现（nearby_users/nearby_groups/trending/suggest）与地图渲染 (`map/canvas`，仅 public 版——响应无查看者字段、可见性不依赖身份)定义于 `api_spec_public.md`；标签类型（`TagItem`/`TagTrend`）定义于 `api_spec_types.md`，标签管理接口见 `api_spec_admin.md`。

## 模块信息

- **命名空间**: `feed` / `search`
- **基础路径**: `/api/v1/feed/`、`/api/v1/search/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 守卫与权限：`api_spec_authz.md`
  - 错误码：`api_spec_errors.md`
  - 匿名镜像版（/public/v1 同名端点）：`api_spec_public.md`

## 分层定位（契约级）

本文件两个命名空间（feed/search）属 **L2 聚合发现层**（见 `api_spec_base.md`「分层与依赖规则」）：只读聚合 L1 领域数据（users/moments/groups/tags 等），**不得写入领域数据、不得绕过 L1 的可见性规则**——聚合结果同样受隐私过滤（如 nearby_users 的 searchable 系列开关）。

## 端点总览

| 端点 | 说明 | 节点 |
|-----|------|------|
| `POST /api/v1/feed/following` | 关注动态（我关注的人发布的瞬间） | /api/v1 |
| `POST /api/v1/feed/nearby_moments` | 附近瞬间（基于位置的发现流） | /api/v1 |
| `POST /api/v1/feed/nearby_users` | 附近的人推荐（双版本，匿名版在 /public/v1） | /api/v1 |
| `POST /api/v1/feed/nearby_groups` | 附近的群组推荐（双版本） | /api/v1 |
| `POST /api/v1/feed/trending` | 热门内容（发现页默认；双版本） | /api/v1 |
| `POST /api/v1/search/suggest` | 综合搜索建议（双版本） | /api/v1 |

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `MomentItem`（Feed 场景下 `is_favorited` 与 `recommendation` 字段会被填充）
- `GroupItem`（Feed 场景下 `viewer_state` 与 `friends_in_group` 字段会被填充）
- `UserItem`（Feed 场景下 `relation` 与 `social_context` 字段会被填充）
- `PaginationParams`、`GeoPoint`、`RelativeLocation`、`UserInfo`、`GroupInfo`、`TagItem`、`MomentSummary`
- `MomentSuggestItem`、`SearchSuggestType`（搜索建议，与 public 版共享）
- 地图结构类型（`MapViewport`、`MapHotspotItem`、`MapHeatBucket` 等；`map/canvas` 仅 public 版，类型定义于 `api_spec_public.md`）

## 信息流 API（feed）

### POST /api/v1/feed/following

关注的人发布的瞬间，按时间倒序排列。

- **节点**: /api/v1
- **排序**: 按 `created_at DESC` 排序，`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `FollowingRequest`
```typescript
interface FollowingRequest {
  reference_point: GeoPoint | null;  // 用于 MomentItem.location 的距离/方位计算；不传时服务端按回退链解析
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

---

### POST /api/v1/feed/nearby_moments

附近范围内的瞬间。当前用户的主要发现流入口。

- **节点**: /api/v1
- **排序**: 主序为服务端计算的 `distance` 升序（相对 `reference_point`）；仅当距离差在服务端定义的近似阈值内时，热度/活跃度作为次级信号参与排序；`moment_id DESC` 作为稳定 tie-breaker；一致性级别为弱一致

**Request**: `NearbyMomentsRequest`
```typescript
interface NearbyMomentsRequest {
  radius: number | null;  // 搜索半径（米）；不传默认 5000 并受 search_radius_max 裁剪（位置发现类缺省口径，见 api_spec_base.md「搜索与地理约束」）
  reference_point: GeoPoint | null;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

---

### POST /api/v1/feed/nearby_users

获取附近的用户推荐（登录视角）。匿名发现（拉新关键路径）使用 `/public/v1/feed/nearby_users`。

**隐私过滤（契约级）**：`location_searchable=false` 的用户不出现（位置发现被关闭）；`searchable=false` 的用户不出现（整体不被发现）。其余用户按各自 `presence_visible` 等隐私设置返回对应字段（null = 隐藏）。

- **节点**: /api/v1（双版本端点：匿名版 `/public/v1/feed/nearby_users` 见 `api_spec_public.md`）
- **排序**: 
  - 主序：服务端计算的 `distance` 升序（相对 `reference_point`）
  - 次序：仅当距离差在服务端定义的近似阈值内时，允许综合"社交信号"（共同好友数、共同标签）与活跃度提升前排质量
  - tie-breaker：`user_id ASC`（与 users/search 统一）；一致性级别为弱一致

**Request**: `NearbyUsersRequest`
```typescript
interface NearbyUsersRequest {
  radius: number | null;  // 搜索半径（米）；不传默认 5000 并受 search_radius_max 裁剪（位置发现类缺省口径，见 api_spec_base.md「搜索与地理约束」）
  reference_point: GeoPoint | null;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<UserItem>>`

---

### POST /api/v1/feed/nearby_groups

获取附近可加入的群组推荐（登录视角）。匿名浏览使用 `/public/v1/feed/nearby_groups`。

- **节点**: /api/v1（双版本端点：匿名版 `/public/v1/feed/nearby_groups` 见 `api_spec_public.md`）
- **排序**: 
  - 主序：服务端计算的 `distance` 升序（相对 `reference_point`）
  - 次序：仅当距离差在服务端定义的近似阈值内时，允许体现"社交信号"（共同好友/同标签）与热度/活跃度
  - tie-breaker：`group_id ASC`（与 groups/search 统一）；一致性级别为弱一致

**Request**: `NearbyGroupsRequest`
```typescript
interface NearbyGroupsRequest {
  radius: number | null;  // 搜索半径（米）；不传默认 5000 并受 search_radius_max 裁剪（位置发现类缺省口径，见 api_spec_base.md「搜索与地理约束」）
  reference_point: GeoPoint | null;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<GroupItem>>`

---

### POST /api/v1/feed/trending

获取热门瞬间（发现页默认内容，无输入场景）。

**与 search/suggest 的边界（契约级）**：
- `feed/trending`：**无输入**的"发现页默认内容"——用户刚进入 App/发现页时，平台主动推荐的热门瞬间。全局热门，不依赖用户输入或位置。
- `search/suggest`：**有输入前缀**的"搜索框联想"——用户在搜索框输入时，按前缀匹配的用户/群组/瞬间/标签建议。

- **节点**: /api/v1（双版本端点：匿名版 `/public/v1/feed/trending` 见 `api_spec_public.md`）
- **排序**: 按热度降序（含时间衰减因子；具体衰减算法为服务端实现细节），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为弱一致

**Request**: `TrendingRequest`
```typescript
interface TrendingRequest {}
// 返回条数由 AppConfig.response_defaults.hot_moments_count 控制
```

**Response**: `ApiResponse<TrendingResult>`
```typescript
interface TrendingResult {
  hot_moments: MomentItem[];   // 热门瞬间列表（登录视角，含查看者字段；匿名版 PublicTrendingResult 见 api_spec_public.md）
}
```

---

## 搜索建议 API（search）

### POST /api/v1/search/suggest

综合搜索建议（标签、用户、群组、瞬间）。

- **节点**: /api/v1（双版本端点：匿名版 `/public/v1/search/suggest` 见 `api_spec_public.md`）
- **排序**:
  - 完全匹配优先
  - 在匹配度相近时按距离优先，热度/活跃度作为辅助信号
  - 各类型结果内部以对应资源 ID（`user_id`/`group_id`/`moment_id`/tag name）作为稳定 tie-breaker；一致性级别为弱一致

**说明**：
- 建议用于搜索框输入提示与全局发现入口。
- **LBS 优化**：`reference_point` 可选，不传时服务端按 `api_spec_types.md`「reference_point 解析规则」回退链解析。
- **类型过滤**：通过 `types` 参数可限定返回特定类型的结果。

**Request**: `SearchSuggestRequest`
```typescript
interface SearchSuggestRequest {
  prefix: string;
  reference_point: GeoPoint | null;
  types: SearchSuggestType[];  // 空数组表示返回所有类型
  radius: number | null;  // 搜索半径（米）；不传默认 5000 并受 search_radius_max 裁剪
}
```

**Response**: `ApiResponse<SearchSuggestResponse>`

```typescript
// MomentSuggestItem / SearchSuggestType 定义于 api_spec_types.md
// 登录视角建议项：user/group 分支为 UserItem/GroupItem（含查看者状态）；
// 匿名版 PublicSearchSuggestResponse 见 api_spec_public.md
type SearchSuggestItem =
  | { type: 'tag'; payload: TagItem }
  | { type: 'user'; payload: UserItem }
  | { type: 'group'; payload: GroupItem }
  | { type: 'moment'; payload: MomentSuggestItem };

interface SearchSuggestResponse {
  items: SearchSuggestItem[];
}
```

> 综合搜索结果（用户 + 群组 + 瞬间）由客户端分别调用 `users/search`、`groups/search`、`moments/search` 拼装，各自独立分页，不提供聚合端点。

---
