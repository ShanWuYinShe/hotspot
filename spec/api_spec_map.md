---
alwaysApply: false
---
# HotSpot API - 地图渲染

> 本文档定义 HotSpot 地图渲染 (`map`) 相关的 RPC 接口。

## 模块信息

- **命名空间**: `map`
- **基础路径**: `/api/v1/map/`
- **引用**: 
  - 基础规范：`@spec/api_spec_base.md`
  - 通用类型：`@spec/api_spec_types.md`
  - 错误码：`@spec/api_spec_errors.md`

## 设计理念

地图模块的核心目标是：

- 按**行政区域**对用户、群组、动态进行划分与聚合；
- 在不同视角（zoom/视口大小）下，返回**合适尺度**的结果：
  - 视角大时：按国家/省/市/区县返回**区域统计**，用于热力分布；
  - 视角小时：返回当前视野内的**具体对象列表**，用于查看/点击明细。

本模块提供两个端点：

- **热力统计（heatmap）**：按行政区聚合统计总量和按类型拆分，不返回任何实体详情；
- **热点明细（hotspots）**：在足够小的视口内返回具体的用户/群组/动态列表，使用游标分页控制返回量。

服务端负责：

- 基于 `viewport.bounds` 计算视口物理跨度（米），动态选择合适的行政级别（国家/省/市/区县）；
- 控制最大返回条数，当数据量过大时只返回最热的前 N 个区域或对象；
- 在大数据量和高并发场景下依然保持查询可扩展（依赖预聚合统计表，而非直接扫描明细）。

## 端点总览

| 端点 | 说明 | 认证 | 幂等 |
|-----|------|------|------|
| `POST /api/v1/map/heatmap`  | 获取按行政区聚合的热力统计 | 可选 | 无 |
| `POST /api/v1/map/hotspots` | 获取当前视野内的热点对象列表 | 可选 | 无 |

## 类型定义

以下类型定义于 `@spec/api_spec_types.md`：
- `GeoPoint`

本节仅补充地图模块特有的请求/响应结构。

### 通用类型

```typescript
// 过滤器：限定参与统计/热点的类别
// - 'all' 或未传：用户 / 群组 / 动态都参与
// - 'user'：仅用户
// - 'group'：仅群组
// - 'moment'：仅动态
type MapDataType = 'user' | 'group' | 'moment' | 'all';

interface Bounds {
  north: number;
  south: number;
  east: number;
  west: number;
}
// 约束：north >= south；east >= west（允许退化视口）；east/west 取值范围 [-180, 180]。
// 暂不支持跨国际日期变更线视口（需由客户端拆分请求）。违反时返回 `10000`。

interface MapViewport {
  bounds: Bounds;
  zoom: number;   // 取值范围 1-20，约束定义见 @spec/api_spec_base.md 通用字段约束
}

interface TimeRangeParams {
  start_time?: string;  // UTC ISO 8601
  end_time?: string;    // UTC ISO 8601
}
// 约束：若同时提供，必须满足 start_time <= end_time。违反时返回 `10000`。
```

## 端点详情

### POST /api/v1/map/heatmap —— 按行政区聚合的热力统计

获取地图热力统计数据，仅返回**按行政区聚合**的统计信息，不返回任何实体详情。适用于全国/省/市/区县等“概览视角”，用于渲染“哪里比较热”。

- **认证**: 可选
- **幂等**: 无

#### 请求

```typescript
interface MapHeatFilter {
  data_type?: MapDataType;
  time_range?: TimeRangeParams;  // 可选，限定统计时间范围
}

interface MapHeatRequest {
  viewport: MapViewport;
  filter?: MapHeatFilter;
}
```

行为约束：

- 服务端根据 `viewport.bounds` 计算视口物理跨度 `span_m`（单位：米）并结合视口中心位置，在内部选择合适的行政区聚合层级（基于 OSM admin_level），但不会在响应中暴露具体层级数值；
- 当 `span_m` 过小（例如 ≤ 3km）时，不再返回行政区热力统计，服务端应返回空列表并在 `toast_message` 中提示“当前视野已足够细，请使用热点接口查看具体对象”；
- 服务端可限制最大返回区域数量（例如 500 个），当区域过多时按热度排序仅返回前 N 个，并在 `toast_message` 中给出说明。

#### 响应

```typescript
interface MapHeatBreakdown {
  user?: number;   // 该行政区内用户数量
  group?: number;  // 该行政区内群组数量
  moment?: number; // 该行政区内动态数量
}

interface MapHeatBucket {
  /**
   * 聚合单元的代表坐标（例如行政区多边形的质心）。
   * 客户端仅将其视为一个热力点，至于如何在地图上展示、是否叠加自己的行政区边界，由客户端自行决定。
   */
  location: GeoPoint;

  /** 当前过滤条件下，该区域内参与统计实体的总数 */
  total_count: number;

  /** 按类型拆分；当 data_type != 'all' 时可选 */
  breakdown?: MapHeatBreakdown;

  /**
   * 可选的调试/展示标签（例如行政区名称）。
   * 客户端可以直接忽略该字段，仅用于在开发/调试或简单列表展示时辅助理解。
   */
  name?: string;
}

interface MapHeatCanvas {
  items: MapHeatBucket[];
  toast_message?: string;
}
```

**错误码**: `10000`（请求参数不合法，如 bounds/zoom/time_range 约束违反）

---

### POST /api/v1/map/hotspots —— 视野内热点对象列表

获取当前视图内的热点对象列表（用户/群组/动态的明细）。仅在视口足够小、缩放级别足够高时使用，用于用户在地图上查看具体对象，而不是统计分布。

- **认证**: 可选
- **幂等**: 无

#### 请求

```typescript
interface PaginationParams {
  limit?: number;   // 单页数量，默认 50，最大不超过 200
  cursor?: string;  // 游标，不透明，由服务端生成和回传
}

interface MapHotspotFilter {
  data_type?: MapDataType;       // 仅用户 / 仅群组 / 仅动态 / 全部
  time_range?: TimeRangeParams;  // 对动态尤为重要
}

interface MapHotspotsRequest {
  viewport: MapViewport;
  filter?: MapHotspotFilter;
  pagination?: PaginationParams;
}
```

行为约束：

- 服务端根据 `viewport.bounds` 计算视口物理跨度 `span_m`，并结合 `zoom` 判断是否允许返回明细：
  - 仅当 `span_m` 不超过实现定义的阈值（例如 3km）**且** `zoom` 足够大（例如 `zoom >= 13`）时返回热点列表；
  - 否则应返回错误（`10000`）或空列表，并在 `toast_message` 中提示“请放大地图后再查看该区域的详细内容”。
- 分页必须使用游标（cursor）进行 keyset 分页，禁止使用 offset-based 分页；
- `limit` 上限固定（例如 200），超出时服务端按上限处理。

#### 响应

```typescript
type HotspotObjectType = 'user' | 'group' | 'moment';

interface MapHotspotItem {
  object_type: HotspotObjectType; // 对象类型
  object_id: string;              // 业务 ID（user_id / group_id / moment_id）
  title: string;                  // 昵称 / 群名 / 动态标题
  subtitle?: string;              // 简短描述 / 地点名等
  cover_url?: string;             // 头像 / 群头像 / 动态封面
  location: GeoPoint;             // 对象实际坐标（WGS-84）
  created_at?: string;            // 对于动态等有时间语义的对象可返回创建时间
  distance_meters?: number;       // 相对视口中心或参考点的距离（可选）
}

interface MapHotspotsCanvas {
  items: MapHotspotItem[];
  next_cursor?: string;    // 若有下一页则返回游标
  toast_message?: string;  // 用于提示视口过大、结果被截断等情况
}
```

**错误码**: `10000`（请求参数不合法或视口/zoom 不在允许范围内）
