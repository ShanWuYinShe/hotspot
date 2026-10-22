# HotSpot 设计系统

> **现行权威**：本文件是双端设计 token 数值的唯一权威文档，与
> `frontend/scripts/check-cross-platform-*.mjs`（CI 强制执行）共同保证双端一致。
> 改色、改字号、改间距、改圆角、改阴影、改动效时，两端必须同步，漂移由
> `bun run check:theme` 在 CI 拦截。强制的同步流程与组件规则见两端 `AGENTS.md`
> （根 `AGENTS.md` / `frontend/AGENTS.md` / `harmonyapp/AGENTS.md`）。

> 单一事实源（Single Source of Truth）。本文件从 Svelte Web 前端（`frontend/`）的
> `--cst-*` token 体系整理而来，覆盖**桌面端 / 平板端 / 移动端**三档设备形态，
> 并给出前端（CSS token）↔ 鸿蒙（`DesignConstants` / `ThemeConstants`）的逐项映射。

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

| 语义 | 前端 | 鸿蒙 | 值 |
|---|---|---|---|
| 移动端（手机） | `platform.platform === 'mobile'` | `!isDesktopForm` | `< 600` |
| 平板端 | `isTablet` | `isTablet` | `600 – 1279` |
| 桌面端 | `isDesktop` | `isDesktopForm` | `≥ 1280` |
| 内容双栏（master-detail） | `isWide` | `isWideForm` | `≥ 960` |

- **chrome 归桌面**：≥1280 切桌面顶栏 + 侧边栏；<1280 走 PhoneLayout（含平板药丸底栏）。
- **内容双栏归 isWide**：960–1279 是「平板药丸底栏 + 内容可双栏」，不是桌面侧边栏。
- 鸿蒙 `Breakpoint.ets` 底层 6 档（xs/sm/md/lg/xl/xxl）仅服务栅格，业务一律用语义派生量。

---

## 3. 颜色系统

三层架构：**primitive**（`--cst-palette-*`）→ **semantic**（`--cst-*`）→ **registry**（Tailwind `@theme`）。

主题族（vibrant / forest / inkwash）双端可选集一致：前端三套齐全；鸿蒙 `default_clean` = forest 清野、`inkwash` = 水墨， vibrant 为默认。每主题族 × 明暗模式。

### 3.1 原始色板（primitive `--cst-palette-*`）

| Token | Vibrant 亮 | Vibrant 暗 | Forest 亮 | Forest 暗 |
|---|---|---|---|---|
| `page-bg` / `bg` | `#F7FAFF` | `#0B1424` | `#F6F7F3` | `#161A16` |
| `text` | `#16233A` | `#E8F1FF` | `#152219` | `#E8F1E9` |
| `text-muted` | `#52647F` | `#A5B6CE` | `#3F5547` | `#A8BDAE` |
| `accent-primary` | `#2B5EB2` | `#69A8FF` | `#39714F` | `#52B878` |
| `on-accent` | `#FFFFFF` | `#071426` | `#FFFFFF` | `#0C1911` |
| `online` | `#0D6A62` | `#37C99A` | `#39714F` | `#52B878` |
| `hot`（危险） | `#C81E4E` | `#FB7185` | `#A04B37` | `#D97C5A` |
| `warning` | `#9C4708` | `#FBBF24` | `#7E602B` | `#C4A75A` |
| `map-user` | `#2B5EB2` | `#69A8FF` | `#39714F` | `#52B878` |
| `map-group` | `#0B7653` | `#34D399` | `#4A703F` | `#88B85A` |
| `map-moment` | `#C2470B` | `#FB923C` | `#8A6426` | `#C4A75A` |

> `workspace-bg` 已删除：历史上与 `page-bg` 两档整层灰并存造成断裂，现桌面舞台改用
> `pageBackground`、侧栏与主面板改为白浮卡。inkwash 主题族数值以 `themes/*.css` 为准。

> 派生档位（`-bg` 淡底 / `-border` 淡边框 / `-glow` 光晕）由 `color-mix(in srgb, 主色 12~14%, transparent)`
> 统一生成，禁止组件/数据层再写 `color-mix`。

### 3.2 语义层（semantic `--cst-*`，跨主题不变）

| 语义 token | 指向 | 用途 |
|---|---|---|
| `--cst-text-primary` / `-secondary` | `palette-text` / `-muted` | 正文 / 辅助文字 |
| `--cst-text-on-accent` | `on-accent` | accent 底上的文字 |
| `--cst-surface-primary` / `-page` | `bg` / `page-bg` | 表面 / 画布 |
| `--cst-accent-default` / `-soft` / `-border` | `accent-primary` / `accent-soft` / `accent-soft-border` | 品牌 / 淡底 / 淡描边 |
| `--cst-border-default` / `-strong` | `border` / `border-pressed` | 常规 / 强分隔 |
| `--cst-status-online/danger/warning` | `online` / `hot` / `warning` | 状态色 |
| `--cst-marker-user/group/moment` | `map-*` | 地图标记 |

### 3.3 鸿蒙侧颜色消费

- 画布：`palette.pageBackground` / `palette.surfaceBackground`（对齐前端
  `--cst-surface-page` / `--cst-surface-primary`）
- **画布契约（克制画廊，全量推广时遵守）**：应用内只允许一个整层底色 = 画布
  `pageBackground`（对齐前端 `--cst-surface-page`）；外壳各层（手机壳 / 桌面舞台 /
  顶栏 / 页面整底 / header bar 变体）只许用画布色连续铺底，层级只靠白卡
  `surfaceBackground` + 发丝描边（`BORDER_WIDTH.HAIRLINE` + `borderColor`）+
  主题投影（`surfaceShadow` / `elevatedShadow`）表达。禁止引入第二档整层灰。
  分组卡样板见设置页 `SettingsSectionCard`（白卡 + 发丝描边 + 投影）、分组小标用
  `textSecondary` eyebrow（`SettingsGroupLabel`）、用户 hero 卡只保留白卡 + 描边（不加光晕）。
- 表面：`palette.surfaceSubtle` / `palette.surfaceElevated` / `palette.chatBackground`
- 文字：`palette.textPrimary` / `palette.textSecondary` / `palette.textTertiary`
- 品牌：`palette.accentBackground`（forest `#3D7A55`，与前端
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
  旧 `settings_dark_forced=true` 用户已迁移为 `dark`）
- 已删除主题族的存量偏好一律映射 vibrant 对应明暗

---

## 4. 字体系统

### 4.1 字号刻度（`--text-*` ↔ `FONT.SIZE`）

| 档 | 前端 | 鸿蒙 `FONT.SIZE` | 用途 |
|---|---|---|---|
| xs-minus | `9` | `XS_MINUS` | 极小标注 |
| xxs | `10` | `XS2` | 极小 uppercase 标签（eyebrow、分组标） |
| xs | `11` | `XS` | 辅助文字（副标题、计数、时间戳） |
| sm | `12` | `SM` | 正文 |
| md | `14` | `MD` | 强调正文（昵称、按钮主标） |
| lg | `16` | `LG` | 正文（**鸿蒙平台适配档**，见下） |
| lg-plus | `18` | `LG_PLUS` | 小标题 |
| xl | `20` | `XL` | 标题 |
| 2xl | `24` | `XXL` | 大标题 |
| 3xl | `30` | `XXXL` | 强调大数字/节标题 |
| 4xl | `36` | `HERO` | 错误码/超大数字（前端最大 text 档） |

> **鸿蒙平台适配例外**：`FONT.SIZE.LG = 16` 为 HarmonyOS 标准正文尺寸（web 端正文 12/14 偏小），
> 是唯一多出的档位，两端有意不同，属已登记豁免，不属于漂移。
> 近邻档（13/15/22/28/32/48/56 与 XXXL_PLUS 36）已按「少档位」原则归并删除，禁止再引入。

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
    对齐前端 `theme-registry.css` 的 `--leading-*` 标尺（`card=1.7` 是前端
    「原 1.55，阅读疲劳修复」后的值）。**多行长文正文用它**（覆盖默认行高）：

    | 语义 | 比例 | 适用 |
    |---|---|---|
    | `DEFAULT` | 1.5 | 基线（对齐前端 base.css 元素级 1.5） |
    | `CARD` | 1.7 | 卡片/聊天气泡正文（前端 `--leading-card`） |
    | `COPY` | 1.75 | 长文正文、简介、公告、动态正文（前端 `--leading-copy`） |

  - 绝对档 `FONT.LINE_HEIGHT.{XS 18, SM 20, MD 22, LG 24, XL 26}` 只用于**单行紧凑**场景。
    ⚠️ 绝对档与字号是解耦的：把 `LINE_HEIGHT.SM`(20) 用在 `SIZE.LG`(16) 上比值只有 1.25。
    实测（2026-10-12）多处出现该误用，比值散落在 1.57~1.83，已统一到上表比例制。
- 字距：`FONT.LETTER_SPACING.{EYEBROW 1.4, LABEL 1.2, FIELD 1.28, ICON_LABEL 0.6}`
  （对齐前端 tracking-eyebrow/label/field 刻度，鸿蒙按 10px 基准折算）。

---

## 5. 间距（`SPACING`）

4px 网格 + 微步，**不强制收敛**（间距的 2px 差异是布局设计的一部分）：

`HALF 2` / `THREE 3` / `XS 4` / `SM2 6` / `SM 8` / `MD2 10` / `MD 12` / `MD3 14` / `LG 16` / `LG2 20` / `XL 24` / `XXL 32`

禁止组件内 `padding({left:16})` 这类裸数——用 `SPACING.LG`。1vp 级微距（top:1 等）与 0 值允许保留（对齐前端间距不强制收敛原则）。

### 5.1 页面级横向 gutter（唯一来源，强制）

**横向留白只能由 `PageShellContainer` 提供**，页面内不得再为自己的正文写
`left`/`right` padding：

| 断点 | gutter | 对齐前端 `--cst-page-gutter` |
|---|---|---|
| xs / sm（手机） | `PAGE_SHELL.GUTTER_MOBILE` = 16 | 1rem |
| md（平板） | `PAGE_SHELL.GUTTER_TABLET` = 24 | 1.5rem |
| lg+（桌面） | `PAGE_SHELL.GUTTER_DESKTOP` = 28 | 1.75rem |

- **禁止**「容器 gutter + 页面自加 padding」双重留白——会叠成 36~40，与只用容器
  gutter 的页面差 16~20vp，各页正文左边缘不在同一条竖线上，滑动时边界跳动。
- **禁止**用 `noHorizontalPadding: true` 关掉容器留白再另写一套值。
- **通栏元素（封面图等）说明**：前端用 `-mx-[var(--cst-page-gutter)]` 抵消 gutter，
  但 **ArkUI 不能照搬**——子元素 `width('100%')` 是相对**父内容盒**解析的，负 margin 只会
  位移不会加宽（实测 2026-10-11：群详情封面加左右各 −20vp 后仍是 x=70..1186 / 屏宽 1256）。
  当前约定：封面/hero 一律与下方内容同宽（收束式 hero），不做通栏；确需通栏须把该页整页改为
  `noHorizontalPadding: true` + 各区块自加 gutter（已登记例外），或改用 `calc` 宽度。
- `SystemPageHeader` 只带纵向节奏、**不再自带横向 padding**（此前自带 20 与
  容器 gutter 叠加，页头标题比正文多缩进一截）。

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

语义形状（`SHAPE`）：`CARD 20 / HERO_CARD 24 / COMPACT_CARD 16 / PREVIEW_CARD 20 / LIST_ITEM 20 / ICON_TILE 12 / INPUT 16 / BUTTON FULL / CHIP FULL / TAB_BAR 24 / SHEET_TOP_RADIUS 28`，按前端语义用法对齐（hero/feature 24、主按钮胶囊 FULL、弹层顶部 28）。近邻档 8/18/22 已删除，禁止再引入。

---

## 7. 边框宽度（`--cst-border-width-*` / `BORDER_WIDTH`）

`HAIRLINE 1` / `REGULAR 1.5` / `STRONG 2` / `HEAVY 3`。禁止 `.border({ width: 1 })` 裸数，一律 `DesignConstants.BORDER_WIDTH.*`。

---

## 8. 阴影与层级

扁平设计默认 `SHADOW.*` 全零；需要投影时用主题注册表
`AppThemeManager.getPaletteById(...).{surfaceShadow, elevatedShadow, controlShadow, tabBarShadow}`
（真值在 `ThemeConstants` 的 ThemeShadowRegistry），禁止组件内 `.shadow({ radius: N, ... })` 手写。

**全端统一中性 raised 投影语言**：主按钮/主图标钮/加入·关注·发布等 accent 实底控件静态用
`surfaceShadow`、hover/按压抬档 `elevatedShadow`，禁止 `accentShadow` 彩色光晕
（`accentShadow` 字段已随 glow 体系删除，对齐前端 ui/button
`shadow-raised-sm hover:shadow-raised-md` 契约）。

### 8.1 跨端投影契约速查

| 场景 | 静态 | hover/按压 |
|---|---|---|
| 主按钮（SystemButton primary/danger） | `surfaceShadow` | `elevatedShadow` |
| accent 实底图标钮（SurfaceIconButton primary） | `surfaceShadow` | `elevatedShadow` |
| 未加入群组/未关注/发布等可点 accent 控件 | `surfaceShadow` | `elevatedShadow` |
| FAB | `elevatedShadow`（对齐前端 `--cst-elevation-fab` 中性投影） | — |
| 白卡/统计卡/徽章/**列表分组容器**（ListGroup / SettingsSectionCard） | `surfaceShadow` | — |
| 列表行（分组内的行） | **无投影**（层级由分组容器 + 发丝分隔线承担） | — |
| 选中胶囊（TabsPillGroup） | `controlShadow`（hover 增量走底色） | — |
| 输入框聚焦 | `surfaceShadow` + 边框加深（不画光晕） | — |
| 菜单/浮层（NotificationCenter、sheet 内浮钮） | `controlShadow` | — |
| 静态内容卡（登录卡/表单区块/提示 banner/空态卡/骨架卡） | `surfaceShadow` | — |

> **档位纪律（2026-10-11 修正）**：`elevatedShadow`（radius 24~44 / offsetY 8~12）**只允许**
> 出现在 FAB、hover/按压抬档、以及桌面外壳（DesktopLayout 侧栏/主面板）三类场景。
> 历史实现里 `AppThemeVisuals` 把「白卡/hero 卡/列表行/统计卡」的静态投影也取成了
> `elevatedShadow`，是「所有页面内容都像悬浮卡片」的根因；已全部收敛到 `surfaceShadow`。

---

## 9. 模糊（`--cst-blur-*`）

`sm 8`（轻徽章/浮层）/ `md 16`（sticky header/overlay）/ `lg 24`（底部导航/tablet dock）。

鸿蒙映射系统 `BlurStyle` 枚举（平台原生，非数值 token）：`BlurStyle.Thin` ↔ sm、
`BlurStyle.Regular` ↔ md、`BlurStyle.Thick` ↔ lg（当前 SDK 的 BlurStyle 枚举无
`Medium` 档，强档一律用 `Thick`）。

---

## 10. 动效与交互反馈

### 10.1 时长与曲线（`ANIMATION`，禁用裸 `duration: N`）

- 时长：`DURATION_FAST 150` / `DURATION_SHORT 200` / `DURATION_MEDIUM 300` /
  `DURATION_CONTENT_SWITCH 400` / `DURATION_SHEET 320`（底部弹层统一）/
  `DURATION_250` / `DURATION_280` / `DURATION_SLOW 1500`（脉冲/shimmer）/
  `DURATION_STAGGER_DELAY 150`。
- 曲线：`CURVE_EASE_IN_OUT` / `CURVE_EASE_OUT` / `CURVE_FAST_OUT_SLOW_IN` / `CURVE_SMOOTH`；
  前端曲线 `standard (0.22,1,0.36,1)` / `bounce (0.34,1.56,0.64,1)`。
- Toast/震动等行为时长保留字面量并注释（非视觉 token 范畴）。

### 10.2 交互反馈（`INTERACTION`，禁用裸 `scale(...)` / 裸 curve）

按压强度用 `SCALE_PRESSED_*`（对齐前端 `tap-*`/`hover-btn` 契约，数值为 HarmonyOS 平台适配档）：

- 主按钮按压 0.95 ↔ 前端 `hover-btn` tap 0.94；行内图标 0.90 ↔ `scale-btn` tap 0.94
- 卡片 0.97 ↔ `tap-97`；瀑布流卡 0.96 ↔ `tap-96`（前端按页微调档同理）
- 输入行 0.98 ↔ `tap-98`；胶囊/筛选 0.94 ↔ `tap-94`
- hover 抬升用 `HOVER_ELEVATION`（ArkUI elevation，对应前端 fine-pointer hover 缩放），
  弹簧动效用 `SPRING_RESPONSE_*` / `SPRING_DAMPING_*`（对应前端 `--cst-ease-bounce`）

---

## 11. 透明度（`OPACITY`）

`DISABLED 0.38` / `PRESSED 0.72` / `LOADING 0.58` / `SUBTLE 0.85` / `MUTED 0.5` / `EMPHASIZED 0.8`
（对齐前端 opacity-50/80 标准档；FADED 0.4 档已归并删除）。

---

## 12. 组件与布局尺寸（禁止裸 `width(N)` / `height(N)`）

- 头像 `AVATAR_SIZE.{XXS 22, XS 32, SMALL 40, SMALL_PLUS 44, MEDIUM 48, PROFILE 68, LARGE 80, XLARGE 96}`；
  按钮 `BUTTON_HEIGHT 42` / `BUTTON_HEIGHT_LG 48` / `ACTION_BUTTON_HEIGHT 28` /
  `ACTION_BUTTON_HEIGHT_MD 40` / `ACTION_BUTTON_HEIGHT_LG 34` / `SHEET_ACTION_HEIGHT 44` /
  `FILTER_CHIP_HEIGHT 30` / `LOGOUT_BUTTON_HEIGHT 50` / `PASSWORD_INPUT_HEIGHT 46`
- 输入框 `INPUT_HEIGHT 42`（通用表单）/ `SEARCH_INPUT_HEIGHT 39`（搜索框：对齐前端
  SearchInput 移动端实测 39.14 = py-2.5(10) + 行高 17.14 + border 2，2026-10-17 逐像素实测）；
  搜索框内图标与文字间距同前端 `gap-2.5`(10)
- 图标 `ICON_SIZE_2XS 14` / `ICON_SIZE_XS 16` / `ICON_SIZE_SM 20` / `ICON_SIZE_SM_PLUS 22` /
  `ICON_SIZE_MD 24` / `ICON_SIZE_LG 32` / `ICON_SIZE_EMPTY 48`（与 `FONT.SIZE` 解耦的独立
  ICON 刻度，图标尺寸一律走此档，不用字号 token）/ `STATUS_ICON_SIZE 28` /
  `LOADING_ICON_SIZE 20` / `LOADING_GLYPH_SIZE 46` / `BRAND_ICON_SIZE 38`；
  容器 `EMPTY_ICON_SIZE 64` / `LOADING_CONTAINER_SIZE 72` / `BANNER_HEIGHT 140` /
  `SKELETON_CARD_HEIGHT 100` / `PLACEHOLDER_AREA_HEIGHT 120` / `IMAGE_PLACEHOLDER_HEIGHT 110` /
  `SHARE_CARD_HEIGHT 130` / `LOG_AREA_HEIGHT 300` / `PROGRESS_STROKE_WIDTH 4`
- 布局 `LAYOUT.{DESKTOP_HEADER_HEIGHT 52, DESKTOP_SIDEBAR_WIDTH 224, RIGHT_SIDEBAR_WIDTH 220,
  MOBILE_BOTTOM_NAV_HEIGHT 56, TABLET_NAV_MAX_WIDTH 680, FAB_SIZE 52, ICON_BUTTON_SIZE 36,
  LABEL_WIDTH 60, HOME_COLUMN_WIDTH 420, FILTER_PANEL_WIDTH 180}`（GLOW_BLOB_* 光晕尺寸已随 glow 体系删除）
  > `TABLET_NAV_MAX_WIDTH` 文档曾写 520，与代码（680）及前端 `--cst-tablet-dock-max: 42.5rem`(680px)
  > 不一致，2026-10-12 核对代码后修正为 680。
- 其他布局尺寸：`BACK_BUTTON_SIZE 42`（文档曾写 40，2026-10-18 核对
  `DesignConstants.ets LAYOUT.BACK_BUTTON_SIZE=42` 与前端批次 3「页面级返回键统一 42」
  后修正；`SystemPageHeader` 内一处过期注释「40」同批修正）/ `MIN_HIT_TARGET 44`（触控热区底线，见 skill
  `mobile-ui-conventions`：官方底线 ≥40vp、推荐 46）/ `UNREAD_BADGE_SIZE 20` /
  `COVER_HEIGHT 220` / `NICKNAME_MAX_WIDTH 56` / `LOGIN_CARD_MAX_WIDTH 400`
- **页面横向 gutter 与纵向节奏**：`PAGE_SHELL.{GUTTER_MOBILE 16, GUTTER_TABLET 24,
  GUTTER_DESKTOP 28}` 与 `PAGE_SHELL.{SECTION_GAP 24, SECTION_INNER_GAP 12,
  PAGE_TOP_GAP 16}` —— **语义与强制规则见第 5.1 / 5.2 节**，此处仅登记存在，
  数值不重复维护（两处重复必漂移）
- 聊天 `CHAT.{BUBBLE_PADDING_H 16, BUBBLE_PADDING_V 10, IMAGE_BUBBLE_PADDING_H 14,
  IMAGE_BUBBLE_PADDING_V 6, IMAGE_MAX_WIDTH 200, ...}`
- 页面内容栏档位（对齐前端 `.page-shell`）：`form 560` / `narrow 720` / `default 1180` /
  `wide 1320`；feed：`feed-narrow 640` / `feed-medium 960`

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
| `forms/SearchInput` | `SystemInput`（通用单行输入）/ `SearchPageHeader`（页头内联搜索） | 搜索输入（原表写的 `SmartSearchComponent` 已不存在，2026-10-12 修正） |
| `navigation/TabBar` / `NavTab` / `RailIcon` | `TabsPillGroup`（默认等宽分段；`freeWidth` 形态对齐前端胶囊行，`TabItem.count` 对齐角标） / `PhoneLayout` 底部导航 | 分类/导航 Tab |
| `list/ListContainer`（bordered / embedded） | `components/ListGroup`（`bordered` 开关）+ `ListRowDivider` / `ListItemGroup.divider` | 列表**分组容器**：唯一一层白卡外框 + 行间发丝分隔线；行本身禁止再叠边框/圆角/投影（同前端 ListContainer 契约） |
| `list/ListItem` | `SettingsEntryRow` / `SettingsToggleRow` / `GroupMemberRow` 等行组件 | 列表行：只写 padding + 按压反馈，不写背景/圆角/投影 |
| `overlays/SheetShell` / `ui/dialog` | `SystemDialog` / `bindSheet` | 弹层/抽屉 |
| `forms/TagPill` | `TabsPillGroup` chip / `FILTER_CHIP_HEIGHT` | 标签/筛选胶囊 |
| `data-display/EmptyState` | `EmptyState` | 空列表态（同名同契约；`fillMode` 对齐前端 fill 模式） |
| `data-display/ListEnd` | `components/ListEnd` | **分页到底收尾提示**（「没有更多了」）。分页列表内容耗尽**必须**渲染，否则用户无法区分「已到底」与「加载卡住」 |
| `data-display/LoadError` | 无独立组件 → `EmptyState` + `actionLabel:'重试'` + `onAction` | 加载失败态。**语义等价**（同为「图标+文案+重试按钮」），故不另造组件；契约要求「失败必须展示错误并提供重试，禁止静默转空态」——实测 9 个失败分支全部满足 |
| `data-display/InlineLoading` | 无独立组件 → `LoadingMoreIndicator`（分页）/ 行内 `LoadingProgress`（局部） | 局部加载态。分页场景一律走 `LoadingMoreIndicator`（含向上分页，如聊天「加载更早消息…」）；行内指示用 `LoadingProgress` |
| `data-display/EmptyState` 整页加载 | `SystemLoadingView` / `Skeleton` / `SkeletonShimmer` | 整页/面板加载与列表骨架 |
| `data-display/MetricRow` / `HeroStats` | `HomePage.buildHeroStats`（页面内 @Builder） | 统计段（原表写的 `StatSegments` 已不存在，2026-10-12 修正；目前只有首页一处消费，暂不抽独立组件） |
| `routes/moments/shared/MomentCard` | `MomentFeedCard` | 动态卡片 |
| `auth/LoginPromptSheet` | `LoginPromptView` | 未登录引导 |
| `messages/ChatView` 等 | `ChatPage` / `ChatPageHeader` | 聊天界面 |
| `routes/(app)/nearby/+page.svelte`（附近聚合） | `features/nearby/pages/NearbyPage` | 附近聚合页（tab 式：附近的人/附近群组/附近动态三分区，各分区拉取并渲染内容； 2026-10-13 更新描述——原「纯导航门户」描述已过期）。手机/平板底栏「附近」tab |
| `data-display/NearbyImageCard` | `components/NearbyImageCard` | 图片优先卡（3:4 图 + 覆盖层徽章），附近用户/群组卡共用 |
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
| 跨端色彩 | `frontend/scripts/check-cross-platform-colors.mjs` | `themes/*.css --cst-palette-*` ↔ `ThemeConstants.ets ColorConfig` 逐值一致（3 主题 × 2 明暗 × 14 组映射）+ 主题 swatch |
| 跨端刻度 | `frontend/scripts/check-cross-platform-scale.mjs` | 字号 9 档（`--text-*` ↔ `FONT.SIZE`）+ 圆角 5 档（`--radius-*` ↔ `RADIUS`）逐值一致；单端加近邻档即拦截。已登记豁免：`FONT.SIZE.LG=16`（HarmonyOS 平台适配档） |

两脚本均挂入 `bun run check:theme`（随 `bun run check` 与 CI 生效），漂移即红。

---

## 15. 已登记例外与平台适配

- `FONT.SIZE.LG = 16`：HarmonyOS 标准正文尺寸，平台有意差异（跨端刻度脚本豁免项）。
- 明暗切换：鸿蒙 `DarkModePreference` 三档（auto/light/dark），web 端可手动切换；
  主题族（vibrant/forest/inkwash）两端可选集一致。
- 布局几何（width/height/gap 的 px）：非主题风格，不受 token 约束。
- 主题选择器 swatch：静态展示目标主题 accent。**当前实现已改为引用 token**
  （`ThemeConstants.VIBRANT_LIGHT_COLORS.accentBackground` 等，见 `SettingsPage` 主题卡），
  不再写字面 hex —— 此处曾描述为「保留字面 hex」，2026-10-12 核对代码后修正。
  对应值 vibrant `#2B5EB2` / forest `#3D7A55`（见 `ThemeConstants`），须与 themes 同步。
- `figma_neu` 主题：鸿蒙存量兼容项，非双端对齐目标。

---

## 16. 氛围画布（Ambient canvas，前端装饰层）

**目的**：解决桌面宽屏 / 内容稀疏 / 空态页「大面积纯白，看起来空荡荡」的观感问题。
借鉴游戏场景的分层做法给页面补「世界感」纵深，而不是往页面里堆内容。

### 16.1 分层结构

实现：`frontend/src/components/layout/ambient/AmbientBackdrop.svelte`（纯装饰层）。

| 层 | 实现 | 作用 |
|---|---|---|
| 天光 sky / glow | 渐变 div | 顶部天光 + 底部中央「地平线主光源」（世界光源） |
| 地平线雾 horizon | 底部渐隐带 | 把远景剪影与画布衔接 |
| 主题世界 world | 单套 SVG（按当前主题只渲染一套） | 见 16.2 |
| 暗角 vignette | 径向渐变 | 边缘收束，聚焦正文 |

### 16.2 主题语言（必须三套，不得共用一套图形）

| 主题 | 世界语言 |
|---|---|
| vibrant 星际 | 星尘（确定性伪随机）+ 三段星座连线 + 双层轨道环 + 城市天际线剪影（远近两层） |
| forest 清野 | 等高线地形（9 条柔顺曲线）+ 暖阳光斑 + 针叶林剪影（远近两层） |
| inkwash 水墨 | 三段墨山 + 朱砂圆相（enso，描边留缺口）+ 飞白墨点 |

色值全部来自 `themes/*.css` 的 `--cst-ambient-*` token（3 主题 × 明暗，共 12 个）：
`glow-1/glow-2`（天光/地平线光源）、`sky`（顶部天光）、`line`（星座/轨道/等高线）、
`dot`（星尘）、`sun`（清野暖阳）、`ink-1/2/3`（远/中/近剪影）、`emblem`（圆相/墨点/徽记）、
`horizon`（地平线雾）、`vignette`（暗角）。新增主题必须补齐全套（`check-theme-parity` 强制）。

### 16.3 挂载点（全站统一，页面不得自建）

- **外壳内容区**：`DesktopLayout`（`.desktop-main-panel` 内）、`TabletLayout`、
  `MobilePortrait`、`MobileLandscape` 各自内容区各挂一次——氛围层固定于可见内容区，
  **不随路由滚动**；`PageViewport` 底色改为透明以透出装饰层（画布底色由外壳提供：
  桌面白浮卡 / 移动·平板 `surface-page`）；
- 认证页（`routes/(auth)/login`）、错误页（`routes/+error.svelte`）；
- 启动占位（根布局 BootSplash 外层）：开屏即「世界 + 徽记」，接近游戏加载画面。

> 历史说明（2026-10-22 同日迭代）：最初挂在 `PageViewport` 内并尝试 `position:sticky`，
> 但页面视口高度固定为视口高、内容溢出其外，sticky 无活动空间 → 背景会随滚动消失；
> 改为外壳挂载后背景常驻。

### 16.4 硬约束

- 装饰层必须 `aria-hidden="true"` + `pointer-events:none`；挂载容器 `relative isolate`，
  层自身 `z-index:-1`（压在画布底色之上、正文之下），不参与布局 / 交互 / 焦点 / 读屏；
- 不允许页面内手写第二套背景装饰（新页面自动继承画布；确需场景化空态时另立组件评审）；
- 动效只用 `transform` / `opacity` 合成动画，循环时长走 `--cst-motion-ambient*`（24s/36s）；
  `prefers-reduced-motion: reduce` 下全部停用；
- **不引入第二整层底色**：装饰层是透明渐变/图形叠加，不改变「`--cst-surface-page` 是唯一
  整层底色」契约（见第 3 节与 `hotspot-frontend-theme` skill）；
- 世界层固定在外壳内容区（不随路由滚动），形成纵深视差；全画布页（消息分栏 / 地图）
  由自身不透明画布覆盖，氛围不可见属预期；
- **窄屏构图修正（≤640px）**：世界层经 `xMidYMid slice` 裁切时右侧母题会出画，手机档把
  星际轨道环/星座、水墨朱砂圆相平移到画面中部并缩小（`utilities.css` 媒体查询）；
  新增主题的世界层若含右侧母题，必须同步补窄屏构图。

### 16.5 已知边界

- 鸿蒙端尚未同步该视觉：需要 ArkUI 装饰层实现 + 三主题 token 评审，单独立项；
- 若后续觉得氛围过淡/过浓，调 `--cst-ambient-*` 的 alpha 即可全站生效（不改组件）。
