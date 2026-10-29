---
name: hotspot-harmony-ui
description: hotspot 鸿蒙端 UI/UX 与组件原语契约：设计 token 强制规则、跨端同步机制、SystemPageHeader/PageShellContainer/ListGroup、路由对齐、SystemButton/SystemInput 原语与已登记例外清单、焦点环/减少动态、动效与媒体、回顶与聊天滚动。触发：修改 harmonyapp 任意 UI 或样式；新增鸿蒙页面；使用设计 token 或颜色字号间距；手写 Button/TextInput 前确认是否该用原语；跨端色彩/刻度同步；鸿蒙端 a11y/触控验收。
---

# hotspot-harmony-ui

> 设计 token 的**全部数值档位**（颜色/字号/间距/圆角/边框/动效/交互反馈/透明度/尺寸/阴影/模糊）、双端共享组件映射表与跨端投影契约速查的唯一权威是 [docs/design-system.md](../../../docs/design-system.md)（仓库根 docs/）。本文件只保留强制规则、原语契约与鸿蒙端专项口径；web 端对应规则见 skill `hotspot-frontend-style` / `hotspot-frontend-layout` / `hotspot-frontend-theme`。

## UI/UX 设计规范（强制）

- **禁止硬编码**：颜色一律使用 `AppThemeManager.getPaletteById(resolvedThemeId)` 的 palette token（或 `DesignConstants` 系统/图片固定色），禁止硬编码色值；字号/间距/圆角/边框/动效/透明度/尺寸一律用 `DesignConstants` 对应 token 档位，禁止裸 `fontSize(N)` / `padding({left:N})` / `width(N)` / `duration: N` / `scale(...)`；`$r('app.color.*')` 资源仅用于启动窗口等系统场景（当前 color.json 只保留 start_window_background）。
- **图标尺寸用专用刻度**：一律 `COMPONENT.ICON_SIZE_*`（`2XS 14 / XS 16 / SM 20 / SM_PLUS 22 / MD 24 / LG 32 / EMPTY 48`），**不要用 `FONT.SIZE.*` 当图标尺寸**——那是字号刻度，混用会让图标尺寸随字号变动、并产生 9 种图标尺寸（实测：`iconSize` 有 133 处取自 FONT.SIZE、仅 17 处用专用刻度）。唯一例外（明示约定）：图标**与同档文字行内并排**时（如聊天气泡的「已编辑」行、群卡 meta 行的图标），允许 `iconSize: FONT.SIZE.X` 与该文字同档——判断口径：该处 ±14 行内存在 `fontSize(...FONT.SIZE.同档)`。已登记未迁移：`LG_PLUS(18)` 11 处（行首图标）、`HERO(36)` 2 处（hero/地图图标）——专用刻度无 18/36，取近档会改像素，待设计定档。
- **行高（多行正文强制用比例制）**：`FONT.LINE_RATIO.{DEFAULT 1.5, CARD 1.7, COPY 1.75}` + `lineHeightFor(fontSize, ratio)`，对齐前端 `--leading-*`（`card` 是前端「原 1.55，阅读疲劳修复」后的值）。适用：聊天气泡/卡片正文 `CARD`；简介、公告、动态正文等长文 `COPY`。`FONT.LINE_HEIGHT` 的绝对档（XS18/SM20/MD22/LG24/XL26）**只用于单行紧凑场景**——它与字号解耦，把 `LINE_HEIGHT.SM`(20) 配 `SIZE.LG`(16) 只有 1.25 倍。实测：多处误用导致比值散落 1.57~1.83，且聊天气泡/群公告**完全没有行高**（长文本挤成一团）。
- **跨端同步机制（强制）**：前端与鸿蒙共享同一套设计 token 刻度，同步由 CI 脚本强制拦截，**改两端任一 token 必须同步改另一端**：

  | 校验 | 脚本 | 覆盖范围 |
  |---|---|---|
  | 跨端色彩 | `frontend/scripts/check-cross-platform-colors.mjs` | `themes/*.css --cst-palette-*` ↔ `ThemeConstants.ets ColorConfig` 逐值一致（3 主题 × 2 明暗 × 14 组映射）+ 主题 swatch |
  | 跨端刻度 | `frontend/scripts/check-cross-platform-scale.mjs` | 字号 9 档（`--text-*` ↔ `FONT.SIZE`）+ 圆角 5 档（`--radius-*` ↔ `RADIUS`）逐值一致；单端加近邻档即拦截。已登记豁免：`FONT.SIZE.LG=16`（HarmonyOS 平台适配档） |

  两脚本均挂入 `bun run check:theme`（随 `bun run check` 与 CI 生效），漂移即红。
- **视觉结构同步约定**：前端 `components/home|messages|profile/` 下组件与鸿蒙 `components/`、`features/*/` 对应实现构成 design-system.md 第 13 节映射表——修改任一端的**共享组件**（视觉契约：布局/字号档/圆角档/投影/动效）时，必须检查映射表对端并同步，commit message 中注明「对齐 <前端组件/commit>」；仅平台适配差异（如 FAB 位置、安全区）可不同步但需在两端代码注释中互指。
- **页面头部（强制 SystemPageHeader）**：所有页面头部必须用 `SystemPageHeader`（或专用组件 `SearchPageHeader` / `ChatPageHeader`）；禁止手搓 `Row { Text(...).fontSize(XL)... Button(...) }` 头部。已登记例外（无标准页面头部）：登录页 `LoginPage`（无主壳）、主壳 `Index`（容器）、地图全屏 `LocationPickerPage`、图片全屏预览 `PhotoPreviewPage`、分享解析中间页 `ShareResolutionPage`（无内容 UI，仅 loading/错误/跳转）、调试面板 `DebugPanel`；`SettingsSupport.ets` 等仅导出页面子组件的文件不算页面。API：`title` / `subtitle` / `showBackButton`+`onBack` / `action`(文本按钮) / `leftAction`+`rightAction`(IconAction，`kind: 'primary'|'surface'|'neutral'`) / `trailing`(@BuilderParam 任意尾随) / `titleAlign` / `titleSize` / `titleWeight` / `backgroundVariant`('bar'|'transparent'|'card')。**页头图标钮跨端口径**：品牌头（首页）= ghost（`kind:'neutral'`，前端 `variant="ghost"`）；标准内页 PageHeader trailing = surface（`kind:'surface'`，前端默认 `variant="surface"`）。web 移动端首页只有「搜索 + 通知」两枚，地图入口在附近域页头；鸿蒙同步移除首页地图钮。安全区由 SystemPageHeader 内聚（自带 `@StorageLink('safeAreaTop')` + `expandSafeArea`），页面无需重复写。内联搜索框头部 → `SearchPageHeader`；浮动卡片+头像头部 → `ChatPageHeader`。
- **页面内容栏（PageShellContainer）**：强制使用；档位数值（form/narrow/default/wide + feed 两档）见 design-system.md 第 12 节与本文「PageShellContainer 档位选择」。
- **页面横向 gutter（唯一来源）**：横向留白**只能**由 `PageShellContainer` 提供（手机 16 / 平板 24 / 桌面 28，对齐前端 `--cst-page-gutter`；手机档由 20 收紧至 16，对齐 iOS inset grouped 16pt / Material 16dp 主流下限，两端同步）。**禁止**双重留白（容器 gutter + 页面自加 left/right padding，会叠成 36~40，与只用容器 gutter 的页面差 16~20vp）；**禁止**用 `noHorizontalPadding: true` 关掉容器留白再另写一套值。`SystemPageHeader` 只带纵向节奏、不自带横向 padding。
  - **通栏元素的坑**：前端用 `-mx-[var(--cst-page-gutter)]` 抵消 gutter，ArkUI **不能照搬**——子元素 `width('100%')` 相对父内容盒解析，负 margin 只位移不加宽（群详情封面加左右 −20vp 后实测仍是 x=70..1186 / 屏宽 1256）。当前约定：封面/hero 与下方内容同宽；确需通栏要把整页改 `noHorizontalPadding: true` + 各区块自加 gutter（已登记例外）。
  - **症状对照**：各页正文左边缘不在同一条竖线上、滑动时内容左右边界跳动、页头标题比正文多缩进一截——都是本规则被违反的表现（实测 25 个页面里同时存在 16/20/24/32 四套留白）。
- **页面纵向节奏（强制）**：分区之间 `PAGE_SHELL.SECTION_GAP`(24) 用**底距**且**每个分区都要有（含首块）**；分区标题→其内容 `PAGE_SHELL.SECTION_INNER_GAP`(12)；页头→首块 `PAGE_SHELL.PAGE_TOP_GAP`(16)。24 : 12 = 2:1 是必需的层次比：分区之间不明显大于标题内距，标题就会飘在两块之间、整页读不出分组。用「上距」或漏给首块加底距，会让标题上方间距随各卡片自身 padding 变化，同类标题间距不一致（实测首页三个分区底距曾混用 20/20/24，且首块统计卡完全没有底距）。
- **页头字号档统一（强制）**：全站 `SystemPageHeader.titleSize` 统一 `FONT.SIZE.XXL`(24)——前端各页实测均为 `desktop ? 'xl'(20) : '2xl'(24)`，**无 3xl 用法**。禁止顶级 Tab 页用 XXXL(30)、二级页用 XXL(24) 的分档（同机切换页面标题大小会跳动；违规者实测首页/附近曾用 XXXL）。
- **列表分组容器（ListGroup，强制）**：任何「可重复的同类条目集合」（设置项、关系列表、搜索结果、群组/分享/订阅/账号列表、公告列表）一律走 `components/ListGroup`（`bordered` 默认 true = 开放页面区域的唯一一层分组外框卡）+ 行间 `ListRowDivider` / `ListItemGroup.divider`，**禁止给每一行加边框/圆角/投影**（那就是「一行一张浮卡」，整屏看起来所有内容都在漂浮）。`bordered=false` 用于嵌入面板/Sheet/侧栏内的集合。
  - 判定依据（官方规范原文）：M3 Cards guidelines「Don't force content into cards when spacing, headlines, or dividers would create a simpler visual hierarchy」；M3 Lists「Use gaps for contained lists / Limit dividers to uncontained lists」；Apple HIG Layout「negative space, container shapes, or separator lines」三选一、HIG Materials「Don't use Liquid Glass in the content layer」；HarmonyOS `ListItemGroupStyle` 默认就是 NONE（无卡片），卡片是显式开关；NN/g「homogenous items → 标准竖排列表，不要卡片」。
  - **卡片只留给「单一主题、内容异构、可独立存在」的块**：动态卡（MomentFeedCard / MomentWallCard）、群组封面卡（NearbyImageCard）、统计/仪表盘块、空态卡、页面级 hero 卡。
  - 分组外框静态投影一律 `surfaceShadow`（白卡档），**禁止 `elevatedShadow`**（弹层/FAB 档）；行本身无投影、无边框、无圆角，只有按压填充（`surfaceSubtle`）。设置页专用分组卡 `SettingsSectionCard` 已按同一契约实现，新设置分组直接复用。
  - **横向留白按列表形态二分**（踩过的坑）：容器已提供 gutter 后——**扁平列表**（无卡片外框，行直接铺在页面底色上，如消息会话列表）行不得有 `left`/`right` padding，列表也不得有横向 `margin`；否则行内容比页头多缩进一个 gutter（实测消息页曾多 16~36vp，「消息」标题 x=73 而「最近」标签 x=109），行间分隔线用 `startMargin: 0`。**分组卡片**（`ListGroup`/`ListItemGroup` 带底色+描边+圆角，如群列表/订阅/账号管理）卡片去掉横向外边距（与外层内容同宽）、行内 padding 保留（那是卡片内边距）；卡片自带 `margin LG` 会让卡片比页头窄 32vp、卡内内容窄 64vp。页面级元素（退出登录按钮、搜索框、升级表单）同样不得自加横向 padding（实测个人中心升级表单自带 `XXL(32)` 叠加容器 gutter 后达 52vp）；分组标题（`SettingsGroupLabel`）也不得自加 `left`，否则比分组卡多缩进（实测 91 → 70）。

## 页面路由对齐（web → harmony）

目录命名对齐前端路由域（复数）、底栏 4 tab / 桌面侧边栏 IA、消息列表排序、聊天记录正序与日期胶囊、附近聚合页与搜索页 Tab、WS 事件类型等对齐事实见 `references/routing-map.md`。

## 组件原语强制（对齐前端 button/input 体系）

- 通用文本操作按钮必须用 `SystemButton`（variant=primary/secondary/outline/ghost/danger/surface/surface-accent/surface-danger，size=compact/small/default/large/sheet，支持 loading/disabled/fullWidth）；禁止手写 `Button('文案') + backgroundColor/fontColor/height/borderRadius` 重复样式。
- **危险操作分两档，别混用**：`danger`（**实底 error 色 + 白字**）只用于**最终确认**（二次确认弹窗的 `SystemDialog.confirmVariant`、弹层里的确认键，如「封禁并移出」）；`surface-danger`（染色底 + error 文字 + 发丝描边，安静档）用于**行级/列表内**的危险操作（账号行删除、消息/聊天操作表、退出群组、取消分享…）。反例：账号管理页每行一个实底 `danger`，眯眼测试下整页形成一列刺眼红色块，视线被破坏性操作抢走、观感像警报页（NN/g：最强颜色档必须留给真正的警示）。
- 通用单行输入框必须用 `SystemInput`（前缀图标/清除按钮/错误态）；登录/表单等密码框可用 `SystemInput({ passwordMode: true })`。
- **搜索输入（2026-10-29 跨端对齐）**：搜索页头部 `SearchPageHeader` 与前端 `forms/SearchInput` 同口径——胶囊（`SHAPE.BUTTON`）+ `SEARCH_INPUT_HEIGHT 39` + `FONT.SIZE.MD 14` + `ICON_SIZE_XS 16` + `SPACING.MD2 10`；静态边框用 `palette.borderColor`（18% alpha，对齐前端 `border-default`，勿用 8% 的 `floatingBorderColor`）、**无投影**（对齐前端 `input-surface` 与 design-system §8）；聚焦用 `palette.accentBackground` 1.5 边框（**不要用 `accentText`**——它是 accent 实底上的文字色，浅色主题为白，作焦点边会不可见）；`SystemInput` 保持 42/16 圆角不变（表单与搜索两派语言），其聚焦边框同用 `accentBackground`（见 `resolveInputBorder`）。
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
  | `components/layout/AmbientBackdrop` | 氛围画布（三主题世界层，装饰层唯一实现）；单 Canvas 静态一次性绘制、`hitTestBehavior(None)`、`accessibilityLevel('no')`；统一挂载 `PageShellContainer`（`fillHeight=true`）内，页面禁止自建第二套背景装饰。token 为 `ThemeConstants.AMBIENT`（6 组合 × 12 色，对齐 web `--cst-ambient-*`，见 design-system 第 16 节） |
  | `components/states/EmptySpotlight` | 场景化空态装饰层（超大空态专用，配合 `EmptyState({ spotlight: true })`）；几何 320 光晕 + 288/168 双环 + 4×4 四向星点@±102，纯 ambient token、无动画、不拦截触摸/不进读屏。`EmptyState` 的 `spotlight` 默认 false，小面板空态不得开启 |
  | `components/states/Skeleton` / `SkeletonShimmer` | 列表/信息流首屏骨架屏（几何基准：行内 12、前导圆 36/40/44/48、文本条 14·55% + 11·35%、条间距 8、圆角 12；脉冲 1500ms）；分页/表单等短促场景保留 `LoadingMoreIndicator`/`LoadingProgress` |
  | `components/media/MediaOverlay` | 整卡图片卡覆盖层原语：`CardScrimTop`(96vp/.60→透明)、`CardScrimBottom`(58%/.88→.60@42%→透明@78%)、`CardFallbackBackground`（主题渐变+≤.10 水印）、`MediaOverlayBadge`（深色强遮罩胶囊+白字）、`MediaActionButton`（白图标/计数、热区 44）。色值只取 palette 的 `cardScrim*`/`mediaScrimStrong`；禁止页面写 rgba |
  | `components/TabsPillGroup` | Tab 原语。**选中态不得只依赖颜色**：`segmented` 选中 = accent 实底 + ✓(`AppIcon.CHECK`) + semibold；`pills` 选中 = semibold（accent-soft 底）；`underline` 选中 = semibold + 下划线。需过灰度截图验收 |

- 已登记例外（可保留手写原生控件）：头像/图形按钮、地图全屏控件、搜索建议行、动态卡片内嵌操作、主题色板 swatch、扫描/贴纸等特殊控件。存量页面手写 Button/TextInput 属遗留债务，改到该文件时优先迁移为原语。当前存量例外清单：ProfilePage 外部主页/升级/编辑流程按钮（走 `getProfileVisualStyle()` 主题样式对象，已 token 化，不套 SystemButton）；创建页「移除」等 48vp 大 chip、群组详情「恢复原头像」ghost 小按钮；SettingsPage「退出登录」（50vp 透明底定制）、DebugPanel 调试按钮；ChatPage 隐藏键盘失焦占位 `Button('')`、LocationPicker 搜索栏内置 searchButton；ProfileEditPage 多行简介输入辅助、ChatPage 输入栏（含发送钮 `Button('发送')`，需 ChatTheme/getChatVisualStyle 定制动效与 isSendInteractive 动态态，见 ChatPage.ets 顶部「聊天发送钮属已登记例外」注释）、CreateGroupPage 人数步进输入。
- **ArkUI 陷阱：`.border()` 写在 `.borderRadius()` 之后会把圆角重置为 0**（双环/圆钮会变方框）——装饰层双环、圆点等一律先 `.border()` 后 `.borderRadius()`；改完端上放大截图确认（实拍案例：ProfilePage 升级表单「取消」直角 vs「确认升级」胶囊）。

## 内容软岛与整卡图片卡（两端同 spec，鸿蒙实现）

- **内容软岛**：`ListGroup`/`SystemCard` = `surfaceIsland` + `surfaceShadow` + `SHAPE.CARD`(20) + padding 8、行透明 + 行间距 2、行内 hover 填充离岛边 8；**禁止 1px hairline 全框**；`SettingsSectionCard` 同。`surfaceIsland` 与 web `--cst-surface-island` 逐值一致（light = `withAlpha(surfaceBackground,0.86)`；dark = `mixOpaque(surfaceBackground,0.94,textPrimary,0.06)`；6 套组合，实测 6/6，记录见 `docs/frontend-ui-audit.md`）。改 token 后按两端代码逐值比对（web `themes/*.css` 的 `--cst-surface-island` ↔ 鸿蒙 `AppThemeManager` 的 `surfaceIsland`，仓库无独立入库校验脚本）。按钮安静化同 web：次级动作 `withAlpha(surfaceSubtle,0.60)` 软填底、无描边、hover 全量；主操作 accent 实底；危险 soft danger；同级动作不混用描边/软填。页面级硬线清零：`SystemPageHeader`/自定义页头/分区头不得有 1px 底线（功能线：聊天输入条/粘底条/表格行/消息分栏发丝线）。页面结构与 web 端同一套规则，见 skill `hotspot-frontend-layout`「内容软岛与长尾/后台页面」。
- **整卡图片卡**（用户明确要求「图片占满整个卡片、信息覆盖在图上」；与 `docs/design-system.md` §17 同源，视觉细则见 skill `hotspot-frontend-style`「媒体与图片」）：适用 `MomentFeedCard variant='overlay'`（首页 feed，4:5）、`NearbyImageCard mediaOverlay`（网格 3:4，含附近三叶/我的瞬间/我的群组/群组精选）；`horizontal` 横排两段式保留为列表例外。卡片无 border、圆角 `SHAPE.CARD`(20)；图片 object-cover 铺满；顶部作者行/徽章 + 底部标题/元信息/白图标操作行压在图；位置只在右上出现一次；多图首图 + 1/N 徽章；无图 fallback = 主题渐变 + ≤.10 水印 + 主题色文字（不压白字）。压图文字对比度 ≥4.5:1（字形区像素实测，亮暗都测）；徽章深色强遮罩 + 白字；点赞/收藏状态用「填充形状 + 白描边」不只色相；操作热区 ≥44。骨架屏同步：`Skeleton` 的 feed 形态按整卡形状（图片块 + 底部文本条），骨架→内容零位移。

## 主题、A11y 与触控验收

- **不要用「底色明度差」判断卡片层次**：前端 `semantic.css` 已明文规定 `--cst-surface-page` 是唯一整层底色；页面级容器不画框后，层级只靠**内容单元软岛**（`surfaceIsland` 表面 + `surfaceShadow` + 圆角 20，无 hairline 描边，如 `ListGroup`/`SystemCard`/设置分组卡/通知列表岛）与留白表达。因此清野主题下 `pageBackground == surfaceBackground`（浅色深色都是 `#F6F7F3` / `#161A16`）**是刻意设计，不是缺陷**，不要为了「拉开层次」去改底色（会破坏跨端色彩校验）。深色下的层级靠**加亮的描边**承载（`ThemeConstants.ets` 注释：深色下黑阴影对比弱，卡片抬升改用更亮的描边，阴影只做轻微加深）；`borderColor` 是 `#AARRGGBB`，验收要算**与底色合成后**的颜色再求对比度（不能只看 alpha）。量化基线（文字对比度、描边合成比、6 组合主题矩阵、forest light 软岛覆写、harmony-only `textTertiary`）见 `references/theme-baselines.md`。
- **无障碍名称**：`dumpLayout` 不暴露 accessibilityText（description 恒空），验收采用「源码级 onClick↔accessibilityText 覆盖扫描 + 运行期结构」；新增可点控件必须显式 `accessibilityText`（纯图标钮/复合卡/中央发布钮等）。
- **触控**：有效热区 ≥44vp（`responseRegion` 外扩、视觉不变）；chip/输入框/地图 marker 等登记例外；边缘 tap 实测（视觉边界外仍命中）验证。原语层已强制：`MIN_HIT_TARGET = 44`（`DesignConstants.LAYOUT`）、`SurfaceIconButton` 外框取 `max(buttonSize, MIN_HIT_TARGET)`、`.responseRegion(...)`（首页 tab 行、消息行、瞬间卡标签、发布页等，行高压到 20vp 也能外扩到 44 不占布局）。**实测剩余缺口（backlog）**：区块标题右侧「查看全部 / 探索群组 / 更多动态」是 `Row.onClick`（文字 ~17vp 高），有效热区低于 44；修法需结构调整（`responseRegion` 在百分比宽度行上无法简单外扩），优先级低（次级跳转、无相邻可点元素、误触代价小）。审计方法：`uitest dumpLayout` 后按 `clickable=true` 过滤节点取 `bounds`，与 `44vp × 屏幕缩放（本机 3.5x ≈ 154px）` 对比，排除整行可点（宽 >900px）。
- **真机限制**：TalkBack / 系统 reduced-motion 开启路径 / 焦点环像素需真机；离线时标注未验证并保留复现步骤。

## 焦点环与减少动态

- **键盘焦点环**：口径 `focusBox`，**2vp accent 描边 + 2vp 外距**（颜色取 `palette.accentBackground`），对齐 web `ring-accent 2px offset 2`；不新造颜色档。实现 `core/accessibility/FocusRing.ets` 的 `FocusRing.box(accentHex)`；`SystemButton` / `SurfaceIconButton` / `Pressable` 已默认接入（`.focusable(true)` + `.focusOnTouch(false)` + `.focusBox(...)`）。输入：`SystemInput` / `SystemTextArea` 聚焦用 **2vp accent 边框**（与 focusBox 等价的可见焦点；避免被输入容器 `.clip(true)` 裁切）。新增可聚焦自定义卡片/行：`.focusable(true).focusOnTouch(false).focusBox(FocusRing.box(getPalette().accentBackground))`；父容器需有 ≥2vp padding（ListGroup 默认 padding 8），否则焦点环会被 `clip(true)` 裁切。验证：编译 + 布局树/代码参数；真机键盘 Tab 截图（真机离线时报告标注待真机）。
- **减少动态 reduced-motion**：数据源 `AccessibilityManager.getInstance().shouldReduceMotion()`（系统「减少动态/移除动画」或应用内关闭动画）。实现 `core/accessibility/MotionUtil.ets`：`spring(response, damping)` reduced 时 `duration=0`；`pressScale(pressed, scale)` reduced 时恒为 1（不缩放）；`shimmerActive()` reduced 时 false（骨架不脉冲）。已接入 `SystemButton` / `SurfaceIconButton` / `Pressable` 按压动画、`SkeletonShimmer` 脉冲。要求：reduced 下禁用位移/缩放/脉冲，透明度淡入可保留；默认（false）行为必须与既有完全一致；新增动画一律走 `MotionUtil`，禁止裸写动画参数。

## 动效与媒体

- **动效唯一入口 `MotionUtil`**：`contentEnter`（180ms ease-out）、`statePop`（220ms，1→1.15→1，keyframeAnimateTo 两段）、`popScale`、`enterOffset`；`reduced` 时全部直接终态（duration 0 / scale 1 / offset 0）。禁止裸写 animateTo 参数或旧 150ms/1.3 手写 pop。内容进场：loading true→false / tab 切换 / 刷新时播，**冷启动首屏跳过、缓存命中不播**；只包内容层，不含页头/滚动容器/系统转场；无布局位移。状态 pop：点赞/收藏/关注/加入仅激活方向播，取消不播。验收：代码级检查 + reduced 分支证据（duration 0/直接终态）+ build/check:theme/parity/布局树；真机手感/中间帧像素待补。
- **图片加载与 URL 优化**：`core/utils/ImageUrlUtil.ets` 与 web `lib/images.ts` 同规则——仅 `images.unsplash.com` 追加/覆盖 `auto=format&fit=crop&q=78&w=<width>`，其他域名原样。宽度档按显示尺寸 2x/2.5x（首屏大图 1080、卡片 900/800、头像 2x）。ArkUI 无 `fetchpriority/loading`：首屏前 2 张高清档 + 立即创建，`LazyForEach` 离屏不创建（平台差异已登记）。加载体验：占位（surfaceSubtle/accentMuted，不新增 shimmer）+ `onComplete` 渐显 200ms ease-out；`MotionUtil.reduced` 直接显示；失败 fallback 保持。新图片组件必须按此接入，禁止裸用远程原图。真机待核：首屏流量、头像 2x vs 3x、渐显像素、reduced 系统路径。
- **媒体占位与降级**：`SystemAvatar` 占位层**始终渲染**（有 name 首字母 trim+slice(0,1)、无 name 人形图标；accentMuted 底 + textSecondary 字），Image 覆盖占位 + `onComplete` 渐显（`MotionUtil.reduced` 直接显示）；`onError` 保留占位不闪烁/不分支切换；decorative/alt 语义与尺寸不变。群封面/通知/搜索/rail/主页瞬间/详情/群 hero 等缩略图：底层占位（渐变/图标/首字母）+ 渐显 + 失败保留；有 Skeleton 的列表不叠加；不新增 token。新图片接入必须同时给「占位 + 失败降级 + 宽度档/URL 优化」；真机占位/渐显/失败像素待补。

## 回顶、页头滚动与聊天滚动

- **页头滚动玻璃**：鸿蒙 TopBar 均在滚动容器外（`grep '.sticky('`=0），**N/A**；如未来引入 sticky 页头再按 web 口径实现并去掉常驻底。
- **回顶**：`components/BackToTopButton.ets`（48vp、ARROW_UP、surface+hairline+shadow、opacity 180ms via MotionUtil/reduced 直接显隐、不可见禁触+无障碍树移除、label「回到顶部」）；已接 Home/Nearby/Notification/Search/MomentsFeed 5 页（阈值 1.5×视口高、scrollTo 300ms/reduced 0ms、右下避底栏/居中 FAB）；GroupPage/Profile/MyShares 待接。SPA 焦点/TalkBack 播报由系统处理（N/A 登记）；320/reflow 为 web-only（N/A）。
- **聊天滚动与回底**：私聊/群聊共用 ChatPage：ListScroller + `onScrollIndex`（+列表头偏移）判定末条可见=近底；ArkUI List 无距底像素 API，平台差异已登记。规则：仅近底自动 `scrollToBottom` + `markAsRead`；上滑时不跳转、计数++、显示 `ChatJumpToBottomButton`；自己发送/初始/切换/分页保持滚底/视口；回底按钮 48vp、count>0 徽标（99+）、150ms via MotionUtil/reduced 直接、不可见禁触+移出 a11y、label 与计数一致；旧 78% 胶囊已删除。新增聊天/列表页时按此契约实现；真机上滑不跳/计数/按钮位置待补验。

## PageShellContainer 档位选择

宽度已统一收口在 `PageShellContainer`（禁止页面各自 `maxWidth`），但**档位要按内容类型选**，选错会在平板/桌面只占中间一条：

| 档位 | 宽度 | 适用 |
|---|---|---|
| `form` | 560 | 表单/创建/设置单项（登录、发布瞬间、建群、编辑资料、设置各页）|
| `narrow` | 720 | 阅读型详情（瞬间详情）|
| `feed-narrow` | 640 | 单条内容页 |
| `feed-medium` | 960 | 内容墙/流（附近瞬间列表）|
| `default` | 1180 | 信息流/列表/枢纽/仪表盘（首页、消息、附近、群组、搜索、管理后台概览、订阅、公告、我的分享…）|
| `wide` | 1320 | 大表格/管理后台宽页 |

`SearchPage`、`AdminOverviewPage`、`MySharesPage`、`SubscriptionsPage`、`AnnouncementsPage`、`RelationshipHubPage` 曾从 `form` 改为 `default`（它们不是表单）。

## 长尾/后台页面软岛映射

- 对应 web 长尾软岛化的鸿蒙落点：`MomentDetailPage` 单帖内容包软岛（surfaceIsland/20/shadow/clip），媒体大图圆角 24+shadow、九宫格 16+shadow，反馈容器/粘底操作条按功能线例外保留，加载态用骨架（媒体块+作者行+正文两行）；他人 `ProfilePage` 身份区软岛（20/p16/shadow），私信/已关注/备注按钮 surface 软填、关注 primary；`GroupDetailPage` hero 圆角 24+shadow+clip、Tab 内容一张软岛、行线保留（内容分隔例外）；`AdminOverviewPage` 权限卡/模块行软岛，其余后台页 web-only（N/A 登记）；首页 onboarding/HeroStats/轮播/附近人列表鸿蒙无对应组件（N/A，不强行造页面）。
- 例外（保留功能边界）：媒体容器/九宫格圆角与阴影、输入框 2vp accent 焦点边、表格行线/内容分隔线、chip/badge、弹层/Sheet、地图控件、hero 封面裁切结构。验证：build + check:theme + 布局树；真机离线时阴影/焦点环/reduced-motion 开启路径标注待真机。

## 审计基线与布局陷阱（指针）

- 审计判据与假阳性规律（「固定高度」正确判据、交互反馈承载方式、性能审计基线：LazyForEach 键/cachedCount/@Reusable/Image 约束/事件生命周期）见 `references/audit-baselines.md`。
- ArkUI 布局陷阱（`.border()`/`.borderRadius()` 顺序、Scroll 短内容居中、Refresh 防抖动契约、裸 `{ }` 块不是容器）见 `references/layout-pitfalls.md`。
- web → harmony 页面路由对齐、底栏/侧栏 IA、消息列表排序与聊天记录、搜索页 Tab 等路由事实见 `references/routing-map.md`。
- 设计 token 对照 HarmonyOS 官方规范与已登记有意偏差见 `references/theme-baselines.md`。

## references

- `references/theme-baselines.md`：文字/描边对比度量化基线、6 组合主题矩阵、forest light 软岛、官方规范对照与有意偏差。
- `references/audit-baselines.md`：固定高度/交互反馈/性能审计的正确判据与脚本假阳性规律。
- `references/layout-pitfalls.md`：ArkUI 渲染与滚动陷阱、Refresh 防抖动契约、裸块陷阱。
- `references/routing-map.md`：web → harmony 路由与信息架构对齐。
