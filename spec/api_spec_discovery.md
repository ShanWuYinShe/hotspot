# HotSpot API - 发现与聚合

> 本文档定义 HotSpot L2 聚合发现层的 RPC 接口：信息流 (`feed`)、搜索建议 (`search`) 与地图渲染 (`map`)。
> 标签类型（`TagItem`/`TagTrend`）定义于 `api_spec_types.md`，标签管理接口见 `api_spec_admin.md`。

## 模块信息

- **命名空间**: `feed` / `search` / `map`
- **基础路径**: `/api/v1/feed/`、`/api/v1/search/`、`/api/v1/map/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 守卫与权限：`api_spec_authz.md`
  - 错误码：`api_spec_errors.md`

## 分层定位（契约级）

本文件三个命名空间属 **L2 聚合发现层**（见 `api_spec_base.md`「分层与依赖规则」）：只读聚合 L1 领域数据（users/moments/groups/tags 等），**不得写入领域数据、不得绕过 L1 的可见性规则**——聚合结果同样受隐私过滤（如 nearby_users 的 searchable 系列开关）。

## 端点总览

| 端点 | 说明 | 认证 |
|-----|------|------|
| `POST /api/v1/feed/following` | 关注动态（我关注的人发布的瞬间） | required |
| `POST /api/v1/feed/nearby` | 附近瞬间（基于位置的发现流） | required |
| `POST /api/v1/feed/nearby_users` | 附近的人推荐 | optional |
| `POST /api/v1/feed/nearby_groups` | 附近的群组推荐 | optional |
| `POST /api/v1/feed/trending` | 热门内容（发现页默认） | optional |
| `POST /api/v1/search/suggest` | 综合搜索建议 | optional |
| `POST /api/v1/map/canvas` | 根据视口返回统计或热点明细 | optional |

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `MomentItem`（Feed 场景下 `is_favorited` 与 `recommendation` 字段会被填充）
- `GroupItem`（Feed 场景下 `viewer_state` 与 `friends_in_group` 字段会被填充）
- `UserItem`（Feed 场景下 `relation` 与 `social_context` 字段会被填充）
- `PaginationParams`、`GeoPoint`、`RelativeLocation`、`UserInfo`、`GroupInfo`、`TagItem`、`MomentSummary`

## 信息流 API（feed）

### POST /api/v1/feed/following

关注的人发布的瞬间，按时间倒序排列。

- **认证**: required
- **排序**: 按 `created_at DESC` 排序，`moment_id DESC` 作为稳定 tie-breaker；一致性级别为强一致

**Request**: `GetFeedFollowingRequest`
```typescript
interface GetFeedFollowingRequest {
  reference_point: GeoPoint | null;  // 用于 MomentItem.location 的距离/方位计算；不传时服务端按回退链解析
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

---

### POST /api/v1/feed/nearby

附近范围内的瞬间。当前用户的主要发现流入口。

- **认证**: required
- **排序**: 主序为服务端计算的 `distance` 升序（相对 `reference_point`）；仅当距离差在服务端定义的近似阈值内时，热度/活跃度作为次级信号参与排序；`moment_id DESC` 作为稳定 tie-breaker；一致性级别为弱一致

**Request**: `GetFeedNearbyRequest`
```typescript
interface GetFeedNearbyRequest {
  radius: number | null;  // 搜索半径（米）
  reference_point: GeoPoint | null;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<MomentItem>>`

---

### POST /api/v1/feed/nearby_users

获取附近的用户推荐。认证 optional——未登录也可浏览附近用户，支持匿名发现（拉新关键路径）。

**隐私过滤（契约级）**：`location_searchable=false` 的用户不出现（位置发现被关闭）；`searchable=false` 的用户不出现（整体不被发现）。其余用户按各自 `presence_visible` 等隐私设置返回对应字段（null = 隐藏）。

- **认证**: optional
- **排序**: 
  - 主序：服务端计算的 `distance` 升序（相对 `reference_point`）
  - 次序：仅当距离差在服务端定义的近似阈值内时，允许综合"社交信号"（共同好友数、共同标签）与活跃度提升前排质量
  - tie-breaker：`user_id ASC`（与 users/search 统一）；一致性级别为弱一致

**Request**: `GetFeedNearbyUsersRequest`
```typescript
interface GetFeedNearbyUsersRequest {
  radius: number | null;
  reference_point: GeoPoint | null;
  pagination: PaginationParams | null;
}
```

**Response**: `ApiResponse<PaginatedData<UserItem>>`

---

### POST /api/v1/feed/nearby_groups

获取附近可加入的群组推荐。认证 optional——未登录也可浏览附近群组。

- **认证**: optional
- **排序**: 
  - 主序：服务端计算的 `distance` 升序（相对 `reference_point`）
  - 次序：仅当距离差在服务端定义的近似阈值内时，允许体现"社交信号"（共同好友/同标签）与热度/活跃度
  - tie-breaker：`group_id ASC`（与 groups/search 统一）；一致性级别为弱一致

**Request**: `GetFeedNearbyGroupsRequest`
```typescript
interface GetFeedNearbyGroupsRequest {
  radius: number | null;
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

- **认证**: optional
- **排序**: 按热度降序（含时间衰减因子；具体衰减算法为服务端实现细节），`moment_id DESC` 作为稳定 tie-breaker；一致性级别为弱一致

**Request**: `GetFeedTrendingRequest`
```typescript
interface GetFeedTrendingRequest {}
// 返回条数由 AppConfig.response_defaults.hot_moments_count 控制
```

**Response**: `ApiResponse<TrendingResult>`
```typescript
interface TrendingResult {
  hot_moments: MomentItem[];   // 热门瞬间列表；查看者字段按登录态返回（匿名时 relation 等为 null、is_favorited 为 false）
}
```

---

## 搜索建议 API（search）

### POST /api/v1/search/suggest

综合搜索建议（标签、用户、群组、瞬间）。

- **认证**: optional
- **排序**:
  - 完全匹配优先
  - 在匹配度相近时按距离优先，热度/活跃度作为辅助信号
  - 各类型结果内部以对应资源 ID（`user_id`/`group_id`/`moment_id`/tag name）作为稳定 tie-breaker；一致性级别为弱一致

**说明**：
- Token 可选；未登录时仅返回公开内容。
- 建议用于搜索框输入提示与全局发现入口。
- **LBS 优化**：`reference_point` 可选，不传时服务端按 `api_spec_types.md`「reference_point 解析规则」回退链解析。
- **类型过滤**：通过 `types` 参数可限定返回特定类型的结果。

**Request**: `SearchSuggestRequest`
```typescript
interface SearchSuggestRequest {
  prefix: string;
  reference_point: GeoPoint | null;
  types: SearchSuggestType[];  // 空数组表示返回所有类型
  radius: number | null;  // 搜索半径（米）；不传默认 5_000 并受 search_radius_max 裁剪
}
```

**Response**: `ApiResponse<SearchSuggestResponse>`

```typescript
// 瞬间建议项（基于 MomentSummary 的轻量扩展，仅用于建议场景）
interface MomentSuggestItem extends MomentSummary {
  author: UserInfo;
  location: RelativeLocation | null;
}

// 搜索建议类型（支持按类型过滤）
type SearchSuggestType = 'tag' | 'user' | 'group' | 'moment';

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

## 地图渲染 API（map）

## 设计理念

地图模块不再依赖任何行政区或 region 概念，完全基于 S2 网格进行空间划分与聚合：

- 所有带位置的实体（用户、群组、动态）在服务端被投影到服务端选定层级的 S2 网格单元（层级由视口跨度与数据密度自动决定，对客户端不透明）；
- 服务端根据当前视口的物理跨度和预估结果规模，自动选择合适的 S2 层级（`level`），并使用 S2 覆盖算法确定视口内的网格集合；
- 在大尺度视野下返回**网格热力统计**，在小尺度视野下返回**具体热点对象列表**，对客户端透明；
- 查询实现依赖预聚合的 S2 统计表，而不是直接扫描明细记录。

### POST /api/v1/map/canvas

根据视口范围和 S2 网格预聚合，自动返回热力统计（`heatmap`）或热点明细（`hotspots`）中的一种。

- **认证**: optional

**Request**: `MapCanvasRequest`

**Response**: `ApiResponse<MapCanvasResponse>`

> 类型定义见下方「类型定义」；`MapCanvasResponse` 为 tagged union（按 `mode` 分支），互斥由结构保证。

```typescript
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
  // 位置统一使用 RelativeLocation（含 distance/bearing），与 feed/search 对齐
  // distance/bearing 相对于视口几何中心计算，由服务端计算
  location: RelativeLocation;
  created_at: string | null;      // 对于动态等有时间语义的对象可返回创建时间
}

interface MapCanvasFilter {
  data_type: MapDataType | null;
  time_range: TimeRangeParams | null;
}

interface MapCanvasRequest {
  viewport: MapViewport;         // 覆盖区域由 bounds 决定
  filter: MapCanvasFilter | null;
  // pagination 仅在返回 hotspots 时使用；limit 覆盖全局默认为 default=50、max=200
  pagination: PaginationParams | null;
  // 注：无 reference_point 字段。distance/bearing 的计算锚点 = viewport 几何中心。
}

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