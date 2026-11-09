---
name: hotspot-frontend-map
description: hotspot 前端自研地图契约：marker 覆盖全部标记（图钉/热力点/我的位置蓝点）且都必须按地理坐标投影、GCJ-02 纠偏同口径、滚轮光标锚缩放、覆盖层 pointer-events 契约、缩放漂移的浏览器实测方法。触发：修改地图相关代码（MapCanvas/MapOverlays/MapPinMarker/MapHeatDot/LocationDesktop/Mobile）；新增地图标记；验证 marker 缩放不漂移与滚轮缩放；排查"缩放越大 marker 越偏"。
---

# hotspot-frontend-map

## marker 语义红线：marker 指所有标记，不只是图钉

「marker 坐标不因缩放而偏移」中的 **marker 是全量语义**，包括四类，缺一不可：

| 标记 | 渲染组件 | 定位方式 |
|---|---|---|
| 图钉 | `routes/(app)/map/shared/MapPinMarker.svelte` | 经纬度投影（`pos.x%/pos.y%`）|
| 热力点 | `components/map/MapHeatDot.svelte` | 经纬度投影 |
| **我的位置蓝点** | `routes/(app)/map/shared/MapMyLocationDot.svelte`（经 `map/shared/MapOverlays.svelte` 挂载） | 经纬度投影（与图钉同口径，缩放/平移后钉在同一地理坐标）|
| **浏览锚点标记** | `routes/(app)/map/shared/MapAnchorMarker.svelte`（经 `MapOverlays` 挂载） | 同口径经纬度投影；44px 热区、点击回到锚点 |

**蓝点语义（红线）**：蓝点代表「**我的地理位置**」，不是
「地图中心」——禁止用 `left:50%;top:50%` 钉视口中心来模拟，任何把二者混同的实现都是 bug。

**反模式**：用 `left:50%;top:50%` 钉视口中心模拟蓝点——以偏离中心的光标为锚
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
  范围 `MAP_MIN_ZOOM` 3 ~ `MAP_MAX_ZOOM` 18（实测上限：高德 z=19 返回 HTTP 200 空白瓦片、`tile.openstreetmap.de` z=19 返回 404，继续放大只会整屏空白）。
- 滚轮监听必须用非 passive 原生监听挂**画布的父级**（Svelte 5 的 `onwheel` 属性是
  passive，preventDefault 不生效会页面跟着滚）；挂父级是因为 marker 覆盖层与画布平级，
  悬停在 marker 上滚动的事件不经过画布。
- 覆盖层 `pointer-events-none` 放行拖拽/滚轮给画布，marker 按钮自身恢复
  `pointer-events:auto`。**覆盖层若整体拦截真实指针事件 → 拖拽/滚轮全失效**，且
  合成事件测试测不出来（缺这段逻辑时滚轮全失效）。

## 浏览锚点（Browse Anchor）

**语义红线**：浏览锚点是「**附近页距离/推荐计算的基准位置**」，不是地图中心，也不是我的位置。地图页必须让用户同时看到三者，否则就是「点了按钮但不知道发生了什么」。

- **存储**：`localStorage.hotspot_last_location` = `{ lat, lng, name? }`（`lib/map-auto-locate.ts` 的 read/write/clear；历史 `[lat,lng]` 数组兼容）。写入/清除派发 `LOCATION_ANCHOR_UPDATED_EVENT`，地图页常驻时必须监听并同步 `anchorPoint`，否则卡片/标记停留在挂载时旧值。
- **名称**：选中地点名优先，否则当前中心地名；「未命名区域 / 我的位置」不落库（`resolveAnchorName`），展示时回退「自定义位置」。
- **三处 UI（桌面/移动共用组件）**：
  1. 左面板 / 底部 Sheet 的 `BrowseAnchorCard`：已保存锚点名 + 「恢复默认」+ 用途说明 + 「地图中心」行（含距锚点距离）；
  2. 地图上的 `MapAnchorMarker`（区别于实体图钉与蓝点：accent 胶囊「锚点 · 名称」，点击回到锚点）；
  3. 地图几何中心的 `MapCenterCrosshair` 准星：拖地图改变的就是它，设锚点时以它为落点。
  桌面左面板与移动/平板底部 Sheet 共用以上三个组件 + `AnchorConfirmButton`：布局不同（左栏 vs Sheet），语义/状态完全一致；改动锚点 UI 必须三端一起验证。移动 Sheet 内容顺序固定为「搜索框 → 搜索建议/选中地点/最近使用 → 浏览锚点卡片 → 贴底确认 CTA」：锚点卡不得回到搜索框正下方（会把「最近使用」空态挤到 mid 档首屏之外）。
- **确认反馈（按钮内联，禁止成功 toast）**：`AnchorConfirmButton` + `anchor-feedback.svelte.ts` 状态机：idle（主按钮「将地图中心设为浏览锚点」/「以此地点为浏览锚点」）→ pending 1.0s（spinner，aria-busy）→ success 1.4s（✓「浏览锚点已生效」）→ idle；失败 error 1.8s（按钮内提示 + 点击重试）。**中心≈锚点时也保持主按钮**（不出现灰色 disabled，避免「以为不能更新」），点击 = 重复确认；「恢复默认」同样不弹 toast，由卡片/标记变化反馈。
- **逻辑**：底部 CTA 落点 = 选中地点优先，否则地图中心；「恢复默认」清除锚点并回到服务端默认位置。**「定位到我」只移动视野/蓝点，禁止静默改写锚点**；进入地图不再自动请求浏览器定位（避免授权弹窗打断探索），已有锚点时以锚点为首屏视野；位置仅在用户显式点「定位到我/回到我的位置」时获取。
- **验收（浏览器）**：清空 localStorage → 卡片「未设置」+ 无标记 + CTA「将地图中心设为浏览锚点」；点 CTA → 按钮内依次出现「正在生效…」→「浏览锚点已生效」→ 恢复主按钮（全程无成功 toast），卡片/标记同步；中心≈锚点时按钮仍是主按钮且可点；注入远端锚点 → 显示距离且 CTA 可点；「恢复默认」→ 三处回到未设置；移动端同流程（Sheet 内卡片 + 粘底 CTA）。

## 浏览器实测方法（agent-browser，合成 wheel 事件即可生效）

1. 打开 `http://localhost:8888/map`（本地 Docker，先用 `./scripts/seed-dev.sh` 注入种子数据；未登录可看公开数据，登录后覆盖个人 marker）。
2. 图钉锚定验证：记录 marker 屏幕坐标 → 派发 `WheelEvent(deltaY:-120)`（光标故意
   偏离中心）→ 缩放一级后应满足 `p' = cursor + 2·(p − cursor)`（相对误差 ≈0）。
3. 缩放级从瓦片 `img.map-tile` 的 `src` 读 `z=`。
4. **蓝点元素查找的坑**：浏览器会把 style 属性规范化（`left:50%` → `left: 50%`），
   按 `getAttribute("style").includes("left:50%")` 找会扑空；按覆盖层
   （`className` 含 `pointer-events-none absolute inset-0`）的子元素找。
5. 多标签时先 `agent-browser tab list` 确认当前标签——常见误判是把线上页当本地测，评估前必须核对标签归属。
6. 结论口径：marker 无漂移 = 所有标记（含蓝点）缩放/平移后仍钉在同一地理坐标。
