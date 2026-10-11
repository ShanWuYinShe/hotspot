---
name: hotspot-harmony-ui
description: hotspot 鸿蒙端 UI/UX 与组件原语契约：设计 token 强制规则、跨端同步机制、SystemPageHeader/PageShellContainer、路由对齐、SystemButton/SystemInput 原语与已登记例外清单。触发：修改 harmonyapp 任意 UI 或样式；新增鸿蒙页面；使用设计 token 或颜色字号间距；手写 Button/TextInput 前确认是否该用原语；跨端色彩/刻度同步。
---

# hotspot-harmony-ui

## UI/UX 设计规范

设计 token 的**全部数值档位**（颜色/字号/间距/圆角/边框/动效/交互反馈/透明度/尺寸/阴影/模糊）、双端共享组件映射表与跨端投影契约速查的唯一权威是 [docs/design-system.md](../../../docs/design-system.md)（仓库根 docs/），改 UI 前必读。本节只保留强制规则与指针。

- **禁止硬编码**：颜色一律使用 `AppThemeManager.getPaletteById(resolvedThemeId)` 的 palette token（或 `DesignConstants` 系统/图片固定色），禁止硬编码色值；字号/间距/圆角/边框/动效/透明度/尺寸一律用 `DesignConstants` 对应 token 档位，禁止裸 `fontSize(N)` / `padding({left:N})` / `width(N)` / `duration: N` / `scale(...)`；`$r('app.color.*')` 资源仅用于启动窗口等系统场景（当前 color.json 只保留 start_window_background）。
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
- 通用单行输入框必须用 `SystemInput`（前缀图标/清除按钮/错误态）；登录/表单等
  密码框可用 `SystemInput({ passwordMode: true })`。
- 已登记例外（可保留手写原生控件）：头像/图形按钮、地图全屏控件、搜索建议行、
  动态卡片内嵌操作、主题色板 swatch、扫描/贴纸等特殊控件。存量页面中的手写
  Button/TextInput 属遗留债务，改到该文件时优先迁移为原语。当前存量例外清单：
  - ProfilePage 外部主页/升级/编辑流程按钮（走 `getProfileVisualStyle()` 主题样式对象，
    已 token 化，不套 SystemButton）；
  - 创建页「移除」等 48vp 大 chip、群组详情「恢复原头像」ghost 小按钮；
  - SettingsPage「退出登录」（50vp 透明底定制）、DebugPanel 调试按钮；
  - ChatPage 隐藏键盘失焦占位 `Button('')`、LocationPicker 搜索栏内置 searchButton；
  - ProfileEditPage 多行简介输入辅助、ChatPage 输入栏（含发送钮 `Button('发送')` 属已登记例外 — 需 ChatTheme/getChatVisualStyle 定制动效与 isSendInteractive 动态态，见 ChatPage.ets 顶部“聊天发送钮属已登记例外”注释）、CreateGroupPage 人数步进输入。
