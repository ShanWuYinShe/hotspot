---
name: hotspot-frontend-layout
description: hotspot 前端页面布局契约：StandardPageLayout / SplitPageLayout 的选择与结构、PageHeader/TabBar/正文滚动、三端分派规则、页面纵向间距与已登记例外。触发：新建或改造前端页面；决定用哪种布局组件；布局分支与设备差异；页面间距。
---

# hotspot-frontend-layout

## 页面布局（普通页 StandardPageLayout / 分栏页 SplitPageLayout）

所有路由页面必须使用统一布局组件，禁止自行拼装 PageHeader/ContentColumn/页面级
滚动/页面级横向 padding：

- **普通页面**：`StandardPageLayout`（`components/layout/page/StandardPageLayout.svelte`），
  正文单列，滚动由 `RoutedContentViewport` 承担；正文内可含多列 grid（如群组详情
  main + aside、个人主页双栏），不属于分栏需求。
- **分栏页面**（桌面主区 + 右 rail）：`SplitPageLayout`
  （`components/layout/page/SplitPageLayout.svelte`），主区与 rail 滚动由组件统一管理，
  footer 固定于主区底部。当前分栏页：通知、动态详情、用户详情。
- **临时豁免/已登记例外**：
  - 聊天页面（桌面端与移动端）因分栏/全屏会话布局特殊，可暂不套用
    `StandardPageLayout`/`SplitPageLayout`；但页头、列表与正文的 gutter/padding
    仍必须与样板一致（沿用 `PageHeader` + `ContentColumn`）。桌面消息分栏左栏
    （`w-75`=300px 窄列）不进 `--cst-page-gutter`，用 `px-4` 压缩 gutter（见 skill
    `hotspot-frontend-style` 的「窄列特例」，已登记例外）。
  - **封面/紧凑型资料头**（`groups/[groupId]`、`users/[userId]`）：标题与封面/头像
    绑定，不在 header 操作行内，故不套 `PageHeader`，自建返回/分享/更多操作行 +
    封面下方 `<h1>`。与 `moments/[momentId]`（有独立标题"动态详情"，套 PageHeader）
    区分。
  - 移动个人主页（`ProfileMobile`）：自建 eyebrow 页眉，非标准列表页 header；
    设置/外观/编辑入口统一收在身份卡的拇指操作区，页眉不放重复入口。
  - 地图（全画布）与登录页（无 MainLayout 外壳）不适用。
- 强制统一布局是为了保证切换页面时右侧内容区不发生水平偏移；存量页面需逐步对齐，
  不得新增其他例外（新增例外必须在上方登记）。

**权威样板页：`src/routes/(app)/nearby/moments/+page.svelte`。** 普通页面以它为基准，
结构固定为：

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

1. **布局**：普通页面必须用 `StandardPageLayout`，分栏页面必须用 `SplitPageLayout`；
   header 以 snippet 传入、正文放 children；不得自行组合
   PageHeader/ContentColumn/页面级滚动/页面级横向 padding（契约见对应组件注释，
   聊天页豁免见上）。gutter 一律由 `ContentColumn` 的 `px-[var(--cst-page-gutter)]` 管理。
2. **页头**：一律用 `PageHeader`（`sticky` 默认开），`onBack` 在非桌面自动显示返回
   按钮；`below` 只放搜索框与 Tab 栏等 header 附属区。**below 首元素必须带
   上间距**（`mt-3`，与样板 nearby/moments 一致）——TabBar/搜索框直接贴 title
   属于间距错误；below 内多行元素之间用 `mt-3` 分隔。
3. **分类/筛选/排序 Tab（强制统一）**：一律使用 `TabBar`
   （`components/navigation/TabBar.svelte`，胶囊样式 + 激活项 accent 实底胶囊 +
   `aria-pressed`，对齐鸿蒙 TabsPillGroup 分段式视觉），放 `below` snippet 内
   随 header 吸顶；tab 状态可存 URL query（如 `?tab=`）以支持深链，切换用
   `goto` 更新 URL。
   **禁止再写手写 tab/chip 变体**（通知/搜索/个人主页/地图 sheet 的手写胶囊已统一为
   TabBar）。表单内的标签/多选 chips 一律用 `TagPill`；网格选择器
   （主题 swatch、发布可见范围）属例外，不套 TabBar。
4. **正文与滚动**：正文直接放 children；`StandardPageLayout` 页面滚动由
   `RoutedContentViewport`（main）唯一承担，`SplitPageLayout` 分栏页的主区/rail 滚动由
   组件统一管理；页面不得自建滚动容器（消息分栏、地图等全画布页面除外）。内嵌
   自带 gutter 的组件时用负 margin（`-mx-[var(--cst-page-gutter)]`）抵消 ContentColumn
   gutter，避免双重内边距（参照 `me/relationships/RelationshipsPage.svelte`）。
5. **三端差异**：共用组件只写一次，设备差异用 `platform.platform` 派生（列数、字号、
   布局分支），不得复制三份页面。
6. **数据获取与状态**：`$effect` + `cancelled` 竞态清理模式（参照样板页）；加载中用
   spinner、空列表用 `EmptyState`（参照样板页 feedState snippet）；数据加载失败
   必须展示 `LoadError`（`components/data-display/LoadError.svelte`）并提供重试
   （`reloadKey` 递增触发 effect 重跑），禁止静默转空态。
   分页/无限滚动列表**内容耗尽时必须渲染 `ListEnd`**（`components/data-display/ListEnd.svelte`，
   文案「没有更多了」）——「加载更多」按钮消失而无收尾提示会让用户无法区分
   「已到底」与「加载卡住」。四种状态各司其职：空列表 `EmptyState`、加载中
   `InlineLoading`、失败 `LoadError`、到底 `ListEnd`，不得互相替代。
7. **列表骨架契约**：列表一律用 `ListContainer` + `ListItem`（`components/list/`）。
   joined 行（默认 tone）**禁止叠加 rounded/border/shadow/ring 卡片类**——行间分隔
   由容器 divide 线承担，圆角外框由容器级（`ListContainer bordered` 唯一外框）或
   `tone="card"`（独立卡片容器 + gap 分隔）承担；调用方 className/style 只允许布局
   类。**高亮形状由组件统一应用**：joined 行 hover/focus/pressed 时背景自带
   `rounded-small`（12px，`--cst-shape-sm`）圆角，非高亮保持直角连接行视觉；
   圆角随背景填充出现并纳入 transition——调用方不得重复设置。双重防线：
   ESLint 自定义规则 `hotspot/no-list-item-card-classes`
   （`eslint-rules/`，Svelte AST 级，IDE 即时报错）挂入 `bun run lint`/`bun run check`；
   ListItem dev 模式运行时警告。
   后续需扩展同类组件级契约时，在 `eslint-rules/` 新增规则并在
   `eslint.config.mjs` 注册（最小化接入：不启用默认规则集）。

## 宽屏密度与断点（2026-10-20 实测教训）

**项目断点是自定义的，不要按 Tailwind 默认值判断**（`src/styles/tailwind.css`）：

| 档 | 值 |
|---|---|
| `sm` | 600px |
| `md` | 960px |
| `lg` | **1280px** |
| `xl` | **1536px** |
| `2xl` | 1792px |

常见笔记本宽度 1397px 只到 **`lg`** —— 用 `xl:` 写的布局在那里**不会生效**（实测踩过一次）。

### 内容宽度规则

- 布局原语的默认内容宽度是 `--cst-page-content-max: 100%`（**流式**）。**页面禁止再传
  `contentMaxWidth`/`maxWidth`** —— 一旦传了窄值（如 768），宽屏下内容只占中间一条、右侧大片空白
  （用户截图即此问题）。需要可读行宽时，用 `max-w-4xl mx-auto` 这类**页内**约束，且
  **宽屏要放宽**（`lg:max-w-5xl`）。
- 宽屏提升密度靠**页内响应式网格**，不是把内容拉宽了事：例如 `/search` 发现态在 `lg:` 起双列
  （左：最近搜索/热门话题/人物推荐；右：热门动态），窄屏回落单列且顺序不变。
- 聊天类页面：消息滚动区与页头用 `max-w-3xl lg:max-w-5xl`（宽屏 1024），气泡自身仍受
  `--cst-chat-bubble-max` 约束 —— 容器变宽不能让行宽失控。

### 侧栏导航层级（2026-10-22 用户审查更新）

- **桌面侧栏**：「附近」域直接列三个叶子入口——附近的人 / 附近的群组 / 附近瞬间
  （`nearby-people` / `nearby-groups` / `nearby-moments`）。用户明确要求桌面少跳一层，
  这是对 2026-10-20「只列域」口径的**定向覆盖**，不推广到其他域。
- **移动 / 平板底栏**保持域级 5 slot（首页/附近/发布/消息/我的）：「附近」落 `/nearby`
  枢纽页，由枢纽页三段分区的「查看全部」进三个叶子页。
- **高亮**：桌面三个叶子各自点亮对应项（`getDesktopNavId` 一对一映射）；枢纽页 `/nearby`
  不再占桌面侧栏项，因此**不点亮任何项**（宁可不亮，不错亮）。移动/平板仍由
  `isPrimaryRouteActive` 统一高亮「附近」。
- **入口可达性**：枢纽页仍可从首页「看看附近」卡片进入，三个叶子的深链不变。

### 跨端信息架构（2026-10-22 现状）

- **桌面**：鸿蒙桌面侧栏本来就有三项（附近瞬间 / 附近的人 / 附近的群组，
  `DESKTOP_PRIMARY_NAV`）；web 桌面侧栏 2026-10-22 起结构对齐（人/群组/瞬间），两端桌面 IA 一致；
- **移动 / 平板**：两端都是域级「附近」单 tab，域内三段分区承载人/群组/瞬间；
- **术语统一（2026-10-22 用户确认）**：全站统一用「瞬间」；web 桌面侧栏第三项、附近瞬间页标题、
  枢纽区块与鸿蒙端均为「附近瞬间」，不要再写「附近的动态」。

结论：**新增导航入口时，先问"它是域还是叶子"**——默认域进侧栏/底栏、叶子进枢纽页；
桌面「附近」是用户指定的例外，改其他域前先对齐用户口径，避免一端列叶子、另一端不列。

## 宽屏回归基线（2026-10-21 实测，用 agent-browser 取证）

视口 **1920×1080**，10 个关键路由（`/`、`/search`、`/messages`、`/me`、`/nearby/moments`、
`/notify`、`/settings`、`/admin`、`/users/user_lumos`、`/groups/group_1`）：

- **内容填充率全部 0.97**（页面内容横向占满可用区域，侧栏宽度扣除后计算）；
- **无横向滚动**（`scrollWidth <= innerWidth + 2`）；
- `/search` 宽屏双列（左：热门话题+人物推荐；右：热门动态）在 1920 下依然成立（截图核对）。

1397×750 视口的同口径基线见前文（全站 0.96，唯一 0.17 是 `/share` 的居中小卡，属预期）。
新增页面或改布局后，用同一口径复测这两个视口即可发现"内容塌回中间一条"的回归。

取证方式：`agent-browser set viewport 1920 1080` → `open <url>` → `eval <测量表达式>`。
（2026-10-21 起红线：浏览器自动化一律走 agent-browser，禁止自研脚本 spawn Chrome。）

## 纵向节奏（2026-10-21 定，页面级间距的唯一档位）

页面内**分区间距只允许三档**，与圆角「禁近邻档」同原则，禁止 6/10/14/18/20 这类近邻值：

| 档 | 值 | 用途 |
|---|---|---|
| 紧凑 | **12** (`SPACING.MD` / `gap-3`) | 同一区块内的行距、图文紧邻 |
| 常规 | **16** (`SPACING.LG` / `gap-4`) | 页面根容器（`PageShellContainer` 内首层 Column）、区块内元素间距 |
| 分节 | **24** (`SPACING.XL` / `gap-6`) | 页面顶/底留白、区块与区块之间 |

细粒度值（< 12：4/6/8）只用于图标与文字之间，不得用于分区间距。

**前端适用范围（2026-10-21 修正）**：上面三档只约束**页面级/区块级**间距（页面根容器、
区块与区块之间、页顶/底留白）。前端组件内部一律沿用 Tailwind 标准刻度（`py-2.5`=10、
`py-3.5`=14、`mt-5`=20 等都是**合法档位**，实测 37/18/18 处，不要为了"三档"去改）——
前端刻度比 vp 体系细，硬压会打乱既有节奏。

**鸿蒙**则确实没有 10/14/18/20 的语义档，已全部归到 12/16/24（2026-10-21 完成）。

### 前端落地口径（2026-10-21 Round 9 逐页归一，已验证）

`routes/(app)/**` 已按下列口径逐页归一（聊天/地图/登录/分享豁免，见「页面布局」一节的登记例外）：

| 位置 | 值 | 实现 |
|---|---|---|
| 页顶留白（页头→首块） | 16 | `contentClassName` 的 `pt-4`（封面型页由封面承担，不另加） |
| 页底留白 | 24 | `pb-6`（白名单例外见下） |
| 页面根容器（ContentColumn 首层布局容器） | 16 | `gap-4` |
| 区块与区块之间 | 24 | 父容器 `gap-6` 或子块 `mb-6`（**单侧提供**） |
| 区块标题 → 其内容 | 12 | `mb-3` / `gap-3` |
| 区块内元素 | 16 | `gap-4` |

- **根容器唯一例外**：根容器直接承载多个带标题分区（如「附近」枢纽页 = HeroStats + 附近的人/瞬间/群组）
  时用 `gap-6`(24)，并在页面注释说明；其余一律 `gap-4`。
- **`contentClassName` 白名单**：标准页 `pt-4 pb-6`（默认值）、长表单（settings、me/edit）`pt-4 pb-10`、
  封面型（groups/[groupId]）`pb-6`、资料总览（ProfilePage 总览）`pt-6 pb-8`、移动端固定底部操作条净空
  `{platform.platform === 'mobile' ? 'pb-28' : 'pb-6'}`（users/[userId]、moments/[momentId]）。
  其它散值（`pb-16`、全端 `pb-28`、`py-5` 等）一律改白名单值。
- **间距只允许单侧提供**：父容器 gap 与子块 mt/mb 不得叠成 40/44 的双倍间距；`contentClassName`
  已带页顶/底留白时，首/末块不再叠 `mt-*`/`pb-*`（同页内同类间距必须同值）。
- **不归三档的部分**：组件内部刻度（`py-2.5`/`py-3.5`/`gap-1.5`/`mt-0.5` 等）、`PageHeader`
  自身上下内距（有 below：16/14；无 below：18/18，属页头组件内部构图）、头像叠压 `-mt-8`、
  封面负 margin、网格单元 gap、纯横向列距（桌面双栏拆成 `gap-y-6 lg:gap-x-*` 保留列距）。
- **横向列距拆分写法**：`grid gap-8` 在移动单列时是纵向 32 → 改为 `gap-y-6 lg:grid-cols-[...] lg:gap-x-8`。
- **Tab 栏→内容的间距（2026-10-22 统一）**：带 `border-b` 的 Tab 栏（含页面内的子 Tab 栏）
  到其内容统一 **24**；同一页内多个 Tab 层级同值，不得「简介 24 / 成员 16」并存
  （groups 详情已按此统一）。列表内「工具行→列表」等块内间距仍按 16。
- **保留现状（已评估，非缺陷）**：`ContextRail` rail 分区间距 16（rail 是紧凑侧栏；改 24 会同时
  影响 notify/users/moments 三处共享原语，收益低）。

验证方法（可重跑）：静态扫描页面根容器/`contentClassName`/页面级块间距，离群档（`gap-5`/`gap-8`/
`mb-5`/`mt-5`/`pb-16` 等）应为 0；浏览器 1920×1080 与 390×844 双视口、亮/暗双主题截图对比，
只允许出现预期的纵向间距变化，不得有水平位移/错位。
