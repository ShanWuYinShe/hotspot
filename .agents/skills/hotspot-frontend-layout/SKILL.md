---
name: hotspot-frontend-layout
description: hotspot 前端页面布局契约：StandardPageLayout / SplitPageLayout 的选择与结构、PageHeader/TabBar/正文滚动、三端分派规则、断点与内容宽度、侧栏导航层级与跨端信息架构、纵向节奏与 contentClassName 白名单、页面级去边框与内容软岛、内容轴（含聊天页）、页头滚动/回顶与聊天滚动。触发：新建或改造前端页面；决定用哪种布局组件；布局分支与设备差异；页面间距；页面级边框/软岛/内容轴；页头滚动与回顶；聊天页布局与滚动。
---

# hotspot-frontend-layout

> 页面视觉细节（圆角/字号/阴影/动效/交互状态/图片管线）见 skill `hotspot-frontend-style`；token 定义见 `hotspot-frontend-theme`。

## 页面布局（普通页 StandardPageLayout / 分栏页 SplitPageLayout）

所有路由页面必须使用统一布局组件，禁止自行拼装 PageHeader/ContentColumn/页面级滚动/页面级横向 padding：

- **普通页面**：`StandardPageLayout`（`components/layout/page/StandardPageLayout.svelte`），正文单列，滚动由 `RoutedContentViewport` 承担；正文内可含多列 grid（如群组详情 main + aside、个人主页双栏），不属于分栏需求。
- **分栏页面**（桌面主区 + 右 rail）：`SplitPageLayout`（`components/layout/page/SplitPageLayout.svelte`），主区与 rail 滚动由组件统一管理，footer 固定于主区底部。当前分栏页：通知、动态详情、用户详情。
- **临时豁免/已登记例外**：
  - 聊天页面（桌面端与移动端）因分栏/全屏会话布局特殊，可暂不套用 `StandardPageLayout`/`SplitPageLayout`；但页头、列表与正文的 gutter/padding 仍必须与样板一致（沿用 `PageHeader` + `ContentColumn`）。桌面消息分栏左栏窄列（`w-80`=320px）用 `px-4` 压缩 gutter、不进 `--cst-page-gutter`（见 skill `hotspot-frontend-style`「窄列特例」，已登记例外）；聊天三块同轴见本文「内容轴」节。
  - **封面/紧凑型资料头**（`groups/[groupId]`、`users/[userId]`）：标题与封面/头像绑定，不在 header 操作行内，故不套 `PageHeader`，自建返回/分享/更多操作行 + 封面下方 `<h1>`。与 `moments/[momentId]`（有独立标题「动态详情」，套 PageHeader）区分。
  - 移动个人主页（`ProfileMobile`）：自建 eyebrow 页眉，非标准列表页 header；设置/外观/编辑入口统一收在身份卡的拇指操作区，页眉不放重复入口。
  - 地图（全画布）与登录页（无 MainLayout 外壳）不适用。
- 强制统一布局是为了保证切换页面时右侧内容区不发生水平偏移；存量页面需逐步对齐，不得新增其他例外（新增例外必须在上面登记）。

**权威样板页：`src/routes/(app)/nearby/moments/+page.svelte`。** 普通页面以它为基准，结构固定为：

```
StandardPageLayout                     # 全站唯一外层布局（components/layout/page/）
├── header snippet
│   └── PageHeader                     # 三端统一页头
│       ├── title / titleSize          # 统一档：桌面 xl(20) / 平板·移动 2xl(24)
│       ├── trailing snippet           # 计数、精选等右上信息
│       └── below snippet              # 搜索框 + 分类 Tab 栏（随 header 吸顶）
└── children                           # 正文，直接放 ContentColumn 内
```

约束：

1. **布局**：普通页面必须用 `StandardPageLayout`，分栏页面必须用 `SplitPageLayout`；header 以 snippet 传入、正文放 children；不得自行组合 PageHeader/ContentColumn/页面级滚动/页面级横向 padding（聊天页豁免见上）。gutter 一律由 `ContentColumn` 的 `px-[var(--cst-page-gutter)]` 管理。
2. **页头**：一律用 `PageHeader`（`sticky` 默认开），`onBack` 在非桌面自动显示返回按钮；`below` 只放搜索框与 Tab 栏等 header 附属区。**below 首元素必须带上间距**（`mt-3`，与样板 nearby/moments 一致）——TabBar/搜索框直接贴 title 属于间距错误；below 内多行元素之间用 `mt-3` 分隔。`PageHeader` 内部 ContentColumn 的上下内距由组件派生（有 below：`pt-4 pb-3.5`；无 below：`pt-4.5 pb-4.5`），页面不要再传 `contentClassName` 覆盖（唯一例外是已登记的窄列面板）。
3. **分类/筛选/排序 Tab（强制统一）**：一律使用 `TabBar`（`components/navigation/TabBar.svelte`，胶囊样式 + 激活项 accent 实底胶囊 + `aria-pressed`，对齐鸿蒙 TabsPillGroup 分段式视觉），放 `below` snippet 内随 header 吸顶；tab 状态可存 URL query（如 `?tab=`）以支持深链，切换用 `goto` 更新 URL。**禁止再写手写 tab/chip 变体**（通知/搜索/个人主页/地图 sheet 的手写胶囊已统一为 TabBar）。表单内的标签/多选 chips 一律用 `TagPill`；网格选择器（主题 swatch、发布可见范围）属例外，不套 TabBar。
4. **正文与滚动**：正文直接放 children；`StandardPageLayout` 页面滚动由 `RoutedContentViewport`（main）唯一承担，`SplitPageLayout` 分栏页的主区/rail 滚动由组件统一管理；页面不得自建滚动容器（消息分栏、地图等全画布页面除外）。内嵌自带 gutter 的组件时用负 margin（`-mx-[var(--cst-page-gutter)]`）抵消 ContentColumn gutter，避免双重内边距（参照 `me/relationships/RelationshipsPage.svelte`）。
5. **三端差异**：共用组件只写一次，设备差异用 `platform.platform` 派生（列数、字号、布局分支），不得复制三份页面。
6. **数据获取与状态**：`$effect` + `cancelled` 竞态清理模式（参照样板页）；加载中用 spinner、空列表用 `EmptyState`（参照样板页 feedState snippet）；数据加载失败必须展示 `LoadError`（`components/data-display/LoadError.svelte`）并提供重试（`reloadKey` 递增触发 effect 重跑），禁止静默转空态。分页/无限滚动列表**内容耗尽时必须渲染 `ListEnd`**（`components/data-display/ListEnd.svelte`，文案「没有更多了」）——「加载更多」按钮消失而无收尾提示会让用户无法区分「已到底」与「加载卡住」。四种状态各司其职：空列表 `EmptyState`、加载中 `InlineLoading`、失败 `LoadError`、到底 `ListEnd`，不得互相替代。**首屏列表/卡优先 Skeleton**；单实体/短列表可 `InlineLoading`（页面注释给理由）。
7. **列表骨架契约**：列表一律用 `ListContainer` + `ListItem`（`components/list/`）。joined 行（默认 tone）**禁止叠加 rounded/border/shadow/ring 卡片类**——行间分隔由容器 divide 线承担，圆角外框由容器级（`ListContainer bordered` 唯一外框）或 `tone="card"`（独立卡片容器 + gap 分隔）承担；调用方 className/style 只允许布局类。**高亮形状由组件统一应用**：joined 行 hover/focus/pressed 时背景自带 `rounded-small`（12px，`--cst-shape-sm`）圆角，非高亮保持直角连接行视觉；圆角随背景填充出现并纳入 transition——调用方不得重复设置。双重防线：ESLint 自定义规则 `hotspot/no-list-item-card-classes`（`eslint-rules/`，Svelte AST 级，IDE 即时报错）挂入 `bun run lint`/`bun run check`；ListItem dev 模式运行时警告。后续同类组件级契约在 `eslint-rules/` 新增规则并在 `eslint.config.mjs` 注册（最小化接入：不启用默认规则集）。

## 断点与内容宽度

**项目断点是自定义的，不要按 Tailwind 默认值判断**（`src/styles/tailwind.css`）：

| 档 | 值 |
|---|---|
| `sm` | 600px |
| `md` | 960px |
| `lg` | **1280px** |
| `xl` | **1536px** |
| `2xl` | 1792px |

常见笔记本宽度 1397px 只到 **`lg`** —— 用 `xl:` 写的布局在那里**不会生效**（实测踩过一次）。

内容宽度按**三类**选择（`ContentColumn` 默认 `--cst-page-content-max: 100%` 流式；其注释口径：「仅真正的阅读/详情列才传 maxWidth，不得用窄宽度来摆放标准页面」）：

- **默认流式**：标准信息流/列表/枢纽页**不传** `contentMaxWidth`/`maxWidth`——传窄值（如 768）会让宽屏内容只占中间一条、右侧大片空白。
- **阅读/详情/表单列（合法例外）**：仅真正的阅读或表单页传列宽（现状：settings `56rem`、me/edit `48rem`、notify `64rem`、瞬间详情/用户详情 `58rem`），由 `ContentColumn`/`SplitPageLayout` 承载。
- **宽屏呼吸档 = 1280（数字字面量）**：长内容呼吸型页面（/me 总览、关系、举报）统一 `contentMaxWidth={1280}`；`PageHeader` 的 `maxWidth` 必须与页面 `contentMaxWidth` **同值成对**（只改其一即轴漂移 46–158px）。细则见「内容轴 → 宽屏呼吸档与列表轴」。
- **页内可读行宽**：需要限制正文行宽时用页内 `max-w-4xl mx-auto`，宽屏随页宽放宽（`lg:max-w-5xl`）——这是**页内**约束，不能替代页面级 `contentMaxWidth`。
- 宽屏提升密度靠**页内响应式网格**，不是把内容拉宽了事：例如 `/search` 发现态在 `lg:` 起双列（左：最近搜索/热门话题/人物推荐；右：热门动态），窄屏回落单列且顺序不变。
- 聊天类页面：消息滚动区与页头用 `max-w-3xl lg:max-w-5xl`（宽屏 1024），气泡自身仍受 `--cst-chat-bubble-max` 约束——容器变宽不能让行宽失控。

## 侧栏导航层级与跨端信息架构

- **桌面侧栏**：「附近」域直接列三个叶子入口——附近的人 / 附近的群组 / 附近瞬间（`nearby-people` / `nearby-groups` / `nearby-moments`）。用户明确要求桌面少跳一层，这是对「只列域」口径的**定向覆盖**，不推广到其他域。
- **移动 / 平板底栏**保持域级 5 slot（首页/附近/发布/消息/我的）：「附近」落 `/nearby` 枢纽页，由枢纽页三段分区的「查看全部」进三个叶子页。
- **高亮**：桌面三个叶子各自点亮对应项（`getDesktopNavId` 一对一映射）；枢纽页 `/nearby` 不再占桌面侧栏项，因此**不点亮任何项**（宁可不亮，不错亮）。移动/平板仍由 `isPrimaryRouteActive` 统一高亮「附近」。
- **入口可达性**：枢纽页仍可从首页「看看附近」卡片进入，三个叶子的深链不变。
- **跨端 IA**：桌面两端一致（鸿蒙桌面侧栏本来就有三项：附近瞬间/附近的人/附近的群组，`DESKTOP_PRIMARY_NAV`；web 桌面侧栏结构已对齐）；移动/平板两端都是域级「附近」单 tab，域内三段分区承载人/群组/瞬间。**术语统一**：全站统一用「瞬间」，web 桌面侧栏第三项、附近瞬间页标题、枢纽区块与鸿蒙端均为「附近瞬间」，不要再写「附近的动态」。
- **结论**：新增导航入口时，先问「它是域还是叶子」——默认域进侧栏/底栏、叶子进枢纽页；桌面「附近」是用户指定的例外，改其他域前先对齐用户口径，避免一端列叶子、另一端不列。

## 纵向节奏（页面级间距的唯一档位）

页面内**分区间距只允许三档**，与圆角「禁近邻档」同原则，禁止 6/10/14/18/20 这类近邻值：

| 档 | 值 | 用途 |
|---|---|---|
| 紧凑 | **12** (`SPACING.MD` / `gap-3`) | 同一区块内的行距、图文紧邻 |
| 常规 | **16** (`SPACING.LG` / `gap-4`) | 页面根容器（`PageShellContainer` 内首层 Column）、区块内元素间距 |
| 分节 | **24** (`SPACING.XL` / `gap-6`) | 页面顶/底留白、区块与区块之间 |

细粒度值（< 12：4/6/8）只用于图标与文字之间，不得用于分区间距。

**前端适用范围**：三档只约束**页面级/区块级**间距（页面根容器、区块与区块之间、页顶/底留白）。前端组件内部一律沿用 Tailwind 标准刻度（`py-2.5`=10、`py-3.5`=14、`mt-5`=20 等都是**合法档位**，实测 37/18/18 处，不要为了「三档」去改）——前端刻度比 vp 体系细，硬压会打乱既有节奏。**鸿蒙**则没有 10/14/18/20 的语义档，已全部归到 12/16/24。

### 前端落地口径（逐页归一，已验证）

`routes/(app)/**` 已按下列口径逐页归一（聊天/地图/登录/分享豁免，见「页面布局」一节的登记例外）：

| 位置 | 值 | 实现 |
|---|---|---|
| 页顶留白（页头→首块） | 16 | `contentClassName` 的 `pt-4`（封面型页由封面承担，不另加） |
| 页底留白 | 24 | `pb-6`（白名单例外见下） |
| 页面根容器（ContentColumn 首层布局容器） | 16 | `gap-4` |
| 区块与区块之间 | 24 | 父容器 `gap-6` 或子块 `mb-6`（**单侧提供**） |
| 区块标题 → 其内容 | 12 | `mb-3` / `gap-3` |
| 区块内元素 | 16 | `gap-4` |

- **根容器唯一例外**：根容器直接承载多个带标题分区（如「附近」枢纽页 = HeroStats + 附近的人/瞬间/群组）时用 `gap-6`(24)，并在页面注释说明；其余一律 `gap-4`。
- **`contentClassName` 白名单**：标准 header 页 `pt-4 pb-6`（页顶 16 由页头自身内距承接）；长表单（settings、me/edit）`pt-4 pb-10`（尾部加大留白）；封面型（groups/[groupId]）`pb-6`；资料总览（ProfilePage 总览）`pt-6 pb-8`；移动端固定底部操作条净空 `pt-4 {platform.platform === 'mobile' ? 'pb-28' : 'pb-6'}`（users/[userId]、moments/[momentId]）；状态/空态区 `py-20`（样板 `feedState` + `EmptyState` 内置）。其它散值（`pb-16`、全端 `pb-28`、`py-5` 等）一律改白名单值。
- **间距只允许单侧提供**：父容器 gap 与子块 mt/mb 不得叠成 40/44 的双倍间距；`contentClassName` 已带页顶/底留白时，首/末块不再叠 `mt-*`/`pb-*`（同页内同类间距必须同值）。
- **不归三档的部分**：组件内部刻度（`py-2.5`/`py-3.5`/`gap-1.5`/`mt-0.5` 等）、`PageHeader` 自身上下内距、头像叠压 `-mt-8`、封面负 margin、网格单元 gap、纯横向列距（桌面双栏拆成 `gap-y-6 lg:gap-x-*` 保留列距）。
- **横向列距拆分写法**：`grid gap-8` 在移动单列时是纵向 32 → 改为 `gap-y-6 lg:grid-cols-[...] lg:gap-x-8`。
- **Tab 栏→内容的间距**：带 `border-b` 的 Tab 栏（含页面内的子 Tab 栏）到其内容统一 **24**；同一页内多个 Tab 层级同值，不得「简介 24 / 成员 16」并存（groups 详情已按此统一）。列表内「工具行→列表」等块内间距仍按 16。
- **保留现状（已评估，非缺陷）**：`ContextRail` rail 分区间距 16（rail 是紧凑侧栏；改 24 会同时影响 notify/users/moments 三处共享原语，收益低）。

验证方法（可重跑）：静态扫描页面根容器/`contentClassName`/页面级块间距，离群档（`gap-5`/`gap-8`/`mb-5`/`mt-5`/`pb-16` 等）应为 0；浏览器 1920×1080 与 390×844 双视口、亮/暗双主题截图对比，只允许出现预期的纵向间距变化，不得有水平位移/错位。

## 页面级容器去边框与硬线清零（连续画布）

用户明确否决「把页面主要内容用一个框圈起来」——屏幕边缘的框线会让界面显得笨重。规则：

1. **外壳不画框**：桌面顶栏/侧栏/主内容面板、移动与平板顶栏/PageHeader 一律不得有包裹整页的 `border`/`rounded-*`/白色底面/硬分隔线（`border-b` 发丝线也算）。四端外壳共享同一张 `--cst-surface-page` 连续画布；层级靠留白、排版、交互态表达。
2. **氛围画布整窗一次**：桌面在 `DesktopLayout` 根挂单实例 `AmbientBackdrop`；不得按区域各挂一份（区域交界会出现渐变接缝）。移动/平板仍挂在各自外壳内容区。
3. **PageHeader**：静止态无底线；滚动后允许柔和投影（不再用硬 `border-bottom`）；sticky 背景 = 画布色/玻璃，不留白条。
4. **页头/分区硬线清零**：/me 自定义页头线、他人主页分区头线、群详情分区标题线、通知/设置/搜索/关系页内分区线、reports 过滤行线一律清零。页面分组用「eyebrow + 留白 + 内容软岛」，不要用横线切分。
5. **保留的功能线（白名单）**：聊天输入条/粘底操作条（`border-t`）、消息分栏竖向发丝线（`border-subtle/40`）、表格/统计行、内容单元内演示分隔。新增页面不得引入新的页面级横线。
6. **页面内允许的「边界」**：内容单元卡片（`ListContainer bordered`、设置分组卡、rail 卡、信息卡）是内容层级，不是页面框，可保留；但要收敛视觉重量（外框读起来像页面框时改投影/间距）。**例外登记**：底部 tab dock、FAB、弹层/Sheet 控件的玻璃浮层属于控件边界，可保留。
7. **自查**：截图看四边是否出现竖线/横线（桌面 1440/2560 与 390 移动端都要看）；`grep` 检查外壳组件是否残留 `border-b`/`rounded-lg border` 包裹体；`grep -rn 'border-b' frontend/src/components frontend/src/routes | grep -vE '功能线白名单'`。

## 内容软岛与长尾/后台页面

**内容软岛（开放页面内容分组，强制）**：

- 形态：`bg-surface-island + shadow-xs + rounded-lg(20)`，**无 1px 描边**；`p-2`(8)、行背景透明、行间 `gap-y-0.5`(2)、hover 填充离岛边 8；**不加 1px 行横线**。开放页面的列表集合只用 `ListContainer bordered`（已改为软岛语义）；调用方不得覆写 border/bg/radius，也不得在外层再包手写卡片（双框）。
- 例外（登记）：弹层/Sheet、hero/封面预览、单帖卡、聊天输入条、表格/统计页。
- **长尾/后台页面软岛化**：页面级自定义卡（单帖卡、群详情 hero/指标/操作、他人主页、首页 onboarding/统计/轮播、admin 模块卡与表格外壳）一律按 design-system §19 软岛化；例外按 §19.5（表格行线/表头线、输入控件、媒体容器、hero 封面、内容分隔线、chip/badge、头像环、弹层、地图面板等保留其功能边界）。单帖卡=内容岛；媒体容器去 1px 描边但保留圆角/阴影/focus-ring；表格外壳去描边但表头/行线/行 hover 保留。
- **表格（长尾/后台）**：390 宽必须可横向滚动到全部列（容器 `overflow-x-auto`，列不强行压缩）；单页 ≤20 行不强制 sticky；分页/筛选/空态/加载/错误统一走既有原语；行按钮次级软填，危险操作 soft danger。长尾页加载态：列表/卡优先 Skeleton；单实体/短列表可 InlineLoading（页面注释给理由）；空态 `EmptyState`、错误 `LoadError` + 重试、到底 `ListEnd`，四态不得互替。
- 不要为「统一」把功能边界（媒体容器/hero/输入控件等）也去掉。

## 内容轴（页面/分栏页/粘底条同轴）

页面/分栏页/粘底动作条必须共用**同一条内容轴**，否则标题会「飘」在内容外面（实测：`/search` 标题 x=284 vs 正文 x=400，差 116px；`/admin` 差 48px；`/me` 差 32px）。

1. **页面不得自设页面级 `max-w-*`**（表单/列表窄列等已登记例外除外），正文一律直接放 `ContentColumn`（`--cst-page-gutter`）；需要可读行宽时用页内 `max-w-4xl mx-auto` 且与页头同宽档。
2. **页头与正文同宽档**：`PageHeader` 的 `maxWidth` 与正文列一致；分栏页主列恒为「主列 + gap + rail」，不随视口漂移。
3. **粘底条同轴**：评论输入条/移动操作条的内部内容与内容卡同左右缘（1440 实测应逐 px 对齐）；满宽铺底时内部仍与卡同轴。
4. **表格/后台页**：正文（模块卡/表格）起点 = 页头 Tab 栏起点（`/admin` 概览卡片 284 = Tab 284）。
5. 自查：1440/1024/390 三视口截图量「同屏标题左缘 vs 正文首元素左缘」，差值必须 ≤1px。

### 宽屏呼吸档与列表轴

1. `PageHeader` 的 `maxWidth` 与 `StandardPageLayout` 的 `contentMaxWidth` 必须同值**成对**出现（审计发现 /me 正文 1280 档而页头全宽，轴漂移 46–158px）。宽屏呼吸档统一 **1280**（数字字面量）；呼吸型页面不再自造 `max-w-5xl/6xl/7xl` / `48rem` 私有档（阅读/表单列的既有 rem 列宽属「断点与内容宽度」的另一类，不属呼吸档）。
2. 列表/卡片分区内层禁止 `mx-auto max-w-2xl/3xl` 再收窄（ProfileUserSection / SectionGroups 已清理）；骨架必须与内容同宽，否则骨架→内容位移。
3. 验收口径：列表岛宽度 / main 宽度 ≥0.70@1920（关系/举报页实测 0.74–0.95）。

### 聊天页内容轴与降噪

聊天页（桌面分栏 / 移动推入）是唯一豁免 StandardPageLayout 的常规页面，但三块内容必须同轴：

1. **唯一列宽来源**：`components/messages/ChatColumn.svelte`（`mx-auto w-full max-w-3xl lg:max-w-5xl`）。聊天头部 / 消息列 / 输入栏内部都包它；页面/组件不得再写各自的 `max-w-*`。实测 1440 三列全部 x=576 / w=852；1920 全部 x=730 / w=1024；1303 全部 x=576 / w=715。错位是「头部飘着、消息挤在一侧」的直接原因（改前群聊头 768 vs 消息 1024，差 128px/侧）。
2. **短会话底对齐**：消息内容容器 `min-h-full flex flex-col justify-end`，空白留在上方；顶部锚定 + 底部 228px 死空白是「消息页很怪」的主因之一。
3. **ListEnd 降噪**：「没有更多了」只在历史确实翻页并耗尽后渲染（首屏一次返回全部、`next_cursor=null` 时不得出现）；日期分节用中性 11px 文本标签，禁止白底胶囊。
4. 输入条整体铺满面板（发丝线是功能线），内部内容经 ChatColumn 与消息列同轴；不得再用 `mx-auto max-w-3xl` 直接居中（会与消息列错 26–128px）。

## 页头滚动、回顶、焦点播报与 Reflow

- **sticky PageHeader**：顶部（scrollTop ≤4）**透明**、与连续画布融合（无白带）；滚动 >4 切 glass（78% surface-page + blur16 + 细底边 + elevation，150ms，reduced 立即）。仅改背景/边框/阴影，不动 sticky 定位（无 CLS）。
- **焦点管理**：客户端导航后焦点落 `main#main-content`（tabindex=-1、preventScroll），`#route-announcer` 播报 h1 标题（首屏不播、1.5s 去重）；不聚焦 h1 避免重复播报；无 h1 才回退并跳过播报。
- **回顶按钮**：滚动 >1.5×视口高出现 44px 圆钮（桌面 bottom-6、移动 bottom-24，避开浮层/底栏）；平滑回顶（reduced 直接跳）、回顶后 200ms 内隐藏；`aria-label` 必填；不可见时移出无障碍树。
- **Reflow**：320px 宽 / 200% 文本缩放关键页无横向滚动（documentElement/主滚动区溢出=0）；装饰层超界须被裁剪。新增浮层不得在窄屏溢出。

## 聊天滚动容器与回底

- 聊天消息区是独立滚动容器（`chat.scrollEl`）；分页 prepend 时用 `scrollHeight - prevHeight + prevTop` 保持视口位置。
- **自动滚只在「近底部（≤120px）」发生**；上滑阅读收到新消息**不得跳转**，只累加计数并显示回底按钮；自己发送/初始/切会话始终回底。
- **回底按钮**：距底 >200px 显示（44px；鸿蒙 48vp），无消息只显箭头、有新消息「N 条新消息」（>99→99+）；`aria-live`、aria-label 与文本一致；150ms 显隐（reduced 直接）；点击滚底+清零+补 markRead；不遮挡输入栏/浮层。overlay 固定于消息区右下（输入栏上方），不得放进滚动流（避免 CLS）。
- **markRead**：仅初始/近底/回底补标（1s 节流），上滑/离开不补；会话切换计数清零。新聊天页面按此契约实现。
- 阈值 120/200px 为当前基线；新增输入栏浮层/表情面板时复测按钮位置。

## 宽屏回归基线

新增页面或改布局后用同一口径复测两个视口，即可发现「内容塌回中间一条」的回归：1920×1080 下 10 个关键路由内容填充率全部 0.97、无横向滚动；1397×750 下全站 0.96（唯一 0.17 是 `/share` 的居中小卡，属预期）。取证方式：`agent-browser set viewport 1920 1080` → `open <url>` → `eval <测量表达式>`（浏览器自动化一律走 agent-browser，禁止自研脚本 spawn Chrome）。完整路由清单与测量表达式见 `references/wide-regression.md`。

## references

- `references/wide-regression.md`：宽屏回归基线的路由清单、测量口径与历史数值。
