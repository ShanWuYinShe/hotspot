---
name: hotspot-harmony-ui
description: hotspot 鸿蒙端 UI/UX 与组件原语契约：设计 token 强制规则、跨端同步机制、SystemPageHeader/PageShellContainer、路由对齐、SystemButton/SystemInput 原语与已登记例外清单。触发：修改 harmonyapp 任意 UI 或样式；新增鸿蒙页面；使用设计 token 或颜色字号间距；手写 Button/TextInput 前确认是否该用原语；跨端色彩/刻度同步。
---

# hotspot-harmony-ui

## UI/UX 设计规范

设计 token 的**全部数值档位**（颜色/字号/间距/圆角/边框/动效/交互反馈/透明度/尺寸/阴影/模糊）、双端共享组件映射表与跨端投影契约速查的唯一权威是 [docs/design-system.md](../../../docs/design-system.md)（仓库根 docs/），改 UI 前必读。本节只保留强制规则与指针。

- **禁止硬编码**：颜色一律使用 `AppThemeManager.getPaletteById(resolvedThemeId)` 的 palette token（或 `DesignConstants` 系统/图片固定色），禁止硬编码色值；字号/间距/圆角/边框/动效/透明度/尺寸一律用 `DesignConstants` 对应 token 档位，禁止裸 `fontSize(N)` / `padding({left:N})` / `width(N)` / `duration: N` / `scale(...)`；`$r('app.color.*')` 资源仅用于启动窗口等系统场景（当前 color.json 只保留 start_window_background）。
- **图标尺寸用专用刻度（强制）**：图标尺寸一律 `COMPONENT.ICON_SIZE_*`
  （`2XS 14 / XS 16 / SM 20 / SM_PLUS 22 / MD 24 / LG 32 / EMPTY 48`），
  **不要用 `FONT.SIZE.*` 当图标尺寸**——那是字号刻度，混用会让图标尺寸随字号变动、
  并产生 9 种图标尺寸（实测 2026-10-12：`iconSize` 有 133 处取自 FONT.SIZE、仅 17 处用专用刻度）。
  - **唯一例外（明示约定）**：图标**与同档文字行内并排**时（如聊天气泡的「已编辑」行、
    群卡 meta 行的图标），允许 `iconSize: FONT.SIZE.X` 与该文字同档——这是刻意对齐，
    不是漂移。判断口径：该处 ±14 行内存在 `fontSize(...FONT.SIZE.同档)`。
  - 已登记未迁移：`LG_PLUS(18)` 11 处（行首图标）、`HERO(36)` 2 处（hero/地图图标）
    ——专用刻度无 18/36，取近档会改像素，待设计定档。
- **行高（多行正文强制用比例制）**：`FONT.LINE_RATIO.{DEFAULT 1.5, CARD 1.7, COPY 1.75}` +
  `lineHeightFor(fontSize, ratio)`，对齐前端 `--leading-*`（`card` 是前端「原 1.55，
  阅读疲劳修复」后的值）。适用：聊天气泡/卡片正文 `CARD`；简介、公告、动态正文等长文 `COPY`。
  `FONT.LINE_HEIGHT` 的绝对档（XS18/SM20/MD22/LG24/XL26）**只用于单行紧凑场景**——它与字号
  解耦，把 `LINE_HEIGHT.SM`(20) 配 `SIZE.LG`(16) 只有 1.25 倍。实测 2026-10-12：多处
  误用导致比值散落 1.57~1.83，且聊天气泡/群公告**完全没有行高**（长文本挤成一团）。
- **跨端同步机制（强制）**：前端与鸿蒙共享同一套设计 token 刻度，同步由 CI 脚本强制拦截，**改两端任一 token 必须同步改另一端**：

  | 校验 | 脚本 | 覆盖范围 |
  |---|---|---|
  | 跨端色彩 | `frontend/scripts/check-cross-platform-colors.mjs` | `themes/*.css --cst-palette-*` ↔ `ThemeConstants.ets ColorConfig` 逐值一致（3 主题 × 2 明暗 × 14 组映射）+ 主题 swatch |
  | 跨端刻度 | `frontend/scripts/check-cross-platform-scale.mjs` | 字号 9 档（`--text-*` ↔ `FONT.SIZE`）+ 圆角 5 档（`--radius-*` ↔ `RADIUS`）逐值一致；单端加近邻档即拦截。已登记豁免：`FONT.SIZE.LG=16`（HarmonyOS 平台适配档） |

  两脚本均挂入 `bun run check:theme`（随 `bun run check` 与 CI 生效），漂移即红。
- **视觉结构同步约定**：前端 `components/home|messages|profile/` 下组件与鸿蒙 `components/`、`features/*/` 对应实现构成 design-system.md 第 13 节映射表——修改任一端的**共享组件**（改动视觉契约：布局/字号档/圆角档/投影/动效）时，必须检查映射表对端并同步，commit message 中注明「对齐 <前端组件/commit>」；仅平台适配差异（如 FAB 位置、安全区）可不同步但需在两端代码注释中互指。
- **页面头部（强制 SystemPageHeader）**：
  - 所有页面头部必须用 `SystemPageHeader`（或专用组件 `SearchPageHeader` / `ChatPageHeader`）；禁止手搓 `Row { Text(...).fontSize(XL)... Button(...) }` 头部。
  - 已登记例外（无标准页面头部）：登录页 `LoginPage`（无主壳）、主壳 `Index`（容器）、地图全屏 `LocationPickerPage`、图片全屏预览 `PhotoPreviewPage`、分享解析中间页 `ShareResolutionPage`（无内容 UI，仅 loading/错误/跳转）、调试面板 `DebugPanel`；`SettingsSupport.ets` 等仅导出页面子组件的文件不算页面。
  - SystemPageHeader API：`title` / `subtitle` / `showBackButton`+`onBack` / `action`(文本按钮) / `leftAction`+`rightAction`(IconAction) / `trailing`(@BuilderParam 任意尾随) / `titleAlign` / `titleSize` / `titleWeight` / `backgroundVariant`('bar'|'transparent'|'card')。
  - 安全区由 SystemPageHeader 内聚（自带 `@StorageLink('safeAreaTop')` + `expandSafeArea`），页面无需重复写。
  - 内联搜索框头部 → `SearchPageHeader`；浮动卡片+头像头部 → `ChatPageHeader`。
- **页面内容栏（PageShellContainer）**：强制使用；档位数值（form/narrow/default/wide + feed 两档）见 design-system.md 第 12 节。
- **页面横向 gutter（唯一来源，强制）**：横向留白**只能**由 `PageShellContainer` 提供（手机 16 / 平板 24 / 桌面 28，
  对齐前端 `--cst-page-gutter`；2026-10-13 手机档由 20 收紧至 16，对齐 iOS inset grouped 16pt /
  Material 16dp 主流下限，两端同步）。**禁止**双重留白（容器 gutter + 页面自加 left/right padding，
  会叠成 36~40，与只用容器 gutter 的页面差 16~20vp）；**禁止**用 `noHorizontalPadding: true`
  关掉容器留白再另写一套值。`SystemPageHeader` 只带纵向节奏、不自带横向 padding。
  - **通栏元素的坑（实测 2026-10-11）**：前端用 `-mx-[var(--cst-page-gutter)]` 抵消 gutter，ArkUI **不能照搬**
    ——子元素 `width('100%')` 相对父内容盒解析，负 margin 只位移不加宽（群详情封面加左右 −20vp
    后实测仍是 x=70..1186 / 屏宽 1256）。当前约定：封面/hero 与下方内容同宽；确需通栏要把整页改
    `noHorizontalPadding: true` + 各区块自加 gutter（已登记例外）。
  - **症状对照**：各页正文左边缘不在同一条竖线上、滑动时内容左右边界跳动、页头标题比正文多缩进一截
    ——都是本规则被违反的表现（实测于 2026-10-11：25 个页面里同时存在 16/20/24/32 四套留白）。
- **页面纵向节奏（强制）**：分区之间 `PAGE_SHELL.SECTION_GAP`(24) 用**底距**且**每个分区都要有（含首块）**；
  分区标题→其内容 `PAGE_SHELL.SECTION_INNER_GAP`(12)；页头→首块 `PAGE_SHELL.PAGE_TOP_GAP`(16)。
  - 24 : 12 = 2:1 是必需的层次比：分区之间不明显大于标题内距，标题就会飘在两块之间、整页读不出分组。
  - 用「上距」或漏给首块加底距，会让标题上方间距随各卡片自身 padding 变化，同类标题间距不一致
    （实测于 2026-10-11：首页三个分区底距曾混用 20/20/24，且首块统计卡完全没有底距）。
- **页头字号档统一（强制）**：全站 `SystemPageHeader.titleSize` 统一 `FONT.SIZE.XXL`(24)
  ——前端各页实测均为 `desktop ? 'xl'(20) : '2xl'(24)`，**无 3xl 用法**。
  禁止顶级 Tab 页用 XXXL(30)、二级页用 XXL(24) 的分档（同机切换页面标题大小会跳动，
  违反者实测于 2026-10-11：首页/附近曾用 XXXL）。
- **列表分组容器（ListGroup，强制）**：任何「可重复的同类条目集合」（设置项、关系列表、搜索结果、群组/分享/订阅/账号列表、公告列表）一律走
  `components/ListGroup`（`bordered` 默认 true = 开放页面区域的唯一一层分组外框卡）+ 行间
  `ListRowDivider` / `ListItemGroup.divider`，**禁止给每一行加边框/圆角/投影**（那就是「一行一张浮卡」，
  整屏看起来所有内容都在漂浮）。`bordered=false` 用于嵌入面板/Sheet/侧栏内的集合。
  - 判定依据（官方规范原文）：M3 Cards guidelines「Don't force content into cards when spacing,
    headlines, or dividers would create a simpler visual hierarchy」；M3 Lists「Use gaps for
    contained lists / Limit dividers to uncontained lists」；Apple HIG Layout「negative space,
    container shapes, or separator lines」三选一、HIG Materials「Don't use Liquid Glass in the
    content layer」；HarmonyOS `ListItemGroupStyle` 默认就是 NONE（无卡片），卡片是显式开关；
    NN/g「homogenous items → 标准竖排列表，不要卡片」。
  - **卡片只留给「单一主题、内容异构、可独立存在」的块**：动态卡（MomentFeedCard / MomentWallCard）、
    群组封面卡（NearbyImageCard）、统计/仪表盘块、空态卡、页面级 hero 卡。
  - 分组外框静态投影一律 `surfaceShadow`（白卡档），**禁止 `elevatedShadow`**（弹层/FAB 档）；
    行本身无投影、无边框、无圆角，只有按压填充（`surfaceSubtle`）。
  - 设置页专用分组卡 `SettingsSectionCard` 已按同一契约实现，新设置分组直接复用。
  - **横向留白按列表形态二分（实测 2026-10-11，踩过的坑）**：容器已提供 gutter 后，
    - **扁平列表**（无卡片外框，行直接铺在页面底色上，如消息会话列表）→ **行不得有
      `left`/`right` padding，列表也不得有横向 `margin`**；否则行内容比页头多缩进
      一个 gutter（实测消息页曾多 16~36vp，「消息」标题 x=73 而「最近」标签 x=109）。
      行间分隔线用 `startMargin: 0`。
    - **分组卡片**（`ListGroup`/`ListItemGroup` 带底色+描边+圆角，如群列表/订阅/账号管理）
      → **卡片去掉横向外边距**（与外层内容同宽），**行内 padding 保留**（那是卡片内边距）。
      卡片自带 `margin LG` 会让卡片比页头窄 32vp、卡内内容窄 64vp。
    - 页面级元素（退出登录按钮、搜索框、升级表单）同样不得自加横向 padding：
      实测个人中心升级表单自带 `XXL(32)` 叠加容器 gutter 后达 52vp。
    - 分组标题（`SettingsGroupLabel`）也不得自加 `left`，否则比分组卡多缩进（实测 91 → 70）。

### 页面路由对齐（web → harmony）

- 目录命名对齐前端路由域（复数）：`features/messages`（消息列表）/
  `features/moments`（瞬间）/ `features/groups`（群组）/ `features/me`（个人中心，
  同时承载前端 `/users/[userId]` 他人主页查看——同一 ProfilePage 复用，属平台例外）/
  `features/nearby` / `features/settings` / `features/search` / `features/map` /
  `features/auth` / `features/chat`（聊天窗口独立域）。
- 底栏 4 tab：首页 / 附近 / 消息 / 我的（`PhoneLayout` TAB_ITEMS + `AppNavigation.BOTTOM_TABS`，
  与前端 `routes/(app)/+page.svelte`、`/nearby` 聚合页对齐）。
- 桌面侧边栏：首页 / 附近瞬间（moments tab）/ 附近的人 / 附近群组 + 消息 / 通知 overlay /
  我的瞬间 / 我的群组 / 关系（对齐前端 desktop nav 的 home / nearby-moments /
  nearby-people / nearby-groups / messages / notifications / moments / my-groups /
  relationships）。
- 消息列表：排序 = 置顶优先 → 最新一条消息时间倒序（`last_message.sent_at` 优先，
  回退 `last_activity_at`），已读/未读不参与排序；列表头部有「全部 / 未读 / 已读」过滤
  （`TabsPillGroup`，对齐前端 messages 页 TabBar）；点击未读会话后徽章立即清零。
- 聊天记录：消息按 `sent_at` 正序渲染（越新越靠下，对齐前端 useChat 翻转）；
  跨天自动插入日期胶囊（`FormatUtil.formatDateLabel` → 今天/昨天/M月D日，
  对齐前端 groupMessagesByDay/formatDateLabel）。
- 动态详情：点赞/收藏/分享/评论（评论列表 + 乐观发布，对齐前端 MomentDetailPage）。
- 附近聚合页 `NearbyPage` 三区块「查看全部」分别 push `NEARBY_USERS` /
  `NEARBY_GROUPS` / `NEARBY_MOMENTS`（MomentsFeedPage 已注册为路由页，push 场景
  带返回按钮，对齐前端 `/nearby/people|groups|moments`）。
- 搜索页 Tab：全部 / 用户 / 动态 / 话题（话题从动态标签聚合，对齐前端搜索
  全部/用户/动态/话题）；群组 Tab 为鸿蒙端扩展（前端搜索无群组 Tab）。
  空查询显示发现态（最近搜索 / 正在关注 / 推荐话题，对齐前端 SearchDiscovery）；
  最近搜索持久化在 PreferencesUtil（键 `search_recent`，最多 5 条）。
- WS 事件类型与 spec/前端对齐：含 `GroupMessageReadCountUpdated` /
  `GroupMemberRoleChanged`（已登记类型与订阅，展示由群详情/聊天页按需刷新）。

## 组件原语强制（对齐前端 button/input 体系）

- 通用文本操作按钮必须用 `SystemButton`（variant=primary/secondary/outline/ghost/danger，
  surface/surface-danger，size=small/default/large/sheet，支持 loading/disabled/fullWidth）；禁止手写
  `Button('文案') + backgroundColor/fontColor/height/borderRadius` 重复样式。
- **危险操作分两档，别混用（实测 2026-10-12）**：
  - `danger`（**实底 error 色 + 白字**）只用于**最终确认**：二次确认弹窗的
    `SystemDialog.confirmVariant`、弹层里的确认键（如「封禁并移出」）。
  - `surface-danger`（染色底 + error 文字 + 发丝描边，安静档）用于**行级/列表内**
    的危险操作（账号行删除、消息/聊天操作表、退出群组、取消分享…）。
  - 反例：账号管理页每行一个实底 `danger`，眯眼测试下整页形成一列刺眼红色块，
    视线被破坏性操作抢走、观感像警报页。NN/g《Visual Hierarchy》：「Reserve warm bright
    colors, like red, for warnings or errors」——最强颜色档必须留给真正的警示。
- 通用单行输入框必须用 `SystemInput`（前缀图标/清除按钮/错误态）；登录/表单等
  密码框可用 `SystemInput({ passwordMode: true })`。
- **其余基础原语清单（唯一登记处，docs/design-system.md 第 13 节引用此处）**：

  | 原语 | 用途 / 关键约束 |
  |---|---|
  | `SystemTextArea` | 多行输入；高度用 `constraintSize({ minHeight })`，**不要固定 height** |
  | `SystemToggle` | 开关；视觉 50×28 固定，外层透明容器扩热区到 `MIN_HIT_TARGET` |
  | `SystemBadge` | 计数/状态徽章（含未读角标） |
  | `SurfaceIconButton` | 纯图标按钮；视觉尺寸可小，**热区必须 `max(视觉, MIN_HIT_TARGET)`**，且**必传 `accessibilityLabel`** |
  | `Pressable` | 通用可按压容器（统一 spring 按压反馈 + 缩放 token） |
  | `SoftDivider` | 发丝分隔线（对齐前端 divide 线） |
  | `LucideIcon` | 业务图标统一渲染（**组件名是历史遗留**——旧 lucide SVG 方案已废，现渲染 `SymbolSpan` + `sys.symbol.*`，与 SystemIcon 同机制；API 多 `paletteRole` 语义色角色）；尺寸用 `ICON_SIZE_*` 刻度 |
  | `SystemIcon` | 系统符号图标（`SymbolSpan` / `sys.symbol.*`），`AppIcon` 枚举定义处；API 直传 color |
  | `NativeMapComponent` | 原生地图组件（平台特有，前端无对应） |
- 已登记例外（可保留手写原生控件）：头像/图形按钮、地图全屏控件、搜索建议行、
  动态卡片内嵌操作、主题色板 swatch、扫描/贴纸等特殊控件。存量页面中的手写
  Button/TextInput 属遗留债务，改到该文件时优先迁移为原语。当前存量例外清单：
  - ProfilePage 外部主页/升级/编辑流程按钮（走 `getProfileVisualStyle()` 主题样式对象，
    已 token 化，不套 SystemButton）；
  - 创建页「移除」等 48vp 大 chip、群组详情「恢复原头像」ghost 小按钮；
  - SettingsPage「退出登录」（50vp 透明底定制）、DebugPanel 调试按钮；
  - ChatPage 隐藏键盘失焦占位 `Button('')`、LocationPicker 搜索栏内置 searchButton；
  - ProfileEditPage 多行简介输入辅助、ChatPage 输入栏（含发送钮 `Button('发送')` 属已登记例外 — 需 ChatTheme/getChatVisualStyle 定制动效与 isSendInteractive 动态态，见 ChatPage.ets 顶部“聊天发送钮属已登记例外”注释）、CreateGroupPage 人数步进输入。


## 主题/深色验收：量化口径与实测基线（2026-10-12 实测）

改主题、改颜色、或做深色验收时，**不要只看截图感觉**，用下面的量化口径跑一遍：

### 1) 文字对比度（WCAG 2.x）
取 `ThemeConstants.ets` 每个主题块的 `pageBackground / surfaceBackground / textPrimary / textSecondary / textTertiary`，
按 WCAG 相对亮度公式算比值，正文要求 **≥4.5:1**。本机实测基线（3 主题 × 2 明暗，全部达标）：

| 主题·模式 | primary | secondary | tertiary |
|---|---|---|---|
| 清野 浅/深 | 15.31 / 15.24 | 7.50 / 8.84 | 5.12 / 5.04 |
| 星际 浅/深 | 14.12 / 16.19 | 5.41 / 8.93 | 4.74 / 5.35 |
| 水墨 浅/深 | 12.84 / 13.99 | 4.82 / 7.02 | **4.68** / 5.20 |

最低值是水墨浅色的 tertiary 4.68 —— **改任何 text/bg token 后必须重跑，低于 4.5 即回归**。

### 2) 卡片与页面底色的层次：**不要用「底色明度差」判断**
前端 `semantic.css` 已明文规定：`--cst-surface-page` 是唯一整层底色，
**层级只靠「白卡 + 发丝描边 + 主题投影」表达**。因此清野主题下
`pageBackground == surfaceBackground`（浅色深色都是 `#F6F7F3` / `#161A16`）**是刻意设计，不是缺陷**，
不要为了「拉开层次」去改底色（会破坏跨端色彩校验）。

### 3) 深色下的层级靠**加亮的描边**承载
`ThemeConstants.ets` 注释原文：「深色下黑阴影对比弱，卡片抬升改用更亮的描边承载
（border 单独加亮），阴影只做轻微加深」。`borderColor` 是 `#AARRGGBB`，
验收时要算**与底色合成后**的颜色再求对比度（不能只看 alpha）。实测基线：

| 主题·模式 | 描边合成后 vs 底色 |
|---|---|
| 清野 浅 / 深 | 1.511 / 1.476 |
| 星际 浅 / 深 | **1.271（偏弱）** / 1.414 |
| 水墨 浅 / 深 | 1.463 / 1.553 |

经验阈值：**≥1.35 清晰、1.2~1.35 偏弱、<1.2 过弱**。星际浅色 1.271 偏弱属跨端一致的
既有设计（前端 `--cst-palette-border` 同值），**修要两端一起改**，不要只改鸿蒙端。


## 「固定高度」审计的正确判据（2026-10-12 实测，避免误报/误改）

本仓 skill 要求「固定高度容器（搜索框、菜单行、pill）改用最小尺寸约束，系统大字体时
要能随内容撑高而不是截断」。据此审计全仓 **81 处** `.height(...)` 且块内含 Text 的位置，
**逐一核实后判定为无缺口、不需要改动**。判据如下，下次别再重复怀疑：

| 类别 | 例子 | 判定 |
|---|---|---|
| **图形容器**（只含 Icon/头像/徽章，无文字） | `EMPTY_ICON_SIZE`、`ICON_SIZE_MD`、`AVATAR_SIZE.*`、`UNREAD_BADGE_SIZE`、`LOADING_ICON_SIZE`、`DIALOG_ICON_TILE_SIZE` | ✅ **固定尺寸正确**——图形不随字体缩放，改 minHeight 反而错 |
| **按钮 / 输入框** | `SystemButton`(BUTTON_HEIGHT*)、`SystemInput`(INPUT_HEIGHT)、`SearchPageHeader`(CHAT.INPUT_HEIGHT) | ✅ **固定高度正确**——前端同为固定高（`button.svelte` 用 `h-10`/`h-11`，全仓 `min-h-*` 仅 14 处且都是特殊场景）。控件固定高是跨端一致的既定契约 + 行业惯例，改成 minHeight 会破坏跨端一致 |
| **可撑高容器** | `SystemTextArea`(`constraintSize({minHeight})`)、`EmptyState`(`EMPTY_FILL_MIN_HEIGHT`)、`CreateMomentSheet` 多行输入 | ✅ 已用 minHeight，正确 |
| **热区扩展** | `TabsPillGroup`：视觉高度由内层固定、外层 `constraintSize({minHeight: MIN_HIT_TARGET})` 只扩命中区 | ✅ 正确范式，可作参考 |

**审计脚本的坑**：「块内 N 行含 Text」这种粗判会把**图标容器**误报为「文本容器」
（81 处里绝大多数是图标容器）——要按**该 height 所属元素自身**是否承载文字判断，
不要用「附近有 Text」。


## 交互反馈审计的判据（2026-10-12 实测，避免误报/误改）

项目原则要求「交互状态一眼可辨：hover / focus / active / disabled 四态差异长在元素本体上」。
按「可点元素链上是否有反馈」做全仓审计，得到 21 处候选，**逐一核实后判定全部为误报**——

原因：按压反馈**不一定写在 onClick 所在的那条属性链上**。以下都是合法承载方式：

| 承载方式 | 例子 | 说明 |
|---|---|---|
| **组件内部承载** | `SystemCard({ clickable: true })` 外层再 `.onClick` | 卡片组件内部用 `isPressed` → `SCALE_PRESSED_CARD` + spring 动画 |
| **状态驱动的样式** | `TabsPillGroup` 的 `.onClick` 只 dispatch | 反馈由 `hoveredIndex` / 选中态驱动背景与缩放（13 处相关代码） |
| **属性开关** | `SystemIcon` 的 `clickable` | 反馈由使用方通过该开关启用 |
| **聚焦态代替按压态** | `SystemInput` | 输入框交互靠 focus border 表达，**不需要**按压缩放 |
| **父级行承载** | 设置行的 `.onClick` | 行组件统一提供按压填充（`surfaceSubtle`） |

**结论：本仓交互原语（Pressable / SystemButton / SystemCard / SurfaceIconButton /
SystemToggle / TabsPillGroup / SystemInput / 设置行）的反馈机制完备，无需补齐。**

**审计脚本的坑**：用「onClick 所在属性链上是否出现 scale/onTouch/animation/backgroundColor」
做启发式，误报率极高（本次 21/21 误报）——因为反馈常在**组件内部**或**由状态间接驱动**。
要判断某处是否真缺反馈，必须**打开该组件与其使用上下文**看，不能只扫属性链。


## 性能审计基线（2026-10-12 实测，全部无缺口）

ArkUI 性能清单四项实测结果与复核方法（改列表/图片相关代码后按此回归）：

| 项 | 结果 | 复核方法 |
|---|---|---|
| LazyForEach/ForEach 键稳定性 | **85/85 稳定** | 键函数是 ForEach 块**收尾行**的 `(item) => 表达式`；逐一核对均为稳定 ID（user_id/moment_id/session_id/图片 url）。两类含 index 的属合法：静态 tab 配置（`\`\${tab.label}-\${index}\``）、骨架占位（`conv-skeleton-\${i}`，键=字符串项自身，HomePage 的 `(_, s) => s` 同理） |
| cachedCount | 12 处长列表全部配置（5~8） | grep cachedCount |
| @Reusable | 18 个组件（含 MomentFeedCard/UserCard/EmptyState 等高频卡） | grep @Reusable |
| Image 尺寸约束 | **20/20** 有 objectFit + 宽高/aspectRatio | 动态图全用 `.width('100%') + .aspectRatio() + objectFit(Cover) + alt + onError` 契约 |
| 事件生命周期 | **9/9 组件 on/off 配对完整**（2026-10-12 补测） | interactionEventHub 订阅组件 4 个 + ChatPage 9on/9off + ProfilePage 双事件双注销 + NativeMapComponent（清定时器/清聚合层/**release 全部 PixelMap**）；地图监听挂组件自有 controller 随组件销毁。新增订阅时必须在 aboutToDisappear 注销同 handler 引用 |

**审计脚本的坑（第三次验证同一规律）**：Image 约束检查报 3 处「缺尺寸」，实际都有
`.width('100%') + .aspectRatio()` —— 脚本只认数字 `.width(N)/.height(N)`。
启发式只产候选清单，判定必须人工开上下文；键函数要按「块收尾行」提取而非开头。


### 4) 主题族运行时矩阵：**6 组合已全部实测通过**（2026-10-12 补齐）

此前只有 vibrant 的明暗两态实测过，本日补齐 forest（清野）/ inkwash（水墨）× 明暗，
用模拟器在设置页逐组合切换截图确认：

| 组合 | 结果 |
|---|---|
| vibrant 浅/深 | ✅（此前多页实测） |
| 清野 浅色 | ✅ 绿调米白底 #F6F7F3、深绿选中 pill 白字、图标徽章绿调 |
| 清野 深色 | ✅ 绿调暗底 #161A16、**亮绿 pill + 深绿文字**（对比清晰）、徽章绿调 |
| 水墨 浅色 | ✅（同批截图确认） |
| 水墨 深色 | ✅ 暖黑底 #181A1C、珊瑚红 pill + 深色文字、徽章红调 |

选中的明暗分段 pill 在深色下均为「亮 accent 底 + 深 accent 文字」——与静态对比度基线
（第 1 节）一致。头像游客色不随主题变（身份色独立于主题 accent，业界惯例，非缺陷）。

## ArkUI 布局陷阱（实拍确认，2026-10-13）

两个真实踩过的渲染坑，都会造成「观感与代码意图不符」：

### 1. `.border()` 写在 `.borderRadius()` 之后会重置圆角为直角

- 现象：同一对并排按钮，无 border 的呈胶囊（borderRadius=999 生效），带 hairline border 的
  呈直角矩形——尽管两者引用同一个 borderRadius 值（SHAPE.BUTTON=RADIUS.FULL）。
- 根因：ArkUI 属性后设覆盖先设；`.border({width,color})` 自带的 radius 字段未显式给出
  时按 0 生效，把先前 `.borderRadius()` 的值覆盖掉。
- 修复：**`.border()` 一律写在 `.borderRadius()` 之前**（后设的 borderRadius 覆盖 border）。
- 排查工具：`.workwork/global-border-scan.py`（扫「.border( 紧随 .borderRadius(」模式，
  全仓曾报 66 处——是否真有视觉影响需逐例核实，部分元素本意即直角）。
- 实拍案例：ProfilePage 升级表单「取消」直角 vs「确认升级」胶囊（commit d05aaacc）。

### 2. Scroll 内容短于视口时默认垂直居中

- 现象：短表单在 `Scroll{...}.layoutWeight(1)` 里「飘」在页中段，页头下方出现大空隙
  （实measured：Scroll 高 597vp、内容 289vp，上下各留 154vp）。
- 根因：Scroll 对小于滚动区的内容默认居中对齐。
- 修复：表单类 Scroll 一律补 `.align(Alignment.Top)`（commit 1ae09938，升级表单）。
- 排查：dump 里 Scroll bounds 与其内容 Column bounds 的 y1 不一致即为居中。

## 已知布局坑（2026-10-19 端上实测）

- **裸 `{ }` 块不是容器**：在 `if/else` 分支里用裸块包裹布局（如把两栏 `if` 简化成
  `else { { Column() {...}.layoutWeight(1) } }`）时，块内 `.layoutWeight()` **不生效**，
  容器高度塌陷 → 内部 `WaterFlow`/`List` 高度为 0 → **页头（标题/计数/筛选）正常但内容一条不渲染**。
  类型检查与编译都不会报错，必须端上或预览器确认。修法：删掉裸块，容器直接作为分支体。
- 自查口径：任何"改完结构后内容区空白"的问题，先看分支体是不是裸块、`.layoutWeight()`
  是否直接挂在**容器组件**上（Column/Row/List/WaterFlow）。

## PageShellContainer 档位选择（2026-10-20 排查）

宽度已统一收口在 `PageShellContainer`（禁止页面各自 `maxWidth`，架构正确），但**档位要按内容类型选**，
选错会在平板/桌面只占中间一条：

| 档位 | 宽度 | 适用 |
|---|---|---|
| `form` | 560 | 表单/创建/设置单项（登录、发布瞬间、建群、编辑资料、设置各页）|
| `narrow` | 720 | 阅读型详情（瞬间详情）|
| `feed-narrow` | 640 | 单条内容页 |
| `feed-medium` | 960 | 内容墙/流（附近瞬间列表）|
| `default` | 1180 | 信息流/列表/枢纽/仪表盘（首页、消息、附近、群组、搜索、管理后台概览、订阅、公告、我的分享…）|
| `wide` | 1320 | 大表格/管理后台宽页 |

2026-10-20 修正：`SearchPage`、`AdminOverviewPage`、`MySharesPage`、`SubscriptionsPage`、
`AnnouncementsPage`、`RelationshipHubPage` 由 `form` 改为 `default`（它们不是表单）。

## 设计 token 对照 HarmonyOS 官方规范（2026-10-21 审计）

依据华为官方 UX/UI 规范（鸿蒙设计规范文档，经 DeepSeek 官方 tools/search 检索、tools/fetch 取原文核实）：
系统通用圆角 **16vp**；卡片之间间距 **12vp**；卡片内主标题 **16sp** / 副标题 **12sp**（90% / 60% 不透明度）；
默认鸿蒙黑体，单页不超过 2 种字体；卡片内最大字号 **24**。

本项目 `DesignConstants` 对照结果：

| 官方项 | 官方值 | 本项目 | 结论 |
|---|---|---|---|
| 卡片之间间距 | 12vp | `SPACING.MD = 12`（实测枢纽/首页卡片列 `space: SPACING.MD`） | ✅ 一致 |
| 系统通用圆角 | 16vp | `RADIUS.MD = 16`（`COMPACT_CARD` / `INPUT` 用它） | ✅ 一致 |
| 卡片主标题 | 16sp | `FONT.SIZE.LG = 16` | ✅ 有对应档 |
| 卡片副标题 | 12sp | `FONT.SIZE.SM = 12` | ✅ 有对应档 |
| 卡片内最大字号 | 24 | `XXL = 24`；更大的 `XXXL 30` / `HERO 36` 只用于页面级标题 | ✅ 限界内 |

**已登记的有意偏差（跨端统一，非缺陷）**

1. **普通卡片圆角 20vp**（`SHAPE.CARD = RADIUS.LG`）而官方通用 16vp —— 对齐前端 `rounded-lg = 20`
   跨端一致优先；紧凑卡/输入框仍用 16 与官方一致。
2. **正文字号 14**（`FONT.SIZE.MD`）小于官方正文量级 16 —— 对齐前端 `text-md = 14`；
   展示型标题仍走 20/24/30/36 档。

新增页面/组件必须从现有档位取值，不得引入近邻档（13/15/18/22/28 已归并过一轮）。
