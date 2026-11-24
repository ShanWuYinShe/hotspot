# ArkUI 布局与滚动陷阱（hotspot harmonyapp）

本文件是 skill `hotspot-harmony-ui` 的明细：实拍确认过的渲染/滚动坑，都会造成「观感与代码意图不符」。

## 1. `.border()` 写在 `.borderRadius()` 之后会把圆角重置为直角

- 现象：同一对并排按钮，无 border 的呈胶囊（borderRadius=999 生效），带 hairline border 的呈直角矩形——尽管两者引用同一个 borderRadius 值（SHAPE.BUTTON=RADIUS.FULL）。
- 根因：ArkUI 属性后设覆盖先设；`.border({width,color})` 自带的 radius 字段未显式给出时按 0 生效，把先前 `.borderRadius()` 的值覆盖掉。
- 修复：**`.border()` 一律写在 `.borderRadius()` 之前**（后设的 borderRadius 覆盖 border）。
- 排查口径（可复现）：扫「`.borderRadius(` 之后 3 行内出现 `.border(`」的候选——
  `grep -rn -A3 '\.borderRadius(' harmonyapp/entry/src/main/ets | grep -B2 '\.border('`
  （扫描会报出候选，但部分元素本意即直角，视觉影响需逐例核实）。
- 实拍案例：ProfilePage 升级表单「取消」直角 vs「确认升级」胶囊。

## 2. Scroll 内容短于视口时默认垂直居中

- 现象：短表单在 `Scroll{...}.layoutWeight(1)` 里「飘」在页中段，页头下方出现大空隙（实测 Scroll 高 597vp、内容 289vp，上下各留 154vp）。
- 根因：Scroll 对小于滚动区的内容默认居中对齐。
- 修复：表单类 Scroll 一律补 `.align(Alignment.Top)`（升级表单即按此处理）。
- 排查：dump 里 Scroll bounds 与其内容 Column bounds 的 y1 不一致即为居中。

## 3. Refresh 内容区结构与防抖动契约（彻底消除下拉刷新鬼畜晃动）

- **必须显式配置 `.pullToRefresh(false)`（强制松手才刷新，晃动最核心根因）**：ArkUI NEXT (API 12+) 的 `Refresh` 属性 `pullToRefresh` 默认值为 `true`！一旦下拉位移超过 `refreshOffset`（默认 64vp），**不等用户手指松开，系统便立即启动刷新并执行吸附回弹动画**；若用户手指仍在向下拉拽，**用户触控向下拉伸与系统吸附回弹在同一时刻剧烈对抗**（手指拉下→系统弹回→手指拉下→系统弹回），形成每秒数十次的高频振荡（俗称「鬼畜晃动」）。**修法**：必须显式配置 `.pullToRefresh(false)`，恢复标准移动端手势（下拉松手后才发起刷新），手指触摸期间绝对不自动吸附打断。
- **嵌套滚动容器必须设置 `.edgeEffect(EdgeEffect.None)`**：`List` 默认带有 `EdgeEffect.Spring`；嵌套在 `Refresh` 内时两个弹性阻尼系统同时响应垂直手势，产生双重弹簧手势竞争。必须在内层 `List` 上显式设置 `.edgeEffect(EdgeEffect.None)`，弹性完全由外层 `Refresh` 单一容器统一接管。
- **自定义刷新指示器统一使用全局锁高的 `builder: RefreshIndicator`**：`EmptyState.ets` 导出的 `RefreshIndicator` 锁高 56vp（`MIN_HIT_TARGET + MD`）。不要只传 `refreshingContent` 而缺省 `builder`：未配 `builder` 时下拉阶段系统使用默认圆圈指示器，进入刷新态才突变为自定义内容，两套指示器高度和布局在树中交替挂载，造成高度突变跳动。全站统一使用 `builder: RefreshIndicator`。
- **列表项必须使用 `Pressable` 原语，禁止在行上写 `onTouch` 修改页面级 `@State`**：在 `ListItem` 上手动写 `.onTouch` 修改父组件 `@State`，会在手指按下和滑动被 List 拦截为 Cancel 时频繁触发布局重算；按压反馈统一收敛在 `Pressable` 内部，保持页面状态无抖动。
- **列表页的 Refresh 内容区用 `List`，不要用 `Scroll{ Column{...} }`**：Refresh 下拉时内容区高度会随指示器区域连续变化，Scroll 包裹动态 ForEach 在该过程中会重排失稳；改 `List` 后行布局稳定。注意 `List` 的行距不要再用 `space`（活动行自带上下 padding，叠 space 会让节奏变松）。
- **不要在 `Refresh.onOffsetChange` 里逐帧写 `@State`**（例如记录位移做渐显动画）：状态更新会触发整页重建 → Refresh 重新计算位移 → 再次回调，形成反馈环。
- 判定口径：端上连续截图（`snapshot_display` 循环）抓「下拉中 / 松手后 / 刷新中」三态，看 spinner 相对页头的位置与内容行位移序列是否单调；不要只截静态终态。

## 4. 裸 `{ }` 块不是容器

- 在 `if/else` 分支里用裸块包裹布局（如把两栏 `if` 简化成 `else { { Column() {...}.layoutWeight(1) } }`）时，块内 `.layoutWeight()` **不生效**，容器高度塌陷 → 内部 `WaterFlow`/`List` 高度为 0 → **页头（标题/计数/筛选）正常但内容一条不渲染**。类型检查与编译都不会报错，必须端上或预览器确认。
- 修法：删掉裸块，容器直接作为分支体。
- 自查口径：任何「改完结构后内容区空白」的问题，先看分支体是不是裸块、`.layoutWeight()` 是否直接挂在**容器组件**上（Column/Row/List/WaterFlow）。判定手法：dump 节点数（本例页头 10 节点 → 正常时 52 节点）。

## 5. `.overlay()` 会拦截滚动容器内子元素的点击与手势

- 现象：聊天页消息点按全部无响应——点图片不开大图预览、点消息不开操作面板；但顶栏返回、Tab 切换、输入栏发送都正常。
- 根因：`List.overlay(buildJumpToBottomOverlay(), { align: Alignment.BottomEnd })`。ArkUI 的 overlay 节点会拦截被覆盖区域的点击/手势；即使按钮内部隐藏时已设 `hitTestBehavior(None)`，**overlay 层本身仍吞掉 List 内的点击**（与 R112 设置页/搜索页「空 overlay 既拦滚动也拦点击」同源，见 `docs/ui-improvement-handoff.md` #33）。
- 修复：浮动按钮改用 **Stack 同级子节点** 承载——列表/滚动容器外层包 `Stack`，按钮放进全屏 `Column`（`justifyContent(End)` / `alignItems(End)`）+ `hitTestBehavior(HitTestMode.Transparent)`，按钮自身的 `margin` 保持原定位；删除 overlay builder。实拍案例：ChatPage 回底按钮（PR #268）。
- 判定口径（可复现）：滚动容器上出现 `.overlay(` 后，端上点按容器内行/卡片无反应，而容器外控件正常 → 先怀疑 overlay；把 `.overlay(...)` 改为 Stack 同级 Transparent 容器后同坐标复测。
- 自查命令：`grep -rn '\.overlay(' harmonyapp/entry/src/main/ets --include=*.ets`。**不要在 `List` 上直接挂 `.overlay()`**；浮动按钮优先用 Stack 同级子节点（第 5 条修复模式）。
- 既有 overlay 端上抽查（本轮）：`NotificationCenter` 挂在 `Refresh` 上、回顶为小按钮，行点击与跳转正常（点已读通知 → 瞬间详情）；`SearchPage` 挂在外层 `Column`、回顶用全屏 `Transparent` 容器，搜索结果行点击正常（点用户 → 他人主页）；`MomentsFeedPage` 与搜索页同构。风险只出现在 `List.overlay` + 小按钮组合（聊天页实测），其它列表若同样挂法按第 5 条修复。

## 6. List 内长按手势的端上验收：`uitest longClick` 不一定触发 ArkUI `LongPressGesture`

- 现象：给聊天图片气泡加 `LongPressGesture({ duration: 500 })` 后，`uitest uiInput longClick` 复测既不弹操作面板也不报错；同组件 `.onClick`（点按）与同页其它点击都正常，`hasActions` 也为 true（点自己新发的文本消息能正常弹面板）。
- 判定：`uitest uiInput longClick` 注入的是「按压后抬起」的事件序列，能触发 `onTouch` 计时器，但不保证 ArkUI 手势识别器（LongPressGesture）判定成功；**LongPressGesture 写在 List/ListItem 内时更不稳定**（与列表滚动手势竞争、抬手即 Cancel 判定的时机差异）。
- 需要「长按必须可端上验收」的场景，改用 `onTouch` 计时器：Down 起 500ms 计时器触发、`Move` 超过 10vp 取消、`Up/Cancel` 取消、`aboutToDisappear` 清理；`onClick` 里用 `longPressFired` 标记避免长按后再触发点按。实拍案例：ChatMessageItem 图片气泡（PR #271）。
- 若坚持用 `LongPressGesture`：至少用 CUA/真实桌面点击或真机手动长按验收，不能只信 `uitest uiInput longClick` 的调用成功（它只表示事件已注入）。
