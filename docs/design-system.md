# HotSpot 设计系统

> 单一事实源（Single Source of Truth）。本文件从 Svelte Web 前端（`frontend/`）的
> `--cst-*` token 体系整理而来，覆盖**桌面端 / 平板端 / 移动端**三档设备形态，
> 并给出前端（CSS token）↔ 鸿蒙（`DesignConstants` / `ThemeConstants`）的逐项映射，
> 作为 `harmonyapp/` 对齐的权威基准。
>
> 改色、改字号、改间距、改圆角、改阴影、改动效时，两端必须同步；漂移由
> `frontend` 的 `bun run check:theme`（`scripts/check-cross-platform-colors.mjs`）在 CI 拦截。

---

## 1. 设计原则

1. **好看**：效果克制、有节奏，不堆叠效果。
2. **交互状态一眼可辨**：hover / focus / active / disabled 四态差异长在元素本体上。
3. **风格统一**：复用既有原语，禁止每页自造交互样式。
4. **风格走 token**：所有风格类变化（颜色、阴影、圆角、字号、字距、行高、透明度、
   边框粗细、动效时长）必须落在 token 体系；页面只消费语义 token，禁止内联硬编码。
5. **少档位 + 强制收敛**：1–2px 近邻档位一律归并；「不够用就归到最近档位」是默认动作。

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

双主题族 × 明暗模式，共 4 套色板。前端默认主题 = **vibrant（星际）**，鸿蒙 `DEFAULT_THEME_MODE` 同为 `vibrant`。

### 3.1 原始色板（primitive `--cst-palette-*`）

| Token | Vibrant 亮 | Vibrant 暗 | Forest 亮 | Forest 暗 |
|---|---|---|---|---|
| `page-bg` / `bg` | `#F7FAFF` | `#0B1424` | `#F6F7F3` | `#161A16` |
| `workspace-bg` | `#EBF1FA` | `#060D19` | `#EBEEE5` | `#0F130E` |
| `text` | `#16233A` | `#E8F1FF` | `#152219` | `#E8F1E9` |
| `text-muted` | `#52647F` | `#A5B6CE` | `#3F5547` | `#A8BDAE` |
| `accent-primary` | `#2E66C0` | `#69A8FF` | `#3D7A55` | `#52B878` |
| `on-accent` | `#FFFFFF` | `#071426` | `#FFFFFF` | `#0C1911` |
| `online` | `#14B8A6` | `#37C99A` | `#3D7A55` | `#52B878` |
| `hot`（危险） | `#F43F5E` | `#FB7185` | `#B4543E` | `#D97C5A` |
| `warning` | `#F59E0B` | `#FBBF24` | `#8A6A2F` | `#C4A75A` |
| `map-user` | `#2E66C0` | `#69A8FF` | `#3D7A55` | `#52B878` |
| `map-group` | `#0B7A56` | `#34D399` | `#4A703F` | `#88B85A` |
| `map-moment` | `#C2470B` | `#FB923C` | `#8A6426` | `#C4A75A` |

> 派生档位（`-bg` 淡底 / `-border` 淡边框 / `-glow` 光晕）由 `color-mix(in srgb, 主色 12~14%, transparent)`
> 统一生成，禁止组件/数据层再写 `color-mix`。

### 3.2 语义层（semantic `--cst-*`，跨主题不变）

| 语义 token | 指向 | 用途 |
|---|---|---|
| `--cst-text-primary` / `-secondary` | `palette-text` / `-muted` | 正文 / 辅助文字 |
| `--cst-text-on-accent` | `on-accent` | accent 底上的文字 |
| `--cst-surface-primary` / `-page` / `-workspace` | `bg` / `page-bg` / `workspace-bg` | 表面 / 画布 / 舞台 |
| `--cst-accent-default` / `-soft` / `-border` | `accent-primary` / `accent-soft` / `accent-soft-border` | 品牌 / 淡底 / 淡描边 |
| `--cst-border-default` / `-strong` | `border` / `border-pressed` | 常规 / 强分隔 |
| `--cst-status-online/danger/warning` | `online` / `hot` / `warning` | 状态色 |
| `--cst-marker-user/group/moment` | `map-*` | 地图标记 |

> 鸿蒙侧对齐方式：`ThemeConstants.ets` 的 `ColorConfig` 是同一套色板的两端表达
> （`forest ≡ default_clean`，`vibrant ≡ VIBRANT`）。`figma_neu` 为鸿蒙存量兼容项，**不作为双端对齐目标**。

---

## 4. 字体系统

### 4.1 字号刻度（`--text-*`）

| 档 | 前端 | 鸿蒙 `FONT.SIZE` | 用途 |
|---|---|---|---|
| xxs | `10` | `XS2` | 极小 uppercase 标签（eyebrow、分组标） |
| xs | `11` | `XS` | 辅助文字（副标题、计数、时间戳） |
| sm | `12` | `SM` | 正文 |
| md | `14` | `MD` | 强调正文（昵称、按钮主标） |
| lg | `18` | `LG_PLUS` | 小标题 |
| xl | `20` | `XL` | 标题 |
| 2xl | `24` | `XXL` | 大标题 |
| 3xl | `30` | `XXXL` | 强调大数字/节标题 |
| 4xl | `36` | `HERO` | 错误码/超大数字（前端最大 text 档） |

> **鸿蒙平台适配例外**：`FONT.SIZE.LG = 16` 为 HarmonyOS 标准正文尺寸（web 端正文 12/14 偏小），
> 是唯一多出的档位，两端有意不同，不属于漂移。
> 近邻档（13/15/22/28/32/48/56 等）已按「少档位」原则归并删除，禁止再引入。

### 4.2 字重 / 行高 / 字距

- 字重：`400 / 500 / 600 / 700`（`REGULAR / MEDIUM / SEMIBOLD / BOLD`）。
- 行高：`card 1.55`（卡片摘要）/ `copy 1.75`（正文阅读），其余用平台默认档。
- 字距：`eyebrow 0.14em` / `label 0.12em` / `field 0.08em` / `title -0.01em`
  （鸿蒙 `LETTER_SPACING` 按 10px 基准折算为 `1.4 / 1.2 / 1.28`）。

---

## 5. 间距

4px 网格 + 微步，**不强制收敛**（间距的 2px 差异是布局设计的一部分）：

`2 / 3 / 4 / 6 / 8 / 10 / 12 / 14 / 16 / 20 / 24 / 32`
（鸿蒙 `SPACING.HALF/THREE/XS/SM2/SM/MD2/MD/MD3/LG/LG2/XL/XXL`）

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

语义形状（`SHAPE`）：`CARD 20 / HERO_CARD 24 / COMPACT_CARD 16 / INPUT 16 / BUTTON FULL / CHIP FULL / TAB_BAR 24 / PREVIEW_CARD 20 / LIST_ITEM 20 / ICON_TILE 12 / SHEET_TOP_RADIUS 28`。

---

## 7. 边框宽度（`--cst-border-width-*` / `BORDER_WIDTH`）

`hairline 1 / regular 1.5 / strong 2 / heavy 3`

---

## 8. 阴影与层级

两套投影语言：

- **双向凸起/凹陷**（控件/卡片）：`raised sm 4/4/9`、`md 5/5/12`、`lg 8/8/18`；`inset sm/md`。
- **Material 单向/弹层**（语义 token）：`card / button / modal / sheet / menu / nav / fab`，
  真值在各主题族 CSS 中定义；vibrant 显影、forest 透明（改由 border 表达层级）。

鸿蒙侧经 `ThemeShadowRegistry` 消费同一语义（surface/elevated/accent/control/tabBar），
`#RRGGBBAA` 由 `withAlpha()` 生成。

---

## 9. 模糊（`--cst-blur-*`）

`sm 8`（轻徽章）/ `md 16`（sticky header/overlay）/ `lg 24`（底部导航/tablet dock）。
鸿蒙映射系统 `BlurStyle.Thin / Regular / Thick`。

---

## 10. 动效（`--cst-motion-*` + `--cst-ease-*`）

- 时长：`instant 100 / rapid 150 / snappy 180 / standard 220 / relaxed 260 / spring 300 / slow 400 / gentle 500 / spinner 700 / pulse 2400`。
- 曲线：`standard (0.22,1,0.36,1)` / `bounce (0.34,1.56,0.64,1)`。

鸿蒙 `ANIMATION` + `INTERACTION`（`SCALE_PRESSED_*`、`SPRING_RESPONSE/DAMPING_*`）按平台适配档表达同一契约。

---

## 11. 透明度

`none 0 / faint .30 / subtle .40 / muted .50 / dim .60 / soft .70 / emphasis .80 / tap .84 / hover .90 / full 1`；
鸿蒙 `OPACITY` 收敛到 `DISABLED .38 / PRESSED .72 / LOADING .58 / SUBTLE .85 / MUTED .5 / EMPHASIZED .8`。

---

## 12. 布局尺寸（`--cst-*` / `LAYOUT` + `PAGE_SHELL`）

| 尺寸 | 前端 | 鸿蒙 |
|---|---|---|
| 桌面侧边栏宽（展开/折叠） | `224 / 72` | `DESKTOP_SIDEBAR_WIDTH 224` |
| 桌面顶栏高 | `52` | `DESKTOP_HEADER_HEIGHT 52` |
| 平板浮动 Dock 最大宽 | `680`（`--cst-tablet-dock-max`） | `TABLET_NAV_MAX_WIDTH 680` |
| 移动端底栏高 | `56` | `MOBILE_BOTTOM_NAV_HEIGHT 56` |
| 封面高（sm / 默认 / lg） | `clamp 120–160 / 220–340 / 220–380` | — |
| 页面横向 gutter | `20 → 24(≥600) → 28(≥1280)` | — |
| 内容栏档位 | `form 560 / narrow 720 / default 1180 / wide 1320` | `PAGE_SHELL.*` |
| feed 档位 | `narrow 640 / medium 960` | `FEED_NARROW_MAX / FEED_MEDIUM_MAX` |

---

## 13. 组件原语与跨端映射

| 前端（Svelte） | 鸿蒙（ArkTS） |
|---|---|
| `StandardPageLayout` / `SplitPageLayout` / `ContentColumn` | `PageShellContainer` |
| `PageHeader` | `SystemPageHeader` / `SearchPageHeader` / `ChatPageHeader` |
| `MainLayout`（desktop/mobile/tablet） | `DesktopLayout` / `PhoneLayout` / `ResponsiveContainer` |
| `list/ListItem` + `ListContainer` | `SystemCard`（listItem 变体）/ `SettingsEntryRow` |
| `forms/SearchInput` | `SearchPageHeader` / `SmartSearchComponent` |
| `navigation/TabBar` / `NavTab` | `TabsPillGroup` / `PhoneLayout` 底部导航 |
| `shared/Avatar` | `SystemAvatar` |
| `data-display/EmptyState` | `EmptyState` |
| `data-display/MetricRow` / `HeroStats` | `StatSegments` |
| `forms/TagPill` | `TabsPillGroup` chip |
| `overlays/SheetShell` / `ui/dialog` | `SystemDialog` / `bindSheet` |
| `routes/moments/shared/MomentCard` | `MomentFeedCard` |
| `messages/ChatView` | `ChatPage` / `ChatPageHeader` |
| `auth/LoginPromptSheet` | `LoginPromptView` |

---

## 14. 跨端 token 对齐校验

- 色彩：`frontend/scripts/check-cross-platform-colors.mjs`（`bun run check:theme`），
  覆盖 2 主题 × 2 明暗 × 11 组映射，CI 漂移即拦截。
- 布局/字号/圆角等：本文件第 4–13 节为权威对照表，改动时人工比对两端。

---

## 15. 已登记例外与平台适配

- `figma_neu` 主题：鸿蒙存量兼容项，非双端对齐目标。
- `FONT.SIZE.LG = 16`：HarmonyOS 标准正文尺寸，平台有意差异。
- 明暗切换：鸿蒙跟随系统，web 端可手动切换；主题族（vibrant/forest）两端可选集一致。
- 布局几何（width/height/gap 的 px）与 StatusBar 模拟 chrome 微圆角：非主题风格，不受 token 约束。
- 主题选择器 swatch：静态展示目标主题 accent，保留字面 hex（vibrant `#2E66C0` / forest `#3D7A55`），须与 themes 同步。
