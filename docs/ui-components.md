# 前端 UI 组件清单（原语目录与统一契约）

> 组件是壳，design token 是壳的大脑（`src/styles/theme/`）。本清单是前端 UI
> 组件的唯一目录：新增任何 UI 前先查这里；一个交互意图只允许一种反馈实现，
> 新需求通过扩展原语的 prop 表达，而不是新建近似组件。
> 审计基线：2026-09-06（ui-list-inset 批次），90 个组件全量盘点。

## 原语目录

### 布局壳（每页恰好一次，页面不得自建）
| 原语 | 职责 |
|---|---|
| `layout/MainLayout` → desktop/tablet/mobile 三壳 | 设备分派、侧栏/底栏/顶栏 |
| `layout/page/RoutedContentViewport` | 三端唯一滚动 owner（`data-desktop-main`） |
| `layout/page/PageViewport` | 路由画布（`data-page-viewport`） |
| `layout/ambient/AmbientBackdrop` | 页面氛围画布（三主题世界层装饰，`z-index:-1`，由 PageViewport/认证页/错误页/启动占位统一挂载） |
| `layout/page/StandardPageLayout` / `SplitPageLayout` / `ContentColumn` | 页面骨架与内容栏档位 |
| `layout/page/PageHeader` | 唯一页面头部（sticky/内容内 `integrated`/搜索/ below 插槽） |

### 列表（行交互的唯一体系）
| 原语 | 职责 |
|---|---|
| `list/ListContainer` | 列表容器。**2026-10-27 Round 3**：`bordered` 语义更新为「开放页面内容软岛」——`bg-surface-island + shadow-xs + rounded-lg(20)`、无 1px 描边、`p-2`(8)、行背景透明、行间 `gap-y-0.5`(2)、hover 填充离岛边 8（不再画行横线）；embedded（面板内集合）× `inset`（内缩圆角行）/ 默认（分隔线）不变。调用方不得再给 `bordered` 覆写 border/bg/radius |
| `list/ListItem` | **唯一行交互原语**。hover/选中填充、12px 圆角、键盘、aria、`tone: joined/card`。行级卡片类禁止（eslint 强制）。列表容器内自动包 `role=listitem`（ListContainer 经 context 声明；ListEnd/InlineLoading 同机制），裸用不变。行点击由内部透明覆盖按钮（`[data-cover]`）承载，行内操作按钮自动抬升（z 序），根不持 button 语义（axe nested-interactive 清零） |
| 行包装组件 | `ChatListItem` / `home/ActivityRow` / `profile/UserCardRow` / `forms/SettingRow`——只做内容编排，交互全部委托 ListItem |

### 卡片（CSS 卡语言 + 一个内容卡原语）
| 原语 | 职责 |
|---|---|
| `data-display/NearbyImageCard` | 图片优先内容卡**唯一实现**（附近瞬间/附近的人/附近群组/首页轮播/群组详情）；根节点必须 `isolate` + `use:cardPress` |
| `data-display/ImageCardActions`（R14） | **图片卡覆盖操作行唯一实现**：点赞/评论/分享/收藏四类操作、计数弹跳、状态语义与 hover/press/focus 全站一致；`size="md"\|"sm"`（sm 用于手机 2 列窄卡） |
| `.media-badge` / `.media-badge--icon`（utilities，R14） | **压图徽章唯一类**：深色遮罩底 + 反色字 + h-5 紧凑档（**禁止 backdrop-filter**，见 design-system §21.11）；分类/位置/距离/1-N/已关注/身份角标不再手写组合 |
| `data-display/Badge`（2026-10-29） | **非压图徽章/标签唯一原语**：尺寸严格两档 `sm`=20px（`h-5 px-2 text-xxs`）/`md`=24px（`h-6 px-2.5 text-xs`）；`tone`=`neutral/accent/solid/danger/warning/online/outline`；数字未读用 `shared/UnreadBadge`（`variant="accent"\|"inverse"`）。旧 `.badge` 类只保留兼容，新代码不得再用。**`style` 逃生口仅限地图 marker 调色板等动态语义色**（值必须是 design token 变量，禁止 hex/rgba；常规场景一律用 `tone`）。|
| `use:cardPress`（lib/motion，R14） | 卡片按压统一入口：卡本体/覆盖按钮按下 → 根节点 `data-pressed` → CSS 按 `--cst-card-tap-scale` 缩放；行内按钮按下不缩放整卡 |
| `card-hover`（utilities） | 可点击卡片唯一 hover 语言：投影抬升 + `--cst-hover-lift`（vibrant）/ 纸面微染+边框（forest）；按压档由 `[data-pressed]` 规则承载（`--cst-card-tap-scale`） |
| `card-raised` / `surface-hero` / `ListItem tone=card` | 静态卡片语言（白底 + 投影），禁止再发明第四种 |
| 单帖卡（瞬间详情） | 内容软岛（无描边 + `surface-island` + shadow-xs + 圆角 20）；媒体容器去 1px 描边、保留圆角/阴影/加载占位/focus-ring；统计与评论分隔发丝线保留（Round 4） |
| Hero 封面（群详情/主页） | 去 1px 描边，保留圆角/阴影/加载占位（媒体 hero 例外，见 design-system §19.5） |
| 表格（admin） | 外壳去描边改软岛；**表头 bg-subtle + 行线 + 行 hover 保留**；行内按钮次级软填 / soft danger；390 横向滚动可及全部列；分页 ≤20 行不强制 sticky；空态/分页/筛选统一（Round 4） |
| `image-hover-frame` | 相册小方格专用（照片墙缩略图），内容卡不得使用 |

### 按钮 / 输入 / 导航 / 弹层 / 反馈
| 原语 | 职责 |
|---|---|
| `ui/button` | 全部动作按钮的唯一壳（表单/CTA/弹层确认/详情动作区；`ui/FollowButton` 等复合钮经它派生）。**R13 新增 `variant="soft"`**：中性软填底（`bg-surface-subtle/60`）+ 无描边 + hover 全量，是「安静次级动作」的唯一形态（与派 C 同源，替代各页手写的软填/描边变体）。**2026-10-29 新增 `shape`**：`"rect"`（默认，派 A `rounded-md`）/`"pill"`（派 B/C，`rounded-full` + `text-xs`）；调用方禁止再用 class 覆盖 `rounded-*` |
| `navigation/ActionIconButton` | 图标按钮（ghost/outline） |
| `hover-btn` / `scale-btn`（utilities） | 行内文字/图标动作的 hover/tap CSS 语言 |
| `hit-slop` / `hit-slop-y`（utilities） | 紧凑控件的触控热区扩展（视觉尺寸不变）。`hit-slop` 四周各 6px，用于图标按钮等**孤立**控件；`hit-slop-y`（2026-10-19 新增）只扩纵向，用于**横向相邻**的紧凑控件（筛选胶囊、分段控制器）——前者的水平 6px 会与相邻项重叠，命中落到 DOM 靠后那颗。注意：两者都靠 `::before` 绝对定位，**会被祖先的 `overflow` 裁剪**，横滚容器需留出纵向 padding（TabBar 的 `py-1` 即为此） |
| `forms/SearchInput` / `forms/Input` / `forms/Toggle` / `forms/ImagePicker` / `forms/TagPill` | 输入类原语。**2026-10-29 输入族重构**：两者共用 `.input-surface` 表面（单层 1px 边 + hover 边界加深 + accent 2px 视觉焦点 + 3px 柔光环 + danger 错误环 + disabled 弱化底，见 utilities.css）；`forms/SearchInput` = 搜索胶囊（rounded-full / 39px / text-md 14 / 图标 16 / 32px 清空钮 + hit-slop 44 命中 / `ariaLabel` 按页面语义 / 清空后回焦 / Esc 清空 / ⌘K 提示槽）；`forms/Input` = 表单 16 圆角（43px ≈ INPUT 42 / text-md / 默认 px-4 py-2.5；`label`/`labelSnippet`、`prefix`、`clearable`、错误态三件套、`compact` 形态均保留）。契约对齐鸿蒙 `SystemInput`（表单）/`SearchPageHeader`（搜索） |
| `layout/page/PageHeader` | 页面页头（sticky）。**R9：顶部透明（与画布融合、无白带）→ 滚动 >4px 加 glass**（`surface-page 78% + blur(16px) + 细底边 + elevation`，150ms，reduced 立即）；仅背景/边框/阴影变化，不改定位；原 `glass` prop 统一为该行为。**`variant="floating"`（2026-10-29，通用沉浸页原语）**：整行玻璃工具条（返回 + 可见标题 + 更多同一面板：`rounded-md` + 细边 + `bg-surface-primary/85` + `backdrop-blur-md` + `shadow-raised-sm`，吸顶）；保留给需要悬浮 chrome 的沉浸页。**瞬间详情页自 2026-10-29 晚已改为「内容内极简工具行 + 画廊 hero」**，不再使用本变体（见「详情页契约」）。**`variant="integrated"`（2026-10-29，hub/列表页「内容内页头」）**：无背景/边框/吸顶；标题（紧凑 `text-lg sm:text-xl`，不再随 titleSize 变页面大标题）+ 副标题/计数 + trailing 工具同一行，below 筛选/操作 `mt-2.5` 紧随，正文 `pt-4` 开始。适用：/nearby、/nearby/people、/nearby/groups、/nearby/moments、/me/relationships、/me/reports、/notify；这些页面**禁止在正文重复页面标题/再放一条分区头**。搜索页（搜索优先+sticky 输入）、设置/表单页、聊天分栏、品牌首页不属于同类，保持各自形态。 |
| `navigation/BackToTop` | 回顶浮动圆钮（R9）：滚动 > 1.5×视口高出现（44px，移动 bottom-24 / 桌面 bottom-6，避开底栏与浮层），点击平滑回顶（reduced 直接跳）、回顶后 200ms 内隐藏；`aria-label="回到顶部"`；fixed 不占流 |
| `lib/route-announcer` | SPA 导航播报（R9）：`#route-announcer`（sr-only `aria-live=polite`/atomic）播报新页 h1 标题；首屏不播、1.5s 去重；焦点落 `main#main-content`（tabindex=-1、preventScroll），不叠加 h1 聚焦 |
| `navigation/TabBar` | 分类/过滤胶囊（唯一 Tab 原语；底栏走 `layout/mobile/NavTab`）。三个变体：`pills`（筛选/分类，可横滚）、`segmented`（二选一/系统级分段，等宽轨道）、`underline`（页级两态切换：全宽等分、无轨道、激活项 + 2px accent 下划线）。**选中态不得只靠颜色**（2026-10-27 Round 2，用户对色弱/色盲的明确要求）：`segmented` 选中 = accent 实底 + ✓ 图标 + `font-semibold`；`pills`/`underline` 选中 = `font-semibold`（underline 另有下划线形状线索）；验收须过灰度截图（`filter:grayscale(1)`）。`disabled`（2026-10-19）整体禁用并置灰；**移动端触控高度**：按钮 `py-3`（视觉 41px）+ 滚动根 `py-1` + `.hit-slop-y` → 有效 ≈49px，桌面 `lg:py-1.5` 保持紧凑 |
| `overlays/SheetShell`、`ui/dialog`、`ui/popover`、`ui/dropdown-menu` | 弹层体系。**2026-10-29**：`Dialog.Content` 新增 `floating`（浮层卡片式底部 sheet：inset-x-4/bottom-4/max-w-md/rounded-xl/shadow-modal 唯一实现）；调用方只传内距 `p-4|p-5`，不得再自拼定位/圆角/阴影；标准 bottom sheet 仍为 `rounded-t-sheet`(28) 贴底全宽 |
| `overlays/MenuPanel` | 弹出菜单面板唯一实现（标题 + 选项行 + 可选底部动作；容器走 `ui/popover` Content）。已有：地图源菜单、地图筛选菜单。新菜单只传数据，不复制行结构 |
| `data-display/EmptyState` / `InlineLoading` / `LoadError` / `ListEnd`、`overlays/ToastViewport`、`shared/ConnectionBanner` | 状态反馈。`EmptyState` 可选 `spotlight`（2026-10-26）：超大空态的信标装饰层（光晕 + 双环 + 四向星点，纯 token、无动画、`aria-hidden`）；小面板空态不得开启 |
| `data-display/Skeleton` / `SkeletonList` / `SkeletonFeed`（2026-10-26 新增，2026-10-27 更新） | 列表/信息流**首屏骨架屏**。`Skeleton` 基础占位块（`aria-hidden`）；`SkeletonList` 行式（前导圆 36/40/44/48 + 两行文本条，`role=status + aria-busy`，ListContainer 内自动补 listitem）；`SkeletonFeed` 卡式（2026-10-27 起同步整卡图片卡：图片块 + 底部文本条，与真实卡同比例/同文字位置，保证零位移；另有 `card`/`portrait` 形态）。几何对齐鸿蒙 `Skeleton`/`SkeletonShimmer`，动效 `--cst-motion-skeleton`(1500ms)，reduced-motion 停用 |
| `shared/Avatar` | 头像（尺寸/形状/状态点/环）。R6：图片经 `lib/images.ts` avatar 档、渐显；**R8：占位层始终渲染**（有 name 首字母 / 无 name UserRound，`aria-hidden`）+ 图片 absolute 覆盖渐显、失败保留占位不闪烁；占位字用 text-primary（accent-bg 上 3 主题×亮暗 ≥4.5） |
| `.media-placeholder`（utilities） | **媒体占位统一底层**（R8）：主题 accent 渐变/图标占位，图片渐显覆盖、失败保留；用于群封面/通知·搜索·rail·主页瞬间缩略图/详情封面与九宫格/群 hero；有 Skeleton 的列表不叠加（design-system §20.10） |
| `lib/search-url.ts` | **搜索 URL 状态契约**（R8）：`?query=&tab=&tag=` 解析/序列化；`people/posts` 为内部 Tab（兼容 `users/moments` 别名）、非法 tab→all、空 query 清空全部参数；replaceState 同步（不增历史）、popstate 恢复。路由契约见 `hotspot-frontend-routing` skill |
| `lib/images.ts` | **图片管线唯一入口**（R6）：`optimizedImageUrl`（仅 unsplash 追加/覆盖 `auto=format&fit=crop&q&w`，其他域名原样）、6 档预设（avatar/thumb/card/cover/hero/full）、`presetSrcSet`、`avatarImageWidths`、`fadeImage` 渐显 action；规范见 design-system §20.8 |
| `lib/motion.ts` | **动效唯一入口**（R7）：`MOTION`（enter 180 / exit 150 / pop 220）、`prefersReducedMotion()`、`motionDuration()`、`pop` action、`playPageEnter()`、`toastExit`；CSS `cst-page-enter` / `cst-state-pop` / `cst-comment-enter`；规范见 design-system §20.9 |

> 四种列表状态各司其职，不得互相替代：空 `EmptyState` / 加载中 `InlineLoading` /
> 失败 `LoadError` / 已到底 `ListEnd`（文案「没有更多了」）。分页与无限滚动列表
> 在内容耗尽时必须渲染 `ListEnd`——「加载更多」按钮消失而无收尾提示，用户无法
> 区分「已到底」与「加载卡住」。

**骨架屏使用口径（2026-10-26 登记）**：
- **首屏加载**高频列表/信息流 → 骨架屏（`SkeletonList` / `SkeletonFeed`），与真实内容同栅格、同列数、同卡片外壳，做到「骨架→内容零位移」；
- **分页加载更多** → `LoadMoreButton` / spinner（不换骨架：契约语义是「更多」而非「首屏」）；
- **短促/异构内容**（表单、设置、详情区块、admin 表格、聊天首屏）→ 保留 spinner（骨架行高与气泡差异会造成滚动跳动，收益低于维护成本）；
- 语义：外层 `role=status` + `aria-busy=true` + `aria-label`（「加载动态中…」等）；占位块 `aria-hidden`，不进读屏/焦点；reduced-motion 下保留骨架形、停止脉冲。

### 按钮语言两派契约（2026-10-18 登记，审计 C1 收敛）

动作按钮只有两种被允许的形态，**按场景分派、同页同场景不得混用**；两种形态都必须
经由 `ui/button` 原语（含其派生 `ui/FollowButton`），禁止绕过原语手写描边/实底钮：

| 派别 | 形态 | 适用场景 | 例 |
|---|---|---|---|
| A 页面级动作（默认） | `shape="rect"`（`rounded-md` 16px）+ `text-sm`，高度档 sm 32 / md 40 / lg 44 | 表单提交、弹层确认、详情页动作区、移动端动作条、错误页动作 | 登录/保存/举报提交/屏蔽/加入群组/私信动作条 |
| B 行内关系 chip | `shape="pill"`（`rounded-full` + `text-xs`），`px-3.5 py-1.5`（行内）/ `px-4 py-2`（详情宽位） | 卡片/列表行内的关系与轻量动作（嵌在内容里的小动作） | 关注/加入/接受/拒绝/私聊入口 chip |
| C 安静动作（2026-10-27 Round 3 补充） | `shape="pill"` + `bg-surface-subtle/60` 软填底、**无描边**、`text-text-primary`；hover → `bg-surface-subtle` 全量 | 设置/关系/通知等次级页面的行内或分组动作（不抢内容） | 编辑资料/设置、发消息、拒绝、已读、会话操作 |

- **同级动作不得混用「描边胶囊」与「软填胶囊」**（2026-10-27 用户视觉一致性要求）；内容单元功能边界（输入框、主题 swatch）不属动作按钮、保留。
- **圆角只由 `shape` 表达**（2026-10-29）：A 用默认 `rect`，B/C 用 `shape="pill"`；**调用方禁止再用 class 传 `rounded-*`**（`button-radius-override` 检查器拦截）。动作按钮禁 `rounded-sm`(12)/`rounded-lg`(20)/`rounded-xl`(24)；
  圆形图标钮走 `ActionIconButton`/`size-* rounded-full`。导航项、菜单行、Toast、
  图片格等非动作控件按各自组件语言，不套本契约。
- 卡片/面板**内嵌**的次级动作钮用 `flat` 去投影（投影语义归容器，见 `ui/button` 注释）。
- 链接跳转用 `<a>` 时沿用派 A 的原语类语言（如错误页「返回首页」），不自造第三种形态。

### 页头动作图标钮口径（2026-10-26 登记，跨端对齐 Lead 决策）

页面头部右侧的纯图标动作（搜索/通知/地图等）按页头形态分派，**两端同一口径**
（前端 `ActionIconButton.variant` ↔ 鸿蒙 `SurfaceIconButton.kind`）：

| 页头形态 | 样式 | 前端 | 鸿蒙 |
|---|---|---|---|
| 品牌头（首页 logo + wordmark） | **ghost**（透明底、裸图标，36px） | `variant="ghost"` | `kind: 'neutral'` |
| 标准 PageHeader（内页） | **surface**（浅底圆钮 + hairline，36px） | 默认 `variant="surface"` | `kind: 'surface'`（`SystemPageHeader.IconAction` 已支持） |

- 同一页头内图标数量保持一致：web 移动端首页只有「搜索 + 通知」（地图入口在附近域页头）；
- 触控热区：web 36px 视觉 + `hit-slop` → 48px；鸿蒙 `SurfaceIconButton` 外扩 ≥44vp；
- 图片/封面上悬浮的返回/更多用 `overlay` 变体（媒体镀铬，平台适配，保留）。

### 图像内容卡形态（2026-10-27 Round 2 修订；原「封面比例三档」作废）

用户明确要求图像是卡片主体：**图片铺满整张卡 + 信息以遮罩形式覆盖在图上**
（完整 spec 见 `docs/design-system.md` §17）。同一视图只用一档：

| 形态 | 规格 | 使用场景 |
|---|---|---|
| 整卡图片 feed 卡 | `aspect-[4/5]` 图铺满 + 顶/底遮罩 + 覆盖信息（白字/白图标）；无 border，`rounded-lg`(20) | 首页信息流卡（`HomeFeedItem` card） |
| 整卡图片网格卡 | `aspect-[3/4]` 图铺满 + 覆盖信息 + 底部 footer 覆盖（白字） | 附近的人/群组/瞬间网格、我的瞬间/我的群组、群组精选卡 |
| 横向紧凑卡（例外） | 固定高 `h-32`，`horizontal` 图左 40% + 右侧信息托盘 | 我的群组手机列表等列表型场景（信息可读性优先，保留两段式） |

- 发现流 BSP 瓦片墙比例是布局算法产物（`minRatio 0.68 / maxRatio 1.48`，有意设计），
  不得据此另造卡片形态；BSP 图块同样走整卡图片 + 覆盖信息。
- 视觉/无障碍硬约束：遮罩 token（`--cst-surface-card-scrim-*`）、压图文字 ≥4.5:1 像素实测、
  徽章一律深色强遮罩 + 白字、无图 fallback 主题渐变 + 主题色文字、多图 1/N、
  操作热区 ≥44、点赞/收藏状态用形状+白描边不只色相（§17）。
- 首页双列人物卡壳由 `ListItem tone="card"` 唯一承担（如仍用列表形态），禁止外层再包手写卡；
  简单关系列表走 `ListContainer bordered` + 连接行，不逐行拆独立卡片。

### R13 视觉原语（2026-10-28，Round 13）

| 原语 | 职责与契约 |
|---|---|
| `data-display/MetricRow` `variant="strip"` | **数字横幅唯一形态**（资料页/附近枢纽/群组详情/后台 KPI）：`flex divide-x divide-border-subtle` + 24px（`size="sm"` 20px）semibold `tabular-nums` 数字 + 12px 次级标签；`align="center"\|"start"`；可点项渲染为 button（focus-ring）。**不再新增灰色药丸指标块**（旧 `capsule` 仅保留既有调用） |
| `home/SectionHead`（全站共享） | **分区标题唯一实现**：icon（16，text-secondary）+ 标题 14 semibold + `count`（12 次级 tabular）+ 副标题 + 右侧 `action` 文字链；`variant="section"\|"rail"\|"eyebrow"`。页面/rail 不再手写「标题 + 查看全部」行 |
| `data-display/EmptyState`（R13 升级） | 非 bare 图标砖 64px（`iconTone="neutral"\|"accent"`，accent = accent-soft 底 + accent 图标）；标题 14 semibold、描述 12 次级；新增 `extra` snippet（价值点/建议 chips，位于描述与动作之间）；spotlight 契约不变 |
| `auth/NeedsAuthNotice`（R13 升级） | **访客引导面板**：spotlight + accent 图标砖 + 标题/描述 + `highlights`（≤3 条 icon+文案）+ 主 CTA（登录/注册）+ 次 CTA（先随便逛逛，默认回首页）；`/me`、`/messages`、`/notify`、me 子页访客态统一使用 |
| `profile/IdentityHero`（R13 新增） | **资料页身份封面唯一实现**（/me、/users/[id]、ProfileMobile）：封面条（桌面 132 / 移动 92，主题渐变/封面图 + 遮罩）→ 头像（88/72，`-mt-11` 压封面 + `ring-4 ring-surface-page`）→ 昵称/handle/在线状态/简介/位置/标签 → `MetricRow strip` → `actions` snippet；桌面两栏（左身份 / 右指标+动作）、移动动作整行。**禁止页面再自写身份卡** |
| `profile/ProfileCompleteness` | 资料状态卡：<100% 分段点 + 可点补全入口；100% 隐藏进度条只留完成态 |
| `profile/ProfileMomentTile` / `profile/ProfileGroupTile` | 我的瞬间/我的群组卡片：整卡图片 + `…` 管理菜单（编辑/删除/退出群组，桌面 `desktop-reveal`、移动常显；删除/退出走 ConfirmDialog + 本地移除） |
| `settings/SettingsControlRow` | 设置单列分组行：icon + 标题/描述 + 整行控件（主题/明暗/地图源 segmented 给整行宽度，杜绝文字截断） |
| `messages/ChatEmptyPanel`（routes 内私有） | 聊天空会话面板：引导 + 最近 3 条会话建议 + 「发起新会话」；小面板不开 spotlight |

> 本轮新原语均由 Lead 或模块负责人在写作用域内实现；新增页面优先组合上述原语，
> 不一致的旧手写形态按模块逐步收敛。

### R14 卡片交互契约（2026-10-28，用户反馈「卡片交互要保持一致 + 移动端层叠错误」）

| 层 | 强制契约 |
|---|---|
| 卡片根 | `card-hover isolate group relative overflow-hidden` + `use:cardPress`；`isolate` **不可省**——否则内部 `z-20` 按钮会逃出卡片堆叠上下文，滚动到页内 sticky header 下方时压住标题栏（用户截图反馈的真实 bug） |
| 覆盖点击层 | 最后绘制 `absolute inset-0 z-10` 的真 `<button data-cover>`；行内控件经根上 `[&_button:not([data-cover])]:relative ...:z-20 [&_a]:relative [&_a]:z-20` 抬升；卡片根不持 button/aria-label |
| 操作行 | 一律 `ImageCardActions`：图标 size-4.5（sm 档 4）、间距 24（sm 12）、计数 text-sm tabular；点赞 = 危险红填充（压图白描边）、收藏 = accent 填充、评论/分享 = 中性；只在激活方向 pop；`aria-label/aria-pressed` 完备；点击 stopPropagation |
| 按压 | 只由 `use:cardPress` + `.card-hover[data-pressed="true"]` 驱动，缩放值走 `--cst-card-tap-scale`（vibrant 0.94 / forest·inkwash 0.97）；不允许再在组件内手写 transform/pointer 监听 |
| 压图徽章 | 一律 `.media-badge`（图标方角标加 `--icon`），禁止再手写 `bg-surface-overlay-strong`/`bg-surface-media-scrim-strong` 组合；非压图徽章统一 `data-display/Badge`（sm/md + tone），禁止手写 h-5/h-6 胶囊 |
| 作者区 | 有 `userId` 时头像/昵称均为可点按钮（hover：头像放大、昵称下划线；点击 stopPropagation → 主页）；无目标时保持纯文本 |
| **行内关注态（2026-10-29 修复）** | 卡片作者行的「已关注/关注」必须是真实按钮（`ui/FollowButton size="chip"`，`aria-pressed` + `aria-label`「取消/关注 + 名字」），**禁止用不可点徽章**：徽章在 `pointer-events-none` 作者行里会穿透到整卡覆盖层，真实点击 = 打开详情（用户反馈 bug）。chip 在作者行内必须显式 `pointer-events-auto` + `stopPropagation` + 乐观取关/关注 + toast；点击绝不触发卡片 onClick。默认只在已关注作者卡显示，取关后就地保留「关注」chip，未关注作者卡不新增常驻按钮（降噪） |
| 行内控件通用 | 所有行内可交互元素必须是 `button/a` 且 `pointer-events-auto`（作者行外层是 `pointer-events-none`）；非交互状态徽章保持纯展示并允许点击穿透（= 点卡片其他区域打开详情是预期） |
| 分享 | 统一 `lib/moment-actions.shareMoment`（系统分享 → 复制链接 + toast），信息流卡与网格卡同一实现 |

### 详情页（瞬间详情）契约（2026-10-29，用户反馈「顶部横条 / 评论输入格格不入 / 图片比例失真」）

| 项 | 强制契约 |
|---|---|
| 页头/工具行 | **不渲染 PageHeader、不出现横栏**：内容顶部是一条与画布同底、非吸顶的极简工具行（`ghost` 返回 + 可见居中标题 + `ghost` 更多，36px，随内容滚走），与正文同一视觉层；加载/错误/空态同样保留。禁止：①整条 sticky glass 通栏硬条；②玻璃悬浮工具条或孤立圆钮压内容。滚动后的返回由浏览器/系统返回与全局「回到顶部」承担。 |
| 媒体（单图） | **画廊级展陈**：图片 hero 位于内容卡首屏（作者/正文在图片之后），整幅出血到卡片左右边缘；主图 **`w-full` + `h-auto`（宽度撑满内容卡、高度按图片自身比例）**：任何分辨率下图片盒子比例恒等于图片宽高比、**左右不出现留白或模糊铺底**（2026-10-29 用户反馈「任何时候都不应该让图片出现两遍空白」）；不再用 `max-h` 压高度——高图只是更高、由页面滚动承载。ambient matte（同图 `scale-110 + blur`）仅保留为图片加载失败时的兜底底色，正常加载时不可见。**多图（2026-10-29 晚画廊化，2026-10-30 细节收口）**：主图 + 底部横向缩略图导航（56px 方图 `object-cover`、选中态 `ring-2 ring-accent-default` + 全不透明、未选中 `opacity-60`、窄屏横滚；点击缩略图**内联切换主图**、不打开全屏），主图右下角页码 `N / M`（非交互，替代旧「共 N 张」角标按钮）；点击主图进全屏查看器。**图片区只保留顶部圆角**（由卡片 `overflow-hidden` 裁剪、与卡片一致），底部直角贴边——禁止「悬浮的圆角矩形」式图片卡（2026-10-30 用户反馈「图片底部不在边缘、圆角矩形不贴合」）。缩略图条容器用 `-mx-1 px-1 py-1` 给选中态 ring（画在按钮外 2px）留绘制空间，禁止让 `overflow-x-auto` 把 ring 裁掉一部分。主图切换用 `{#key imageIndex}` 重挂载 + 渐显，避免残留旧图。依据 MDN object-fit（contain 保比例但可能留边）、web.dev aspect-ratio、NN/g 图像内容（大图且避免过多空白）。 |
| 动作条 | 所有端都内嵌在内容中（`MomentPostBody` 内），**移动端不再有第二条底部动作条**，避免与输入器叠成两条通栏横杠。 |
| 评论输入器 | 页面私有 `CommentInputBar`，两种登记形态：`inline`（**桌面 + 平板，即 ≥600 宽**：嵌评论区顶部，`rounded-md` + 软底浅槽，宽度随内容）；`floating`（**仅手机 <600**：悬浮 dock `mx-3` + `rounded-md` + 细边 + 玻璃底 + `shadow-raised-md` + 底部安全区）。2026-10-29 晚修正：此前平板（600–1279）误用 floating，宽屏下悬浮输入框压住内容；现仅手机悬浮。两者都只用 39px 输入壳 + **40px 发送钮**（与聊天发送钮同档；旧评论 36 vs 聊天 40 已统一）。**禁止整条白底 `border-t` 通栏输入条**。 |
| 与评论的关系 | 输入器是评论区的一部分：`inline` 时位于 `#moment-comments` 顶部（`commentComposer` snippet 注入）；移动悬浮 dock 仍归属评论动作；回复目标 chip 在输入器上方。 |

## 统一交互契约

1. **行高亮**：hover / 选中 / 键盘焦点同形状（12px 圆角 `--cst-shape-sm`），
   由 ListItem 内聚；容器 `inset` 时高亮天然离边 8px，禁止行高亮顶容器边。
2. **卡片 hover**：只走 `card-hover`；参数全部 token 化
   （`--cst-hover-lift` / `--cst-hover-image-zoom` / `--cst-card-tap-scale`），
   主题在 `themes/{vibrant,forest,inkwash}.css` 各自演绎。卡片根必须 `isolate`（R14，防压 sticky header）。
3. **按压**：行 = opacity-tap；卡片 = `use:cardPress` + `--cst-card-tap-scale`（R14 唯一入口）；按钮 = `hover-btn`。
4. **明暗**：一律 token，禁止组件写死色值/色 mix。
5. **焦点可见（Round 4 强制）**：所有 interactive（按钮/链接/行/输入/自定义点击区/表格行按钮）
   必须有可见焦点环：2px accent + 2px offset（`focus-ring`）；键盘 Tab 顺序可达、skip link 首 Tab 可见；
   **去描边不得连带丢焦点环**；亮/暗都测；父容器 overflow 不得裁切焦点环。
6. **减少动态（Round 4 强制）**：全局 `prefers-reduced-motion` 下禁用位移/缩放/脉冲/shimmer/悬停过渡
   （过渡降为 ≈0）；透明度淡入可留；新增动画不得裸写，走既有全局规则/`MotionUtil`。
7. **加载/空/错误（Round 4 收口）**：首屏列表/卡片优先 `Skeleton`；单实体/短列表可用 `InlineLoading`
   （需在页面注释给理由：结构轻/<1s/避免二次位移）；空 `EmptyState`、失败 `LoadError` + 重试、
   到底 `ListEnd` 四态不得互替。
8. **语义结构（Round 5，axe 基线）**：整卡可点必须用「覆盖按钮 + 行内动作 auto-lift」，卡片根不得持
   button/aria-label 语义；`role=list` 子节点必须 `role=listitem`，listitem 不挂 aria-label；`dl` 只包 `dt/dd`，
   指标组用 `div role=group`；无 role 的 div 不挂 aria-label；每页有 `main`；可滚动非键盘可达容器加
   `tabindex=0` + `role=region` + `aria-label`；装饰图 `alt=""` + `aria-hidden`。验收：axe-core 0 serious/critical。
9. **触控目标（Round 5）**：移动端可点元素有效热区 ≥44×44（`hit-slop` / 新增 `hit-slop-44` / ArkUI
   `responseRegion`），视觉尺寸可小；横向相邻元素不得互相覆盖；例外登记（表单控件/chip/内联链接/地图
   marker/工具条/skip link）。验收：390 视口扫描覆盖率 + 命中测试。
10. **主题验收（Round 5）**：三主题×明暗关键页必测；软岛/画布亮度比与文字对比记录进 `frontend-ui-audit.md`；
   主题特有 token 变更两端同值并重跑 parity（design-system §20.7）。
11. **图片加载（Round 6，design-system §20.8）**：所有 `<img>` 走 `lib/images.ts` 档位 + srcset/sizes；
   首屏仅可见图 eager+high、其余 lazy+async；占位 + 0.22s 渐显（reduced 直接显示）；失败走既有 fallback；
   非 unsplash/本地/blob 原样。`ImageViewer` 用 full 档（1600/q85）打开即加载。
12. **键盘导航（Round 6）**：搜索页 ↑↓ 在真实结果链接/最近/话题间移动焦点、Enter 打开、Esc 清空；
   不引入 combobox/aria-activedescendant；Tab 顺序与焦点环不变。
13. **动效（Round 7，design-system §20.9）**：enter 180/exit 150/pop 220，仅 transform/opacity；页面进场仅客户端导航
   （首屏不播护 LCP）；骨架→内容仅实际展示骨架时 fade；toast 退场 150ms；JS 动画必须显式 reduced 判断；禁止动画 LCP/引入 CLS/列表逐项 stagger。
14. **状态反馈（Round 7）**：点赞/收藏/关注/加入仅在用户**激活**操作时 pop（220ms，1→1.18→1），取消不 pop，
   初始数据同步/首帧已激活不播；评论新增仅新行 fade+4px（160ms）。
15. **聊天滚动（Round 10，design-system §20.12）**：近底部（≤120px）才自动滚；上滑阅读收到新消息**不跳转**，只累加计数；
   自己发送始终滚底；初始/切会话直接回底；分页 prepend 保持视口；markRead 仅初始/近底/回底补标（1s 节流），上滑/离开不补。
16. **回底按钮（Round 10）**：距底 >200px 显示（44px/鸿蒙 48vp），无消息只显箭头、有新消息「N 条新消息」（>99→99+），
   `aria-live=polite`、`aria-label` 与文本一致；150ms 显隐（reduced 直接）；点击滚底+清零+补 markRead；回底 200ms 内隐藏；
   不遮挡输入栏/浮层。鸿蒙 List 无距底像素 API，用 `onScrollIndex` 末尾可见近似（平台差异登记）。

### 输入族契约（2026-10-29 重构，强制）

| 语义 | 原语 | 几何 | 说明 |
|---|---|---|---|
| 搜索（发现入口） | `forms/SearchInput` | 胶囊 `rounded-full`，39px（`px-3.5 py-2` + `text-md`），图标 16，清空钮 32px 视觉 + `hit-slop`（44 命中） | 页面不得再传 padding/圆角/字号覆盖；`ariaLabel` 按页面语义（搜索会话/用户/标签…）；清空后回焦；Esc 有内容时先清空（`stopPropagation`，弹层内不会连关），空输入才交还弹层；Enter/Escape 判 IME `isComposing`；输入带 `inputmode=search`/`enterkeyhint`/`autocomplete=off` |
| 单行输入 | `forms/Input` | `rounded-md`(16)；`density="default"` 43px（`px-4 py-2.5` + `text-md`）/`density="compact"` 39px（`px-3.5 py-2`） | label `tracking-field`；错误态三件套；`prefix`/`suffix` snippet；`reserveErrorSpace` 预留错误行（登录/设置防跳动）；清空钮同搜索 |
| 多行文本 | `forms/Textarea` | 壳 `rounded-md`(16) + `.input-surface` + `px-3.5 py-3`；高度由 `rows`/`inputClass`(min-h) 决定 | label/错误态契约同 Input；`footer` snippet 放字符计数（同一壳内）；不套单行 39/43 档；全站多行输入统一用它 |
| 下拉选择 | `forms/Select` | `variant="field"`：`.input-surface` + `rounded-md` + `text-md`（default 43 / compact 39）+ 自绘 chevron；`variant="ghost"`：透明底 + accent 文字（行内排序/轻量切换） | 保留原生 `<select>` 可访问性；后台筛选/表单统一 `field`，行内排序统一 `ghost`；页面不得再写裸 select |
| 复选框 | `forms/Checkbox` | 原生 checkbox 16px + `accent-accent-default` 语义色 + `focus-ring` | 权限勾选/多选/危险确认统一用它；可见文案走 children，无文案传 `ariaLabel` |
| 表面状态 | 单一来源 `.input-surface`（utilities.css） | 静态 = `--cst-field-bg` + 1px `--cst-field-border`；hover = border-strong；focus = accent 1px + 内缘 1px（视觉 2px）+ 3px `--cst-field-ring`；error = danger + `--cst-field-ring-danger`；disabled = `--cst-field-bg-disabled` | 主题文件禁止覆写 `.input-surface`；输入表面禁止 `backdrop-filter`；placeholder 锁 `text-secondary` + `opacity:1`（改前叠 0.6 → 2.43–4.0:1） |
| 快捷键 | `lib/search-shortcut.ts`（纯决策 + 9 条单测）+ `(app)/+layout.svelte` + 搜索页 trailing kbd 胶囊 | ⌘K（Mac）/ Ctrl+K | 任意页唤出搜索；已在搜索页则聚焦全选（导航后 rAF 显式补聚焦，避免被路由焦点管理抢到 main）；弹层打开 / 非搜索页表单编辑中让位；提示与监听必须同时存在 |
| 页面宽度 | `/search` hero | 桌面 `max-w-3xl`(768)，移动满宽 | 布局例外登记见 skill `hotspot-frontend-layout` |

源码契约测试：`src/styles/__tests__/input-surface-contract.test.ts`（锁表面状态/单一来源/placeholder 对比度修复，禁止回退到「双层描边 + 双向内凹阴影 + vibrant 毛玻璃」）。

### 同类型一致性契约（2026-10-29 第二轮收口，强制）

**原则：一种语义（搜索/单行/多行/下拉/复选/文件/Tab/按钮/卡片/列表行/空态…）在全部页面只允许一种实现路径**。同类型组件在所有地方的视觉（几何/字号/圆角/状态）与行为（交互/可访问性）必须一致；上下文差异只能通过原语的**登记变体**（如 Input `density`、Select `field|ghost`）表达，**禁止页内近似实现**（手写 input/select/textarea/checkbox、复制粘贴的卡片样式、自绘 Tab、给原语传几何覆盖类）。

已强制覆盖的原语族（源码级检查器 `scripts/check-component-consistency.mjs` + 16 条规则，2026-10-29 第二轮从 10 条扩到 16 条）：

| 规则 | 强制内容 |
|---|---|
| `raw-search` | 任何搜索语义控件（placeholder/aria-label 含「搜索」）必须用 `forms/SearchInput` |
| `search-override` | SearchInput 调用方不得传 padding/圆角/字号/投影覆盖 |
| `search-aria` | 每个 SearchInput 必须传 `ariaLabel`（按页面语义命名） |
| `raw-input` | 单行输入必须用 `forms/Input`（聊天/评论输入条为登记例外） |
| `raw-textarea` | 多行文本必须用 `forms/Textarea`（聊天气泡内联编辑为登记例外） |
| `raw-select` | 下拉必须用 `forms/Select`（field/ghost 两种变体） |
| `raw-checkbox` | 复选框必须用 `forms/Checkbox` |
| `raw-file-input` | 文件上传必须走 `forms/ImagePicker`（头像/封面徽标为登记例外） |
| `input-surface-outside` | `.input-surface` 只能由输入族原语消费，页面不得手写输入表面 |
| `tab-role-outside` | Tab 必须用 `navigation/TabBar`，页面不得自绘 `role=tab/tablist` |
| `raw-badge` | 非压图徽章/标签必须用 `data-display/Badge`（压图用 `.media-badge`，未读用 `UnreadBadge`），禁止旧 `.badge` 或手写 h-5/h-6 胶囊 |
| `button-radius-override` | Button 调用方不得覆盖 `rounded-*`；派 B/C 用 `shape="pill"`，派 A 用默认 `rect` |
| `untiered-radius` | 禁止 `rounded-2xl/3xl` 等未注册圆角档，必须落到 design-system 登记的 4 档 + full/sheet |
| `handwritten-cta` | accent 实底胶囊 CTA 必须用 `Button`（主操作身份），禁止手写拼语言 |
| `raw-avatar-img` | 头像必须用 `shared/Avatar`（裸 `<img rounded-full>` 会丢占位/兜底/渐显；品牌 logo 除外） |
| `sheet-radius-override` | Dialog 圆角由组件档位决定：浮层卡片用 `floating`，禁止调用方 class 覆盖 `rounded-*` |

- **执行**：`bun run check:ui-consistency`（已挂入 `bun run check`，CI frontend job 漂移即拦截）。
- **例外**：只能登记 `frontend/scripts/ui-consistency-allowlist.json`（`rule + file + reason`），并在 `docs/frontend-ui-audit.md` 记录理由；禁止用 ignore 注释绕过。
- **新增同类型 UI 的流程**：先查本表 → 有原语就复用/扩展 prop → 没有则**先建原语 + 契约测试 + 检查器规则**再使用，禁止直接在各页写近似实现。
- 非输入族的同类型偏差由 `docs/frontend-ui-audit.md` 的「UI 一致性」节登记与收敛（按钮/卡片/列表行等历史 backlog 见下节），新代码不得新增近似实现。

> 多行文本域统一 `forms/Textarea` 的 `px-3.5 py-3` 壳（不再各页传 `p-4`/`px-4 py-3`）；高度由 rows/min-h 决定，不套单行 39/43 档；不要为了「完全统一」把 textarea 压成单行高度。

## 近重复收敛 backlog（按危害排序）

1. **手写按钮**：58 个文件用 `hover-btn` 手写 `<button>`，`ui/button` 引用 23 个文件
   （2026-10-08 刷新：设置页/LoadMoreButton 已迁；**2026-10-12 复核：60 / 24**，
   统计口径 `grep -rl` 全 src *.svelte）。
   规则：改到某文件时把该文件的简单动作按钮顺手迁 `ui/button`/`ActionIconButton`。
   约束（2026-10-07 LoadMoreButton 迁移实测）：`ui/button` secondary 自带的
   `shadow-raised-sm` 是自定义主题 token，**不参与 tailwind-merge 冲突消解**——
   className 里写 `shadow-none` 也消不掉它，且生成 stylesheet 里它排在后面
   （浏览器实测 computed box-shadow 落到 raised-sm）。无投影的胶囊按钮
   （如 LoadMoreButton）在原语支持「去投影」开关之前**不得迁移**，否则静默
   加一层投影。`LoadMoreButton.test.ts` 的「视觉形态守恒」用例已锁定当前形态
   （token 边界匹配），任何加投影/改圆角/改字色的迁移会在此失败。
 2. **表单 Input 原语（2026-10-29 第二轮已全量收口）**：`forms/Input`（含 `density` 两档、
    `reserveErrorSpace`、`suffix`）+`forms/Textarea`+`forms/Select`+`forms/Checkbox` 建成，
    auth/settings/me-edit/users/admin 六页/群组弹层/overlay/nearby 排序全部迁移，
    裸 `input`/`textarea`/`select`/`checkbox`/`.input-surface` 由
    `bun run check:ui-consistency` 拦截（`scripts/check-component-consistency.mjs`）。
    保留的登记例外：聊天输入条（`ChatInputBar`）、评论输入条（`CommentInputBar`）、
    气泡内联编辑（`MessageBubble`）、头像/封面上传徽标（隐藏 file input）、
    `ImagePicker` 内部 file input —— 均为「输入条/媒体上传」等不同语义，
    见 `scripts/ui-consistency-allowlist.json` 理由。
3. **分组卡壳已收敛**（2026-09-10 / 2026-10-18 / 2026-10-27 Round 3 收口）：移动端 `ProfileMobile`
   与桌面 `ProfilePage` 统一为「内容软岛」（无描边 + `surfaceIsland` + `shadow-xs` + 圆角 20）；
   设置页「账号信息/外观与显示/地图与位置/系统与关于」等分组统一为 `ListContainer bordered inset`
   + `ListItem`（软岛语言，与通知/隐私/账号安全分组一致）。新增卡片默认先问：这是「列表集合」
   还是「独立内容卡」——列表集合必须走 ListContainer（软岛）；独立内容卡可保留 hero/封面例外。
4. ~~**触屏列表未启用 inset**~~（2026-10-27 Round 3 已统一）：notify / me 个人数据 / 关系 / 搜索等
   开放页列表统一为软岛 `bordered` 形态（`inset` 行、行透明、gap 无发丝线），不再保留分隔线形态。

## 已完成的收敛（2026-09-10 全量审查批次）

- **关系分区路由归一**：`/me/friends`、`/me/following`、`/me/fans` 三条与
  `/me/relationships?tab=` 完全重复且无入站入口的孤儿路由，改为 308 重定向
  （仅保留 `+page.ts`，UI 全部由 `me/relationships/RelationshipsPage.svelte` 承载）；
  tab id 与深链构造的单一事实源见 `lib/relationships.ts`。
- **个人主页入口去重**：`ProfilePage` / `ProfileMobile` 同一页曾各有 2 个设置入口
  （页眉图标 + 身份卡按钮），现统一只保留身份卡的拇指操作区那一个。
- **列表语言收口（2026-10-18）**：关系页三 tab 的独立卡网格改为连接列表行；
  管理后台概览手机端模块入口改为行式列表；我的举报行补内边距/行分隔并把 `ListEnd`
  移出列表卡；通知入群邀请行动作组窄屏换行；首页双列人物卡去掉外层重复卡壳。
- **标题层级统一**：全站 `PageHeader` 的 `titleSize` 统一为
  `device === 'desktop' ? 'xl' : '2xl'`（原首页/附近/消息页用 `3xl`，动态详情页
  固定 `xl`，四套并存）。
- **无 hover 反馈的按钮清零**：21 个自带底色/描边却因命中全局 hover 排除集而
  完全没有 hover 的按钮，按「实底升一档投影 / 描边白底走 `hover:bg-accent-soft` /
  遮罩按钮走 `hover:opacity-hover`」三档补齐。
- **死类名清零 + 护栏**：11 个从未注册的类名（`shadow-raised-xs` 18 处、
  `bg-surface-subtle`、`bg-surface-secondary`、`bg-accent-hover`、`text-text-tertiary`、
  `duration-fast`、`ease-bounce`、`scrollbar-none`、`rounded-2xl`、`desktop-reveal` 等）
  已全部修好，并新增 `scripts/check-utility-classes.mjs` 挂入 `bun run check:theme`
  防复发。

## 流程要求

任何 UI 改动必须**截图对比审查**（改前/改后、双主题、受影响断面），
不允许只靠代码 diff 判断视觉结果；与市面成熟设计（Telegram/Slack/macOS
System Settings 等）的模式对表后再动手。

## 同类元素基准与密度原则（2026-10-20 实测，收敛中）

用户审查目标：**同类元素必须同风格**，同时"在提高信息密度的前提下保持界面简洁，让用户自然找到功能与内容"。

### 1. 密度原则（定档依据）

- **紧凑优先，但不断行**：能一行说完的信息不用两行；行高由内容决定，档位只有 3 档（见下），禁止近邻并存（如 62/64/68 三档并存 → 合并）。
- **留白只为分组，不为装饰**：区块内用 8/12 节奏，区块间 24；"看起来空"应通过增加可见信息量解决，不要靠加 padding。
- **一眼可辨的三层信息**：标题（15px semibold）→ 正文（13~14）→ 元信息（11~12 次级色）。超过三层即视为信息过载，降级而非缩小。
- **交互元素只在不打断扫读的前提下出现**：行尾 `⋯` / 图标按钮常显时必须是"操作"；辅助操作走 hover/focus 显现（`desktop-reveal`）。

### 2. 实测基准（390×844 / dark / vibrant，CDP 实测）

| 家族 | 基准 | 已登记例外 |
|---|---|---|
| 卡片容器 | `radius 16`（`rounded-md`）+ `border 1px` + `bg surface-primary` | 资料/英雄卡 `radius 20`（`rounded-lg`，`/me` 顶部）|
| 胶囊 chip | 紧凑 `20`（badge）/ 标准 `24`（标签）/ 筛选 `24~28`（TagPill sm/md）/ **行内动作 `28`（`h-7`）** | — |
| 列表行 | **`58`（单行无头像）/ `62`（单行带头像）/ `68`（双行）/ `74`（带控件设置行）** | — |
| 图标按钮 | `36`（header 动作）/ `44`（触控下限） | 卡片内操作 `28~32` |
| 按钮（Button 原语）| `sm h-8` / `md h-10` | — |
| 区块标题 | 15px / semibold（`SectionHead`） | — |

**2026-10-20 首轮收敛结果**（逐项核实：CDP 取 `getBoundingClientRect` + `getComputedStyle` 拿到类名再改，不按估计改元素）：
- ✅ `/notify` 行内「接受/拒绝」原为内容撑高（29 vs 31，兄弟按钮差 2px）→ 固定 `h-7` → 实测两者同为 `28`；
- ✅ `/me` 统计网格 `MetricRow` 由 `rounded-sm`(12) 改为 `rounded-md`(16) → 实测 `rad=16`；
- ✅ 销案两项：`/nearby/groups` 的 32px 胶囊是 `Button` `sm` 档（允许）；`/search` 的 58/68 是单行/双行两档（已补进基准表）。

### 3. 收敛顺序（按收益排序）

1. ✅ **胶囊高度**：内容撑高的胶囊改为固定档（已修 `/notify` 接受/拒绝，实测 29/31 → 同 28）；
2. ✅ **卡片底色**：全站扫描（32 路由 × 移动端，判别条件「radius ≥ 12 + 有边框 + 透明底 + 无图片后代」）
   命中 **0** —— 卡片底色本已统一；透明底只在"叠在图片上"的封面区出现（允许）；
3. ✅ **列表行**：实测值全部落在 58/62/68/74 四档内（58 单行无头像、62 单行带头像、68 双行、74 带控件设置行）；
4. ✅ **区块标题**：管理后台 6 处 `text-sm`(13px) 对齐到 `text-md`(15px)；管理后台之外无 `text-sm` 的 h2。

**结论（2026-10-20）**：本轮清单四项全部处理完毕，基准表转为**硬约束**——新增 UI 必须先选档位，
不新建中间值；如需例外，在本文件登记并写明理由。

> 收敛完成后，本清单转为**硬约束**：新 UI 必须先选档位，不新建中间值。