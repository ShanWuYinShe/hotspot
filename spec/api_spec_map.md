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

**服务端智能决策**：客户端只负责"渲染"，不再手动决定"何时拉聚合/何时拉列表"。

**视口驱动 (Viewport-Driven)**：完全基于当前视口 (`bounds`) 和缩放级别 (`zoom`)，服务端根据算法（如网格聚合/K-Means）动态返回最适合当前视野的展示项。

**高德模式**：
- Low Zoom (远)：返回行政区/热门区域的聚合点 (`type: 'cluster'`, `label: '海淀区 500+'`)
- Mid Zoom (中)：分裂为更小的聚合或展示头部热点 (`type: 'cluster'`, `preview_urls: [...]`)
- High Zoom (近)：直接展示具体的 Poi (`type: 'user'/'moment'`, `entity: {...}`)

## 端点总览

| 端点 | 说明 | 认证 | 幂等 |
|-----|------|------|------|
| `POST /api/v1/map/render` | 获取地图渲染数据 | 可选 | 无 |

## 类型定义

以下类型定义于 `@spec/api_spec_types.md`：
- `GeoPoint`

## 端点详情

### POST /api/v1/map/render

获取地图渲染数据。认证可选，字段可见性规则见 `@spec/api_spec_base.md`。位置保护：服务端可对匿名不可见实体进行聚合或降精度处理。

- **认证**: 可选
- **幂等**: 无

**Request**: `MapRenderRequest`
```typescript
interface MapRenderRequest {
  viewport: MapViewport;
  filter?: {
    data_type?: MapDataType;
    time_range?: TimeRangeParams;
    only_friends?: boolean;  // 仅登录态生效；未登录时忽略此参数（视为 false）；已登录但未传时同样视为 false
  };
}
```

**Response**: `ApiResponse<MapCanvas>`
```typescript
type MapDataType = 'user' | 'group' | 'moment' | 'all';  // 过滤器：限定返回哪些类别的数据

// 返回结果中每项的实际类型。当缩放级别较低时，即使 data_type 指定了具体类别，
// 仍可能返回 cluster 类型的聚合项（cluster 内聚合的数据类别与 data_type 一致）。
type MapItemType = 'cluster' | 'user' | 'group' | 'moment';

interface Bounds {
  north: number;
  south: number;
  east: number;
  west: number;
}
// 约束：north >= south；east >= west（允许退化视口）；east/west 取值范围 [-180, 180]。暂不支持跨国际日期变更线视口（需由客户端拆分请求）。违反时返回 `10000`

interface MapViewport {
  bounds: Bounds;
  zoom: number;  // 取值范围 1-20，约束定义见 @spec/api_spec_base.md 通用字段约束
}

interface TimeRangeParams {
  start_time?: string;  // UTC ISO 8601
  end_time?: string;    // UTC ISO 8601
}
// 约束：若同时提供，必须满足 start_time <= end_time。违反时返回 `10000`

interface MapRenderItem {
  render_id: string;  // 渲染项唯一 ID（聚合时由服务端生成，实体时为 payload_id）
  type: MapItemType;
  location: GeoPoint;  // 渲染坐标（聚合点为聚合中心，实体为实际坐标）；地图场景无需距离/方位，仅需坐标用于标记定位
  
  cluster?: {
    count: number;
    label?: string;
    preview_urls?: string[];
  };

  entity?: {
    title: string;
    subtitle?: string;
    cover_url?: string;
    payload_id: string;  // 原始业务 ID（user_id / group_id / moment_id）
    category?: string;   // 业务分类
  };
}

interface MapCanvas {
  items: MapRenderItem[];  // 最大返回 500 个渲染项，超出时服务端自动聚合
  toast_message?: string;
}
```

**错误码**: `10000`
