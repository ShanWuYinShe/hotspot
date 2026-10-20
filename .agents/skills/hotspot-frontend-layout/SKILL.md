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

### 侧栏导航层级

- 侧栏**只列"域"级入口**（首页/附近/消息/我的…），**不列叶子页**；叶子页入口由枢纽页承担
  （如「附近」枢纽页的「查看全部」）。2026-10-20 移除了「附近的人/群组/瞬间」三个子项，
  原因：与枢纽页重复且让菜单拥挤。
- 访问子页路由时**高亮父项**（`getDesktopNavId` 把 `nearby-people/groups/moments` 都映射到 `nearby`）：
  侧栏表达"在哪个域"，不表达"在域的哪个叶子"。
