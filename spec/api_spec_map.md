# HotSpot API - 地图渲染

> 本文档定义 HotSpot 地图渲染 (`map`) 相关的 RPC 接口。

## 模块信息

- **命名空间**: `map`
- **基础路径**: `/api/v1/map/`
- **引用**: 
  - 基础规范：`api_spec_base.md`
  - 通用类型：`api_spec_types.md`
  - 错误码：`api_spec_errors.md`

## 设计理念

地图模块不再依赖任何行政区或 region 概念，完全基于 S2 网格进行空间划分与聚合：

- 所有带位置的实体（用户、群组、动态）在服务端被投影到服务端选定层级的 S2 网格单元（层级由视口跨度与数据密度自动决定，对客户端不透明）；
- 服务端根据当前视口的物理跨度和预估结果规模，自动选择合适的 S2 层级（`level`），并使用 S2 覆盖算法确定视口内的网格集合；
- 在大尺度视野下返回**网格热力统计**，在小尺度视野下返回**具体热点对象列表**，对客户端透明；
- 查询实现依赖预聚合的 S2 统计表，而不是直接扫描明细记录，以保证在大数据量和高并发场景下的可扩展性。

本模块通过单一端点实现上述能力：

- **地图画布（canvas）**：基于视口范围和 S2 网格预聚合，自动返回热力统计（heatmap）或热点明细（hotspots）中的一种。

## 端点总览

| 端点 | 说明 | 认证 |
|------|------|------|

| `POST /api/v1/map/canvas` | 根据视口返回统计或热点明细 | 可选 |
## 端点详情

### POST /api/v1/map/canvas

根据视口范围和 S2 网格预聚合，自动返回热力统计（`heatmap`）或热点明细（`hotspots`）中的一种。

- **认证**: 可选

**Request**: `MapCanvasRequest`

**Response**: `ApiResponse<MapCanvasResponse>`

> 类型定义见下方「类型定义」；服务端可观测行为见「行为约束」。`MapCanvasResponse` 为 tagged union（按 `mode` 分支），互斥由结构保证。

## 类型定义

以下基础类型定义于 `api_spec_types.md`：

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
// 暂不支持跨国际日期变更线视口（需由客户端拆分请求）。

// 地图视口：仅由经纬度边界决定，不需要客户端提供 zoom
interface MapViewport {
  bounds: Bounds;
}

interface TimeRangeParams {
  start_time?: string;  // UTC ISO 8601
  end_time?: string;    // UTC ISO 8601
}
// 约束：若同时提供，必须满足 start_time <= end_time。
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

  /** 按类型拆分；当 data_type = 'all' 时返回完整三分类拆分，否则仅返回对应类型计数（此时可省略） */
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
  // 位置统一使用 RelativeLocation（含 distance/bearing_degrees），与 feed/search 对齐
  // distance/bearing 相对于视口几何中心计算（见 MapCanvasRequest 说明），由服务端计算
  location: RelativeLocation;
  created_at?: string;            // 对于动态等有时间语义的对象可返回创建时间（UTC ISO 8601）
}

// hotspots 明细即 PaginatedData<MapHotspotItem>（游标分页，见下方 MapCanvasResponse）
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
  // pagination 仅在返回 hotspots 时使用；limit 覆盖全局默认为 default=50、max=200（base.md 允许模块显式覆盖）
  pagination?: PaginationParams;
  // 注：无 reference_point 字段。distance/bearing_degrees 的计算锚点 = viewport 几何中心
  //     （由 bounds 四角求平均得到）。客户端无需传入锚点；服务端按视口中心计算所有 hotspot 的相对距离/方位。
}

// 响应为 tagged union：模式由服务端按视口尺度与数据密度决定（大尺度→heatmap，小尺度→hotspots）
// 客户端按 mode 分支渲染；结构本身编码了互斥规则，无需散文约束
type MapCanvasResponse =
  | { mode: 'heatmap'; heatmap: MapHeatCanvas }
  | { mode: 'hotspots'; hotspots: PaginatedData<MapHotspotItem> }
  | { mode: 'empty' };   // 视口内无足够数据时返回空态（既无 heatmap 也无 hotspots）
```

约束：

- `MapCanvasResponse` 为 tagged union，`mode` 字段互斥地指明返回形态（`heatmap` / `hotspots` / `empty`），互斥规则由结构本身保证，无需客户端额外校验；
- 视口内无足够数据时返回 `mode: 'empty'`（不再用"两者都省略"表达空态）。

### 行为约束（契约级）

服务端在实现 `/api/v1/map/canvas` 时，对客户端承诺以下**可观测行为**；具体 S2 层级选择、覆盖算法与内部阈值属于实现细节，不对外暴露。

#### 视口与结果类型选择

- 服务端基于 `viewport.bounds` 计算视口物理跨度，并结合预估结果规模，自动决定返回统计还是明细
- 对客户端而言行为稳定可预期：
  - **大范围 / 高密度**：倾向返回 `heatmap`（热力统计）
  - **小范围 / 低密度**：倾向返回 `hotspots`（热点明细）
- **互斥保证**：同一次响应中 `heatmap` 与 `hotspots` 最多出现一个字段，不会同时返回
- 当视口范围内无任何已知对象时，既不返回 `heatmap` 也不返回 `hotspots`

#### 明细分页

- 仅在返回 `hotspots` 时支持游标分页（`pagination` 生效）
- `limit` 默认 50、最大 200（本端点显式覆盖 base.md 的全局默认 20/100，因为地图热点明细单次需返回较多点位）；返回的明细条数受服务端单次上限裁剪
- **排序**：主序为服务端计算的 `distance` 升序（相对视口几何中心），`object_type` + `object_id` 组合作为稳定 tie-breaker（游标翻页依赖确定性顺序）；一致性级别为弱一致（聚合统计场景允许短暂延迟，同一 `cursor` 会话内保持排序基准一致）

#### 实现中立

- 聚合与查询必须基于 S2 网格与视口范围，不得引入行政区 / region 概念作为契约维度
- `cell_token`、`level` 对客户端是不透明标识，客户端不应依赖其具体含义或编码格式
