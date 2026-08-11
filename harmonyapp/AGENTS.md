# HotSpot HarmonyOS App

## 功能边界

HarmonyOS ArkTS 客户端。覆盖认证(含多账户切换)/发现与搜索/信息流/生活瞬间与评论/媒体上传/聊天(私聊+群聊)/群组/通知与公告/举报/地图(原生组件+位置选择器)/附近/个人中心。管理后台功能仅 Web 端提供,本端不含。支付与交易、端侧离线同步不在范围内。

订阅管理:契约待定,当前 spec 未定义订阅相关 API,待后续版本补充。

API 契约见 `spec/api_spec_*.md`（仓库根 `spec/` 目录，只读权威）。

## 构建 & 测试

DevEco Studio 工具路径（node/hvigor/ohpm 以本机 `/Applications/DevEco-Studio.app/Contents/tools/` 下实际目录为准，使用前 `ls` 验证；JAVA_HOME 见全局「本机环境」Android Studio JBR 条目，变动时用 `mdfind` 定位）：

```
NODE=/Applications/DevEco-Studio.app/Contents/tools/node/bin/node
HVIGOR=/Applications/DevEco-Studio.app/Contents/tools/hvigor/bin/hvigorw.js
OHPM=/Applications/DevEco-Studio.app/Contents/tools/ohpm/bin/ohpm
DEVECO_SDK_HOME=/Applications/DevEco-Studio.app/Contents/sdk
JAVA_HOME=/Applications/Android Studio.app/Contents/jbr/Contents/Home
PATH=$JAVA_HOME/bin:$PATH
```

- `$OHPM install --all --registry https://ohpm.openharmony.cn/ohpm/ --strict_ssl true` — 安装依赖
- `$NODE $HVIGOR --mode project -p product=default assembleApp --analyze=normal --parallel --incremental --daemon` — 编译整个应用
- `$NODE $HVIGOR assembleHap` — 仅编译 HAP 包（快速检查用）
- `$NODE $HVIGOR clean` — 清理构建产物

如果修正 `DEVECO_SDK_HOME` / `JAVA_HOME` 后仍复用旧环境，先运行 `$NODE $HVIGOR --stop-daemon`，再重新构建。

Lint & Format: DevEco Studio 内置，无独立 CLI。代码检查由 IDE 在编辑时自动执行。

## 技术栈（名称级）

- HarmonyOS ArkTS (.ets)、ArkUI 原生组件、hvigor 构建工具
- `@ohos.net.http`（网络）、`@ohos.data.preferences`（持久化）

## 目录与分层（必须）

- `entry/src/main/ets/entryability/`：应用入口 Ability
- `entry/src/main/ets/components/`：公共 ArkUI 组件
- `entry/src/main/ets/constants/`：常量定义（AppConstants、DesignConstants、SystemConstants）
- `entry/src/main/ets/core/`：核心逻辑层（按用途分子目录，不逐一举出）
  - `api/ApiClient.ets`：按业务域封装的 API 调用静态方法
  - `config/ApiEndpoints.ets`：API 路径常量（唯一路径定义处）
  - `network/HttpUtil.ets`：统一 HTTP 客户端（POST、Token 刷新、幂等键）
  - `network/WebSocketService.ets`：WebSocket 单例
  - `state/`：全局状态管理（GlobalStateManager、AuthAccountManager、PersistentStateManager）
  - `types/`：API 类型定义（对齐 spec）、WebSocket 事件类型
- `entry/src/main/ets/features/`：按业务域划分的页面（每域一个目录，新增页面按域归类，禁止在 core/ 里放页面）

## API 调用规范

- 禁止硬编码路径：必须通过 `core/config/ApiEndpoints.ets`
- 统一入口：通过 `core/network/HttpUtil.ets` 的 `post()` 方法（`HttpUtil.post()` / `ApiClient`）发起请求
- 成功判定：仅使用 `code === 0`，禁止依赖 HTTP 状态码或 `message` 字段
- Token 自动携带：从 `PreferencesUtil` 读取，收到 `code=10100`（认证失败）时自动 refresh 并重试
- 幂等键：MUST 幂等接口以 `spec/api_spec_base.md`「必须使用幂等键的接口（MUST）」列表为准；实现中 `MUST_IDEMPOTENT_ENDPOINTS` 须与该列表同步（增删端点时需更新），自动注入 `Idempotency-Key`
- Cursor 不透明：请求时透传 `cursor`，收到响应后将 `next_cursor` 作为下一页的 `cursor` 透传，不解析、不构造
- RPC + JSON：仅 POST，不实现 REST 风格

## Token 与安全

- Token 通过 `PreferencesUtil` 存取
- 使用 `Authorization: Bearer` 方式携带
- 禁止解析 JWT，登录态按业务接口返回值判断
- 错误码 `10100`（认证失败）须尝试刷新 Token 或跳转登录
- 错误码 `10201`（账号封禁）须强制登出并展示封禁页

## 状态管理

- AppStorage + PersistentStorage：HarmonyOS 原生状态管理
- GlobalStateManager：单例，管理全局状态（currentUser、isLoggedIn、unreadCount 等）
- AuthAccountManager：多账户支持，持久化 accounts 和 current_user_id
- UI 绑定使用 `@StorageLink` 等装饰器
- 持久化键使用有意义的名称（currentUser、isLoggedIn 等）

## WebSocket 规范

- 事件名与 payload 结构以 `spec/api_spec_websocket.md` 为准
- `WebSocketService` 单例管理连接，支持鉴权、心跳、自动重连
- 事件类型定义在 `core/types/WebSocketTypes.ets`，必须与 spec 一致
- 可定义镜像类型，不得扩展 spec 中未定义的事件

## 位置与距离

- 位置请求使用 `Location`（point + location_name），响应使用 `RelativeLocation`
- 坐标转换仅在 `CoordinateConverter.ets` 单一边界层执行一次

## 数据格式（契约级）

- 所有时间字段为 UTC ISO 8601，客户端负责时区转换
- 所有 `xxx_id` 字段按 `string` 处理，不做数值运算

## 图标规范

- 统一使用 `SymbolSpan` / `SymbolGlyph`，优先使用 `sys.symbol.*` 符号字体
- 所有图标名称在 `components/SystemIcon.ets` 的 `AppIcon` 枚举中集中管理
- 禁止在业务代码中直接写 `$r('sys.symbol.xxx')`，必须通过 `AppIcon` 枚举引用
- 新增图标前须在 HarmonyOS 主题图标库中确认符号名存在

## 组件与样式

- 颜色一律使用 `AppThemeManager.getPaletteById(resolvedThemeId)` 的 palette token（或
  `DesignConstants` 系统/图片固定色），禁止硬编码色值；`$r('app.color.*')` 资源
  仅用于启动窗口等系统场景（当前 color.json 只保留 start_window_background）
- 字体大小、间距等使用 `DesignConstants` 常量
- 核心服务使用单例模式（`GlobalStateManager.getInstance()`）

## 组件原语强制（对齐前端 button/input 体系）

- 通用文本操作按钮必须用 `SystemButton`（variant=primary/secondary/outline/ghost/danger，
  size=small/default/large，支持 loading/disabled/fullWidth）；禁止手写
  `Button('文案') + backgroundColor/fontColor/height/borderRadius` 重复样式。
- 通用单行输入框必须用 `SystemInput`（前缀图标/清除按钮/错误态）；登录/表单等
  密码框可用 `SystemInput({ passwordMode: true })`。
- 已登记例外（可保留手写原生控件）：头像/图形按钮、地图全屏控件、搜索建议行、
  动态卡片内嵌操作、主题色板 swatch、扫描/贴纸等特殊控件。存量页面中的手写
  Button/TextInput 属遗留债务，改到该文件时优先迁移为原语。

## UI/UX 设计规范

以下为内联设计规范（原 `.claude/rules/design-tokens.md` 已并入本文件），改 UI 前必读，避免硬编码色值/字号/间距、避免手搓页面头部。

### 设计系统（design-tokens）

所有颜色、字号、间距、圆角、阴影只用 DesignConstants / AppThemeManager token，对齐前端 design token 系统（vibrant / forest 双主题，单一事实源）。禁止在页面/组件里硬编码十六进制色值或裸数字 `fontSize(N)` / `padding({left:N})`。

颜色：
- 画布：`palette.pageBackground` / `palette.surfaceBackground`（对齐前端
  `--cst-surface-page` / `--cst-surface-primary`）
- 表面：`palette.surfaceSubtle` / `palette.surfaceElevated` / `palette.chatBackground`
- 文字：`palette.textPrimary` / `palette.textSecondary` / `palette.textTertiary`
- 品牌：`palette.accentBackground`（forest `#3D7A55`，与前端
  `--cst-pr-accent-primary`（`--cst-accent-default`）一致）
- 语义：`palette.success` / `palette.warning` / `palette.error` / `palette.info`
- 地图标记：`palette.mapUser / mapGroup / mapMoment`（对齐前端 `--cst-marker-user/group/moment`，随主题/明暗变）；状态淡底：`palette.dangerSoft / warningSoft`（对齐前端 `--cst-status-danger/warning-bg` 12~14% 预混档位；success/info 淡底用 `getSoftStatusBackground('success' | 'info')` 派生）
- 主题解析：`AppThemeManager.getPaletteById(resolvedThemeId)` → `@StorageProp(KEY_RESOLVED_THEME_ID)`
- 明暗差异：本端明暗始终跟随系统（`isDarkMode`），与 Web 端可手动切换
  `isDark` 不同；主题族（vibrant/forest/figma_neu）双端可选集一致

字号（FONT.SIZE，禁用裸 `fontSize(N)`）：`XS_MINUS 9` / `XS2 10` / `XS 11` / `SM 12` / `SM_PLUS 13` / `MD 14` / `MD_PLUS 15` / `LG 16` / `LG_PLUS 18` / `XL 20` / `XL_PLUS 22` / `XXL 24` / `XXL_PLUS 28` / `XXXL 32` / `XXXL_PLUS 36` / `HERO 48` / `HERO_LARGE 56`。字重 `FONT.WEIGHT.{REGULAR,MEDIUM,SEMIBOLD,BOLD}`。行高 `FONT.LINE_HEIGHT.{XS 18, SM 20, MD 22, LG 24, XL 26}`；字距 `FONT.LETTER_SPACING.{EYEBROW 1.4, LABEL 1.2, FIELD 1.28, ICON_LABEL 0.6}`（对齐前端 tracking-eyebrow/label/field 刻度）。

间距（SPACING，4px 网格 + 微步）：`HALF 2` / `THREE 3` / `XS 4` / `SM2 6` / `SM 8` / `MD2 10` / `MD 12` / `MD3 14` / `LG 16` / `LG2 20` / `XL 24` / `XXL 32`。禁止组件内 `padding({left:16})` 这类裸数——用 `SPACING.LG`。1vp 级微距（top:1 等）与 0 值允许保留（对齐前端间距不强制收敛原则）。

边框宽度（BORDER_WIDTH，对齐前端 `--cst-border-width-*`）：`HAIRLINE 1` / `REGULAR 1.5` /
`STRONG 2` / `HEAVY 3`。禁止 `.border({ width: 1 })` 裸数，一律 `DesignConstants.BORDER_WIDTH.*`。

圆角（RADIUS / SHAPE）：`SM 8` / `MD 12` / `LG 16` / `XL 18` / `XXL 22` / `FULL 999`（HarmonyOS vp 适配档，非前端逐像素复制——前端为 12/16/20/24 四档收敛 + sheet 28）。语义形状 `SHAPE.{CARD 22, HERO_CARD 24, COMPACT_CARD 18, PREVIEW_CARD 22, LIST_ITEM 22, ICON_TILE 12, INPUT 18, BUTTON 28, CHIP FULL, TAB_BAR 24, SHEET_TOP_RADIUS 20}`，按前端语义用法对齐（hero/feature 24、主按钮 28、胶囊 chip FULL）。

动效（ANIMATION，禁用裸 `duration: N`）：`DURATION_SHORT 200` / `DURATION_MEDIUM 300` / `DURATION_CONTENT_SWITCH 400` / `DURATION_SHEET 320`（底部弹层统一）/ `DURATION_250` / `DURATION_280` / `DURATION_FAST 150` / `DURATION_INSTANT 100` / `DURATION_SLOW 1500`（脉冲/shimmer）/ `DURATION_STAGGER_DELAY 150`；曲线 `CURVE_EASE_IN_OUT` / `CURVE_EASE_OUT` / `CURVE_FAST_OUT_SLOW_IN` / `CURVE_SMOOTH`。Toast/震动等行为时长保留字面量并注释（非视觉 token 范畴）。

交互反馈（INTERACTION，禁用裸 `scale(...)` / 裸 curve）：按压强度用
`SCALE_PRESSED_*`（对齐前端 `tap-*`/`hover-btn` 契约，数值为 HarmonyOS 平台适配档）：
- 主按钮按压 0.95 ↔ 前端 `hover-btn` tap 0.94；行内图标 0.90 ↔ `scale-btn` tap 0.94
- 卡片 0.97 ↔ `tap-97`；瀑布流卡 0.96 ↔ `tap-96`（前端按页微调档同理）
- 输入行 0.98 ↔ `tap-98`；胶囊/筛选 0.94 ↔ `tap-94`；返回 0.88 ↔ `tap-88`
- hover 抬升用 `HOVER_ELEVATION`（ArkUI elevation，对应前端 fine-pointer hover 缩放），
  弹簧动效用 `SPRING_RESPONSE_*` / `SPRING_DAMPING_*`（对应前端 `--cst-ease-bounce`）

玻璃效果（毛玻璃背景）：用系统 `BlurStyle` 枚举（平台原生，非数值 token），
按模糊强度对应前端 `--cst-blur-*` 三档：
- `BlurStyle.Thin` ↔ 前端 `--cst-blur-sm`（8px，轻徽章/浮层）
- `BlurStyle.Regular` ↔ 前端 `--cst-blur-md`（16px，sticky header/overlay）
- `BlurStyle.Medium`/`Thick` ↔ 前端 `--cst-blur-lg`（24px，底部导航/tablet dock）

双端共享组件映射（新增/修改组件时先查此表，避免两端各自发明同类组件）：

| 前端（Svelte） | 鸿蒙（ArkTS） | 说明 |
|---|---|---|
| `layout/page/PageHeader` | `SystemPageHeader` / `SearchPageHeader` / `ChatPageHeader` | 页面头部（含 below/搜索/聊天空态） |
| `StandardPageLayout` / `SplitPageLayout` / `ContentColumn` | `PageShellContainer` | 页面内容栏档位（form/narrow/default/wide/feed） |
| `MainLayout`（desktop/mobile/tablet） | `layout/DesktopLayout` / `PhoneLayout` / `ResponsiveContainer` | 设备外壳分派 |
| `EmptyState` | `EmptyState` | 空态（同名同契约） |
| `shared/Avatar` | `SystemAvatar` | 头像（尺寸/形状/状态点） |
| `forms/SearchInput` | `SearchPageHeader` / `SmartSearchComponent` | 搜索输入 |
| `navigation/TabBar` / `NavTab` / `RailIcon` | `TabsPillGroup` / `PhoneLayout` 底部导航 | 分类/导航 Tab |
| `list/ListItem` / `ListContainer` | `SystemCard`（listItem 变体）/ `SettingsEntryRow` | 列表行（行级卡片禁止规则同前端） |
| `overlays/SheetShell` / `ui/dialog` | `SystemDialog` / `bindSheet` | 弹层/抽屉 |
| `forms/TagPill` | `TabsPillGroup` chip / `FILTER_CHIP_HEIGHT` | 标签/筛选胶囊 |
| `data-display/EmptyState` 加载态 | `SystemLoadingView` / `Skeleton` | 加载/骨架 |
| `data-display/MetricRow` / `HeroStats` | `StatSegments` | 统计段 |
| `routes/moments/shared/MomentCard` | `MomentFeedCard` | 动态卡片 |
| `auth/LoginPromptSheet` | `LoginPromptView` | 未登录引导 |
| `messages/ChatView` 等 | `ChatPage` / `ChatPageHeader` | 聊天界面 |
| `routes/(app)/nearby/+page.svelte`（附近聚合） | `features/nearby/pages/NearbyPage` | 附近聚合页（搜索 + 人/群组/动态三区块），手机/平板底栏「附近」tab |

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
- 附近聚合页 `NearbyPage` 三区块「查看全部」分别 push `NEARBY_USERS` /
  `NEARBY_GROUPS` / `NEARBY_MOMENTS`（MomentsFeedPage 已注册为路由页，push 场景
  带返回按钮，对齐前端 `/nearby/people|groups|moments`）。
- 搜索页 Tab：全部 / 用户 / 动态 / 话题（话题从动态标签聚合，对齐前端搜索
  全部/用户/动态/话题）；群组 Tab 为鸿蒙端扩展（前端搜索无群组 Tab）。

透明度（OPACITY）：`DISABLED 0.38` / `PRESSED 0.72` / `LOADING 0.58` / `SUBTLE 0.85` / `FADED 0.4` / `MUTED 0.5` / `EMPHASIZED 0.8`（对齐前端 opacity-40/50/80 标准档）。

尺寸（COMPONENT / LAYOUT，禁止裸 `width(N)` / `height(N)`）：
- 头像 `AVATAR_SIZE.{XS 32, SMALL 40, SMALL_PLUS 44, MEDIUM 48, LARGE 80, XLARGE 96}`；按钮高度 `BUTTON_HEIGHT 42` / `BUTTON_HEIGHT_LG 48` / `ACTION_BUTTON_HEIGHT 28` / `ACTION_BUTTON_HEIGHT_MD 40` / `ACTION_BUTTON_HEIGHT_LG 34` / `SHEET_ACTION_HEIGHT 44` / `INLINE_BUTTON_HEIGHT 38` / `FILTER_CHIP_HEIGHT 30` / `LOGOUT_BUTTON_HEIGHT 50` / `PASSWORD_INPUT_HEIGHT 46`
- 图标 `ICON_SIZE_XS 16` / `ICON_SIZE_SM 20` / `ICON_SIZE_MD 24` / `STATUS_ICON_SIZE 28` / `LOADING_ICON_SIZE 20` / `LOADING_GLYPH_SIZE 46` / `BRAND_ICON_SIZE 38`；容器 `EMPTY_ICON_SIZE 64` / `LOADING_CONTAINER_SIZE 72` / `BANNER_HEIGHT 140` / `SKELETON_CARD_HEIGHT 100` / `SKELETON_AVATAR_SIZE 66` / `PLACEHOLDER_AREA_HEIGHT 120` / `SHARE_CARD_HEIGHT 130` / `LOG_AREA_HEIGHT 300` / `PROGRESS_STROKE_WIDTH 4`
- 布局 `LAYOUT.{DESKTOP_HEADER_HEIGHT 52, DESKTOP_SIDEBAR_WIDTH 224, RIGHT_SIDEBAR_WIDTH 220, MOBILE_BOTTOM_NAV_HEIGHT 56, TABLET_NAV_MAX_WIDTH 520, FAB_SIZE 52, ICON_BUTTON_SIZE 36, LABEL_WIDTH 60, HOME_COLUMN_WIDTH 420, FILTER_PANEL_WIDTH 180, GLOW_BLOB_LARGE 180 / MEDIUM 160 / SMALL 120}`
- 聊天 `CHAT.{BUBBLE_PADDING_H 16, BUBBLE_PADDING_V 10, IMAGE_BUBBLE_PADDING_H 14, IMAGE_BUBBLE_PADDING_V 6, IMAGE_MAX_WIDTH 200, ...}`

阴影：扁平设计默认 `SHADOW.*` 全零；需要投影时用主题注册表 `AppThemeManager.getPaletteById(...).{surfaceShadow, elevatedShadow, accentShadow, controlShadow}`（真值在 `ThemeConstants` 的 ThemeShadowRegistry），禁止组件内 `.shadow({ radius: N, ... })` 手写。

页面头部（强制 SystemPageHeader）：
- 所有页面头部必须用 `SystemPageHeader`（或专用组件 `SearchPageHeader` / `ChatPageHeader`）。
- 禁止手搓 `Row { Text(...).fontSize(XL)... Button(...) }` 头部。
- 已登记例外（无标准页面头部）：登录页 `LoginPage`（无主壳）、主壳 `Index`（容器）、
  地图全屏 `LocationPickerPage`、图片全屏预览 `PhotoPreviewPage`、分享解析中间页
  `ShareResolutionPage`（无内容 UI，仅 loading/错误/跳转）、调试面板 `DebugPanel`；
  `SettingsSupport.ets` 等仅导出页面子组件的文件不算页面。
- SystemPageHeader API：`title` / `subtitle` / `showBackButton`+`onBack` / `action`(文本按钮) / `leftAction`+`rightAction`(IconAction) / `trailing`(@BuilderParam 任意尾随) / `titleAlign` / `titleSize` / `titleWeight` / `backgroundVariant`('bar'|'transparent'|'card')。
- 安全区由 SystemPageHeader 内聚（自带 `@StorageLink('safeAreaTop')` + `expandSafeArea`），页面无需重复写。
- 内联搜索框头部 → `SearchPageHeader`；浮动卡片+头像头部 → `ChatPageHeader`。

页面内容栏（PageShellContainer）：档位（对齐前端 `.page-shell`）：`form 560` / `narrow 720` / `default 1180` / `wide 1320`；feed：`feed-narrow 640` / `feed-medium 960`。
