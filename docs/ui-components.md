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
| `layout/page/PageHeader` | 唯一页面头部（sticky/搜索/ below 插槽） |

### 列表（行交互的唯一体系）
| 原语 | 职责 |
|---|---|
| `list/ListContainer` | 列表容器。**2026-10-27 Round 3**：`bordered` 语义更新为「开放页面内容软岛」——`bg-surface-island + shadow-xs + rounded-lg(20)`、无 1px 描边、`p-2`(8)、行背景透明、行间 `gap-y-0.5`(2)、hover 填充离岛边 8（不再画行横线）；embedded（面板内集合）× `inset`（内缩圆角行）/ 默认（分隔线）不变。调用方不得再给 `bordered` 覆写 border/bg/radius |
| `list/ListItem` | **唯一行交互原语**。hover/选中填充、12px 圆角、键盘、aria、`tone: joined/card`。行级卡片类禁止（eslint 强制）。列表容器内自动包 `role=listitem`（ListContainer 经 context 声明；ListEnd/InlineLoading 同机制），裸用不变。行点击由内部透明覆盖按钮（`[data-cover]`）承载，行内操作按钮自动抬升（z 序），根不持 button 语义（axe nested-interactive 清零） |
| 行包装组件 | `ChatListItem` / `home/ActivityRow` / `profile/UserCardRow` / `forms/SettingRow`——只做内容编排，交互全部委托 ListItem |

### 卡片（CSS 卡语言 + 一个内容卡原语）
| 原语 | 职责 |
|---|---|
| `data-display/NearbyImageCard` | 图片优先内容卡**唯一实现**（附近瞬间/附近的人/附近群组/首页轮播/群组详情） |
| `card-hover`（utilities） | 可点击卡片唯一 hover 语言：投影抬升 + `--cst-hover-lift`（vibrant）/ 纸面微染+边框（forest） |
| `card-raised` / `surface-hero` / `ListItem tone=card` | 静态卡片语言（白底 + 投影），禁止再发明第四种 |
| 单帖卡（瞬间详情） | 内容软岛（无描边 + `surface-island` + shadow-xs + 圆角 20）；媒体容器去 1px 描边、保留圆角/阴影/加载占位/focus-ring；统计与评论分隔发丝线保留（Round 4） |
| Hero 封面（群详情/主页） | 去 1px 描边，保留圆角/阴影/加载占位（媒体 hero 例外，见 design-system §19.5） |
| 表格（admin） | 外壳去描边改软岛；**表头 bg-subtle + 行线 + 行 hover 保留**；行内按钮次级软填 / soft danger；390 横向滚动可及全部列；分页 ≤20 行不强制 sticky；空态/分页/筛选统一（Round 4） |
| `image-hover-frame` | 相册小方格专用（照片墙缩略图），内容卡不得使用 |

### 按钮 / 输入 / 导航 / 弹层 / 反馈
| 原语 | 职责 |
|---|---|
| `ui/button` | 全部动作按钮的唯一壳（表单/CTA/弹层确认/详情动作区；`ui/FollowButton` 等复合钮经它派生） |
| `navigation/ActionIconButton` | 图标按钮（ghost/outline） |
| `hover-btn` / `scale-btn`（utilities） | 行内文字/图标动作的 hover/tap CSS 语言 |
| `hit-slop` / `hit-slop-y`（utilities） | 紧凑控件的触控热区扩展（视觉尺寸不变）。`hit-slop` 四周各 6px，用于图标按钮等**孤立**控件；`hit-slop-y`（2026-10-19 新增）只扩纵向，用于**横向相邻**的紧凑控件（筛选胶囊、分段控制器）——前者的水平 6px 会与相邻项重叠，命中落到 DOM 靠后那颗。注意：两者都靠 `::before` 绝对定位，**会被祖先的 `overflow` 裁剪**，横滚容器需留出纵向 padding（TabBar 的 `py-1` 即为此） |
| `forms/SearchInput` / `forms/Input` / `forms/Toggle` / `forms/ImagePicker` / `forms/TagPill` | 输入类原语。`forms/Input`（2026-10-12 新增）：单行输入，受控 value+onChange；`label`/`labelSnippet`（tracking-field 档位或自定义行）与 input 以 id 关联；`prefix` snippet 前缀图标；`clearable` 清除钮；**错误态三件套** `hasError`+`errorText`→ aria-invalid + aria-describedby + role="alert" 就近错误行（样式由 input-surface 的 :has(aria-invalid) 承担）。padding 由调用方 class 提供（同 SearchInput 约定，如 `rounded-md px-4 py-3`）；另支持 `compact` 形态（input-surface 直贴 input，无包裹/label/prefix/清除，紧凑位专用）。契约对齐鸿蒙 `SystemInput` |
| `layout/page/PageHeader` | 页面页头（sticky）。**R9：顶部透明（与画布融合、无白带）→ 滚动 >4px 加 glass**（`surface-page 78% + blur(16px) + 细底边 + elevation`，150ms，reduced 立即）；仅背景/边框/阴影变化，不改定位；原 `glass` prop 统一为该行为 |
| `navigation/BackToTop` | 回顶浮动圆钮（R9）：滚动 > 1.5×视口高出现（44px，移动 bottom-24 / 桌面 bottom-6，避开底栏与浮层），点击平滑回顶（reduced 直接跳）、回顶后 200ms 内隐藏；`aria-label="回到顶部"`；fixed 不占流 |
| `lib/route-announcer` | SPA 导航播报（R9）：`#route-announcer`（sr-only `aria-live=polite`/atomic）播报新页 h1 标题；首屏不播、1.5s 去重；焦点落 `main#main-content`（tabindex=-1、preventScroll），不叠加 h1 聚焦 |
| `navigation/TabBar` | 分类/过滤胶囊（唯一 Tab 原语；底栏走 `layout/mobile/NavTab`）。三个变体：`pills`（筛选/分类，可横滚）、`segmented`（二选一/系统级分段，等宽轨道）、`underline`（页级两态切换：全宽等分、无轨道、激活项 + 2px accent 下划线）。**选中态不得只靠颜色**（2026-10-27 Round 2，用户对色弱/色盲的明确要求）：`segmented` 选中 = accent 实底 + ✓ 图标 + `font-semibold`；`pills`/`underline` 选中 = `font-semibold`（underline 另有下划线形状线索）；验收须过灰度截图（`filter:grayscale(1)`）。`disabled`（2026-10-19）整体禁用并置灰；**移动端触控高度**：按钮 `py-3`（视觉 41px）+ 滚动根 `py-1` + `.hit-slop-y` → 有效 ≈49px，桌面 `lg:py-1.5` 保持紧凑 |
| `overlays/SheetShell`、`ui/dialog`、`ui/popover`、`ui/dropdown-menu` | 弹层体系 |
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
| A 页面级动作（默认） | `rounded-md`(16px) + `text-sm`，高度档 sm 32 / md 40 / lg 44 | 表单提交、弹层确认、详情页动作区、移动端动作条、错误页动作 | 登录/保存/举报提交/屏蔽/加入群组/私信动作条 |
| B 行内关系 chip | `rounded-full` + `text-xs`，`px-3.5 py-1.5`（行内）/ `px-4 py-2`（详情宽位） | 卡片/列表行内的关系与轻量动作（嵌在内容里的小动作） | 关注/加入/接受/拒绝/私聊入口 chip |
| C 安静动作（2026-10-27 Round 3 补充） | `rounded-full` + `bg-surface-subtle/60` 软填底、**无描边**、`text-text-primary`；hover → `bg-surface-subtle` 全量 | 设置/关系/通知等次级页面的行内或分组动作（不抢内容） | 编辑资料/设置、发消息、拒绝、已读、会话操作 |

- **同级动作不得混用「描边胶囊」与「软填胶囊」**（2026-10-27 用户视觉一致性要求）；内容单元功能边界（输入框、主题 swatch）不属动作按钮、保留。
- 圆角禁档：动作按钮不得使用 `rounded-sm`(12)/`rounded-lg`(20)/`rounded-xl`(24)；
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

## 统一交互契约

1. **行高亮**：hover / 选中 / 键盘焦点同形状（12px 圆角 `--cst-shape-sm`），
   由 ListItem 内聚；容器 `inset` 时高亮天然离边 8px，禁止行高亮顶容器边。
2. **卡片 hover**：只走 `card-hover`；参数全部 token 化
   （`--cst-hover-lift` / `--cst-hover-image-zoom` / `--cst-card-tap-scale`），
   主题在 `themes/{vibrant,forest,inkwash}.css` 各自演绎。
3. **按压**：行 = opacity-tap；卡片 = `--cst-card-tap-scale`；按钮 = `hover-btn`。
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
 2. **表单 Input 原语**：**已落地（2026-10-12）**——`forms/Input` 建成（10 个契约测试），
    首个消费方 `me/edit` 昵称输入已迁移（登录态截图对比审查通过）。
    确认码输入亦已迁（compact 形态，2026-10-12，运行时截图验证通过）——me/edit 全页输入原语化。
    **其余消费方评估后保留手写（2026-10-12 全量盘点）**：剩余 20 处 input 全部是
    **compact 变体**（`input-surface` 直贴 input，admin 六页/群组弹层 ×3/安全设置 ×6/
    map 面板 ×2）+ 应保持原生的 file/checkbox。compact 迁移收益≈0：样式半边已合规、
    表单无错误态需求，迁移只改写 markup 不增行为，反有回归风险。策略改为：
    ① wrapper 型新表单一律用 `forms/Input`；② compact 消费方改到该文件且有新增行为需求
    （如清除钮/错误播报）时，先给原语加 compact/size 变体再迁；
    ③ AccountSecuritySection 密码簇若要补 AuthForm 式显隐切换，属 UX 增强需设计确认。
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
