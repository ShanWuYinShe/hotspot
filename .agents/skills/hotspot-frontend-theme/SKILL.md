---
name: hotspot-frontend-theme
description: hotspot 前端主题与 CSS Token 架构：三层 token（shared 基础/语义/themes 调色板）、配色与硬编码约束、画布/软岛/图片卡遮罩/氛围画布 token 契约、状态色 on-soft 档、导航三态 token、字体自托管与禁止第三方 @import、明暗三档与主题验收。触发：改颜色/主题/design token；新增 token 或注册 Tailwind 类；跨端色彩与刻度同步；字体或第三方样式表引入；主题/深色验收。
---

# hotspot-frontend-theme

## 配色硬约束（强制）

**所有涉及配色的取值——字体颜色、背景、边框、hover 颜色、active 颜色、阴影、遮罩、状态色、图钉/标记色等——必须来自 design token，严格禁止在页面/组件中单独定义色值**（禁止 `text-white`、`color:#fff`、`bg-black/50`、内联 `rgba(...)` 等字面量）。所有 token 必须能随主题（vibrant/forest/inkwash）与明暗模式正确变化。

- **前缀约定**：全部 design token 使用 `cst-` 前缀（`--cst-text-primary`、`--cst-surface-primary`、`--cst-elevation-card`…），一眼可识别 token 来源；定义层（shared/semantic/themes/layout/shadcn）禁止出现无前缀变量。
- **Tailwind 注册层例外**：`theme-registry.css` 的 `@theme inline` 注册键（`--color-*`、`--text-xxs` 等字号、`--radius-*`、`--shadow-*`、`--ease-standard`）是 Tailwind 工具类生成机制，**必须保持原名**；其值引用 `var(--cst-*)`。
- **Token 两层链**：`--cst-text-primary` 等语义名 → `--cst-palette-*` 原始值（无旧名兼容层）。
- **新增 token 必须同步注册 Tailwind 类**：需以 `bg-*/text-*/border-*` 类名消费的 token，必须在 `theme-registry.css` 的 `@theme inline` 补 `--color-*` 注册（如 `--color-text-on-status`）；只经 `var()` 内联引用的才可跳过。未注册即产生死类名静默失效（机制与自查见 skill `hotspot-frontend-style` 自查清单第 6 条）。
- **新增组件缺语义 token 时**：先在 themes/{vibrant,forest,inkwash}.css 补 `--cst-*` 定义（light + dark），不得在组件里写死色值绕过。

### 配色映射速查（页面中如何选 token）

- 正文/辅助文字：`--cst-text-primary` / `--cst-text-secondary`。
- 主题色背景上的文字：`--cst-text-on-accent`（按钮/选中态；**随明暗模式切换深浅**——light 白配深色 accent、dark 深配浅色 accent）。
- 状态色背景上的文字：`--cst-text-on-status`（online/warning 实底，如连接横幅）；danger 实底按钮一律走 `--cst-text-on-accent`（随明暗翻转：light 白 / dark 深；vibrant light danger `#C81E4E` 配白字 5.60，对齐鸿蒙 error 同值，与各确认按钮一致）。
- 图片/遮罩上的反色文字：`--cst-content-inverse`（**恒白**，仅用于图片/遮罩/深色 overlay 上，禁止用于 accent/status 实色背景——那些背景在 dark 下是浅色，恒白文字会低对比隐形）。
- 图钉色背景上的文字：`--cst-content-on-marker`（地图标记）。
- **accent 背景上的文字/图标一律用 `--cst-text-on-accent` 系列**（含次级文字用 `opacity-70/80` 派生，参照 ListItem/ChatListItem selected 态）；图钉等 marker 色与 accent 同色（`--cst-palette-map-user` = `--cst-palette-accent-primary`），在 selected 行内需换实色底衬托（参照 `PinMedia` 的 `selected` prop）。
- 遮罩/玻璃：`--cst-surface-scrim`（弹层遮罩）、`--cst-surface-media-scrim-strong`（照片遮罩）、`--cst-surface-cover-scrim`（封面渐变）、`--cst-surface-glass-badge`（玻璃徽章）。
- hover/active：优先 `hover:bg-accent-soft` 等 token 类；背景填充走 `--cst-interactive-hover-fill` / `--cst-interactive-active`。
- 弱化嵌入面（分段控件底轨、置顶会话行等「比卡片低一档」的静态填充）：`--cst-surface-subtle`（`bg-surface-subtle`）。与 `bg-interactive-hover` 的区别是语义：前者是「静态的凹陷/次表面」，后者是「悬停反馈」，不可互换。该表达式经 `--cst-text-primary` 自适配明暗（light 微沉 / dark 微浮），因此只在 themes/*.css 的 light 块定义一次，**不算缺失 dark 覆盖**。
- **数据层禁止携带颜色**：mock 数据/API 响应只返回语义标识（tag/type/category），不得包含 `color`/`bg` 等展示字段；视觉色由前端按语义键映射到 token（参照 `notify/shared/notify-data.ts` 的 `TYPE_CONFIG` 语义键 → token 映射模式）。
- **组件/数据层禁止 color-mix 表达式**：需要淡底/淡边框/光晕时引用主题预定义档位 token——每个语义色族在 themes/{vibrant,forest,inkwash}.css 定义三档：`-bg`（淡底）、`-border`（淡边框）、`-glow`（光晕），如 `--cst-status-warning-bg/-border/-glow`、`--cst-marker-user-bg/-border/-glow`。color-mix 只允许出现在 `styles/theme/` 定义体系内部（themes/*.css 的 token 值、semantic.css 的语义派生档位、utilities.css/base.css 的既有工具类——值即最终定义）；页面/组件/数据层仍然禁止，需要新档位时先在定义体系补 token。

## 画布、软岛与媒体表面 token（强制）

- **连续画布契约**：`--cst-surface-page`（= page-bg）是应用内唯一的整层底色。外壳各层（外框/内框/视口/状态栏/安全区/导航底座透明区/桌面顶栏/侧栏/主内容）一律只许用它连续铺底，**不得再加**包裹整页的描边/圆角/白底/硬分隔线（会在屏幕边缘形成「框线」，用户明确否决）。层级只靠**内容单元软岛**与留白表达；内容单元格本身不是页面框，可保留。禁止引入第二个整层底色（原整层底色 `workspace` token 已删除；layout.css 仅保留 `--cst-workspace-gap`/`-inset` 间距变量，不是底色）。分壳规则：四端外壳直接暴露画布（`[data-page-viewport]` 透明），桌面氛围画布为**整窗单实例**（外壳根挂一次）；`PageHeader` 实底走 `.page-header-solid`（一律画布色，**无底边线**，滚动时才允许柔和投影），sticky glass 仍走 `surface-nav-glass`。唯一例外：聊天页独立画布 `--cst-surface-chat`（= chat-bg，微差于页面画布，对齐鸿蒙 `chatBackground`），仅聊天页根/头部/输入条使用。
- **内容岛表面**：`--cst-surface-island`（light = `surface-primary 86%` 半透合成画布；dark = `surface-primary 94% + text-primary 6%`；三主题×明暗 6 处，registry 映射 `bg-surface-island`）。开放页面内容分组用「软岛」：无硬描边 + `shadow-xs` + `rounded-lg`(20) + `p-2` + 行透明/gap；**禁止再给 `ListContainer bordered` 覆写 border/bg/radius**。详见 design-system §19；页面结构侧规则见 skill `hotspot-frontend-layout`「内容软岛与长尾/后台页面」。
- **图片优先卡遮罩**：整卡图片卡只用 `--cst-surface-card-scrim-top` / `--cst-surface-card-scrim-bottom` / `--cst-surface-card-fallback`（三主题×明暗定义在 `themes/*.css`，语义见 `docs/design-system.md` §17）；禁止页面里写 rgba 遮罩。**可读性优先**：压图文字必须满足 4.5:1，必要时加深 token alpha，不要靠浅色玻璃底压白字。
- **氛围画布**：内容稀疏/空态页用主题化装饰层补纵深，唯一实现 `components/layout/ambient/AmbientBackdrop.svelte`：canvas 变体挂四端外壳内容区（桌面主内容卡 / 平板 / 移动）/ 认证页 / 错误页 / 启动占位；chrome 变体挂桌面顶栏与侧栏（只铺天光、不画世界层）。页面禁止自建第二套背景装饰。装饰层是**透明渐变/图形叠加**，不引入第二整层底色，因此不违反画布契约；它必须 `aria-hidden` + `pointer-events:none` + `z-index:-1`（容器 `relative isolate`）。色值走 `themes/*.css` 的 `--cst-ambient-*` 全套 token（三主题 × 明暗；新增主题必须补齐，`check-theme-parity` 强制），完整分层/主题语言/边界见 `docs/design-system.md` 第 16 节。
- **图钉 SVG 资源（`assets/markers/*.svg`）**：主题色 fill/stroke 一律用 `currentColor`，由 `location-shared.ts` 的 `markerSvg()` 以 `style="color:var(--cst-marker-*)"` 染色，随主题/明暗变化；资源内仅保留图标固有固定对比色（深色底 `#102A43`、白色高光 `#FFFFFF`）不随主题，属合理例外。新增 marker 类 SVG 必须遵循：主题色走 currentColor，禁止内嵌主题色 hex。
- **主题选择器 swatch（stores/theme.svelte.ts `THEME_OPTIONS`）**：色板需在主题激活前静态展示**目标主题**的 accent 代表色，而 CSS 变量只能解析「当前」主题，故保留字面 hex（vibrant `#2B5EB2` / forest `#39714F` / inkwash `#A63A2E`）作为已登记例外；必须与 themes/{vibrant,forest,inkwash}.css 的 `--cst-palette-accent-primary` 保持同步（check-cross-platform-colors.mjs 直比）。

## 输入表面 token（2026-10-29）

- 输入族表面统一走 `.input-surface`（utilities.css）；颜色全部由 semantic.css 的 `--cst-field-*` 派生：`bg` / `bg-disabled` / `border` / `border-hover` / `border-focus` / `ring` / `ring-danger`（由 accent / status-danger / surface 系混出，三主题×明暗自动成立）。
- **主题文件（themes/*.css）禁止覆写 `.input-surface`**：vibrant 曾给全部输入加 `backdrop-filter: blur(16px)` + 白化混合底，导致大面积毛玻璃（违反性能红线）且与 forest/inkwash 纸面语言不一致；2026-10-29 全部移除。
- 输入表面不使用投影（静态/hover/focus 都不加）：层级由单层 hairline 边界 + 字段底 + 焦点环表达，纸面主题因此零主题分支。
- 新增字段色档只能在 semantic.css 用 `color-mix` 派生（定义层最终值）；组件/页面仍禁止 color-mix。`--cst-field-*` 不注册 Tailwind 颜色类（组件不消费类名）。

## 状态色 on-soft 档

- 「tint 淡底 + 同色语义文字」必须用 on-soft 文字档：`text-text-accent-on-soft` / `text-text-status-{online,warning,danger}-on-soft`（= 语义色 80% + text-primary 20%，`semantic.css` 派生）；**仅限 tint 软底**（徽章/胶囊/软底容器/危险菜单 hover），实底/画布正文继续用 `text-status-*` / `text-accent-default`。
- 悬停才出现 tint 的场景写成 `text-status-* hover:text-text-status-*-on-soft` + `hover:bg-*`；背景 mix 比例不动。
- 状态色/徽章的对比度**验收**（三主题×明暗 ≥4.5、矩阵与 `check_status_parity.py` 命令）单点权威在 skill `hotspot-frontend-style`「质量门与性能红线」；本文件只定义 token 与用法，不重复验收口径。

## 导航三态 token

- `--cst-surface-navigation-hover` = `color-mix(text-primary 12%, surface-page)`（中性低彩度，hover 靠明度）；
- `--cst-surface-navigation-active` = `color-mix(accent-default 26%, surface-page)`（选中底色）；
- `--cst-surface-navigation-active-hover` = `color-mix(accent-default 34%, surface-page)`（选中+hover）。
- 三者定义在 `semantic.css`（随主题 × 明暗自动派生，**不要**写进 themes/*.css palette，跨端 parity 不比对）；`theme-registry.css` 注册 `--color-surface-navigation-{hover,active,active-hover}` 以启用 `bg-surface-navigation-*` 类（Tailwind v4 未注册 utility 静默失效）。
- 灰度验收口径：hover 底/画布 ≥1.2、选中底/画布 ≥1.4，且选中至少 2 种非颜色线索（指示条/内缘描边/字重/图标）；`filter:grayscale(1)` 截图（三主题 × 明暗）留档。hover 禁用 accent 色相（色盲会把 hover 误读成选中）。旧值（text-primary 9% 混 surface-primary）实测 hover 底/画布仅 1.07–1.19:1、灰度不可辨，已被取代。

## 焦点/动效相关 token 口径

- 焦点环颜色只用 `--cst-accent-default`（accent token），2px + 2px offset；**不得新造焦点色**。
- 减少动态走全局 `@media (prefers-reduced-motion: reduce)`（`styles/theme/utilities.css`），不新增动画 token；新增动画必须复用既有 motion token + 全局降级规则，禁止裸写 duration/iteration。
- 软岛/长尾例外不改颜色 token：媒体/hero/表格行线等边界由结构/阴影/发丝线表达，颜色仍走既有语义 token。

## 主题覆盖与验收

- **forest light 软岛覆写**：`--cst-surface-island = color-mix(surface-primary 92%, text-primary 8%)`（合成 #E4E6E2，岛/画布 ratio 1.17）；其余 5 组保持 86%（light）/94%+6%（dark）。修改此值时两端同改并重跑 parity。
- **方块验收**：三主题 × 明暗关键页截图 + 软岛/画布亮度比 + 文字对比（≥4.5）记录到 `frontend-ui-audit.md`；禁止只验 vibrant。
- **harmony-only 档**：`textTertiary`（web 无对应）允许单端调整（forest light #5F6C64→#58645C），但需在文档标注。
- **状态徽章对比**：验收规则与命令见 skill `hotspot-frontend-style`「质量门与性能红线」，此处不重复。

## 主题 & CSS Token 架构

`src/styles/theme/` 目录按三层架构组织：

```
theme/
  index.css              # 入口，按依赖顺序导入所有分片
  shadcn.css             # Layer 0: shadcn/ui 基础变量
  shared.css             # Layer 0: z-index/shape/motion/ease（与主题无关）
  semantic.css           # Layer 1: 语义 token（--cst-text-primary 等 → --cst-palette-*）
  themes/vibrant.css     # Layer 2: 星际主题（全部 --cst-palette-* 值）
  themes/forest.css      # Layer 2: 清野主题（全部 --cst-palette-* 值）
  themes/inkwash.css     # Layer 2: 水墨主题（全部 --cst-palette-* 值）
  layout.css             # 布局尺寸变量（context-rail 宽/间距、workspace-gap/inset 等；无颜色 token）
  base.css               # @layer base（排版 + 按钮基线）
  theme-registry.css     # @theme inline Tailwind 注册（字号 text-*/字距 tracking-*/行高
                         #   leading-*/圆角 radius-*/阴影 shadow-raised-*+inset-*+glow-*/动效 ease-*）
  utilities.css          # @layer utilities（组件工具类 + hover-* 交互类）
```

## 明暗与主题切换

- 主题切换通过 `<html>` 上的 `.vibrant` / `.forest` / `.inkwash` + `.dark` 类实现。
- 明暗三档（`stores/theme.svelte.ts`）：`colorScheme='auto'|'light'|'dark'`（持久化键 `colorScheme`），`auto`=跟随系统（`matchMedia('(prefers-color-scheme: dark)')` 初值 + change 监听刷新，监听由根布局持有/清理）；`light`/`dark`=手动档（忽略系统变化）。老 `isDark` 键只做一次性迁移（存在→对应手动档并删键，不存在→`auto`），首帧脚本（`app.html`）同步三态逻辑防闪。设置页与 `AppearanceMenu` 用 `TabBar` 三段选择（跟随系统/浅色/深色），`toggleDark` 仅作手动档间切换的兼容入口。
- 浏览器 chrome 配色（`<meta name="theme-color">`）随当前主题画布色同步：首帧由 app.html 内联脚本按持久化偏好设置四态映射，之后由 theme store 的 `applyClasses` 读取 `--cst-surface-page` 动态更新。
- 悬停/按压反馈统一用 `hover-btn/action/bg`、`scale-btn`、`tap-*`、`header-action` CSS 类（主题差异由 themes/*.css 的 `@layer utilities` 覆盖承载），避免 JS 状态动画。**行内次要动作的显现**统一用 `desktop-reveal`（仅 fine pointer 默认隐藏，行 `.group` hover / 自身 `:focus-visible` / `aria-expanded="true"` 时显现；触屏无 hover 则常显）——禁止各页用 `opacity-0 group-hover:opacity-100` 自行拼装。
- 主题专属组件覆盖（`.vibrant .foo` / `.forest .foo` / `.inkwash .foo`）统一放对应 themes/*.css 的 `@layer utilities` 块，utilities.css 只保留与主题无关的通用类。

## 字体（自托管，禁止第三方阻塞式引入）

- **Inter 自托管**：`static/fonts/inter-latin-wght.woff2`（latin 子集，可变字重轴 100–900，48 KB），`@font-face` 定义在 `src/styles/fonts.css`，首帧 `<link rel="preload">` 在 `src/app.html`（字体请求必须带 `crossorigin`，否则同源也会下载两次）。
- **中文字形走系统字体栈**，不自托管 CJK 字体：`"Inter", "PingFang SC", "HarmonyOS Sans SC", "Microsoft YaHei", "Noto Sans CJK SC", ui-sans-serif, system-ui…`（定义在 `theme/base.css`）。与 harmonyapp 使用系统字体的策略一致，也避免为 CJK 背上数 MB 分片。
- **禁止在 CSS 里 `@import` 任何第三方样式表**（字体/图标/CDN 主题等）。`@import` 在样式表内部是串行且阻塞的：浏览器会把整张应用样式表挂起，直到该第三方资源返回。本项目实测代价——`@import url("https://fonts.googleapis.com/css2?...")` 122 KB / 2.4–5.1s，样式生效时间 2999ms；删除后同一构建 159ms（首次绘制 2908ms → 364ms）。需要外部字体/图标时改为自托管，或用 `media="print"` + `onload` 切回的非阻塞 `<link>` 并注明理由。
- **许可**：字体许可全文随资源存放（`static/fonts/OFL.txt`），并在 `frontend/ATTRIBUTIONS.md` 登记来源与许可类型。
