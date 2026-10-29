---
name: hotspot-frontend-map
description: hotspot 前端自研地图契约：marker 覆盖全部标记（图钉/热力点/我的位置蓝点）且都必须按地理坐标投影、GCJ-02 纠偏同口径、滚轮光标锚缩放、覆盖层 pointer-events 契约、缩放漂移的浏览器实测方法。触发：修改地图相关代码（MapCanvas/MapOverlays/MapPinMarker/MapHeatDot/LocationDesktop/Mobile）；新增地图标记；验证 marker 缩放不漂移与滚轮缩放；排查"缩放越大 marker 越偏"。
---

# hotspot-frontend-map

## marker 语义红线：marker 指所有标记，不只是图钉

「marker 坐标不因缩放而偏移」中的 **marker 是全量语义**，包括三类，缺一不可：

| 标记 | 渲染组件 | 定位方式 |
|---|---|---|
| 图钉 | `routes/(app)/map/shared/MapPinMarker.svelte` | 经纬度投影（`pos.x%/pos.y%`）|
| 热力点 | `components/map/MapHeatDot.svelte` | 经纬度投影 |
| **我的位置蓝点** | `routes/(app)/map/shared/MapMyLocationDot.svelte`（经 `map/shared/MapOverlays.svelte` 挂载） | 经纬度投影（与图钉同口径，缩放/平移后钉在同一地理坐标）|

**蓝点语义（用户 2026-10-10 权威澄清，红线）**：蓝点代表「**我的地理位置**」，不是
「地图中心」——禁止用 `left:50%;top:50%` 钉视口中心来模拟，任何把二者混同的实现都是 bug。

**反模式（已被取代）**：曾用 `left:50%;top:50%` 钉视口中心模拟蓝点——以偏离中心的光标为锚
滚轮缩放一级（14→15）时底图按光标锚移动、蓝点不动，实测等效漂移 1339px，拖拽后同样失真。
现由 `MapMyLocationDot` 按「我的位置 lat/lng → 地理投影」实现；任何把「我的位置」与
「地图中心」混同的实现都是 bug。

## 地理投影契约（防「缩放越大 marker 越偏」的根因）

- **同空间原则**：bounds 与 marker 坐标必须投到同一坐标系再算百分比。底图为高德
  （GCJ-02 瓦片）时，WGS-84 的 marker 坐标先经 `wgs84ToGcj02` 纠偏；OSM 底图恒等直通。
  混用两个空间是国内场景 zoom 放大后 marker 与底图地标错位的根因（注释见
  `MapPinMarker.svelte`，勿删）。
- `gcj02ToWgs84` 必须迭代到厘米级，否则高 zoom 下「我的位置」肉眼可见滑 off
  （测试：`__tests__/map-projection.gcj.test.ts`）。
- 新增任何标记一律走经纬度投影，**禁止**用视口百分比（`left:50%` 之类）模拟地理位置。

## 交互契约

- 滚轮缩放：以**光标位置为锚**（标准地图行为），deltaY 累计 120 触发一级，
  范围 `MAP_MIN_ZOOM` 3 ~ `MAP_MAX_ZOOM` 19。
- 滚轮监听必须用非 passive 原生监听挂**画布的父级**（Svelte 5 的 `onwheel` 属性是
  passive，preventDefault 不生效会页面跟着滚）；挂父级是因为 marker 覆盖层与画布平级，
  悬停在 marker 上滚动的事件不经过画布。
- 覆盖层 `pointer-events-none` 放行拖拽/滚轮给画布，marker 按钮自身恢复
  `pointer-events:auto`。**覆盖层若整体拦截真实指针事件 → 拖拽/滚轮全失效**，且
  合成事件测试测不出来（历史修复 0f6528aa，线上 v1.3.1 因缺此提交滚轮全失效）。

## 浏览器实测方法（agent-browser，合成 wheel 事件即可生效）

1. 打开 `http://localhost:8888/map`（本地 Docker，先用 `./scripts/seed-dev.sh` 注入种子数据；未登录可看公开数据，登录后覆盖个人 marker）。
2. 图钉锚定验证：记录 marker 屏幕坐标 → 派发 `WheelEvent(deltaY:-120)`（光标故意
   偏离中心）→ 缩放一级后应满足 `p' = cursor + 2·(p − cursor)`（相对误差 ≈0）。
3. 缩放级从瓦片 `img.map-tile` 的 `src` 读 `z=`。
4. **蓝点元素查找的坑**：浏览器会把 style 属性规范化（`left:50%` → `left: 50%`），
   按 `getAttribute("style").includes("left:50%")` 找会扑空；按覆盖层
   （`className` 含 `pointer-events-none absolute inset-0`）的子元素找。
5. 多标签时先 `agent-browser tab list` 确认当前标签——评估跑错标签会得出假结论
   （2026-10-09 实际踩过：把线上页当本地测）。
6. 结论口径：marker 无漂移 = 所有标记（含蓝点）缩放/平移后仍钉在同一地理坐标。
