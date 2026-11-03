# HotSpot 设计系统

> **现行权威**：本文件是双端设计 token 数值的唯一权威文档，与
> `frontend/scripts/check-cross-platform-*.mjs`（CI 强制执行）共同保证双端一致。
> 改色、改字号、改间距、改圆角、改阴影、改动效时，两端必须同步，漂移由
> `bun run check:theme` 在 CI 拦截。强制的同步流程与组件规则见两端 `AGENTS.md`
> （根 `AGENTS.md` / `frontend/AGENTS.md` / `harmonyapp/AGENTS.md`）。

> 单一事实源（Single Source of Truth）。本文件整理 Svelte Web 前端的 `--cst-*` token 体系
> 与鸿蒙 `DesignConstants` / `ThemeConstants` 的逐项映射，覆盖**桌面端 / 平板端 / 移动端**三档设备形态。
>
> **token 落点**（前端，均在 `frontend/src/styles/theme/`）：
> `semantic.css`（语义别名与派生公式）、`shared.css`（z-index / shape / 边框宽度 / 动效 / 缓动 / 模糊 /
> 弹层高度 / 聊天宽度 / 页面 gutter / 大屏缩放）、`layout.css`（布局壳尺寸）、
> `themes/*.css`（三主题 × 明暗的色板与情境覆写）、`utilities.css`（入场时长等类内 token）、
> `theme-registry.css`（Tailwind v4 `@theme inline` 注册）。
> 鸿蒙侧统一走 `DesignConstants`（尺寸/动效/形状/间距/token）与 `ThemeConstants`（主题色/阴影/氛围色板）。

---

## 1. 设计原则

1. **好看**：效果克制、有节奏，不堆叠效果。
2. **交互状态一眼可辨**：hover / focus / active / disabled 四态差异长在元素本体上。
3. **风格统一**：复用既有原语，禁止每页自造交互样式。
4. **风格走 token**：所有风格类变化（颜色、阴影、圆角、字号、字距、行高、透明度、
   边框粗细、动效时长）必须落在 token 体系；页面只消费语义 token，禁止内联硬编码。
5. **少档位 + 强制收敛**：1–2px 近邻档位一律归并；「不够用就归到最近档位」是默认动作。
6. **页面不空荡**：内容稀疏/空态页必须有视觉纵深，禁止大面积纯白空屏；用主题化「氛围画布」
   背景层兜底（见第 16 节），不靠堆卡片或占位文案凑密度。

---

## 2. 断点与设备形态

两端统一采用「三档设备形态 + 一处内容双栏阈值」：

| 语义 | 前端（`platform.svelte.ts`） | 鸿蒙（`core/responsive/Breakpoint.ets`） | 值 |
|---|---|---|---|
| 移动端（手机） | `platform === 'mobile'` | `!isTablet` | `< 600` |
| 平板端 | `platform === 'tablet'` | `isTablet` 且 `!isDesktopForm` | `600 – 1023` |
| 桌面端 | `isDesktop` | `isDesktopForm` | `≥ 1024`（两端一致） |
| 内容双栏（master-detail） | `isWideTablet` | `isWideForm` | `≥ 960` |

- **宽度来源**：鸿蒙 `EntryAbility` 把窗口宽 vp 写入 `AppStorage.currentWidthVp`，`Breakpoint` 语义函数按宽度精确判定（阈值 600/960/1024，以 web 为基准）；无宽度来源时回退命名断点（xl/xxl 视为桌面）。
- **chrome 归桌面**：两端 ≥1024 切桌面顶栏 + 侧边栏；其余走 Phone/TabletLayout（**移动端和平板端的导航栏永远在底部**，禁止在左侧出现 side rail 侧边栏）。
- **内容双栏归 isWide**：≥960 内容可双栏；960–1023 仍是「平板药丸底栏 + 内容可双栏」，不是桌面侧边栏。
- 鸿蒙 `Breakpoint.ets` 底层 6 档（xs/sm/md/lg/xl/xxl，阈值 320/600/840/1200/1600vp）只服务 ArkUI 栅格；业务一律用语义派生量（宽度判定优先）。
- 前端 Tailwind 断点在 `tailwind.css` 覆写为：`sm` 600 / `md` 960 / `lg` 1024 / `xl` 1280 / `2xl` 1536，与 `platform.svelte.ts` 的语义阈值咬合（**不是 Tailwind 默认 640/768/1024/1280/1536**）。

---

## 3. 颜色系统

三层架构：**primitive**（`--cst-palette-*`）→ **semantic**（`--cst-*`）→ **registry**（Tailwind `@theme`）。

主题族（vibrant / forest / inkwash）双端可选集一致：前端三套齐全；鸿蒙 `default_clean` = forest 清野、`inkwash` = 水墨， vibrant 为默认。每主题族 × 明暗模式。

### 3.1 原始色板（primitive `--cst-palette-*`）

| Token | Vibrant 亮 | Vibrant 暗 | Forest 亮 | Forest 暗 | Inkwash 亮 | Inkwash 暗 |
|---|---|---|---|---|---|---|
| `page-bg` | `#EFF3F9` | `#0B0E14` | `#F6F7F3` | `#161A16` | `#F5F2EA` | `#181A1C` |
| `bg` | `#FFFFFF` | `#121622` | `#F6F7F3` | `#161A16` | `#FBF9F3` | `#212426` |
| `chat-bg` | `#EDF3FC` | `#0E121B` | `#F2F3F2` | `#0E100E` | `#F1EDE3` | `#1D2022` |
| `text` | `#16233A` | `#F1F5F9` | `#152219` | `#E8F1E9` | `#2B2A26` | `#EAE6DA` |
| `text-muted` | `#52647F` | `#94A3B8` | `#3F5547` | `#A8BDAE` | `#6B675D` | `#A8A49A` |
| `accent-primary` | `#2B5EB2` | `#5B9CF6` | `#39714F` | `#52B878` | `#A63A2E` | `#DF7F6A` |
| `on-accent` | `#FFFFFF` | `#0B0E14` | `#FFFFFF` | `#0C1911` | `#FFFFFF` | `#1F1512` |
| `online` | `#0D6A62` | `#34D399` | `#39714F` | `#52B878` | `#3F7256` | `#82B297` |
| `hot`（危险） | `#C81E4E` | `#F87171` | `#A04B37` | `#D97C5A` | `#9E2140` | `#DE8496` |
| `warning` | `#9C4708` | `#FBBF24` | `#7E602B` | `#C4A75A` | `#7F6422` | `#D6A54E` |
| `map-user` | `#2B5EB2` | `#5B9CF6` | `#39714F` | `#52B878` | `#A63A2E` | `#DF7F6A` |
| `map-group` | `#0B7653` | `#34D399` | `#4A703F` | `#88B85A` | `#4A7048` | `#7FA88B` |
| `map-moment` | `#C2470B` | `#FB923C` | `#8A6426` | `#C4A75A` | `#8A6426` | `#C89B5A` |

> 三主题各 37 项 `--cst-palette-*`（每项在 light/dark 各一处定义）；上表为主色板，其余为派生/情境原料：`border` / `border-pressed` /
> `accent-soft` / `accent-soft-border` / `hover` / `active-hover` / `glow` / `shadow-light` / `shadow-dark` /
> `shadow-card` / `shadow-btn` / `overlay-strong` / `opacity-*` 等，真值与完整清单见 `themes/*.css`。
> 三主题变量集合由 `check-theme-parity.mjs` 强制一致（light/dark 各自比对）。

> `workspace-bg` 整层底色不存在（与 `page-bg` 两档整层灰并存会造成断裂）：桌面舞台不用白浮卡包住侧栏/主面板，
> 整窗为一张连续画布，侧栏、主内容、顶栏均无外框、无硬分隔线，层级只由内容单元卡片与留白表达
> （见 §17 与 theme skill「连续画布契约」）。

> 派生档位（`-bg` 淡底 / `-border` 淡边框 / `-glow` 光晕）由 `color-mix(in srgb, 主色 12~14%, transparent)`
> 统一生成，禁止组件/数据层再写 `color-mix`。

### 3.2 语义层与情境 token（`semantic.css`）

语义 token 是组件唯一消费面：跨主题/明暗同名，值随主题解析。别名映射集中定义在
`semantic.css`，主题相关覆写在 `themes/*.css`（见 §3.4）。

**基础语义**

| 语义 token | 指向 / 公式 | 用途 |
|---|---|---|
| `--cst-text-primary` / `-secondary` | `palette-text` / `palette-text-muted` | 正文 / 辅助文字 |
| `--cst-text-on-accent` | `palette-on-accent` | accent 实底上的文字 |
| `--cst-surface-primary` / `-page` / `-chat` | `palette-bg` / `page-bg` / `chat-bg` | 卡片面 / 画布 / 聊天画布 |
| `--cst-surface-subtle` / `-raised` / `-panel` | 主题层：primary 混 8% text-primary（弱化嵌入面） / primary / primary | 分段底轨、置顶行等低一档填充 / 浮起控件 / 面板 |
| `--cst-surface-island` | light `color-mix(surface-primary 86%, transparent)`（forest 92% + text-primary 8%）；dark 94% primary + 6% text-primary（vibrant 显式 `#181E2B`） | 内容岛表面（§19） |
| `--cst-surface-overlay` / `-overlay-strong` | 主题层（遮罩色随主题/明暗覆写） / `palette-overlay-strong` | 遮罩 / 强遮罩 |
| `--cst-surface-scrim` / `-glass-badge` / `-media-scrim-card` / `-media-scrim-strong` | 主题层（见 §3.4） | 弹层底遮罩 / 压图小徽章玻璃 / 图片卡底部渐隐 / 强压暗 |
| `--cst-content-inverse` / `-muted` / `-secondary` | 主题层（白系） | 压图/深底上的反色文字三档 |
| `--cst-content-secondary` | `text-secondary` | 次级内容 |
| `--cst-text-on-status` / `--cst-content-on-marker` | 主题层（深色） | 状态/标记实底上的文字 |
| `--cst-accent-default` / `-soft` / `-border` | `accent-primary` / `accent-soft` / `accent-soft-border` | 品牌 / 淡底 / 淡描边 |
| `--cst-accent-soft-bg` / `-border-soft` | accent 12% / 30% `color-mix(... transparent)` | accent 淡底 / 淡边框 |
| `--cst-glow-accent` / `-danger` | `palette-glow` / `rgba(0,0,0,.12)` | 微阴影光晕原料 |
| `--cst-border-default` / `-strong` / `-subtle` / `-inverse` | `border` / `border-pressed` / `border-default` / 主题层 | 常规 / 强 / 弱分隔 / 反色描边 |
| `--cst-interactive-hover` / `-active` / `-hover-fill` / `-active-fill` | `palette-hover` / `active-hover`（fill 与其同源） | 悬停/按压实底 |
| `--cst-surface-hover` | accent 7% 混 primary | accent 系悬停面 |
| `--cst-hover-lift` / `-image-zoom` | `2px` / `1.045` | 卡片悬停抬升 / 图片放大 |
| `--cst-status-online` / `-danger` / `-warning` | `online` / `hot` / `warning` | 状态色 |
| `--cst-status-*-bg` / `-border` | 14% / 28% `color-mix(... transparent)` | 状态淡底 / 淡边框 |
| `--cst-text-accent-on-soft` / `-status-*-on-soft` | 80% 语义色 + 20% text-primary | 淡软底上的同色系文字（对比度 ≥4.5） |
| `--cst-marker-user/group/moment` | `map-*` | 地图标记主色 |
| `--cst-marker-*-bg` / `-border` / `-glow` | 14% / 30% / 50% `color-mix(... transparent)` | 标记淡底 / 边框 / 光晕 |
| `--cst-elevation-shadow-light` / `-dark` | `palette-shadow-light` / `-shadow-dark` | 投影原料 |
| `--cst-elevation-card` / `-button` | `palette-shadow-card` / `-shadow-btn` | 卡片 / 按钮投影（鸿蒙侧镜像字段） |

**输入族（field surface，唯一输入表面语言）**

| token | 值 | 用途 |
|---|---|---|
| `--cst-field-bg` / `-bg-disabled` | surface-primary / surface-subtle | 常态 / 禁用底 |
| `--cst-field-border` / `-border-hover` / `-border-focus` | border-default / border-strong / accent-default | 边界三态 |
| `--cst-field-ring` / `-ring-danger` | accent 22% / danger 24% `color-mix(... transparent)` | 焦点 / 错误柔光环 |

**导航、交互与浮层（组件情境）**

| token | 值 / 来源 | 用途 |
|---|---|---|
| `--cst-navigation-content` / `-content-active` / `-active-indicator` | text-secondary / accent-default / accent-default | 桌面侧栏文字 / 选中 / 指示条 |
| `--cst-surface-navigation-hover` / `-active` / `-active-hover` | text-primary 12% / accent 26% / accent 34% 混 surface-page | 侧栏三态底（灰度可分） |
| `--cst-surface-toggle-off` / `-thumb` | 主题层（灰 22% / panel） | 开关轨道关态 / 滑块 |
| `--cst-elevation-toggle-on` / `-off` / `-thumb` | 主题层 | 开关 inset 与滑块投影 |
| `--cst-surface-card-menu` / `--cst-border-card-menu` | surface-primary / border-subtle | 卡片菜单表面 / 描边 |
| `--cst-elevation-menu` / `-modal` / `-sheet` / `-side-sheet` / `-control-thumb` / `-fab` / `-nav` / `-rail-hover` / `-profile-avatar` | 主题层（见 §8） | 各浮层/控件级投影（Tailwind `shadow-menu` 等注册） |
| `--cst-surface-nav-glass` / `-border-nav-glass` | panel 90% / accent 18% | 顶栏玻璃与描边 |
| `--cst-surface-map-control` | 主题层 | 地图控件面 |
| `--cst-card-tap-scale` | 1（主题覆写 0.94 / 0.97） | 卡片按压缩放 |
| `--cst-surface-cover-scrim` / `-gallery-scrim` / `-gallery-label` / `-cover-placeholder` / `-card-fallback` / `-card-scrim-top` / `-card-scrim-bottom` | 主题层（渐变 / 占位） | 封面、画廊、整卡图片卡的遮罩与占位 |

> 新增语义 token 的完整同步链：`themes/*.css` 三主题 × 明暗补齐 → `semantic.css` 别名/公式 →
> `theme-registry.css` 注册 Tailwind 类 → `check-theme-parity` / `check-cross-platform-colors` 通过。
> Tailwind 未注册的 utility 会静默失效，注册层不能漏。

### 3.3 鸿蒙侧颜色消费

- 画布：`palette.pageBackground` / `palette.surfaceBackground`（对齐前端
  `--cst-surface-page` / `--cst-surface-primary`）
- **画布契约（克制画廊 / 连续画布）**：应用内只允许一个整层底色 =
  画布 `pageBackground`（对齐前端 `--cst-surface-page`）；外壳各层（手机壳 / 桌面舞台 /
  顶栏 / 页面整底 / header bar 变体）只许用画布色连续铺底。**页面级容器不用卡片外框**：
  桌面壳的侧栏/主面板、手机/平板顶栏与页头都不得出现包裹整页的描边、圆角、白底或硬分隔线
  （会形成屏幕边缘的「框线」观感；详情页 DetailHeader 外层恒为 `bg-transparent`，禁止附加通栏底色或毛玻璃白带，交互胶囊悬浮于画布之上，与瞬间/群组详情一致）。层级只由**内容单元软岛**
  （`surfaceIsland` 表面 + 轻投影 + 圆角 20，**无 1px 描边**，如设置分组卡、通知列表岛、rail 卡片）
  与留白表达；内容单元格本身不属于「页面框」，可保留。软岛语言见 §19；分组小标用
  `textSecondary` eyebrow（`SettingsGroupLabel`）。
- 表面：`palette.surfaceSubtle` / `palette.surfaceElevated` / `palette.chatBackground`
- 文字：`palette.textPrimary` / `palette.textSecondary` / `palette.textTertiary`
- 品牌：`palette.accentBackground`（forest `#39714F`，与前端
  `--cst-palette-accent-primary`（`--cst-accent-default`）一致）
- 语义：`palette.success` / `palette.warning` / `palette.error` / `palette.info`
- 地图标记：`palette.mapUser / mapGroup / mapMoment`（随主题/明暗变）；状态淡底：
  `palette.dangerSoft / warningSoft`（对齐前端 `--cst-status-danger/warning-bg` 12~14%
  预混档位；success/info 淡底用 `getSoftStatusBackground('success' | 'info')` 派生）
- 主题解析：`AppThemeManager.getPaletteById(resolvedThemeId)` → `@StorageProp(KEY_RESOLVED_THEME_ID)`
- 明暗三档：`DarkModePreference`（`auto` 跟随系统 / `light` 手动浅色 / `dark` 手动深色，
  缺省 `auto`，与 `ThemeMode` 主题族正交，持久化键 `dark_mode_preference`，单通道经
  `AppThemeManager.setDarkModePreference` 读写）；`auto` 时解析看系统 `isDarkMode`，
  `manual` 时直接取偏好（`onSystemColorModeChange` 仅 `auto` 时改变结果）；
  `setColorMode` 对应关系：`dark`→`COLOR_MODE_DARK` / `light`→`COLOR_MODE_LIGHT` /
  `auto`→`COLOR_MODE_NOT_SET`（`EntryAbility` 三处 + 设置页切换处统一按此三选一，
  `settings_dark_forced=true` 的存量偏好按 `dark` 读取）
- 不存在主题族的存量偏好一律映射 vibrant 对应明暗

### 3.4 主题层（`themes/*.css`，三主题 × 明暗）

主题层是主题相关 token 的唯一真值：每主题 `light` 与 `dark` 两组变量，集合必须完全一致
（`check-theme-parity.mjs` 以 vibrant 为锚逐主题比对；缺变量会静默回落到其他主题/模式，造成跨主题漂移）。

| 分组 | 代表 token | 用途 |
|---|---|---|
| 色板原料 | `--cst-palette-*`（37 项：主色 / border / hover / opacity / shadow / overlay / chat / glow） | 一切语义 token 的输入 |
| 画布与表面 | `--cst-surface-primary` / `-page` / `-chat` / `-island` / `-subtle` / `-panel` / `-raised` / `-overlay` / `-overlay-strong` | 层级与遮罩 |
| 压图与反色 | `--cst-content-inverse`（三档） / `--cst-content-on-marker` / `--cst-text-on-status` / `--cst-surface-glass-badge` / `--cst-surface-media-scrim-*` / `--cst-surface-cover-*` / `--cst-surface-card-scrim-*` / `--cst-surface-card-fallback` / `--cst-surface-gallery-*` | 图片上的文字、徽章与占位 |
| 交互与导航 | `--cst-surface-navigation-*` / `--cst-surface-hover` / `--cst-surface-toggle-*` / `--cst-surface-card-menu` / `--cst-border-card-menu` / `--cst-card-tap-scale` / `--cst-hover-lift` / `--cst-hover-image-zoom` | 交互态 |
| 投影与玻璃 | `--cst-elevation-*`（menu / modal / sheet / side-sheet / control-thumb / fab / nav / rail-hover / profile-avatar / toggle-*） / `--cst-surface-nav-glass` / `--cst-border-nav-glass` | 层级与毛玻璃 |
| 氛围画布 | `--cst-ambient-*`（glow1/2、sky、line、dot、sun、ink1–3、emblem、horizon、vignette） | §16 装饰层 |
| 状态与标记 | `--cst-status-*-bg` / `-border` / `--cst-marker-*-bg` / `-border` / `-glow` / `--cst-accent-soft-bg` / `--cst-accent-border-soft` | 淡底、边框、光晕 |

> 改主题色只改 `themes/*.css` 与 `ThemeConstants` 两端同源值，不要改 `semantic.css` 的派生公式。
> 覆盖范围：`ColorConfig` 中参与跨端映射的字段（14 组 palette 映射 + 合成色规则）由 `check-cross-platform-colors.mjs` 逐值校验，
> 其余装饰字段（glow / tabBar 等）与 `ThemeShadows` 为手工镜像；
> `AmbientColorConfig`（氛围画布 12 字段，预合成 `#AARRGGBB`）由 `check-cross-platform-ambient.mjs` 逐值校验。

---

## 4. 字体系统

### 4.1 字号刻度（`--text-*` ↔ `FONT.SIZE`）

| 档 | 前端（px @16 基准） | 鸿蒙 `FONT.SIZE` | 用途 |
|---|---|---|---|
| xxs | `10`（`--text-xxs` 0.625rem） | `XS2` | 极小 uppercase 标签（eyebrow、分组标） |
| xs | `11`（`--text-xs` 0.6875rem） | `XS` | 辅助文字（副标题、计数、时间戳） |
| sm | `12`（`--text-sm` 0.75rem） | `SM` | 正文 |
| md | `14`（`--text-md` 0.875rem） | `MD` | 强调正文（昵称、按钮主标） |
| lg | `18`（Tailwind 默认 `text-lg`） | `LG_PLUS` | 小标题 |
| xl | `20`（Tailwind 默认） | `XL` | 标题 |
| 2xl | `24`（Tailwind 默认） | `XXL` | 大标题 |
| 3xl | `30`（Tailwind 默认） | `XXXL` | 强调大数字 / 节标题 |
| 4xl | `36`（Tailwind 默认，前端最大 text 档） | `HERO` | 错误码 / 超大数字 |

> 前端只覆写 xxs/xs/sm/md 四档（`theme-registry.css` @theme），lg 及以上沿用 Tailwind 默认（18/20/24/30/36）。
> **鸿蒙平台适配例外**：`FONT.SIZE.LG = 16` 为 HarmonyOS 标准正文尺寸（web 端正文 12/14 偏小），
> 是唯一多出的档位，两端有意不同，属已登记豁免，不属于漂移；前端无 16px 档，9px `XS_MINUS` 档不存在（最小档 10px）。
> 近邻档（13/15/22/28/32/48/56 等）一律归并到上表档位，禁止引入。
>
> **大屏缩放**：前端字号 token 用 rem 表达（0.625/0.6875/0.75/0.875rem），16px 基准下等价上表 px；
> ≥1920 视口随根字号等比放大（见 §23），鸿蒙端保持设备独立字号。`check-cross-platform-scale.mjs`
> 按 16px 基准把 rem 折算回 px 后与 `FONT.SIZE` 逐档比对（含 lg+ Tailwind 默认档与行高）。

### 4.2 字重 / 行高 / 字距

- 字重：`FONT.WEIGHT.{REGULAR, MEDIUM, SEMIBOLD, BOLD}`（400/500/600/700）。
- **行高（三档口径，按场景选）**：
  - **各字号默认行高（最常用）：`textLineHeight(fontSize)`**，对齐前端 Tailwind 各档
    `text-*` 自带行高。**凡是单行/普通文本，必须显式调用它**——ArkUI Text 不设
    `lineHeight` 时行盒 ≈ 字号 × 1.17，而前端是 1.33~1.56，不设会系统性矮约 15%：

    | 字号 | 行高 | 来源比例 |
    |---|---|---|
    | 10（XS2） | 15 | 1.5（项目新增档走 base 1.5） |
    | 11（XS） | 15 | 16/12 = 1.3333 |
    | 12（SM） | 17 | 20/14 = 1.4286 |
    | 14（MD） | 21 | 1.5 |
    | 16（LG） | 25 | 28/18 ≈ 1.5556 |
    | 18（LG_PLUS） | 28 | 28/18 ≈ 1.5556 |
    | 20（XL） | 28 | 28/20 = 1.4 |
    | 24（XXL） | 32 | 32/24 = 1.3333 |
    | 30（XXXL） | 36 | 36/30 = 1.2 |

    该映射由 `check-cross-platform-scale.mjs` 逐档断言，CI 漂移即拦截。

  - 比例制 `FONT.LINE_RATIO.{DEFAULT 1.5, CARD 1.7, COPY 1.75}` + `lineHeightFor(fontSize, ratio)`，
    对齐前端 `theme-registry.css` 的 `--leading-*` 标尺（`card=1.7`）。
    **多行长文正文用它**（覆盖默认行高）：

    | 语义 | 比例 | 适用 |
    |---|---|---|
    | `DEFAULT` | 1.5 | 基线（对齐前端 base.css 元素级 1.5） |
    | `CARD` | 1.7 | 卡片/聊天气泡正文（前端 `--leading-card`） |
    | `COPY` | 1.75 | 长文正文、简介、公告、动态正文（前端 `--leading-copy`） |

  - 绝对档 `FONT.LINE_HEIGHT.{XS 18, SM 20, MD 22, LG 24, XL 26}` 只用于**单行紧凑**场景。
    ⚠️ 绝对档与字号是解耦的：把 `LINE_HEIGHT.SM`(20) 用在 `SIZE.LG`(16) 上比值只有 1.25。
    误用会让比值散落 1.57~1.83；多行长文一律走上表比例制。
- 字距：`FONT.LETTER_SPACING.{EYEBROW 1.4, LABEL 1.2, FIELD 1.28, ICON_LABEL 0.6, TITLE -0.12}`
  （对齐前端 `--tracking-{eyebrow 0.14em, label 0.12em, field 0.08em, title -0.01em}` 刻度，鸿蒙按 10px 基准折算）。
- 前端行高/字距注册：`--tracking-*`（上四档）、`--leading-{card 1.7, copy 1.75}`；其余行高走 Tailwind 默认
  （`leading-none` 1 / `leading-snug` 1.375 / `leading-relaxed` 1.625 等）。禁止 `tracking-[…]` / `leading-[…]` arbitrary 类。

---

## 5. 间距（`SPACING`）

4px 网格 + 微步，**不强制收敛**（间距的 2px 差异是布局设计的一部分）：

`HALF 2` / `THREE 3` / `XS 4` / `SM2 6` / `SM 8` / `MD2 10` / `MD 12` / `MD3 14` / `LG 16` / `LG_PLUS 18` / `LG2 20` / `XL 24` / `XXL 32`

禁止组件内 `padding({left:16})` 这类裸数——用 `SPACING.LG`。1vp 级微距（top:1 等）与 0 值允许保留（对齐前端间距不强制收敛原则）。

前端对应：间距用 Tailwind 默认 4px 阶梯（`p-1`/`p-2`/`gap-3` 等）；页面级另有 `--cst-page-gutter`（§5.1）
与布局 token（`--cst-workspace-*`/`--cst-context-rail-*`/`--cst-cover-height-*` 等，见 §12），
并通过 `@theme` 注册 `spacing-cover-height{,-sm,-lg}`/`spacing-page-gutter` 类。

### 5.1 页面级横向 gutter（唯一来源，强制）

**横向留白只能由 `PageShellContainer` 提供**，页面内不得再为自己的正文写
`left`/`right` padding：

| 断点 | 鸿蒙 `PAGE_SHELL` gutter | 前端 `--cst-page-gutter` |
|---|---|---|
| 手机（< 600） | `GUTTER_MOBILE` = 16 | 1rem |
| 平板（600 – 1279） | `GUTTER_TABLET` = 24 | 1.5rem |
| 桌面（≥ 1280） | `GUTTER_DESKTOP` = 28 | 1.75rem |

> 两端同一阈值：鸿蒙 `PageShellContainer` 读 `AppStorage.currentWidthVp` 精确解析（无宽度时回退命名断点），
> 与 web 的 600 / 1280 断点一致；改阈值必须两端同步。

- **禁止**「容器 gutter + 页面自加 padding」双重留白——会叠成 36~40，与只用容器
  gutter 的页面差 16~20vp，各页正文左边缘不在同一条竖线上，滑动时边界跳动。
- **禁止**用 `noHorizontalPadding: true` 关掉容器留白再另写一套值。
- **通栏元素（封面图等）说明**：前端用 `-mx-[var(--cst-page-gutter)]` 抵消 gutter，
  但 **ArkUI 不能照搬**——子元素 `width('100%')` 是相对**父内容盒**解析的，负 margin 只会
  位移不会加宽（实测：群详情封面加左右各 −20vp 后仍是 x=70..1186 / 屏宽 1256）。
  当前约定：封面/hero 一律与下方内容同宽（收束式 hero），不做通栏；确需通栏须把该页整页改为
  `noHorizontalPadding: true` + 各区块自加 gutter（已登记例外），或改用 `calc` 宽度。
- `SystemPageHeader` 只带纵向节奏、**不带横向 padding**（自加 20 与
  容器 gutter 叠加会让页头标题比正文多缩进一截）。

### 5.2 页面纵向节奏（强制）

| 语义 | 值 | 用途 |
|---|---|---|
| `PAGE_SHELL.PAGE_TOP_GAP` | 16 | 页头与首个分区 |
| `PAGE_SHELL.SECTION_GAP` | 24 | 相邻分区之间（用**底距**，含首块） |
| `PAGE_SHELL.SECTION_INNER_GAP` | 12 | 分区标题与其内容 |

- 分区之间必须**明显大于**「分区标题→其内容」（24 : 12 = 2:1），标题才在视觉上归属
  下方内容；否则标题飘在两块之间，整页读不出分组。
- 分区间距统一用**底距**且**每个分区都要有**（含首个分区）：用「上距」或漏给首块加
  底距时，标题上方间距会随各卡片自身 padding 变化，同类标题间距不一致。

---

## 6. 圆角（`--radius-*` / `RADIUS` + `SHAPE`）

| 档 | 值 | 用途 |
|---|---|---|
| sm | `12` | 小控件、卡片角标、pill |
| md | `16` | 按钮、输入框、中等卡片 |
| lg | `20` | 面板、列表容器、大卡片 |
| xl | `24` | 封面、hero 区、feature 卡 |
| full | `999` | 圆/胶囊 |
| sheet | `28` | 底部弹层专用大圆角 |

前端实现：`--radius-{sm,md,lg,xl,full}` 映射 `--cst-shape-{sm,md,lg,xl,full}`（shared.css），`--radius-sheet` = 1.75rem；`rounded-*` 类由 `@theme` 注册，禁止 arbitrary 圆角。

语义形状（`SHAPE`）：`CARD 20 / HERO_CARD 24 / COMPACT_CARD 16 / PREVIEW_CARD 20 / LIST_ITEM 20 / ICON_TILE 12 / INPUT 16 / BUTTON FULL / CHIP FULL / TAB_BAR 24 / SHEET_TOP_RADIUS 28`，按前端语义用法对齐（hero/feature 24、主按钮胶囊 FULL、弹层顶部 28）。近邻档 8/18/22 不存在，禁止引入。**输入族两派**：搜索输入 = 胶囊（前端 `rounded-full`，鸿蒙 `SearchPageHeader` 用 `SHAPE.BUTTON`）；表单输入保持 `INPUT 16`（鸿蒙 `SystemInput` 不变）。

---

## 7. 边框宽度（`--cst-border-width-*` / `BORDER_WIDTH`）

前端：`--cst-border-width-{hairline 1px, regular 1.5px, strong 2px, heavy 3px}`（shared.css），`@theme` 注册 `border-width-*`——`border` 默认 = hairline、`border-2` = strong。
鸿蒙：`BORDER_WIDTH.{SUBPIXEL 0.5, HAIRLINE 1, REGULAR 1.5, STRONG 2, HEAVY 3}`。
禁止 `.border({ width: 1 })` / `border-[1.5px]` 这类裸数，一律走 token。

---

## 8. 阴影与层级

### 8.1 层级（z-index，`shared.css`）

| token | 值 | 用途 |
|---|---|---|
| `--cst-z-dropdown` | 10 | 下拉 / 菜单 |
| `--cst-z-sticky` | 20 | 页内 sticky 头 / 筛选条（滚动内容之上、浮层之下） |
| `--cst-z-chrome` | 30 | 桌面顶栏 / 侧栏（页内 sticky 之上、遮罩之下） |
| `--cst-z-overlay` | 40 | 遮罩 |
| `--cst-z-modal` | 50 | 模态 |
| `--cst-z-toast` | 60 | Toast |

### 8.2 投影（前端 token）

原料：`--cst-elevation-shadow-light` / `-dark`（来自 `palette-shadow-*`）；组件级投影值由 `themes/*.css` 预定义。

| 刻度 | token | 值 / 用途 |
|---|---|---|
| raised | `--shadow-raised-sm` / `-md` / `-lg` | 0/1/3 · 0/2/8 · 0/6/16：控件/按钮 → 卡片 → 大卡片/封面 |
| inset | `--shadow-inset-sm` / `-md` | 双向凹陷：输入聚焦、选中态 |
| glow（accent） | `--shadow-glow-sm` / `-md` / `-lg` | 中性投影 + `--cst-glow-accent`，accent 实底控件 hover/按压 |
| glow（danger） | `--shadow-danger-glow-sm` / `-md` | 危险操作 hover 抬档 |
| toggle | `--shadow-toggle-on` / `-off` / `-thumb` | 开关 inset 与滑块 |
| overlay | `--shadow-menu` / `-modal` / `-sheet` / `-side-sheet` / `-control-thumb` | 浮层统一投影（菜单/Toast/模态/上下 sheet/侧滑面板） |

- `@theme` 把上表注册为 `shadow-*` 类；组件禁止 `shadow-[var(--cst-elevation-*)]` arbitrary 写法。
- 组件级：`--cst-elevation-{fab, nav, rail-hover, profile-avatar, modal, sheet, side-sheet, menu, control-thumb}`，值在 `themes/*.css`。
- 悬停/按压缩放另走 `--cst-hover-lift`（2px）与 `--cst-hover-image-zoom`（1.045）。

### 8.3 鸿蒙投影契约（`SHADOW` / `ThemeShadowRegistry`）

扁平设计默认 `SHADOW.*` 全零；需要投影时用主题注册表
`AppThemeManager.getPaletteById(...).{surfaceShadow, elevatedShadow, controlShadow, tabBarShadow}`
（真值在 `ThemeConstants` 的 ThemeShadowRegistry），禁止组件内 `.shadow({ radius: N, ... })` 手写。

**全端统一中性 raised 投影语言**：主按钮/主图标钮/加入·关注·发布等 accent 实底控件静态用
`surfaceShadow`、hover/按压抬档 `elevatedShadow`，禁止 `accentShadow` 彩色光晕
（`accentShadow` 字段已随 glow 体系删除，对齐前端 ui/button
`shadow-raised-sm hover:shadow-raised-md` 契约）。

### 8.4 跨端投影契约速查

| 场景 | 静态 | hover/按压 |
|---|---|---|
| 主按钮（SystemButton primary/danger） | `surfaceShadow` | `elevatedShadow` |
| accent 实底图标钮（SurfaceIconButton primary） | `surfaceShadow` | `elevatedShadow` |
| 未加入群组/未关注/发布等可点 accent 控件 | `surfaceShadow` | `elevatedShadow` |
| FAB | `elevatedShadow`（对齐前端 `--cst-elevation-fab` 中性投影） | — |
| 内容岛/统计卡/徽章/**列表分组容器**（ListGroup / SettingsSectionCard） | `surfaceShadow`（软岛用 `shadow-xs`） | — |
| 列表行（分组内的行） | **无投影**（层级由分组容器 + 发丝分隔线承担） | — |
| 选中胶囊（TabsPillGroup） | `controlShadow`（hover 增量走底色） | — |
| 输入框静态 | 字段表面色（surface-primary）+ 1px border-default，无投影（纸面主题一致） | hover：边框升 border-strong |
| 输入框聚焦 | accent 1px 边界 + 内缘 1px（视觉 2px）+ 3px accent 柔光环 | — |
| 菜单/浮层（NotificationCenter、sheet 内浮钮） | `controlShadow` | — |
| 静态内容卡（登录卡/表单区块/提示 banner/空态卡/骨架卡） | `surfaceShadow` | — |

> **档位纪律**：`elevatedShadow`（radius 24~44 / offsetY 8~12）**只允许**
> 出现在 FAB、hover/按压抬档、以及桌面外壳（DesktopLayout 侧栏/主面板）三类场景。
> 「白卡/hero 卡/列表行/统计卡」的静态投影一律 `surfaceShadow`——取 `elevatedShadow`
> 会让所有页面内容都像悬浮卡片。

---

## 9. 模糊（`--cst-blur-*`）

`--cst-blur-sm 8px`（轻徽章/浮层）/ `--cst-blur-md 16px`（sticky header/overlay）/ `--cst-blur-lg 24px`（底部导航/tablet dock）；三档强制收敛，值在 `shared.css`（可按主题覆写），禁止组件内联手写 blur 半径。

鸿蒙映射系统 `BlurStyle` 枚举（平台原生，非数值 token）：`BlurStyle.Thin` ↔ sm、
`BlurStyle.Regular` ↔ md、`BlurStyle.Thick` ↔ lg（当前 SDK 的 BlurStyle 枚举无
`Medium` 档，强档一律用 `Thick`）。

---

## 10. 动效与交互反馈

### 10.1 时长与曲线（禁用裸 `duration: N` / 裸 curve）

**前端 token（`shared.css` 定义，`@theme` 注册）**

| token | 值 | 用途 |
|---|---|---|
| `--cst-motion-instant` | 100ms | 即时状态切换 |
| `--cst-motion-rapid` | 150ms | 快速反馈 / 退场 |
| `--cst-motion-snappy` | 180ms | 紧凑过渡 |
| `--cst-motion-standard` | 220ms | 状态 pop / 默认过渡 |
| `--cst-motion-relaxed` | 260ms | 宽松面板过渡 |
| `--cst-motion-spring` | 300ms | 弹性位移 |
| `--cst-motion-slow` | 400ms | 大面板 / 弹层 |
| `--cst-motion-gentle` | 500ms | 缓慢强调 |
| `--cst-motion-spinner` | 700ms | 加载旋转 |
| `--cst-motion-pulse` | 2400ms | 占位 / 关注脉冲 |
| `--cst-motion-reduced` | 1400ms | reduced-motion 降级周期 |
| `--cst-motion-skeleton` | 1500ms | 骨架屏 shimmer 循环（仅骨架屏；对齐鸿蒙 `DURATION_SLOW 1500`） |
| `--cst-motion-ambient` / `-ambient-slow` | 24000ms / 36000ms | 氛围画布超长循环（仅装饰层） |
| `--cst-ease-standard` | `cubic-bezier(.22,1,.36,1)` | 标准曲线（`ease-standard`） |
| `--cst-ease-bounce` | `cubic-bezier(.34,1.56,.64,1)` | 弹簧 overshoot（`ease-bounce`） |

- 页面进场/内容淡入：`utilities.css` 的 `--cst-page-enter-duration` 120ms、`--cst-content-fade-duration` 110ms；
  JS 侧 `lib/motion.ts` 的 `MOTION.{enter 120, contentFade 110, exit 150, pop 220}` 与之同源，改一处必须同步。
- `@theme` 注册：`--transition-duration-{instant,rapid,snappy,standard,relaxed,spring,slow,gentle}`、`--animate-fade-in`。

**鸿蒙 `ANIMATION`**：`DURATION_FAST 150` / `DURATION_SHORT 200` / `DURATION_MEDIUM 300` /
`DURATION_CONTENT_SWITCH 400` / `DURATION_SHEET 320`（底部弹层统一）/ `DURATION_SLOW 1500`（脉冲/shimmer）/
`DURATION_STAGGER_DELAY 150`；曲线 `CURVE_EASE_IN_OUT` / `CURVE_EASE_OUT` / `CURVE_FAST_OUT_SLOW_IN` / `CURVE_SMOOTH`。
骨架屏脉冲两端同源（`--cst-motion-skeleton: 1500ms` ↔ `DURATION_SLOW 1500`），勿单独改一端。
Toast/震动等行为时长保留字面量并注释（非视觉 token 范畴）。

### 10.2 交互反馈（`INTERACTION`，禁用裸 `scale(...)` / 裸 curve）

按压强度用 `SCALE_PRESSED_*`（对齐前端 `tap-*`/`hover-btn` 契约，数值为 HarmonyOS 平台适配档）：

- 主按钮按压 0.95 ↔ 前端 `hover-btn` tap 0.94；行内图标 0.90 ↔ `scale-btn` tap 0.94
- 卡片 0.97 ↔ `tap-97`；瀑布流卡 0.96 ↔ `tap-96`（前端按页微调档同理）
- 输入行 0.98 ↔ `tap-98`；胶囊/筛选 0.94 ↔ `tap-94`
- hover 抬升用 `HOVER_ELEVATION`（ArkUI elevation，对应前端 fine-pointer hover 缩放），
  弹簧动效用 `SPRING_RESPONSE_*` / `SPRING_DAMPING_*`（对应前端 `--cst-ease-bounce`）

---

## 11. 透明度

前端 `--cst-opacity-*`（`semantic.css` 映射 `--cst-palette-opacity-*`，`@theme` 生成 `opacity-*` 类）：

| token | 默认值 | 用途 |
|---|---|---|
| `--cst-opacity-none` | 0 | 完全透明 |
| `--cst-opacity-faint` | 0.30 | 极弱提示 |
| `--cst-opacity-subtle` | 0.40 | 弱化元素 |
| `--cst-opacity-muted` | 0.50 | 次要文字 / 图标 |
| `--cst-opacity-dim` | 0.60 | 弱化控件 |
| `--cst-opacity-soft` | 0.70 | 软提示 |
| `--cst-opacity-emphasis` | 0.80 | 强调 / 次级 |
| `--cst-opacity-tap` | 0.84 | 按压反馈 |
| `--cst-opacity-hover` | 0.90 | 悬停反馈 |
| `--cst-opacity-full` | 1 | 不透明 |
| `--cst-opacity-active` | 0.13 | 选中叠加（mix 档） |
| `--cst-opacity-glow` | 按主题 / 明暗覆写 | 光晕与主题强调档（见 `themes/*.css`） |
| `--cst-opacity-watermark` | 0.08 | 无图卡片水印（跨端一致） |

- 组件只消费 `opacity-*` 语义类，禁止内联 opacity 字面量（AGENTS.md UI 原则 #4）。
- 鸿蒙 `OPACITY`：`DISABLED 0.38` / `PRESSED 0.72` / `LOADING 0.58` / `SUBTLE 0.85` / `MUTED 0.5` / `EMPHASIZED 0.8`；
  FADED 0.4 档不存在。

---

## 12. 组件与布局尺寸（禁止裸 `width(N)` / `height(N)`）

### 12.1 前端布局与尺寸 token（`layout.css` / `shared.css` / `@theme`）

| 分组 | token | 值 | 用途 |
|---|---|---|---|
| 内容栏 | `--cst-content-breath` | `clamp(80rem, 60vw, 100rem)` | 呼吸式页面（首页/发现） |
| | `--cst-content-feed` | `clamp(64rem, 52vw, 80rem)` | 信息流 |
| | `--cst-content-detail` | `clamp(58rem, 50vw, 70rem)` | 详情页 |
| | `--cst-content-form` | `clamp(56rem, 42vw, 64rem)` | 表单页 |
| | `--cst-content-form-narrow` | `clamp(48rem, 36vw, 56rem)` | 窄表单 |
| | `--cst-page-content-max` | 100% | `ContentColumn` 默认流式宽度 |
| 工作区 / 侧栏 | `--cst-workspace-gap` | `clamp(0.875rem, 1.5vw, 1.25rem)` | 桌面工作区间距 |
| | `--cst-workspace-inset` | `0.75rem` | 浮动卡片工作区距舞台内缩 |
| | `--cst-context-rail-width` | `clamp(18rem, 20vw, 21rem)` | 右 rail 宽 |
| | `--cst-context-rail-gap` | `clamp(1rem, 1.75vw, 1.5rem)` | 主区与 rail 间距 |
| | `--cst-context-rail-inset` | `1rem` | rail 内边距 |
| | `--cst-desktop-sidebar-width` / `-collapsed` | `14rem` / `4.5rem`（224 / 72px） | 桌面侧栏展开 / 折叠 |
| | `--cst-desktop-header-height` | `3.25rem`（52px） | 桌面顶栏高 |
| | `--cst-tablet-dock-max` | `42.5rem`（680px） | 平板浮动 dock 最大宽 |
| 移动壳 | `--cst-mobile-rail-width` / `-item` | `4rem` / `3.25rem`（64 / 52px） | 横屏左侧 rail |
| | `--cst-bottom-nav-pad-top` / `-bottom` | `0.5rem` / `1.25rem`（8 / 20px） | 竖屏底栏留白（与 safe-area 取大） |
| 封面 | `--cst-cover-height-sm` | `clamp(7.5rem, 18vw, 10rem)` | 编辑资料小封面 |
| | `--cst-cover-height` | `clamp(13.75rem, 28vw, 21.25rem)` | 群 / 个人封面 |
| | `--cst-cover-height-lg` | `clamp(13.75rem, 38vw, 23.75rem)` | 动态详情大封面 |
| 弹层 | `--cst-sheet-height` / `-max-height` | `78vh` / `700px` | center 模态弹层 |
| | `--cst-sheet-height-lg` / `-xl` / `-2xl` | `85%` / `90%` / `92%` | 底部 sheet 三档 |
| 聊天 | `--cst-chat-bubble-max-width` | `min(75%, 40rem)` | 气泡最大宽度（窄屏占比 75%，宽屏封顶 640px） |
| 大屏 | `--cst-ui-scale` | 1（JS 同步） | 数值型内联尺寸缩放（§23） |
| 注册类 | `--spacing-page-gutter` / `--spacing-cover-height{,-sm,-lg}` | 同上 | `@theme` 生成对应 spacing 类，消除 arbitrary 写法 |

> 布局 token 统一 rem 表达；≥1920 视口随根字号等比放大（§23）。页面正文宽度用 `ContentColumn` 档位，
> 禁止页面自设 `max-w-*`（例外见 §21.13）。

### 12.2 鸿蒙尺寸常量（`DesignConstants`）

- 头像 `AVATAR_SIZE.{XXS 22, XS 32, SMALL 40, SMALL_PLUS 44, MEDIUM 48, PROFILE 68, LARGE 80, XLARGE 96}`；
  按钮 `BUTTON_HEIGHT 42` / `BUTTON_HEIGHT_LG 48` / `ACTION_BUTTON_HEIGHT 28` /
  `ACTION_BUTTON_HEIGHT_MD 40` / `ACTION_BUTTON_HEIGHT_LG 34` / `SHEET_ACTION_HEIGHT 44` /
  `FILTER_CHIP_HEIGHT 30` / `LOGOUT_BUTTON_HEIGHT 50` / `PASSWORD_INPUT_HEIGHT 46`
- 输入框 `INPUT_HEIGHT 42`（通用表单）/ `SEARCH_INPUT_HEIGHT 39`（搜索框）。输入族
  两端同口径：搜索框 = 胶囊 + 39px（前端 py-2(8) + text-md 行高 21 + border 2 = 39），
  图标 16、字号 14、图标与文字间距 `gap-2.5`(10)；表单输入 = 16 圆角 + 43px（前端 px-4 py-2.5 +
  行高 21 + border 2 = 43，≈ INPUT_HEIGHT 42）
- 图标 `ICON_SIZE_2XS 14` / `ICON_SIZE_XS 16` / `ICON_SIZE_SM 20` / `ICON_SIZE_SM_PLUS 22` /
  `ICON_SIZE_MD 24` / `ICON_SIZE_LG 32` / `ICON_SIZE_EMPTY 48`（与 `FONT.SIZE` 解耦的独立
  ICON 刻度，图标尺寸一律走此档，不用字号 token）/ `STATUS_ICON_SIZE 28` /
  `LOADING_ICON_SIZE 20` / `LOADING_GLYPH_SIZE 46` / `BRAND_ICON_SIZE 38`；
  容器 `EMPTY_ICON_SIZE 64` / `LOADING_CONTAINER_SIZE 72` / `BANNER_HEIGHT 140` /
  `SKELETON_CARD_HEIGHT 100` / `PLACEHOLDER_AREA_HEIGHT 120` / `IMAGE_PLACEHOLDER_HEIGHT 110` /
  `SHARE_CARD_HEIGHT 130` / `LOG_AREA_HEIGHT 300` / `PROGRESS_STROKE_WIDTH 4`
- 布局 `LAYOUT.{DESKTOP_HEADER_HEIGHT 52, DESKTOP_SIDEBAR_WIDTH 224, RIGHT_SIDEBAR_WIDTH 220,
  MOBILE_BOTTOM_NAV_HEIGHT 56, TABLET_NAV_MAX_WIDTH 680, FAB_SIZE 48, ICON_BUTTON_SIZE 36,
  LABEL_WIDTH 60, HOME_COLUMN_WIDTH 420, FILTER_PANEL_WIDTH 180}`（无 `GLOW_BLOB_*` 光晕尺寸；
  完整字段见 `DesignConstants.ets` 的 `LAYOUT` / `COMPONENT`）
- 其他布局尺寸：`BACK_BUTTON_SIZE 42`（`DesignConstants.ets LAYOUT.BACK_BUTTON_SIZE=42`，与前端
  「页面级返回键统一 42」一致）/ `MIN_HIT_TARGET 44`（触控热区底线，见 skill
  `mobile-ui-conventions`：官方底线 ≥40vp、推荐 46）/ `UNREAD_BADGE_SIZE 20` /
  `COVER_HEIGHT 220` / `NICKNAME_MAX_WIDTH 56` / `LOGIN_CARD_MAX_WIDTH 400`
- **页面横向 gutter 与纵向节奏**：`PAGE_SHELL.{GUTTER_MOBILE 16, GUTTER_TABLET 24,
  GUTTER_DESKTOP 28}` 与 `PAGE_SHELL.{SECTION_GAP 24, SECTION_INNER_GAP 12,
  PAGE_TOP_GAP 16}` —— **语义与强制规则见第 5.1 / 5.2 节**，此处仅登记存在，
  数值不重复维护（两处重复必漂移）
- 聊天 `CHAT.{BUBBLE_PADDING_H 16, BUBBLE_PADDING_V 10, IMAGE_BUBBLE_PADDING_H 14,
  IMAGE_BUBBLE_PADDING_V 6, IMAGE_MAX_WIDTH 200, ...}`
- 页面内容栏档位（`PAGE_SHELL`）：`SHELL_FORM_MAX 560` / `SHELL_NARROW_MAX 720` /
  `SHELL_DEFAULT_MAX 1180` / `SHELL_WIDE_MAX 1320`；feed：`FEED_NARROW_MAX 640` / `FEED_MEDIUM_MAX 960`。
  前端对应为 `ContentColumn` 的 `maxWidth` 档 + `--cst-content-*` 流式宽度（§12.1），无 `.page-shell` 常量类

---

## 13. 组件原语与跨端映射

双端共享组件映射（新增/修改组件时先查此表，避免两端各自发明同类组件）。

> **本表覆盖范围**：**业务组件**（有明确跨端对应物的）。基础原语
> （`SystemButton` / `SystemToggle` / `SystemTextArea` / `SystemBadge` /
> `SurfaceIconButton` / `Pressable` / `SoftDivider` / `LucideIcon` / `SystemIcon` /
> `NativeMapComponent`）登记在 skill `hotspot-harmony-ui` 的「组件原语强制」章节，
> 不重复列在这里。

| 前端（Svelte） | 鸿蒙（ArkTS） | 说明 |
|---|---|---|
| `layout/page/PageHeader` | `SystemPageHeader` / `SearchPageHeader` / `ChatPageHeader` | 页面头部（含 below/搜索/聊天空态） |
| `StandardPageLayout` / `SplitPageLayout` / `ContentColumn` | `PageShellContainer` | 页面内容栏档位（form/narrow/default/wide/feed） |
| `MainLayout`（desktop/mobile/tablet） | `layout/DesktopLayout` / `PhoneLayout` / `ResponsiveContainer` | 设备外壳分派 |
| `EmptyState` | `EmptyState` | 空态（同名同契约） |
| `shared/Avatar` | `SystemAvatar` | 头像（尺寸/形状/状态点） |
| `forms/SearchInput` | `SystemInput`（通用单行输入）/ `SearchPageHeader`（页头内联搜索） | 搜索输入（搜索两端统一胶囊 39px / 字号 14 / 图标 16，表单 `SystemInput` / 前端 `forms/Input` 仍 16 圆角） |
| `navigation/TabBar` / `NavTab` / `RailIcon` | `TabsPillGroup`（默认等宽分段；`freeWidth` 形态对齐前端胶囊行，`TabItem.count` 对齐角标） / `PhoneLayout` 底部导航 | 分类/导航 Tab |
| `list/ListContainer`（bordered / embedded） | `components/ListGroup`（`bordered` 开关）+ `ListItemGroup` | 列表**分组容器**。开放页面 `bordered` = 内容软岛（无 1px 描边 + `surfaceIsland` + `surfaceShadow`/`shadow-xs` + 圆角 20 + 行 gap、行透明；无行间发丝线），见 §19；embedded 面板形态仍保留行分隔线。行本身禁止再叠边框/圆角/投影 |
| `list/ListItem` | `SettingsEntryRow` / `SettingsToggleRow` / `GroupMemberRow` 等行组件 | 列表行：只写 padding + 按压反馈，不写背景/圆角/投影 |
| `overlays/SheetShell` / `ui/dialog` | `SystemDialog` / `bindSheet` | 弹层/抽屉 |
| `forms/TagPill` | `TabsPillGroup` chip / `FILTER_CHIP_HEIGHT` | 标签/筛选胶囊 |
| `data-display/EmptyState` | `EmptyState` | 空列表态（同名同契约；`fillMode` 对齐前端 fill 模式） |
| `data-display/ListEnd` | `components/ListEnd` | **分页到底收尾提示**（「没有更多了」）。分页列表内容耗尽**必须**渲染，否则用户无法区分「已到底」与「加载卡住」 |
| `data-display/LoadError` | 无独立组件 → `EmptyState` + `actionLabel:'重试'` + `onAction` | 加载失败态。**语义等价**（同为「图标+文案+重试按钮」），故不另造组件；契约要求「失败必须展示错误并提供重试，禁止静默转空态」——实测 9 个失败分支全部满足 |
| `data-display/InlineLoading` | 无独立组件 → `LoadingMoreIndicator`（分页）/ 行内 `LoadingProgress`（局部） | 局部加载态。分页场景一律走 `LoadingMoreIndicator`（含向上分页，如聊天「加载更早消息…」）；行内指示用 `LoadingProgress` |
| `data-display/EmptyState` 整页加载 | `SystemLoadingView` / `Skeleton` / `SkeletonShimmer` | 整页/面板加载与列表骨架 |
| `data-display/Skeleton` / `SkeletonList` / `SkeletonFeed` | `components/states/Skeleton` / `SkeletonShimmer` | **列表/信息流首屏骨架屏**（前端三原语）。几何同源：行内 12、前导圆 36/40/44/48、文本条 14·55% + 11·35%、条间距 8、圆角 12；`role=status + aria-busy`，占位块 `aria-hidden` |
| `layout/ambient/AmbientBackdrop` | `components/layout/AmbientBackdrop` | 氛围画布（三主题世界层，装饰层唯一实现）。鸿蒙为静态 Canvas 一次性绘制，挂载于 `PageShellContainer`；详见第 16 节 |
| `data-display/EmptyState`（`spotlight` 开启） | `EmptyState`（`spotlight`）+ `components/states/EmptySpotlight` | 场景化空态装饰层（超大空态专用）。几何/token 见第 16.6 节，两端同一 spec |
| `data-display/MetricRow` / `HeroStats` | `HomePage.buildHeroStats`（页面内 @Builder） | 统计段（只有首页一处消费，暂不抽独立组件） |
| `routes/moments/shared/MomentCard` | `MomentFeedCard` / `MomentWallCard` | 动态卡片。网格/列表形态均为整卡图片 + 遮罩覆盖信息（3:4；首页 feed 由 `HomeFeedItem` ↔ `MomentFeedCard variant='overlay'` 4:5），spec 见 §17 |
| `auth/LoginPromptSheet` | `LoginPromptView` | 未登录引导 |
| `messages/ChatView` 等 | `ChatPage` / `ChatPageHeader` | 聊天界面 |
| `routes/(app)/nearby/+page.svelte`（附近聚合） | `features/nearby/pages/NearbyPage` | 附近聚合页（tab 式：附近的人/附近群组/附近瞬间三分区，各分区拉取并渲染内容）。手机/平板底栏「附近」tab |
| `data-display/NearbyImageCard` | `components/NearbyImageCard` | 图片优先卡。图片铺满整卡 + 遮罩覆盖信息（3:4），徽章/操作压在图上（footer 覆盖、白字/白图标、对比度硬约束）；`horizontal` 横排两段式保留为列表场景例外。`HomeFeedItem` card ↔ 鸿蒙 `MomentFeedCard variant='overlay'`（4:5）。spec 见 §17 |
| `routes/(app)/notify/+page.svelte` + `notify/shared/NotifyRow` | `components/NotificationCenter` | 通知中心（列表 + 行） |
| `shared/ConnectionBanner`（近似） | `components/NotificationBanner` | 顶部横幅提示（前端对应为连接横幅，语义近似；鸿蒙端承载更广） |

**本端特有组件（前端无对应，属平台适配，已登记）**：
`IntelligentLocationCard`（智能位置卡）、`AnchorEntry`（浏览锚点入口）、
`PerformanceIndicator`（性能指示器）、`SystemCard` / `MasonryCard`（通用卡与瀑布流卡）。
这些不参与跨端同步，改动只需本端自查。

---

## 14. 跨端 token 对齐校验

前端与鸿蒙共享同一套设计 token 刻度，同步由 CI 脚本强制拦截，**改两端任一 token 必须同步改另一端**：

| 校验 | 脚本 | 覆盖范围 |
|---|---|---|
| 主题变量一致性 | `frontend/scripts/check-theme-parity.mjs` | vibrant / forest / inkwash × light/dark 的 `--cst-*` 变量集合必须一致；`semantic.css` 引用的 `--cst-palette-*` 必须在每个主题中定义 |
| 跨端色彩 | `frontend/scripts/check-cross-platform-colors.mjs` | `themes/*.css --cst-palette-*` ↔ `ThemeConstants.ets ColorConfig` 逐值一致（3 主题 × 2 明暗 × 映射组）+ 主题 swatch |
| 跨端刻度 | `frontend/scripts/check-cross-platform-scale.mjs` | 字号 9 档（`--text-*` ↔ `FONT.SIZE`，含 lg+ Tailwind 默认档）+ 圆角 5 档 + 各档行高（`textLineHeight()`）逐值一致；单端加近邻档即拦截。已登记豁免：`FONT.SIZE.LG=16`（HarmonyOS 平台适配档） |
| 类名有效性 | `frontend/scripts/check-utility-classes.mjs` | 源码消费的 token 类名必须已在 `@theme` 注册（含变体前缀与功能类白名单）；未注册的 utility 会静默失效 |
| 跨端氛围画布 | `frontend/scripts/check-cross-platform-ambient.mjs` | web `--cst-ambient-*`（6 组合 × 12 字段）按 `color-mix` 公式预合成 ↔ 鸿蒙 `ThemeConstants.AMBIENT` 逐值一致 |

以上脚本串入 `bun run check:theme`（随 `bun run check` 与 CI 生效），漂移即红；
组件原语同类型一致性另由 `check-component-consistency.mjs` 在 `bun run check` 中校验。

---

**状态色 on-soft 校验**：`python3 harmonyapp/scripts/check_status_parity.py` 断言 web `--cst-text-*-on-soft`（80% 语义色 + 20% text-primary）与鸿蒙 `ColorConfig.*OnSoft` 双端一致，且 3 主题×明暗×4 语义色对最差软底（含 danger /50）≥4.5；改状态色/新增语义色时与 `check:theme` 一起跑。

## 15. 已登记例外与平台适配

- `FONT.SIZE.LG = 16`：HarmonyOS 标准正文尺寸，平台有意差异（跨端刻度脚本豁免项；前端无 16px 档）。
- `BORDER_WIDTH.SUBPIXEL 0.5`：鸿蒙仅有的亚像素档（前端无对应，用于极细分割）。
- `RADIUS.FULL`：前端 9999 / 鸿蒙 999，均为胶囊语义，数值差异不判漂移。
- `XS_MINUS`（9px）档不存在：两端最小字号档为 10px（`XS2`）。
- 明暗切换：鸿蒙 `DarkModePreference` 三档（auto/light/dark），web 端可手动切换；
  主题族（vibrant/forest/inkwash）两端可选集一致。
- 布局几何（width/height/gap 的 px）：非主题风格，不受 token 约束。
- 主题选择器 swatch：静态展示目标主题 accent，引用 token
  （`ThemeConstants.VIBRANT_LIGHT_COLORS.accentBackground` 等，见 `SettingsPage` 主题卡），
  不写字面 hex。对应值 vibrant `#2B5EB2` / forest `#39714F`（见 `ThemeConstants`），须与 themes 同步。
- `figma_neu` 主题：鸿蒙存量兼容项，非双端对齐目标。

---

## 16. 氛围画布（Ambient canvas，双端装饰层）

**目的**：解决桌面宽屏 / 内容稀疏 / 空态页「大面积纯白，看起来空荡荡」的观感问题。
借鉴现代界面环境光设计，为页面赋予通透深邃的「环境光源」纵深，杜绝粗暴生硬的具象背景插画与繁杂剪影。

### 16.1 分层结构

实现（两端同构，共用同一套高质感环境光语言与 token）：

- 前端：`frontend/src/components/layout/ambient/AmbientBackdrop.svelte`（高质感渐变光晕；**动画默认关闭**，
  仅稀疏页面用 `<AmbientBackdrop motion />` 显式开启，见 16.4 与 §21.11）；
- 鸿蒙：`harmonyapp/entry/src/main/ets/components/layout/AmbientBackdrop.ets`（单 Canvas 一次性绘制，静态无逐帧动画）。

| 层 | 实现 | 作用 |
|---|---|---|
| 天光 sky | 渐变 div | 顶部天光，柔化视口上边缘，提供从容阅读引导 |
| 环境微光 glow | 双层径向渐变 div (a/b) | 顶部广角微漫射 + 底部地平线微漫射，提供通透空气感与景深 |
| 地平线雾 horizon | 底部渐隐带 | 地平线轻雾，自然收敛视口底部 |
| 暗角 vignette | 径向渐变 | 边缘收束，聚焦核心正文，提升品质感 |

### 16.2 主题色彩与质感设计（去除具象插画，主色精细化）

三套主题均去除具象插画/天际线/树林/墨山等背景图案，转而通过精心打磨的主题主色与环境光搭配传递风格：

| 主题 | 风格定位与色彩质感 |
|---|---|
| vibrant 星际 | 现代电气质感：浅色高透冷灰蓝晶透画布，深色深邃星海绀蓝，搭配湛蓝/青绿环境天光 |
| forest 清野 | 自然有机质感：浅色温润淡雅晨雾画布，深色沉稳松柏曜黑，搭配林冠暖绿与琥珀天光 |
| inkwash 水墨 | 东方素雅质感：浅色宣纸凝脂微暖画布，深色冷烟松烟砚黑，搭配淡墨微光与朱砂点缀 |

色值全部来自 `themes/*.css` 的 `--cst-ambient-*` token（3 主题 × 明暗）：
`glow-1/glow-2`（天光/地平线光源）、`sky`（顶部天光）、`horizon`（地平线雾）、`vignette`（暗角）。
保证底纹纯净通透，绝不喧宾夺主，永远让位正文内容。

### 16.3 挂载点（全站统一，页面不得自建）

- **外壳内容区**：`DesktopLayout` 根节点挂**一次整窗实例**（
  顶栏/侧栏/主内容共享同一世界层，不按区域各挂一份，避免区域交界出现渐变接缝）；
  `TabletLayout`、`MobilePortrait`、`MobileLandscape` 内容区各挂一次（canvas 完整版）——
  氛围层固定于可见内容区，**不随路由滚动**；`PageViewport` 底色透明以透出装饰层
  （画布底色由外壳统一提供 = `surface-page`）；
- `variant="chrome"`（轻量天光）：仅在确实需要给导航 chrome 单独铺光的场景使用；桌面连续画布
  下不使用（整窗已共享世界层）。
- 认证页（`routes/(auth)/login`）、错误页（`routes/+error.svelte`）；
- 启动占位（根布局 BootSplash 外层）：开屏即「世界 + 徽记」，接近游戏加载画面。
- **鸿蒙端**：统一挂在 `PageShellContainer`（`fillHeight=true` 的页面视口容器）内、内容之下；Tab 页与 NavDestination 推入页自动继承。消息 Tab 未登录分支在 `PhoneLayout` 显式接线（底板 → AmbientBackdrop → LoginPromptView，登录提示不得给不透明底色，否则会盖住氛围层）。页面不得自建第二套背景装饰。

> 挂载点必须在外壳内容区：`PageViewport` 高度固定为视口高、内容溢出其外，`position:sticky`
> 无活动空间，背景会随滚动消失。

### 16.4 硬约束

- 装饰层必须 `aria-hidden="true"` + `pointer-events:none`；挂载容器 `relative isolate`，
  层自身 `z-index:-1`（压在画布底色之上、正文之下），不参与布局 / 交互 / 焦点 / 读屏；
- 不允许页面内手写第二套背景装饰（新页面自动继承画布；确需场景化空态时另立组件评审）；
- **动效默认关闭（强制）**：全屏渐变/星层无限动画会让浏览器 GPU
  进程在页面完全空闲时持续合成（实测首页 idle GPU 进程 24.6%→0%、滚动 19.3%→14.9%、
  1.6%→0%），因此基础类不带 `animation`；只有显式 `<AmbientBackdrop motion />`
  （→ `.ambient-backdrop--motion`）才播放，且仅限空态/稀疏页面使用；
  动画只允许 `transform` / `opacity` 合成属性，时长走 `--cst-motion-ambient*`（24s/36s，仅 motion opt-in 时生效）；
  `prefers-reduced-motion: reduce` 下全部停用（含 opt-in）；
- `variant="chrome"` 只允许天光/边光（不得画世界层/剪影/暗角），用于顶栏、侧栏这类
  窄而高的导航 chrome；主内容区必须用默认 canvas 变体；
- **不引入第二整层底色**：装饰层是透明渐变/图形叠加，不改变「`--cst-surface-page` 是唯一
  整层底色」契约（见第 3 节与 `hotspot-frontend-theme` skill）；
- 世界层固定在外壳内容区（不随路由滚动），形成纵深视差；全画布页（消息分栏 / 地图）
  由自身不透明画布覆盖，氛围不可见属预期；
- **窄屏构图修正（≤640px）**：世界层经 `xMidYMid slice` 裁切时右侧母题会出画，手机档把
  星际轨道环/星座、水墨朱砂圆相平移到画面中部并缩小（`utilities.css` 媒体查询）；
  新增主题的世界层若含右侧母题，必须同步补窄屏构图。鸿蒙端按同一 1600×900 设计空间与
  slice 裁切口径复刻（窄屏变换同 web）。
- **鸿蒙端 Canvas 约束**：静态一次性绘制（尺寸/主题变化才重绘）、`hitTestBehavior(HitTestMode.None)`、
  `accessibilityLevel('no')`、无逐帧状态更新；每个 `fillHeight` 页面一块画布实例（Tab 页常驻，
  实测 4→5 实例、PSS 218.7→330.2MB 含页面数据，单画布理论 ≈12MB）。实例数/内存优化是已登记
  backlog（见 `docs/ui-improvement-handoff.md`），未优化前不得再增加挂载点。

### 16.5 已知边界

- 鸿蒙端实现：`AmbientBackdrop.ets` + `ThemeConstants.AMBIENT`（6 组合 × 12 色，预合成 `#AARRGGBB`）。
  字段语义与 web `--cst-ambient-*` 一一对应；数值按 web `color-mix(palette 色 N%, transparent)` 的同一 N 预合成，
  由 `check-cross-platform-ambient.mjs` 逐值校验（改 web alpha 后同步重算，否则 CI 红）；
- **平台适配差异（已登记）**：web 与鸿蒙默认都为静态世界（web 动画默认关闭，
  统一以性能优先）；web 额外支持 `motion` 逐页 opt-in，鸿蒙保持静态 Canvas；
  鸿蒙每页一块画布（见 16.4）；两端同主题同明暗的世界语言/构图一致；
- 若后续觉得氛围过淡/过浓，调 `--cst-ambient-*` 的 alpha 即可全站生效（不改组件），
  鸿蒙需按同口径重算 `ThemeConstants.AMBIENT` 的 ARGB 值；
- **待优化 backlog**：鸿蒙多画布实例的内存/首帧成本（上提到外壳单实例 or 共享绘制缓存）。

### 16.6 场景化空态装饰（Spotlight，超大空态专用）

**目的**：给整页/整块内容区的超大空态（访客引导、我的内容空、搜索无结果等）一个「信标」
意象，与氛围画布协同一体；小面板空态保持纯图标/文案，**不得开启**（堆装饰反而吵）。

**实现（两端同一 spec）**：

- 前端：`EmptyState` 的 `spotlight?: boolean`（默认 false）+ `.empty-spotlight*`；
- 鸿蒙：`EmptyState` 的 `spotlight` + `components/states/EmptySpotlight.ets`。

**几何（vp/px 同值，1:1）**：

| 层 | 尺寸 | 样式 | token |
|---|---|---|---|
| 中心光晕 | 320×320 | 径向渐变，中心色 → 70% 透明 | `--cst-ambient-glow-2` / `AMBIENT.glow2` |
| 外环 | 288×288 | 1 发丝描边、无填充 | `--cst-ambient-line` / `AMBIENT.line` |
| 内环 | 168×168 | 1 发丝描边、无填充 | 同上 |
| 四向星点 | 4×4 ×4 | 外环半径 144 的 45° 等分位（相对中心 ±102, ±102） | `--cst-ambient-dot` / `AMBIENT.dot` |

**硬约束**：无动画；装饰层 `aria-hidden` / `accessibilityLevel('no')`、`pointer-events:none` /
`hitTestBehavior(None)`、位于内容之下（前端 `z-index:-1` + 父级 `relative isolate`）；
颜色只取 ambient token（随三主题 × 明暗自适应）；ArkTS 侧 `.border()` 必须写在
`.borderRadius()` 之前（否则圆环被重置为直角，实拍确认）。
---

## 17. 整卡图片卡（Image-first overlay card）

**目的**：图像是卡片主体——图片占满整个卡片，信息覆盖在图片上方，文字与动作以
遮罩层形式压在图上，而不是「图片 + 白色信息托盘」两段式。

### 17.1 适用范围

- 适用：首页信息流卡（`HomeFeedItem` card）、附近的人/群组/瞬间网格卡（`NearbyImageCard` 非 horizontal）、
  我的瞬间/我的群组卡、群组详情精选卡等**图像优先的内容卡**；鸿蒙 `MomentFeedCard` / `NearbyImageCard` 同。
- 保留两段式（已登记例外）：横排紧凑列表卡（`horizontal`，图左 40% + 右侧信息托盘，用于手机我的群组等
  列表场景）、搜索/关系等非图片列表行、详情页 hero/封面（已是覆盖式）。

### 17.2 结构与几何

- 图片 `object-cover` 铺满整卡；卡片**不设 border**，圆角 `rounded-lg`(20) / 鸿蒙 `SHAPE.CARD`；
  投影只在 hover 抬升时出现（`card-hover` 语言）。
- 比例：feed 卡 `aspect-[4/5]`；网格卡（附近人/群、我的瞬间/群组、群组精选）`aspect-[3/4]`；
  同一视图只用一档。BSP 瓦片墙按布局单元铺满（登记例外）；横排 `horizontal` 紧凑卡保留两段式（例外）。
- 三层覆盖（装饰层 `pointer-events-none`，交互按钮自身可点）：
  1. **顶部遮罩** `--cst-surface-card-scrim-top`（高 128；三段 alpha 0.86 / 0.64@40% / transparent）：
     作者行（头像 32 + 白字昵称/时间）、关注/推荐徽章、右上分类/位置/多图计数徽章
     （深色强遮罩胶囊 + 白字）；高度必须覆盖到时间行底（≈52px），96px 会不足；
  2. **底部遮罩** `--cst-surface-card-scrim-bottom`（高 62%，feed 卡 58%；四段 alpha：亮 0.94/0.86/0.64/0，
     暗 0.96/0.88/0.68/0）：标题/摘要（白字，≤2 行）+ 元信息（「时间 · 标签」，位置只在右上出现一次）+
     操作行（点赞/评论/收藏/分享，白色图标 18 + 计数）；
  3. 中间保持画面干净，不叠半透明层。
- 内边距 16（移动）/ 20（平板·桌面）；feed 标题 16 semibold；网格卡 171px 宽为登记例外
  （正文 `text-sm`=12 / 作者 `text-md`=14，避免 16px 两行挤压操作行）；元信息 `text-xs`=11（非 12，
  以 `theme-registry.css` 实际刻度为准）。
- 多图：封面取首图 + 右上「1/N」计数徽章（详情页仍可看全部）。
- 无图/加载失败 fallback：`--cst-surface-card-fallback` 主题渐变 + 居中 64px 主题水印图标
  （`--cst-opacity-watermark` = 0.08，aria-hidden）+ **主题色文字**（不得压白字）。
- 骨架屏：整卡图片占位 + 顶部作者行/底部文本条占位，与真实卡同比例、同文字位置（零位移）。
- 覆盖态按钮（`onMedia` 变体）：关注/加入/已加入/审核中等压图按钮 = 深色强遮罩胶囊 + 白字 + 白描边
  （未加入主按钮可用 accent 实底 + `text-on-accent`）；亮图/暗图都稳。

### 17.3 硬约束

- 压图文字对比度 **≥4.5:1**（像素级实测：纯白底图最坏情况下的字形区 vs 背景），亮暗双主题都要测；
  徽章一律深色强遮罩 + 白字（禁止浅玻璃底 + 白字：4.16:1 不达标）。
- 操作行热区 ≥44、纯图标按钮有 `aria-label`；整卡点击仍走覆盖按钮，行内动作抬 z 序可点
  （沿用 `NearbyImageCard` 既有「覆盖按钮 + auto-lift」模式，不得嵌套交互元素）。
- 点赞/收藏等**状态**不依赖色相：填充形状 + 白色描边共同表达（色弱可辨）。
- 新增 scrim/fallback token 必须三主题 × 明暗齐备（前端 `themes/*.css`；鸿蒙 6 套 palette 同值），
  其余颜色一律走 token，禁止页面内写 rgba。
- 对比度实测方法：卡内封面替换纯白 data-URI（最坏情况）→ 元素「有字/无字」两帧原子截图取字形区 →
  按 WCAG 逐像素算 min/P5，全部 ≥4.5:1 才算通过。基线：全局最低 4.81（附近群组群名 light）；
  顶部作者行必须用「三段 0.86/0.64@40%/transparent + 高 128」遮罩，实测昵称 7.47/时间 4.89（light）、
  7.69/5.04（dark），全项达标。
- 触控：操作行图标 18px + `hit-slop` → ≥44；覆盖按钮热区同样 ≥44。

---

## 18. 选中态不得仅依赖颜色（TabBar / TabsPillGroup）

**选中态不能只靠颜色区分**（色弱/色盲用户不可辨）。所有「有选中态」的控件必须同时
提供**非颜色线索**（形状 / 图标 / 字重 / 下划线 / 位置）：

| 形态 | 选中态要求 |
|---|---|
| `segmented`（分段控制器：首页「为你推荐/正在关注」、设置明暗、登录模式） | accent 实底胶囊 + **✓ 图标**（`Check` / `AppIcon.CHECK`，12–14px）+ **font-semibold**；未选中 normal |
| `pills`（筛选/分类胶囊） | accent-soft 底 + accent 字 + **font-semibold** |
| `underline`（页级两态切换） | **下划线**（2px accent）+ **font-semibold** + 主文字色 |

- ✓ 图标与文字 gap 4；图标尺寸不得盖过文字；有 swatch/icon 的分段项仍加 ✓（不挤时可放文字后）。
- 验收：强制灰度（`filter: grayscale(1)`）截图中，选中项必须仍一眼可辨；两端（web / 鸿蒙）同口径。

---

## 19. 内容岛（Content Island）

**背景**：页面不要明显的边界。页面级/壳层不画框；开放页面的**内容分组**统一为「软岛」：
无描边、表面色 + 轻投影 + 圆角 20 分层——设置/通知/关系/搜索/个人中心不再是一格一格的框，
而是若干安静的浮起内容块。

### 19.1 形态（强制）

| 项 | 值 |
|---|---|
| 容器 | `bg-surface-island` + `shadow-xs` + `rounded-lg`(20)，**无 border** |
| 内边距 | `p-2`(8)；行 hover/focus 填充离岛边 ≥8 |
| 行 | 行背景透明，行间 `gap-y-0.5`(2px)；**不加 1px 行横线**；hover 填充 + 12 圆角 |
| 岛内元素 | chip/segmented 等用既有 `surface-subtle`/`surface-primary` 单元，不得套第二层带框卡片 |
| 例外（登记） | 弹层/Sheet、hero/封面预览、单帖卡、聊天输入条等功能/内容单元不纳入岛统一 |

### 19.2 token `--cst-surface-island`（三主题 × 明暗 6 处，必须一致）

| 组合 | 定义 | 合成到画布实测 | 岛/画布亮度比 |
|---|---|---|---|
| light | vibrant/inkwash：`color-mix(surface-primary 86%, transparent)`；**forest 覆写**：`color-mix(surface-primary 92%, text-primary 8%)` | vibrant #FDFDFE / **forest #E4E6E2** / inkwash #FAF8F2 | vibrant 1.11 / forest **1.17** / inkwash 1.06（forest 纸面主题画布与卡片同色，仅阴影不足以分层，故用 8% text-primary 提亮；岛屿文字 primary 13.11 / secondary 6.42） |
| dark | vibrant 显式覆写 `#181E2B`；forest / inkwash：`color-mix(in srgb, var(--cst-surface-primary) 94%, var(--cst-text-primary) 6%)` | vibrant #181E2B / forest #232723 / inkwash #2D3031 | 1.18–1.33（暗色阴影不可见，必须靠表面差分层） |

- registry：`--color-surface-island: var(--cst-surface-island)` → `bg-surface-island`。
- 新增主题必须同时补 light/dark 两处，`check-theme-parity` 会校验变量集合。
- 鸿蒙镜像：palette `surfaceIsland`（同 6 值），`ListGroup`/`SystemCard` 走同形态。

### 19.3 页面级硬线

- 页头/分区标题下**不得有 1px 硬线**（用留白 + eyebrow 分组）；页面内允许保留的只有功能线：
  聊天输入条/粘底操作条（`border-t`）、消息分栏竖线（降为 `border-subtle/40` 发丝线）、
  表格/统计行、内容单元内演示分隔。
- 每次新增页面按 §19.1/§19.3 自查，并在审计表登记「保留的功能线」。

### 19.4 按钮语言（安静化）

- 主操作：accent 实底 + `text-on-accent`（保存/接受/加入）。
- 次级/安静：`bg-surface-subtle/60` 软填底、无描边、`text-text-primary`；hover → `bg-surface-subtle` 全量；
  focus-ring / active scale 保留（编辑资料、设置、发消息、拒绝、已读、会话操作）。
- 危险：软填 danger 底 + danger 文字，保留二次确认（注销、退出登录）。
- **同级动作不得混用「描边胶囊」与「软填胶囊」**；功能边界（输入框、主题 swatch）保留。

### 19.5 例外清单（类型 / 理由 / 是否保留 1px）

| 类型 | 位置示例 | 理由 | 保留 1px |
|---|---|---|---|
| 表格行线 / 表头线 | admin 全部表格 TH/TR | 数据对齐与扫读必需 | ✅ 保留 |
| 表格外壳 / 容器框 | TABLE_WRAP | 内容分组，非功能线 | ❌ 移除 |
| 表单输入框 / select / textarea | SearchInput、admin 筛选 | 控件功能边界（可点击区域/输入态） | ✅ 保留 |
| 媒体容器（单帖大图/行卡配图/九宫格） | MomentPostBody、HomeFeedItem | 图片与背景边界靠圆角+阴影+占位即可 | ❌ 移除（圆角/阴影/focus-ring 保留） |
| Hero 封面（群详情/主页封面） | group-detail hero | 媒体 hero 例外 | ❌ 移除描边（保留圆角/阴影/占位） |
| 内容分隔发丝线 | 单帖统计/评论分隔、群详情 tab 下 | 内容分区，非卡片边界 | ✅ 保留 |
| 评论输入条 / 底部操作条 | moment detail、group detail | 功能栏 | ✅ 保留 |
| 内容 chip / badge / 筛选 pill | 标签、状态胶囊、tab pill | 内容单元边界 + 可点控件 | ✅ 保留 |
| 头像圆环 | 头像、评论头像 | 头像与背景分离的功能边界 | ✅ 保留 |
| 弹层 / Sheet / Dialog / 菜单 | overlays/*、sheets | 浮层容器 | ✅ 保留 |
| 地图桌面面板 | map/desktop/* | 功能浮层（与地图强耦合） | ✅ 保留 |
| demo-only 演示卡 | `/?demo=` 雷达/轮播演示 | 非默认用户路径，避免为改而改 | ✅ 保留（不改） |
| 举报裁决 segmented | admin/reports 弹窗 | 表单控件选中态边界 | ✅ 保留 |
| 浮动保存条 | admin/config sticky bar | 功能反馈条，去 1px 但保留阴影/模糊 | ❌ 移除 1px |
| 悬浮 Tab 容器 | 首页桌面分段器 | 浮岛，底色/投影表达 | ❌ 移除 1px |

---

## 20. 交互状态与可访问性基线

### 20.1 焦点可见（focus-visible，强制）

- 所有 interactive（按钮/链接/行/输入/自定义卡片点击区/表格行按钮）必须有可见焦点环：
  **2px accent + 2px offset**（web `ring-accent` / 鸿蒙 `focusBox`）。
- 键盘 Tab 顺序可到达；skip link 首 Tab 可见；**去描边不得连带丢焦点环**。
- 亮/暗主题都要验证；父容器 overflow/裁切不得吃掉焦点环（ArkUI 需父级 padding ≥2vp）。
- 验收：键盘 Tab 截图 + computed outline（web）/ 布局树 + 代码参数（鸿蒙）。

### 20.2 减少动态（prefers-reduced-motion，强制）

- 全局 `@media (prefers-reduced-motion: reduce)` 覆盖：位移动画/脉冲/骨架 shimmer/悬停过渡禁用
  （duration ≈0 / 单次）；透明度淡入可保留。
- web 走 `styles/theme/utilities.css` 全局规则；鸿蒙走 `MotionUtil`（`shouldReduceMotion()`），
  新增动画一律走统一入口，禁止裸写动画参数；默认（未开启）行为必须与既有完全一致。

### 20.3 hover / press / disabled

- 卡片 hover 抬升 + 图片微缩放（`card-hover`）；按钮 active scale .94–.98；行 hover 填充；
  disabled 不可点且可辨（opacity + pointer-events）；同级元素手感一致。

### 20.4 加载 / 空 / 错误态

- 首屏加载：列表/卡片优先 Skeleton；单实体/短列表可 InlineLoading（需给理由：结构轻、<1s、
  避免二次位移）。空态/错误态统一 `EmptyState` / `LoadError` + 重试；admin/长尾页一致。
- 回归基线：39 页 0 console error / 0 横向溢出 / h1=1。

### 20.5 语义结构规则（axe-core 基线）

自动化基线：axe-core 4.10.2（`frontend/node_modules/axe-core/axe.min.js`，经 `agent-browser --init-script` 注入；`bun run audit:a11y` 即全量跑）；
验收目标 **0 serious / 0 critical**。规则：

- **整卡可点**：卡片根不得持 button/aria-label + 内层交互（nested-interactive）；用「覆盖按钮（承载点击/键盘/aria）
  + 行内动作 auto-lift」模式（NearbyImageCard / HomeFeedItem / ListItem 均用此模式）。
- **列表语义**：`role=list` 的直接子节点必须是 `role=listitem`（自定义行也要补）；`role=listitem` 上不挂 `aria-label`，
  名称由覆盖按钮或可见文本承载。
- **定义列表**：`dl` 只能直接包含 `dt/dd`；指标组/统计组用 `div role=group`（MetricRow 即此实现）。
- **aria-prohibited**：无 role 的 `div` 不得挂 `aria-label`；用可见文本 / `sr-only` 文本 / 覆盖按钮承载。
- **地标**：每页有 `main`（登录/注册/错误页也要），必需 landmark 不重复；`lang` 正确。
- **可滚动区域**：可滚动且非键盘可达的容器（如 admin config 的 `pre`）加 `tabindex=0` + `role=region` + `aria-label`。
- **图片**：装饰图 `alt=""` + `aria-hidden`；内容图 alt 提供信息，避免与相邻文本重复（image-redundant-alt）。

## 21. 视觉语言：锚点、同轴与安静动作

> 本节登记**可验收的规则**（页面/组件按此执行，新增页面不得另起一套）；数值全部复用既有
> token（无新增 palette、无新增字号/圆角档位）。

### 21.1 三个母题

1. **一个锚点**：每屏只允许一个视觉主角（一张图 / 一个人 / 一组数字），其余退到
   `text-secondary` + 11–12px；**容器宽度服从内容**，不得为填满容器而拉伸
   （反面：1024 宽身份卡右半 400px 全空、精选横卡右半 350px 空）。
2. **同一张卡**：所有图片优先卡共用一套几何与覆盖语法——一个左上分类/标签胶囊、
   一个右上距离/隐私徽标、底部一行标题 + 一行 meta + 一行动作；**比例只随断点变一档**
   （移动 3/4，≥600px 4/5；BSP 瓦片墙为算法产物例外）。
3. **一条轴，一种安静**：页头标题、正文列、粘底动作条共用同一条内容轴；
   安静次级动作只有一种形态（`ui/button variant="soft"` 中性软填、无描边）。

### 21.2 内容轴（强制）

- 页面**不得自设页面级 `max-w-*`**（除已登记的表单/列表窄列例外），正文与
  `PageHeader` 共用 `ContentColumn` 的 `--cst-page-gutter`；分栏页主列恒为
  「主列 + gap + rail」，不随视口漂移。
- **gutter 只有一层**：`ContentColumn` 是左右 gutter 的唯一管理者。布局组件/页面
  不得在它外层再叠一层页面级左右内距（`SplitPageLayout` 外层只保留 rail 的右侧
  内距）；否则正文比标准页右移一档 gutter（1920 桌面实测 312 vs 284，切页横向跳动）。
- **全宽页正文轴 = 主内容面板左缘 + `--cst-page-gutter`**（1920 桌面：284）。
  分栏页主列同为 284；带内容上限的页面（`--cst-content-*` 档，如 settings/me/
  notify/moments/users）按上限居中，但页头、正文、粘底条仍共用同一条轴。
- 粘底条（评论输入/移动操作条）内部内容与内容卡同轴（1440 实测应完全对齐：
  例 /moments/[id] 284–1060 = 卡片 284–1060）。
- 表格/后台页正文与模块 Tab 栏同轴（`/admin` 概览卡片起点 = Tab 栏起点）。

### 21.3 指标条（MetricRow strip）

- 用法：`MetricRow variant="strip"`（详见 `docs/ui-components.md`）。
- 数字 24px（紧凑 20）semibold `tabular-nums` `tracking-tight`；标签 12px 次级；
  `divide-x divide-border-subtle` 发丝分隔；**每屏最多一处强调色**（如"在线"绿色 + 圆点），
  其余中性。
- 禁止：灰色药丸指标块、每个数字套圆角矩形、「卡片里的卡片」。

### 21.4 资料身份封面（IdentityHero）

- 封面条：桌面 132 / 移动 92，主题渐变或封面图 + 遮罩（token）；
  头像 88/72，`-mt-11` 压封面 + `ring-4 ring-surface-page`；
  桌面两栏（左身份 / 右指标+动作），移动动作整行；stats 用 21.3。
- 动作：主操作 = accent 实底；次操作 = `soft`。他人主页动作形态二选一：
  未关注（关注 primary / 私信 soft）、已关注（私信 primary / 已关注 soft），**禁止第三种描边胶囊**。

### 21.5 安静次级动作（soft 变体）

- `ui/button variant="soft"`（`bg-surface-subtle/60`、无描边、hover 全量）是页面/行内
  次级动作唯一形态：编辑资料/设置/发消息/私信/拒绝/新会话/全部已读/表格行内动作等。
- 描边只留给「可切换的筛选 chip」（TagPill/TabBar）与「功能边界」（输入框、主题 swatch）。
- 状态型关系 chip（已关注/已加入）同族：未激活 = surface-accent，已激活 = soft。

### 21.6 分区标题（SectionHead）

- icon 16 + 标题 14 semibold + count 12 次级 + 右侧「查看全部」文字链（12 accent）；
  rail 变体（11 semibold）用于右侧栏卡片；`eyebrow` 只用于表单/微型分组标。
- 页面不再手写「标题 + 查看全部」行；同一页多个分区间距 24，标题→内容 12。

### 21.7 空态与访客引导

- 页面级空态：新版 EmptyState（64px 软底图标砖 + 14 semibold 标题 + 12 描述 + 动作），
  超大空态开 `spotlight`；小面板不开。
- 访客态整页/整块：`NeedsAuthNotice` 面板（spotlight + 3 条价值点 + 主/次 CTA），
  页面按语境传 highlights（如消息：私聊与群聊会话 / 在线好友与群组 / 消息实时同步）。

### 21.8 卡片文字层级（图片优先卡）

- 标题：`text-base sm:text-lg` semibold `leading-tight`（压图仍白字，禁止降低遮罩对比度）；
- 作者行：头像 24–32 + 昵称 14 semibold + 时间 12；meta 行 12；动作计数 12 `tabular-nums`、动作间距 24；
- 压图对比度保持 §17（徽章强遮罩 + 白字；位置信息只出现一次）。

### 21.9 卡片交互

- **层叠**：图片内容卡根 = `card-hover isolate group relative overflow-hidden`；`isolate` 让卡内
  `z-20` 操作限制在卡内，页内 sticky header 稳定压住整卡（移动端「卡片 UI 压标题栏」的根因即卡根缺 `isolate`）。
- **按压**：`use:cardPress` + `data-pressed` + `--cst-card-tap-scale`（vibrant 0.94 / forest·inkwash 0.97），
  仅卡本体/覆盖按钮按下缩放；行内按钮不带动整卡。
- **操作行**：`data-display/ImageCardActions` 唯一实现（点赞红/收藏 accent/评论分享中性；pop 仅激活方向；
  md/sm 两档；计数 `tabular-nums`；aria 完备），信息流大卡与网格卡共用。
- **压图徽章**：`.media-badge`（图标方角标 `--icon`）唯一实现：深色遮罩 + 反色字 + h-5；
  **禁止 backdrop-filter**（见 §21.11）。
- **作者区**：有 `userId` 时可点进主页（头像/昵称同 hover 语言）；分享统一 `lib/moment-actions.shareMoment`。

### 21.11 性能纪律（强制）

**背景**：首页打开后浏览器进程管理器 GPU 占用高。实测（headless Chrome，GPU 进程 CPU%；采集环境为 MSW mock 前端，结论对真实后端同样适用）定位到三个叠加项：

| 因素 | 实测影响（首页 1440x900） | 处理 |
|---|---|---|
| 氛围画布 3 条无限动画（全屏渐变 x2 + 星层 opacity） | 空闲 GPU **24.6% -> 0%**（对照：关闭后 0%） | 默认关闭，motion opt-in |
| .media-badge 的 backdrop-filter（一屏 18 个） | 滚动 GPU 19.3%->14.9% 中约 1/3 | 禁止使用（深色遮罩已保证对比度） |
| .hover-action 常驻 will-change（首页 48 个） | 合成层数 79->12（CPU 影响小，显存/层数大） | 禁止常驻 |

现状实测：首页空闲 GPU 0.0%、滚动 10.5%（剩余为右栏 glass 与吸顶胶囊的少量毛玻璃，保留设计语言）；滚动 + 关 backdrop 的对照为 6.1%。

**规则（新增 UI 自检）**：

1. 全屏/大面积**无限循环动画默认不允许**；需要时逐页 opt-in，并给出 idle GPU 实测；
2. will-change 只允许加在「持续动画的固定图层」上，禁止挂到批量列表/卡片内的交互类（.hover-btn/.hover-action 明确禁止；transform 过渡 Chrome 会自动提升）；
3. backdrop-filter 只用于**少量、大面积有设计意义的 glass 面板**（右栏玻璃卡、吸顶胶囊），禁止用于批量小角标；新增毛玻璃必须评估滚动成本；
4. 性能改动必须给出可复现量测：见 docs/frontend-ui-audit.md 的「性能」采样方法（GPU 进程 CPU / document.getAnimations() / 合成层计数），并保留对照数据。

### 21.10 验收（可重跑）

- 全量截图矩阵：3 主题 × 2 明暗 × 桌面 1440 / 平板 768·1024 / 移动 390 × 登录/访客/管理员三态；
- `bun run audit:a11y -- --matrix --parallel 2`（3×2，0 serious/critical）；
- `bun run audit:touch`（390 起，未登记离群 0）；
- 轴与密度：同屏标题左缘 = 正文首元素左缘（±1px）；1440 内容包围盒 / 容器包围盒 ≥70%，
  容器内侧无 >120px 空带（截图量测）；
- 交互：hover/focus/press/reduced-motion 仍走既有基线（§20）。
- 性能（§21.11）：内容页空闲 GPU 进程 CPU 约 0%（document.getAnimations() 为 0）；
  bun run test 含 src/styles/__tests__/perf-contract.test.ts 源码契约守卫。

### 21.12 运行时响应与导航可读性（强制）

> 背景：可感知迟滞不是单次交互慢（Event Timing INP 16–64ms、hover 24ms、60fps），而来自
> ① dev 首访路由按需编译（58–900ms，生产 8–49ms）+ ② 页面进场动画从 `opacity:0` 起、180ms
> （点击后「空白一拍」再淡入）+ ③ 唯一滚动容器上的全局 `scroll-smooth` 把程序化滚动动画化 766ms。
> 同口径 dev 导航基线「点击→DOM / 点击→完全可见」：关系 **29.9/127.6ms**，通知 **26.9/167.4ms**，
> 设置 **31.7/133.6ms**（1440×900）。

**1. 页面进场预算（强制）**

- 客户端导航进场 ≤**120ms**，起点 `opacity ≥ 0.6`、位移 ≤2px；**禁止从 opacity:0 起**
  （先空白再淡入 = 可感知的「慢一拍」，特效保留但去掉不可见窗口）。
- 骨架→内容 fade ≤**110ms**。首屏不播进场（保护 LCP）；reduced-motion 由 JS 直接跳过（不加类）。
- 时长唯一来源：CSS `.page-enter` / `.content-fade` 的 `--cst-page-enter-duration` /
  `--cst-content-fade-duration` 与 `lib/motion.ts` 的 `MOTION`，由
  `lib/__tests__/motion.test.ts` 源码级锁定（改一处必须同步另一处）。

**2. 导航意图预取（强制，button 导航必须补齐）**

- SvelteKit 的 `data-sveltekit-preload-data="hover"` 只覆盖 `<a>`；本项目侧栏/顶栏是
  `<button>` + `goto()`，必须经 `lib/nav-preload.ts`：`preloadRoute()`（`preloadCode`、
  Set 去重、失败静默、当前路径跳过）+ `scheduleIdlePreload()`（5s 后 requestIdleCallback
  预热常用路由，saveData 跳过）。事件委托放 `DesktopLayout` 的 `<svelte:window>`
  （**必须在模板顶层**，放进元素内会编译失败）。
- 只允许预取**代码**（preloadCode）：本应用没有会取数的路由 load 函数，改成 preloadData
  会在真实后端产生额外请求。
- dev 必须配 `vite.config.ts` → `server.warmup.clientFiles`（全部路由 `+page.svelte` +
  布局/消息/导航组件；glob 用 `src/routes/*/**` 绕开 `(app)` 括号陷阱），否则首访路由
  按需编译 58–720ms，就是「每个新页面都要等一拍」的根因。
- 验收：冷 hover 300ms 后点击导航，click→DOM ≤50ms；未 hover 直点不回归。

**3. 全局滚动（禁止 scroll-smooth）**

- 唯一滚动容器（`RoutedContentViewport` 的 `<main>`）禁止 `scroll-smooth`：实测把任何
  `scrollTop` 赋值 / 无参 `scrollTo` 动画化 766–768ms，破坏滚动恢复、回底与分页视口保持；
  需要平滑的入口（BackToTop）自己传 `behavior:'smooth'`。
- 键盘 PageDown/Home 的 ~134ms 平滑是浏览器原生行为，与 CSS 无关（不要为此回退）。

**4. 合成层（§21.11 补充）**

- will-change 只允许「持续动画的固定图层」；`.image-hover-frame__image` 与
  `.avatar-entry__avatar img` 不得加常驻 will-change，`NearbyImageCard.svelte` 不得带
  `will-change-transform` Tailwind 类（Tailwind 类不受 CSS 删除影响）。新增同类交互禁止再加。

**5. 侧栏导航三态（强制）**

| 状态 | 底色 | 图标/文字 | 非颜色信号 |
|---|---|---|---|
| 普通 | 透明 | secondary / 1.5 描边 | — |
| hover（未选中） | `--cst-surface-navigation-hover`（text-primary 12% 混 surface-page，中性低彩度） | primary | 图标 scale 1.1 |
| 选中 | `--cst-surface-navigation-active`（accent 26% 混 surface-page） | accent 图标 / primary 文字 | 左侧 3×20px 指示条 + 1px accent 内缘描边 + semibold |
| 选中 + hover | `--cst-surface-navigation-active-hover`（accent 34%） | 同选中 | 同选中 |

- hover 不得使用 accent 色相（避免与选中混淆，且色盲/灰度可辨）；token 定义在
  `semantic.css`、Tailwind 类在 `theme-registry.css` 注册（`bg-surface-navigation-*`）。
- **灰度验收口径**：hover 底/画布 ≥1.2、选中底/画布 ≥1.4，且选中至少 2 种非颜色线索；
  `filter:grayscale(1)` 截图留档（三主题 × 明暗）。
- 全站按钮光标：`button:not(:disabled)` / `[role="button"]:not([aria-disabled="true"])`
  由 `base.css` 全局给 `cursor:pointer`——浏览器对 `<button>` 默认是箭头，会让 hover 反馈
  「不像可点」（实测侧栏/顶栏/TabBar 默认全是箭头）。

**6. 消息页（聊天）内容轴与降噪（强制）**

- 页头 / 消息列 / 输入栏必须共用 `components/messages/ChatColumn.svelte` 的同一列宽
  （`max-w-3xl lg:max-w-5xl`），禁止各自写 `max-w-*`；面板窄于该档自然 w-full。
  实测三列：1440 全部 x=576 / w=852；1920 全部 x=730 / w=1024。
- 短会话消息**底对齐**（`min-h-full` + `justify-end`，空白留在上方）——顶部锚定 + 底部
  228px 死空白是「消息页很怪」的主因之一。
- 「没有更多了」（ListEnd）只在历史确实分页并耗尽后出现；首屏一次返回全部
  （`next_cursor=null`）不得显示。
- 字号：消息正文 `text-md`(14) + `leading-card`；时间戳 `text-xs`(11)；群聊发送者名
  `text-sm`(12)；输入框 `text-md`(14)。accent 实底气泡内文字用 `text-text-on-accent`
  全量、**不得叠 opacity**（叠 0.7 只有 3.97:1，不达标）。
- 日期分节 = 中性 11px 文本标签（无底色/无投影），禁止白底胶囊（像按钮）。
- 会话行三态：选中 accent 实底 + 行首长指示条（on-accent）+ semibold；未读 淡 tint +
  行首短指示条（accent）+ 数字徽标 + semibold；普通无条 medium。选中行内文字 on-accent 全量。

**7. 验收**

- `bun run check` 0 error / 0 warning（含 eslint + theme 四联）；`bun run test` 117 files / 631 tests 全绿。
- 消息页三视口三列同轴（±1px）、短会话底对齐（最后气泡距输入条 27px）、无 ListEnd 噪音；
  正文/时间/日期/选中行对比度实测 ≥4.5（6 主题 × 明暗）。
- 侧栏 3 主题 × 明暗：普通文字 4.93–8.45、hover 10.28–12.05、选中 8.61–12.05；
  hover 底/画布 1.17–1.25、选中底/画布 1.27–1.33；灰度 L* 梯 hover→选中 4.3–6.8 且多 4 项非颜色信号。
- 页面性能纪律：内容页 `getAnimations()`=0、will-change 0、longtask 0、滚动 60fps；
  `main` computed `scroll-behavior:auto`。

### 21.13 宽屏填充与列表轴（强制）

> 背景：页头与正文轴漂移、宽屏填充率低、BSP 瓦片墙左缘偏移，根因都是「页面自设窄列但页头
> 不跟随」与「组件内层再收窄」。规则收口：

1. **页头必须跟随正文宽度**：`PageHeader` 的 `maxWidth` 与 `StandardPageLayout` 的
   `contentMaxWidth` 必须同值（一对 prop 一起传）；只改其一即轴漂移。
2. **宽屏呼吸档 = 1280px**（数字字面量，布局几何）：需要「长内容呼吸型」的页面
   （/me 总览、关系、举报）统一用 `contentMaxWidth={1280}`，不再自造
   `max-w-5xl/6xl/7xl` 或 `48rem` 私有档。
3. **列表组件不得内层再收窄**：ProfileUserSection（好友/关注/粉丝/黑名单）、
   SectionGroups 等列表/卡片分区内层禁止 `mx-auto max-w-2xl/3xl`——宽度只由页面
   ContentColumn 一层决定。骨架与内容必须同宽（骨架→内容零位移的前提）。
4. **列表行 hover 次级文字对比度（无障碍，强制）**：行 hover 底色是半透明
   palette-hover，叠加后行内 `.text-text-secondary` 实测跌到 3.2–4.7:1。ListItem
   在 hover（fine pointer）与 focus-within 时把行内次级文字升 `--cst-text-primary`
   （pressed/on-accent 行排除）。新增行类组件必须按此口径自检，不得只测静态态。
5. **BSP 瓦片墙轴对齐**：BSP 几何从 `gap/2` 内缩起算，容器外扩 `±gap/2`
   （BspCardCanvas 内置），保证首卡左缘 = 页头正文左缘；调用方不要再手补 margin。
6. **媒体占位动画限时**：图片占位 pulse 类动画最多 **3 个周期**（`.media-placeholder--pulse`，
   ≈4.5s）后转静态；禁止 infinite 占位动画（慢图下批量常驻会拖 GPU，见 §21.11）。
7. **首屏图优先级口径（补齐）**：每个列表首屏**前 2 张**内容图 eager + fetchpriority=high
   （GroupCard 现支持 `priority` prop）；行式列表头像小图、rail 缩略图保持 lazy。
8. **同屏信息不双写**：同一指标（如「N 在线」）同屏只允许一个表达位（Hero MetricRow
   或页头 chip 二选一）。

**验收**：/me 全页族 h1 与内容左缘 Δ0@1440；关系/举报列表岛填充 0.95@1440、
0.74@1920（≥0.70）；BSP slot0 x = h1 x（Δ0@1440）；ListItem hover 对比度 6 组合
10.61–13.86（全 ≥4.5）；/search 390 CLS 0×8 采样；/nearby 与 /nearby/groups 首屏
eager 图各 2 张且像素 diff 0 回归。

### 21.14 卡片 hover 的合成器纪律（强制）

> 背景：快速晃动鼠标跨卡滑动时 GPU/CPU 占用飙升（首页 / 附近的人 / 附近群组 / 附近瞬间 /
> 我的瞬间 / 我的群组等全部卡片页）。定位方法：agent-browser 驱动 headless Chrome（真 GPU：
> ANGLE Metal）+ CDP trace 分线程归因 + 逐进程 CPU 时间采样，视口 1440×1000 @DPR2。
> 根因：每次 hover 同时启动 4 条动画——卡片上浮（transform）、box-shadow、图片缩放
> （transform）、整卡覆盖按钮背景色；其中 box-shadow 与整卡按钮底色是「整卡面积逐帧重绘」
> 的 paint 动画（GPU 进程 CPU 0.14–0.15 cores，空闲仅 0.02；renderer 主线程 0.18 cores）。

1. **整卡命中面 hover 底色改伪元素 opacity（视觉等价）**：`[data-cover]::before`
   （`position:absolute; inset:0; border-radius:inherit; background:var(--cst-interactive-hover-fill);
   opacity:0; pointer-events:none; transition:opacity 160ms`）+ `[data-cover]:hover` 自身
   `background-color:transparent`（压过全局 button hover 填充）。颜色 token 与 160ms 时长
   与既有行为一致，只有执行位置从主线程重绘换成合成器；卡片与 ListItem 的覆盖按钮共用。
   **例外**：图片内容卡（`.card-hover`，已有合成器上浮 + 配图 1.045 缩放）不显示整卡底色——
   `.card-hover [data-cover]:hover::before { opacity: 0 }`（无 layer 区、特异性覆盖）。
   列表行没有抬升/缩放，**保留**底色反馈。
2. **卡片 hover 动画限定合成器属性**：`.card-hover` 只过渡 `transform` + `border-color`；
   抬升用 transform（vibrant 2px / forest·inkwash 0），配图缩放仍是 transform。
3. **卡片/配图在 fine pointer 下常驻合成层**：`@media (hover:hover) and (pointer:fine)` 内
   `.card-hover, .image-hover-frame__image { will-change: transform; }`——避免快速跨卡时
   反复提升/销毁层并重栅格化整卡；批量交互小类（`.hover-btn/.hover-action`）仍禁止。
4. **禁止死类**：卡片 className 不得写 `hover:shadow-raised-sm`（被 `.card-hover:hover` 的
   `!important` 覆盖）与 `transition-all`（被 `.card-hover` 过渡覆盖）——它们不生效，
   会让人误以为在承载 hover 视觉。

**验收**（worktree 8898 与主环境 8888 同 harness 交替采样，DPR2）

- GPU 进程 CPU：**0.141–0.146 → 0.101–0.109 cores（-27%）**；renderer 0.182–0.187 → 0.177–0.182（-3%）。
- hover 视觉增量像素级一致：/me/moments 单卡 hover-delta 178,601 vs 178,622 px，重叠 178,586 px
  （差异 3,294 px = 1.8%，全部为亚像素抗锯齿/合成取整）。
- 空闲 GPU 0.013–0.016 cores 不变；Chrome RSS ±4MB；滚动 GPU 0.127 → 0.090 cores（副作用为正向）。
- 契约守卫：`src/styles/__tests__/perf-contract.test.ts` 的「卡片 hover 只走合成器属性」一组。

## 22. 页面锚点岛（Page Island，桌面端）

**目标**：把「会随滚动离开视口、但用户随时可能需要」的常驻信息固定到 52px 顶栏中央，
用户自由滚动也不丢失上下文；短页/无滚动页不出岛（没有信息丢失，就不打扰）。

**结构（三段式，唯一实现：DesktopLayout `[data-desktop-dynamic-island]`）**

- `leading`：返回按钮 / 头像（作者名片，20px）；普通页面无 leading。
- `title + meta`：标题 12px semibold 单行截断；meta 11px secondary，最多两个事实。
- `trailing`：每页 ≤2 个行动控件（允许写操作：保存 / 全部已读 / 收藏等），状态与页面同步。

**触发与交接（信息丢失驱动）**

1. 吸顶页头（PageHeader 默认 / DetailHeader）：滚动 >120px 标题淡出（透明度过渡，保留高度不产生 CLS），
   由岛承接；回到 <72px 恢复（滞回防抖）。
2. 非吸顶页头（PageHeader `integrated` / `sticky={false}`）：标题离开视口即入岛
   （IntersectionObserver，rootMargin 扣掉 52px 顶栏）；回到视口退出。
3. 详情页：`DetailHeader` 滚动 >160px 后收敛到同一顶栏岛（作者/群/用户名 + 地点·时间 + 行动区）；
   移动/平板保留页面内吸顶岛。
4. 岛绝不因滚动方向自动隐藏；点击岛标题平滑回到锚点/页首（reduced-motion 立即）。
5. 无页面滚动 / 头部常驻不丢信息的页面不出岛：`/messages`（左右列头部常驻）、`/map`（全画布）、
   `/login`、`/register`（无外壳）；面板内嵌 PageHeader 传 `island={false}`。
6. 首页桌面：无页面标题，灵动岛承载信息流分段器（`components/home/HomeFeedIsland.svelte` + `stores/home-feed.svelte.ts`），内容区不再渲染吸顶分段器；移动/平板仍在页内品牌头下。首页不注册 `headerIsland`。

**API / 实现**

- 状态：`stores/header-island.svelte.ts`（PageIslandStore：token 注册 + title/meta/author/trailing + visible）。
- 入口：`PageHeader` 的 `island`（默认 true）/ `islandMeta`（缺省 subtitle）/ `islandTrailing`；
  `DetailHeader` 走同一 store 契约。
- 视觉：h-9 胶囊、`bg-surface-primary/90 + backdrop-blur-xl + border + shadow-raised-sm`；
  进入动画 180ms（fade + zoom），reduced-motion 立即。
- 无障碍：岛内文字不重复播报（标题按钮有 aria-label；h1 仍在 DOM/landmark）；不抢焦点；带返回的页头入岛后返回动作移到岛 leading，页头标题/返回同步淡出（同一动作不出现两份）。
- 性能：每页一个触发源（滚动阈值或 IO），无共享全局 scroll 监听；岛为顶栏 overlay，不参与布局 → 无 CLS。
- 测试钩子：页头暴露 `data-island-visible` / `data-island-handoff`（触发与交接断言用；供单测与排障，不是样式契约）。

## 23. 大屏 UI 等比放大（≥1920）

**问题**：2560/3440/4K 下 UI 元素尺寸不变，文字/图标/控件相对视口过小、不易读；单纯封顶内容宽度不会放大元素，不能解决可读性。

**契约**（等效浏览器 zoom，版面比例按「1920 视口」保持）：

- **根字号**：`html { font-size: clamp(1rem, calc(100vw / 120), 2rem) }`（`styles/theme/shared.css`）：1920→16px、2560→21.33px、3840→32px，上限 2×。所有 rem 尺寸（Tailwind spacing/字号/圆角、布局 token、组件 rem 化尺寸）自动等比放大；≤1920 与移动/平板仍为 16px，基线零变化。
- **字号 token 用 rem**：`--text-xxs/xs/sm/md` = 0.625/0.6875/0.75/0.875rem（10/11/12/14px @16 基准）；跨端刻度脚本按 16px 基准折算回 px 与鸿蒙 FONT.SIZE 对齐。
- **数值型内联尺寸**：lucide `size={n}` 图标由 `svg.lucide[width][height]:not([class*='size-']) { zoom: var(--cst-ui-scale) }` 统一缩放（`app.html` 首帧写入、`platform` store resize 同步）；Avatar/ActionIconButton/PinMedia/Toggle 等在组件内 rem 化。新增代码禁止裸写 px 尺寸（1px 发丝线、地图瓦片 256px 除外）。
- **JS 阈值**：滚动阈值、IO rootMargin 等乘 `platform.uiScale`（PageHeader/DetailHeader 已接）。
- **验收**：1920 与既有基准一致；2560/3840 读 `getComputedStyle(document.documentElement).fontSize` 与侧栏/顶栏/正文字号，确认比值 = 视口宽度比（±1px）、无横向溢出。


## 24. 数据量自适应（0 / 1 / 2 / 3+ / 大量）

**硬约束**：任何数据量下都要好看且和谐。骨架、卡片尺寸、间距、字号恒定，只改排列与空态；禁止孤卡贴边、卡片拉成巨卡、大片无解释空白、加载/到底控件空转。

- **统一判定**：`components/layout/AdaptiveCardGrid.svelte` + `lib/adaptive-grid.ts`（纯函数，已单测）。
  - 列数按**视口**断点（手机 2 / 平板 3 / 桌面 auto-fill 反推；fixed 模式按 base/lg/xl），轨道宽按**容器**计算；
  - `count < 可见列数` → 整行居中，轨道宽 = 密排宽（不放大/不缩小）；`count ≥ 列数` → 显式列数铺开（最后一行保持列对齐）；
  - BSP 网格传 `centerSparse={layouts.every(tier === '1x1')}`；`semantic-bsp` 在数据 ≤3 时强制 1x1，避免跨列撑破居中轨道。
- **0 条**：紧凑空态（图标 + 原因 + 动作），不占首屏；`查看全部` / 分页 / 到底在无更多时隐藏。
- **覆盖**：附近的人 / 附近群组 / 附近瞬间 / 枢纽页预览 / 个人主页瞬间 / 首页 feed 默认网格；新增网格必须复用，不得各页重写 auto-fill + 稀疏分支。
- **验收**：390/768/1440/1920/3840 × {0,1,2,3,8,20} 截图矩阵；同断点卡片宽高恒定（count 1→4→8 不变）、居中/列对齐正确、无横向溢出。
