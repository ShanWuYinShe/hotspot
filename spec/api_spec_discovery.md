# HotSpot API - 搜索建议

> 本文档定义登录版搜索建议 (`search`) 接口。
> 匿名版搜索建议定义于 `api_spec_public.md`；条件搜索（`users/search`、`groups/search`、`moments/search`，含附近用户/群组的位置发现）定义于各自模块文档；附近瞬间流（`moments/fetch_nearby`）定义于 `api_spec_moment.md`；地图渲染 (`map/canvas`，仅 public 版——响应无查看者字段、可见性不依赖身份)定义于 `api_spec_public.md`；标签类型（`TagItem`）定义于 `api_spec_types.md`，标签管理接口见 `api_spec_admin.md`。

## 模块信息

- **命名空间**: `search`
- **基础路径**: `/api/v1/search/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 守卫与权限：`api_spec_authz.md`
  - 错误码：`api_spec_errors.md`
  - 匿名镜像版（/public/v1 同名端点）：`api_spec_public.md`

## 分层定位（契约级）

本命名空间属 **L2 聚合发现层**（见 `api_spec_base.md`「分层与依赖规则」）：只读聚合 L1 领域数据（users/moments/groups/tags 等），**不得写入领域数据、不得绕过 L1 的可见性规则**——聚合结果同样受隐私过滤（如 searchable 系列开关）。

**发现架构三分（契约级）**：`*/search` = 条件查询（关键字/标签/半径组合，附近用户/群组发现即 `users/search`/`groups/search` 显式传 `radius` + `sort_by='distance'`）；`moments/fetch_*` = 无输入的流（created/favorites/following/nearby）；`map/canvas` = 空间聚合渲染。feed 域已废除（存废审查记录见 `api_spec_index.md`）。

## 端点总览

| 端点 | 说明 | 节点 |
|-----|------|------|
| `POST /api/v1/search/suggest` | 综合搜索建议（双版本） | /api/v1 |

## 类型定义

以下类型定义于 `api_spec_types.md`：
- `UserItem`（建议场景下 `relation` 与 `social_context` 字段会被填充）
- `GroupItem`（建议场景下 `viewer_state` 与 `friends_in_group` 字段会被填充）
- `PaginationParams`、`GeoPoint`、`RelativeLocation`、`UserInfo`、`GroupInfo`、`TagItem`、`MomentSummary`
- `MomentSuggestItem`、`SearchSuggestType`（搜索建议，与 public 版共享）
- 地图结构类型（`MapViewport`、`MapHotspotItem`、`MapHeatBucket` 等；`map/canvas` 仅 public 版，类型定义于 `api_spec_public.md`）

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

> 综合搜索结果（用户 + 群组 + 瞬间）由客户端分别调用 `users/search`、`groups/search`、`moments/search` 拼装，各自独立分页，不提供聚合端点；发现页无输入首屏使用 `moments/search` 的热门模式（仅 `sort_by='popular'`）或 `moments/fetch_nearby`。

---
