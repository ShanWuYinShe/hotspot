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

地图模块不再依赖任何行政区或 region 概念，完全基于 S2 网格进行空间划分与聚合：

- 所有带位置的实体（用户、群组、动态）在服务端被投影到固定层级的 S2 网格单元；
- 服务端根据当前视口的物理跨度和预估结果规模，自动选择合适的 S2 层级（`level`），并使用 S2 覆盖算法确定视口内的网格集合；
- 在大尺度视野下返回**网格热力统计**，在小尺度视野下返回**具体热点对象列表**，对客户端透明；
- 查询实现依赖预聚合的 S2 统计表，而不是直接扫描明细记录，以保证在大数据量和高并发场景下的可扩展性。

本模块通过单一端点实现上述能力：

- **地图画布（canvas）**：基于视口范围和 S2 网格预聚合，自动返回热力统计（heatmap）或热点明细（hotspots）中的一种。

## 端点总览

| 端点 | 说明 | 认证 | 幂等 |
|------|------|------|------|
| `POST /api/v1/map/canvas` | 根据视口返回统计或热点明细 | 可选 | 无 |

## 类型定义

以下基础类型定义于 `@spec/api_spec_types.md`：

- `GeoPoint`
- `PaginationParams`

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

// 地图视口：仅由经纬度边界决定，不需要客户端提供 zoom
interface MapViewport {
  bounds: Bounds;
}

interface TimeRangeParams {
  start_time?: string;  // UTC ISO 8601
  end_time?: string;    // UTC ISO 8601
}
// 约束：若同时提供，必须满足 start_time <= end_time。违反时返回 `10000`。
```

### 统计与明细类型

> 下列类型由本模块统一端点 `/api/v1/map/canvas` 复用，用于承载统计结果和热点明细结果。

```typescript
interface MapHeatBreakdown {
  user?: number;   // 该网格内用户数量
  group?: number;  // 该网格内群组数量
  moment?: number; // 该网格内动态数量
}

// S2 网格的标识。为避免 JS 精度问题，对外暴露的是 S2 cell 的字符串 token。
type S2CellToken = string;

interface MapHeatBucket {
  /**
   * 网格标识：
   * - cell_token：S2 单元格的字符串 token，可用于前端作为 key 或调试。
   * - level：该单元格所在的 S2 level。
   * 客户端不需要理解 token 细节，只需视为不透明字符串。
   */
  cell_token: S2CellToken;
  level: number;

  /**
   * 聚合单元的代表坐标（例如 S2 单元格的中心点）。
   * 客户端仅将其视为一个热力点，如何展示由客户端自行决定。
   */
  location: GeoPoint;

  /** 当前过滤条件下，该网格内参与统计实体的总数 */
  total_count: number;

  /** 按类型拆分；当 data_type != 'all' 时可选 */
  breakdown?: MapHeatBreakdown;
}

interface MapHeatCanvas {
  items: MapHeatBucket[];
}

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
}
```

### 请求与响应

```typescript
interface MapCanvasFilter {
  data_type?: MapDataType;       // 与 MapHeatBreakdown / MapHotspotItem 含义一致
  time_range?: TimeRangeParams;  // 时间过滤
}

interface MapCanvasRequest {
  viewport: MapViewport;         // 覆盖区域由 bounds 决定
  filter?: MapCanvasFilter;
  pagination?: PaginationParams; // 仅在返回 hotspots 时使用
}

interface MapCanvasResponse {
  heatmap?: MapHeatCanvas;       // 有统计数据时出现
  hotspots?: MapHotspotsCanvas;  // 有明细数据时出现
}
```

约束：

- 同一次响应中，`heatmap` 与 `hotspots` 最多出现一个字段：要么只返回统计，要么只返回明细，不会同时出现；
- 当某类结果没有任何数据时，不返回对应字段：没有 bucket 就不返回 `heatmap`，没有热点对象就不返回 `hotspots`。

## 行为约束

服务端在实现 `/api/v1/map/canvas` 时，必须遵循以下行为约束。

### 视口与 S2 网格选择

- 服务端基于 `viewport.bounds` 计算视口的大致物理跨度（例如对角线长度，单位米）；
- 根据视口大小和预估结果规模，选择一个内部使用的 S2 `level`：
  - 目标是在该 `level` 下，视口覆盖的 S2 单元格数量处于一个合理区间（例如 50–200）；
  - 具体 heuristic 可根据实际数据分布调整，但必须保证在全局范围和局部范围下都具有可扩展性；
  - 该 `level` 对客户端不透明，不作为请求参数出现。
- 使用 S2 覆盖算法（例如 `RegionCoverer`）在选定的 `level` 上计算 `viewport.bounds` 覆盖的 S2 单元格集合，记为 `CoveringCells`。
- 若 `CoveringCells` 为空，视为当前视口范围内没有任何已知对象，此时响应中既不返回 `heatmap`，也不返回 `hotspots`。

### 统计视图（heatmap）与明细视图（hotspots）的选择

服务端基于当前视口、过滤条件以及预估结果规模，在“热力统计”和“明细列表”之间进行自动选择：

- 定义实现层面的阈值（示例，具体数值由实现决定）：
  - `MAX_HOTSPOT_ITEMS`：单次请求允许返回的热点对象最大数量（例如 200）；
  - `MAX_HEATMAP_BUCKETS`：单次请求允许返回的热力网格最大数量（例如 500）。
- 典型行为模式：
  1. 服务端先基于预聚合统计和/或近似估算，判断在当前视口和过滤条件下的对象规模：
     - 若预估对象数量远大于 `MAX_HOTSPOT_ITEMS`：
       - 返回 **热力统计**：
         - 在内部选定的 S2 `level` 上，对 `CoveringCells` 做聚合，得到每个单元格的 `total_count` 及按类型拆分的 `breakdown`（若需要）；
         - 仅返回前 `MAX_HEATMAP_BUCKETS` 个最“热”的单元格（按 `total_count` 降序截断）；
         - 在响应中只设置 `heatmap` 字段，不返回 `hotspots` 字段。
     - 否则（对象规模较小或视口范围足够小）：
       - 返回 **热点明细**：
         - 在当前视口范围内（由 `viewport.bounds` 定义），按空间范围和过滤条件查询具体对象；
         - 支持基于游标的分页，`limit` 默认 50，最大不超过 200（由 `PaginationParams` 中的限制规则约束）；
         - 在响应中只设置 `hotspots` 字段，不返回 `heatmap` 字段。
  2. 当选定的结果类型对应的数据为空时：
     - 响应体中既不返回对应字段，也不返回另一种类型字段。

实现方可以根据实际数据量和性能需求调整判断逻辑，但必须满足：

- 对客户端而言，接口行为稳定可预期：在大范围/高密度场景下倾向返回 `heatmap`，在小范围/低密度场景下倾向返回 `hotspots`；
- 任何情况下都不会同时返回 `heatmap` 和 `hotspots`；
- 聚合和查询实现必须基于 S2 网格和视口范围，不得重新引入行政区或 region 概念。

## 错误码

- `10000`：请求参数不合法，例如：
  - `viewport.bounds` 超出允许范围或 east/west 不满足约束；
  - `TimeRangeParams` 中 `start_time > end_time`；
  - 其他违反本模块参数约束的情况。
